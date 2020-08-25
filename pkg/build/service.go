package build

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/zmb3/spotify"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
)

// Service manages building the actual spotify playlists
type Service struct {
	store store.Store
	auth  spotify.Authenticator
	log   *zap.SugaredLogger
}

type trackFetcher func(client *spotify.Client, tracks []spotify.ID, trackSource store.TrackSource) ([]spotify.ID, error)

var trackFetchers map[store.ExtractMethod]map[store.TrackSourceType]trackFetcher

func init() {
	// Prebuild map of functions to fetch tracks
	trackFetchers = map[store.ExtractMethod]map[store.TrackSourceType]trackFetcher{
		store.Latest: map[store.TrackSourceType]trackFetcher{
			store.AlbumSrc:    getTopAlbumTracks,
			store.LikedSrc:    getTopLikedTracks,
			store.PlaylistSrc: getTopPlaylistTracks,
		},
		store.Randomly: map[store.TrackSourceType]trackFetcher{
			store.AlbumSrc:    getRandomAlbumTracks,
			store.LikedSrc:    getRandomLikedTracks,
			store.PlaylistSrc: getRandomPlaylistTracks,
		},
	}
}

// New returns a pointer to a new BuildService
func New(store store.Store, auth spotify.Authenticator, log *zap.SugaredLogger) *Service {
	return &Service{
		store: store,
		auth:  auth,
		log:   log,
	}
}

// BuildPlaylist uses the configuration from playlistID to build a spotify playlist for userID
func (s *Service) BuildPlaylist(userID, playlistID uuid.UUID) {
	// Tell DB that playlist is currently being built
	err := s.store.UpdatePlaylistStartBuild(playlistID)
	if err != nil {
		// This shouldn't go wrong but if it does we want to just return b/c db was not changed
		s.log.Errorw("failed to update playlist into building state", "err", err.Error())
		return
	}

	// Get playlist configuration
	playlist, err := s.store.GetPlaylist(playlistID)
	if err != nil {
		s.logBuildError(userID, playlistID, err)
		return
	}

	// Build and validate output
	output := store.Output{
		Name:        playlist.Name,
		Description: playlist.Description,
		Public:      playlist.Public,
	}

	// Build spotify client
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		s.logBuildError(userID, playlistID, err)
		return
	}
	token := oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenType:    user.TokenType,
		Expiry:       user.TokenExpiry,
	}
	client := s.auth.NewClient(&token)

	// Unfollow a possibly pre-existing spotify playlist
	if playlist.SpotifyID != nil {
		err = client.UnfollowPlaylist(spotify.ID(user.SpotifyID), spotify.ID(*playlist.SpotifyID))
		if err != nil {
			s.logBuildError(userID, playlistID, err)
			return
		}
	}

	// Build the playlist
	spotifyPlaylistID, err := buildPlaylist(&client, user.SpotifyID, playlist.Input, output)
	if err != nil {
		s.logBuildError(userID, playlistID, err)
		return
	}

	// Update database for successful case
	err = s.store.UpdatePlaylistGoodBuild(playlistID, string(*spotifyPlaylistID))
	if err != nil {
		s.logBuildError(userID, playlistID, err)
		return
	}

	err = s.store.IncrementUserBuildCount(userID)
	if err != nil {
		// This really shouldn't go wrong but if it does all we can do is log it
		s.log.Errorw("failed to increment build count", "err", err.Error(), "userID", userID)
	}
}

// DeletePlaylist deletes both the actual spotify playlist and the configuration in the db
func (s *Service) DeletePlaylist(userID, playlistID uuid.UUID) {
	// Get playlist configuration
	playlist, err := s.store.GetPlaylist(playlistID)
	if err != nil {
		s.logDeleteError(userID, playlistID, err)
		return
	}

	// Build spotify client
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		s.logDeleteError(userID, playlistID, err)
		return
	}
	token := oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenType:    user.TokenType,
		Expiry:       user.TokenExpiry,
	}
	client := s.auth.NewClient(&token)

	// Unfollow spotify playlist
	if playlist.SpotifyID != nil {
		err = client.UnfollowPlaylist(spotify.ID(user.SpotifyID), spotify.ID(*playlist.SpotifyID))
		if err != nil {
			s.logDeleteError(userID, playlistID, err)
			return
		}
	}

	// Delete playlist configuration
	err = s.store.DeletePlaylist(playlistID)
	if err != nil {
		s.logDeleteError(userID, playlistID, err)
		return
	}
}

func (s *Service) logBuildError(userID, playlistID uuid.UUID, errIn error) {
	s.log.Errorw("failure while building playlist", "err", errIn.Error())
	err := s.store.UpdatePlaylistBadBuild(playlistID, errIn.Error())
	if err != nil {
		// This really shouldn't happen, but all we can do is log it
		s.log.Errorw("failed to update playlist config to failure state", "err", err.Error())
	}

	err = s.store.IncrementUserBuildCount(userID)
	if err != nil {
		// This really shouldn't happen, but all we can do is log it
		s.log.Errorw("failed to increment build count", "err", err.Error())
	}
}

func (s *Service) logDeleteError(userID, playlistID uuid.UUID, errIn error) {
	s.log.Errorw("failure while deleting playlist", "err", errIn.Error())
	err := s.store.UpdatePlaylistBadDelete(playlistID, errIn.Error())
	if err != nil {
		s.log.Errorw("failed to update playlist config to failure state", "err", err.Error())
	}
}

func buildPlaylist(client *spotify.Client, userID string, input store.Input, output store.Output) (*spotify.ID, error) {
	var tracks []spotify.ID
	var err error

	for _, trackSource := range input.TrackSources {
		tracks, err = trackFetchers[trackSource.Method][trackSource.Type](client, tracks, trackSource)
		if err != nil {
			return nil, err
		}
	}

	// Build spotify playlist
	playlist, err := client.CreatePlaylistForUser(userID, output.Name, output.Description, output.Public)
	if err != nil {
		return nil, err
	}

	// Add tracks to spotify playlist
	playlistID, err := addTracksToPlaylist(client, playlist.ID, tracks)
	if err != nil {
		// TODO add clean up logic here to unfollow playlist?
		return nil, err
	}
	return playlistID, nil
}

func addTracksToPlaylist(client *spotify.Client, playlistID spotify.ID, tracks []spotify.ID) (*spotify.ID, error) {
	start := 0
	stop := 0
	for {
		if stop >= len(tracks) {
			break
		}
		start = stop
		if (start + 100) > len(tracks) {
			stop = len(tracks)
		} else {
			stop = start + 100
		}
		_, err := client.AddTracksToPlaylist(playlistID, tracks[start:stop]...)
		if err != nil {
			return nil, err
		}
	}
	return &playlistID, nil
}

func getTopAlbumTracks(client *spotify.Client, tracks []spotify.ID, trackSource store.TrackSource) ([]spotify.ID, error) {
	// TODO implement
	return nil, errors.New("not implemented")
}

func getTopLikedTracks(client *spotify.Client, tracks []spotify.ID, trackSource store.TrackSource) ([]spotify.ID, error) {
	count := 0
	offset := 0
	var limit int

	for {
		if trackSource.Count-count <= 0 {
			break
		} else if trackSource.Count-count < 50 {
			limit = trackSource.Count - count
		} else {
			limit = 50
		}
		opts := spotify.Options{
			Limit:  &limit,
			Offset: &offset,
		}

		trackPage, err := client.CurrentUsersTracksOpt(&opts)
		if err != nil {
			return nil, err
		} else if len(trackPage.Tracks) != limit {
			// Not enough songs. Treat as error for now TODO don't treat as error
			return nil, fmt.Errorf("expected %d songs in Liked Songs but did not find that many", trackSource.Count)
		}

		count += limit
		offset += limit

		for _, track := range trackPage.Tracks {
			tracks = append(tracks, track.ID)
		}
	}
	return tracks, nil
}

func getTopPlaylistTracks(client *spotify.Client, tracks []spotify.ID, trackSource store.TrackSource) ([]spotify.ID, error) {
	count := 0
	offset := 0
	var limit int

	for {
		if trackSource.Count-count <= 0 {
			break
		} else if trackSource.Count-count < 50 {
			limit = trackSource.Count - count
		} else {
			limit = 50
		}
		opts := spotify.Options{
			Limit:  &limit,
			Offset: &offset,
		}

		trackPage, err := client.GetPlaylistTracksOpt(trackSource.ID, &opts, "items(track(id))") // TODO fields are wrong here
		if err != nil {
			return nil, err
		} else if len(trackPage.Tracks) != limit {
			// Not enough songs. Treat as error for now TODO don't treat as error
			return nil, fmt.Errorf("expected %d songs in playlist but did not find that many", trackSource.Count)
		}

		count += limit
		offset += limit

		for _, track := range trackPage.Tracks {
			tracks = append(tracks, track.Track.ID)
		}
	}
	return tracks, nil
}

func getRandomAlbumTracks(client *spotify.Client, tracks []spotify.ID, trackSource store.TrackSource) ([]spotify.ID, error) {
	// TODO implement
	return nil, errors.New("not implemented")
}

func getRandomLikedTracks(client *spotify.Client, tracks []spotify.ID, trackSource store.TrackSource) ([]spotify.ID, error) {
	// TODO implement
	return nil, errors.New("not implemented")
}

func getRandomPlaylistTracks(client *spotify.Client, tracks []spotify.ID, trackSource store.TrackSource) ([]spotify.ID, error) {
	// TODO implement
	return nil, errors.New("not implemented")
}

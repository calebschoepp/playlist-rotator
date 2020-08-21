package build

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
)

// Service manages building the actual spotify playlists
type Service struct {
	store store.Store
	auth  spotify.Authenticator
}

type trackFetcher func(client *spotify.Client, tracks []spotify.ID, plInput store.PlaylistInput) ([]spotify.ID, error)

var trackFetchers map[store.ExtractMethod]map[bool]trackFetcher

func init() {
	trackFetchers = map[store.ExtractMethod]map[bool]trackFetcher{
		store.Top: map[bool]trackFetcher{
			false: getTopPlaylistTracks,
			true:  getTopSavedTracks,
		},
		store.Random: map[bool]trackFetcher{
			false: getRandomPlaylistTracks,
			true:  getRandomSavedTracks,
		},
	}
}

// New returns a pointer to a new BuildService
func New(store store.Store, auth spotify.Authenticator) *Service {
	return &Service{
		store: store,
		auth:  auth,
	}
}

// BuildPlaylist uses the configuration from playlistID to build a spotify playlist for userID
func (s *Service) BuildPlaylist(userID, playlistID uuid.UUID) {
	// Get playlist configuration
	playlist, err := s.store.GetPlaylist(playlistID)
	if err != nil {
		s.logBuildError(userID, playlistID, err)
		// TODO handle error (jump to function to write failure info to db)
		fmt.Printf("ERROR 1: %v", err)
		return
	}

	// Build and validate input
	var input store.Input
	err = json.Unmarshal([]byte(playlist.Input), &input)
	if err != nil {
		s.logBuildError(userID, playlistID, err)
		// TODO handle error
		fmt.Printf("ERROR 2: %v", err)
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
		// TODO handle error
		fmt.Printf("ERROR 3: %v", err)
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
			// TODO handle this case
			// Do nothing for now
			fmt.Printf("Failed to unfollow playlist: %v", err)
		}
	}

	// Build the playlist
	spotifyPlaylistID, err := buildPlaylist(&client, user.SpotifyID, input, output)
	if err != nil {
		s.logBuildError(userID, playlistID, err)
		// TODO handle error
		fmt.Printf("ERROR 4: %v", err)
		return
	}

	// Update database for successful case
	err = s.store.UpdatePlaylistGoodBuild(playlistID, string(*spotifyPlaylistID))
	if err != nil {
		s.logBuildError(userID, playlistID, err)
		// TODO handle error
		fmt.Printf("ERROR 5: %v", err)
		return
	}

	err = s.store.IncrementUserBuildCount(userID)
	if err != nil {
		// TODO handle error
		// What do I do here, call out to metrics?
		fmt.Printf("SOMETHING HAS GONE SUPER EXTRA WRONG")
	}
}

func (s *Service) logBuildError(userID, playlistID uuid.UUID, err error) {
	err = s.store.UpdatePlaylistBadBuild(playlistID, err.Error())
	if err != nil {
		fmt.Printf("SOMETHING HAS GONE VERY WRONG: %v", err)
	}

	err = s.store.IncrementUserBuildCount(userID)
	if err != nil {
		// TODO handle error
		// What do I do here, call out to metrics?
		fmt.Printf("SOMETHING HAS GONE SUPER EXTRA WRONG")
	}
}

func buildPlaylist(client *spotify.Client, userID string, input store.Input, output store.Output) (*spotify.ID, error) {
	var tracks []spotify.ID
	var err error

	for _, plInput := range input.PlaylistInputs {
		tracks, err = trackFetchers[plInput.Method][plInput.IsSaved](client, tracks, plInput)
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
		// TODO confirm I don't need to keep using snapshot ID here
		_, err := client.AddTracksToPlaylist(playlistID, tracks[start:stop]...)
		if err != nil {
			return nil, err
		}
	}
	return &playlistID, nil
}

func getTopPlaylistTracks(client *spotify.Client, tracks []spotify.ID, plInput store.PlaylistInput) ([]spotify.ID, error) {
	count := 0
	offset := 0
	var limit int

	for {
		if plInput.Count-count <= 0 {
			break
		} else if plInput.Count-count < 50 {
			limit = plInput.Count - count
		} else {
			limit = 50
		}
		opts := spotify.Options{
			Limit:  &limit,
			Offset: &offset,
		}

		trackPage, err := client.GetPlaylistTracksOpt(plInput.PlaylistID, &opts, "items(track(id)))")
		if err != nil {
			return nil, err
		} else if len(trackPage.Tracks) != limit {
			// Not enough songs. Treat as error for now TODO don't treat as error
			return nil, fmt.Errorf("expected %d songs in playlist but did not find that many", plInput.Count)
		}

		count += limit
		offset += limit

		for _, track := range trackPage.Tracks {
			tracks = append(tracks, track.Track.ID)
		}
	}
	return tracks, nil
}

func getTopSavedTracks(client *spotify.Client, tracks []spotify.ID, plInput store.PlaylistInput) ([]spotify.ID, error) {
	count := 0
	offset := 0
	var limit int

	for {
		if plInput.Count-count <= 0 {
			break
		} else if plInput.Count-count < 50 {
			limit = plInput.Count - count
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
			return nil, fmt.Errorf("expected %d songs in Liked Songs but did not find that many", plInput.Count)
		}

		count += limit
		offset += limit

		for _, track := range trackPage.Tracks {
			tracks = append(tracks, track.ID)
		}
	}
	return tracks, nil
}

func getRandomPlaylistTracks(client *spotify.Client, tracks []spotify.ID, plInput store.PlaylistInput) ([]spotify.ID, error) {
	return nil, errors.New("not implemented")
}

func getRandomSavedTracks(client *spotify.Client, tracks []spotify.ID, plInput store.PlaylistInput) ([]spotify.ID, error) {
	return nil, errors.New("not implemented")
}

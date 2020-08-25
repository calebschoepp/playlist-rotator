package server

import (
	"context"
	"math/rand"

	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"github.com/google/uuid"
)

type ctxKey int

const (
	userIDCtxKey ctxKey = iota
)

// TODO use crypto/rand?
func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func getUserID(ctx context.Context) *uuid.UUID {
	userID, ok := ctx.Value(userIDCtxKey).(*uuid.UUID)
	if !ok {
		return nil
	}
	return userID
}

// TODO IMPORTANT verify that I'm properly getting all of the potential sources
func getPotentialSources(s store.Store, auth *spotify.Authenticator, userID *uuid.UUID) ([]tmpl.PotentialSource, error) {
	// Build spotify client
	user, err := s.GetUserByID(*userID)
	if err != nil {
		return nil, err
	}
	token := oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenType:    user.TokenType,
		Expiry:       user.TokenExpiry,
	}
	client := auth.NewClient(&token)

	// Add liked songs
	pss := []tmpl.PotentialSource{}
	pss = append(pss, tmpl.PotentialSource{Name: "Liked Songs", ID: "", Type: store.LikedSrc})

	// Find and add playlists
	limit := 50
	playlists, err := client.CurrentUsersPlaylistsOpt(&spotify.Options{
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}
	// TODO check total to see if there is > 50 playlists, if so grab the rest
	for _, playlist := range playlists.Playlists {
		ps := tmpl.PotentialSource{
			Name: playlist.Name,
			ID:   string(playlist.ID),
			Type: store.PlaylistSrc,
		}
		pss = append(pss, ps)
	}

	// Find and add albums
	albums, err := client.CurrentUsersAlbumsOpt(&spotify.Options{
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}
	// TODO check total to see if there is > 50 albums, if so grab the rest
	for _, album := range albums.Albums {
		ps := tmpl.PotentialSource{
			Name: album.FullAlbum.SimpleAlbum.Name,
			ID:   string(album.FullAlbum.SimpleAlbum.ID),
			Type: store.AlbumSrc,
		}
		pss = append(pss, ps)
	}

	return pss, nil
}

package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/calebschoepp/playlist-rotator/pkg/motify"
	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
	"github.com/google/uuid"
	zs "github.com/zmb3/spotify"
)

type ctxKey int

const (
	userIDCtxKey ctxKey = iota
)

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
func generateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func getUserID(ctx context.Context) *uuid.UUID {
	userID, ok := ctx.Value(userIDCtxKey).(*uuid.UUID)
	if !ok {
		return nil
	}
	return userID
}

func getPotentialSources(s store.Store, spotify *motify.Spotify, userID *uuid.UUID) ([]tmpl.PotentialSource, error) {
	// Build spotify client
	user, err := s.GetUserByID(*userID)
	if err != nil {
		return nil, err
	}
	client := spotify.NewClient(&user.Token)

	// Add liked songs
	pss := []tmpl.PotentialSource{}
	pss = append(pss, tmpl.PotentialSource{Name: "Liked Songs", ID: "", Type: store.LikedSrc})

	// Find and add playlists
	limit := 50
	playlists, err := client.CurrentUsersPlaylistsOpt(&zs.Options{
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
	albums, err := client.CurrentUsersAlbumsOpt(&zs.Options{
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

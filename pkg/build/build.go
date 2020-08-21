package build

import "github.com/google/uuid"

// Builder provides methods for working with real Spotify playlists
type Builder interface {
	BuildPlaylist(userID, playlistID uuid.UUID)
}

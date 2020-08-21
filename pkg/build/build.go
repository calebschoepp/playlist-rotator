package build

import "github.com/google/uuid"

type Build struct {
}

type BuildServicer interface {
	BuildPlaylist(userID, playlistID uuid.UUID)
}

package playlist

import (
	"time"

	"github.com/google/uuid"
)

type Playlist struct {
	ID     uuid.UUID `db:"id"`
	UserID uuid.UUID `db:"user_id"`

	Input       string `db:"input"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Public      bool   `db:"public"`
	SpotifyID   string `db:"spotify_id"`
	FailureMsg  string `db:"failure_msg"`

	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	LastBuiltAt *time.Time `db:"last_built_at"`
}

type PlaylistServicer interface {
	CreatePlaylist(userID uuid.UUID, input INPUTODOREMOVE, name, description string, public bool) error // TODO should this return playlist?
	UpdatePlaylist(id uuid.UUID, name string, userID uuid.UUID) error
	GetPlaylist(id uuid.UUID) (*Playlist, error)
	GetPlaylists(userID uuid.UUID) ([]Playlist, error)
	UpdatePlaylistGoodBuild(id uuid.UUID, spotifyID string) error
	UpdatePlaylistBadBuild(id uuid.UUID, failureMsg string) error
}

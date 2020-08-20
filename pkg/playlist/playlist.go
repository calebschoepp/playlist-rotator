package playlist

import (
	"time"

	"github.com/google/uuid"
)

type Playlist struct {
	ID          uuid.UUID  `db:"id"`
	Name        string     `db:"name"`
	UserID      uuid.UUID  `db:"user_id"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	LastBuiltAt *time.Time `db:"last_built_at"`
}

type PlaylistServicer interface {
	CreatePlaylist(name string, userID uuid.UUID) error // TODO should this return playlist?
	UpdatePlaylist(id uuid.UUID, name string, userID uuid.UUID) error
	GetPlaylist(id uuid.UUID) (*Playlist, error)
	GetPlaylists(userID uuid.UUID) ([]Playlist, error)
}

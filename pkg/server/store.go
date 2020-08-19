package server

import (
	"time"

	"github.com/google/uuid"
)

// User TODO move this somewhere better
type User struct {
	ID             uuid.UUID `db:"id"`
	SpotifyID      string    `db:"spotify_id"`
	SessionToken   string    `db:"session_token"`
	SessionExpiry  time.Time `db:"session_expiry"`
	PlaylistsBuilt int       `db:"playlists_built"`
	AccessToken    string    `db:"access_token"`
	RefreshToken   string    `db:"refresh_token"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

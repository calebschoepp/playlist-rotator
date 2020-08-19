package user

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

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

type UserServicer interface {
	GetUserByID(id uuid.UUID) (*User, error)
	GetUserBySpotifyID(spotifyID string) (*User, error)
	UserExists(spotifyID string) (bool, error)
	GetSessionExpiry(sessionToken string) (*time.Time, error)
	CreateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error
	UpdateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error
}

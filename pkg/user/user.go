package user

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type User struct {
	ID             uuid.UUID `db:"id"`
	SpotifyID      string    `db:"spotify_id"`
	PlaylistsBuilt int       `db:"playlists_built"`

	SessionToken  string    `db:"session_token"`
	SessionExpiry time.Time `db:"session_expiry"`

	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	TokenType    string    `db:"token_type"`
	TokenExpiry  time.Time `db:"token_expiry"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type UserServicer interface {
	GetUserByID(id uuid.UUID) (*User, error)
	GetUserBySpotifyID(spotifyID string) (*User, error)
	GetUserID(sessionToken string) (*uuid.UUID, error)
	UserExists(spotifyID string) (bool, error)
	GetSessionExpiry(sessionToken string) (*time.Time, error)
	CreateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error // TODO should this return the user
	UpdateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error
}

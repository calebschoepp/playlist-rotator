package postgres

import (
	"database/sql"
	"errors"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/user"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
)

type UserService struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *UserService {
	return &UserService{
		db: db,
	}
}

func (u *UserService) GetUserByID(id uuid.UUID) (*user.User, error) {
	return nil, errors.New("not implemented")
}

func (u *UserService) GetUserBySpotifyID(spotifyID string) (*user.User, error) {
	return nil, errors.New("not implemented")
}

// UserExists determines if there is a user for the give spotifyID already
func (u *UserService) UserExists(spotifyID string) (bool, error) {
	var exists bool
	query := `
SELECT exists
	SELECT *
	FROM users
	WHERE spotify_id=$1;
`
	err := u.db.QueryRow(query, spotifyID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

func (u *UserService) GetSessionExpiry(sessionToken string) (*time.Time, error) {
	query := `
SELECT session_expiry
FROM users
WHERE session_token=$1
`
	var sessionExpiry time.Time
	err := u.db.Get(&sessionExpiry, query, sessionToken)
	if err != nil {
		return nil, err
	}
	return &sessionExpiry, nil
}

// CreateUser creates a new user row in the DB
func (u *UserService) CreateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error {
	query := `
INSERT INTO users (
	spotify_id,
	session_token,
	session_expiry,
	playlists_built,
	access_token,
	refresh_token
)
VALUES 
	($1, $2, $3, $4, $5, $6);
`
	_, err := u.db.Exec(
		query,
		spotifyID,
		sessionToken,
		sessionExpiry,
		0,
		token.AccessToken,
		token.RefreshToken,
	)
	if err != nil {
		return err
	}
	return nil
}

// UpdateUser updates an existing user with new sesion and oauth2 token data
func (u *UserService) UpdateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error {
	query := `
UPDATE users SET
	session_token=$1,
	session_expiry=$2,
	access_token=$3,
	refresh_token=$4
WHERE
	spotify_id=$5;
`
	_, err := u.db.Exec(
		query,
		sessionToken,
		sessionExpiry,
		token.AccessToken,
		token.RefreshToken,
		spotifyID,
	)
	if err != nil {
		return err
	}
	return nil
}

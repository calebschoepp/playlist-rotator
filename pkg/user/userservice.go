package user

// import (
// 	"database/sql"
// 	"errors"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/jmoiron/sqlx"
// 	"golang.org/x/oauth2"
// )

// type UserService struct {
// 	db *sqlx.DB
// }

// func New(db *sqlx.DB) *UserService {
// 	return &UserService{
// 		db: db,
// 	}
// }

// // GetUserByID returns a User matching the given id
// func (u *UserService) GetUserByID(id uuid.UUID) (*User, error) {
// 	var user User
// 	query := `
// SELECT *
// FROM users
// WHERE id=$1;
// `
// 	err := u.db.Get(&user, query, id)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &user, nil
// }

// func (u *UserService) GetUserBySpotifyID(spotifyID string) (*User, error) {
// 	return nil, errors.New("not implemented")
// }

// // GetUserID returns the UUID for a user based off of a session token
// func (u *UserService) GetUserID(sessionToken string) (*uuid.UUID, error) {
// 	var id uuid.UUID
// 	query := `
// SELECT id
// FROM users
// WHERE session_token=$1;
// `
// 	err := u.db.Get(&id, query, sessionToken)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &id, nil
// }

// // UserExists determines if there is a user for the give spotifyID already
// func (u *UserService) UserExists(spotifyID string) (bool, error) {
// 	var exists bool
// 	query := `
// SELECT exists (
// 	SELECT *
// 	FROM users
// 	WHERE spotify_id=$1
// );
// `
// 	err := u.db.QueryRow(query, spotifyID).Scan(&exists)
// 	if err != nil && err != sql.ErrNoRows {
// 		return false, err
// 	}
// 	return exists, nil
// }

// func (u *UserService) GetSessionExpiry(sessionToken string) (*time.Time, error) {
// 	query := `
// SELECT session_expiry
// FROM users
// WHERE session_token=$1;
// `
// 	var sessionExpiry time.Time
// 	err := u.db.Get(&sessionExpiry, query, sessionToken)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &sessionExpiry, nil
// }

// // CreateUser creates a new user row in the DB
// func (u *UserService) CreateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error {
// 	query := `
// INSERT INTO users (
// 	spotify_id,
// 	playlists_built,
// 	session_token,
// 	session_expiry,
// 	access_token,
// 	refresh_token,
// 	token_type,
// 	token_expiry
// )
// VALUES
// 	($1, $2, $3, $4, $5, $6, $7, $8);
// `
// 	_, err := u.db.Exec(
// 		query,
// 		spotifyID,
// 		0,
// 		sessionToken,
// 		sessionExpiry,
// 		token.AccessToken,
// 		token.RefreshToken,
// 		token.TokenType,
// 		token.Expiry,
// 	)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// // UpdateUser updates an existing user with new sesion and oauth2 token data
// func (u *UserService) UpdateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error {
// 	query := `
// UPDATE users SET
// 	session_token=$1,
// 	session_expiry=$2,
// 	access_token=$3,
// 	refresh_token=$4,
// 	token_type=$5,
// 	token_expiry=$6
// WHERE
// 	spotify_id=$7;
// `
// 	_, err := u.db.Exec(
// 		query,
// 		sessionToken,
// 		sessionExpiry,
// 		token.AccessToken,
// 		token.RefreshToken,
// 		token.TokenType,
// 		token.Expiry,
// 		spotifyID,
// 	)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// // IncrementUserBuildCount increments playlists_built by one for the given userID
// func (u *UserService) IncrementUserBuildCount(userID uuid.UUID) error {
// 	query := `
// UPDATE users SET
// 	playlists_built=playlists_built+1
// WHERE id=$1;
// `
// 	_, err := u.db.Exec(query, userID)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

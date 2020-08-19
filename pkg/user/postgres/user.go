package postgres

import (
	"errors"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/user"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
)

type UserService struct {
	DB *sqlx.DB
}

func New(db *sqlx.DB) *UserService {
	return &UserService{
		DB: db,
	}
}

func (s *UserService) GetUserByID(id uuid.UUID) (*user.User, error) {
	return nil, errors.New("not implemented")
}

func (s *UserService) GetUserBySpotifyID(spotifyID string) (*user.User, error) {
	return nil, errors.New("not implemented")
}

func (s *UserService) GetSessionExpiry(sessionToken string) (*time.Time, error) {
	return nil, errors.New("not implemented")
}

func (s *UserService) CreateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error {
	return errors.New("not implemented")
}

func (s *UserService) UpdateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error {
	return errors.New("not implemented")
}

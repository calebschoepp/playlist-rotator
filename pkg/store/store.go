package store

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// Store provides methods for getting data on users, playlists, and more
type Store interface {
	// Users
	GetUserByID(id uuid.UUID) (*User, error)
	GetUserBySpotifyID(spotifyID string) (*User, error)
	GetUserID(sessionToken string) (*uuid.UUID, error)
	UserExists(spotifyID string) (bool, error)
	GetSessionExpiry(sessionToken string) (*time.Time, error)
	CreateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error // TODO should this return the user
	UpdateUser(spotifyID, sessionToken string, sessionExpiry time.Time, token oauth2.Token) error
	IncrementUserBuildCount(userID uuid.UUID) error

	// Playlists
	CreatePlaylist(userID uuid.UUID, input Input, name, description string, public bool, schedule Schedule) error // TODO should this return playlist?
	UpdatePlaylistConfig(id uuid.UUID, playlist Playlist) error
	GetPlaylist(id uuid.UUID) (*Playlist, error)
	GetPlaylists(userID uuid.UUID) ([]Playlist, error)
	UpdatePlaylistGoodBuild(id uuid.UUID, spotifyID string) error
	UpdatePlaylistBadBuild(id uuid.UUID, failureMsg string) error
	UpdatePlaylistStartBuild(id uuid.UUID) error
}

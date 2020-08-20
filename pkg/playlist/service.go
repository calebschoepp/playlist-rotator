package playlist

import (
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PlaylistService struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *PlaylistService {
	return &PlaylistService{
		db: db,
	}
}

// CreatePlaylist inserts a new playlist into the DB
func (p *PlaylistService) CreatePlaylist(name string, userID uuid.UUID) error {
	query := `
INSERT INTO playlists (
	name,
	user_id
)
VALUES (
	$1,
	$2
);
`
	_, err := p.db.Exec(query, name, userID)
	if err != nil {
		return err
	}
	return nil
}

func (p *PlaylistService) UpdatePlaylist(id uuid.UUID, name string, userID uuid.UUID) error {
	return errors.New("not implemented")
}

func (p *PlaylistService) GetPlaylist(id uuid.UUID) (*Playlist, error) {
	return nil, errors.New("not implemented")
}

// GetPlaylists retrieves all the playlists associated with a given userID
func (p *PlaylistService) GetPlaylists(userID uuid.UUID) ([]Playlist, error) {
	playlists := []Playlist{}
	query := `
SELECT *
FROM playlists
WHERE user_id=$1;
`
	err := p.db.Select(&playlists, query, userID)
	if err != nil {
		return playlists, err
	}
	return playlists, nil
}

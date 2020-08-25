package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TODO split into two embedded interfaces and generally improve this type
// 1) Playlist configuration
// 2) Spotify build information
// Playlist is the configuration used to build a new Spotify playlist
type Playlist struct {
	ID     uuid.UUID `db:"id"`
	UserID uuid.UUID `db:"user_id"`

	Input       Input
	InputString string   `db:"input"`
	Name        string   `db:"name"`
	Description string   `db:"description"`
	Public      bool     `db:"public"`
	Schedule    Schedule `db:"schedule"`
	SpotifyID   *string  `db:"spotify_id"`
	FailureMsg  *string  `db:"failure_msg"`
	Building    bool     `db:"building"`
	Current     bool     `db:"current"`

	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	LastBuiltAt *time.Time `db:"last_built_at"`
}

func (p *Playlist) MarshalInput() error {
	b, err := json.Marshal(&p.Input)
	if err != nil {
		return err
	}
	p.InputString = string(b)
	return nil
}

func (p *Playlist) UnmarshalInput() error {
	var i Input
	err := json.Unmarshal([]byte(p.InputString), &i)
	if err != nil {
		return err
	}
	p.Input = i
	return nil
}

// CreatePlaylist inserts a new playlist into the DB
func (p *Postgres) CreatePlaylist(userID uuid.UUID, input Input, name, description string, public bool, schedule Schedule) error {
	b, err := json.Marshal(&input)
	if err != nil {
		return err
	}
	inputJSON := string(b)

	query := `
INSERT INTO playlists (
	user_id,
	input,
	name,
	description,
	public,
	schedule
)
VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6
);
`
	_, err = p.db.Exec(query, userID, inputJSON, name, description, public, schedule)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePlaylistConfig updates the part of a playlist row that configures how Spotify playlists are built
func (p *Postgres) UpdatePlaylistConfig(id uuid.UUID, playlist Playlist) error {
	query := `
UPDATE playlists SET
	input=$1,
	name=$2,
	description=$3,
	public=$4,
	schedule=$5,
	current=FALSE
WHERE id=$6;
`
	err := playlist.MarshalInput()
	if err != nil {
		return err
	}

	_, err = p.db.Exec(
		query,
		playlist.InputString,
		playlist.Name,
		playlist.Description,
		playlist.Public,
		playlist.Schedule,
		id,
	)
	if err != nil {
		return err
	}
	return nil
}

// GetPlaylist returns the playlist with a given id
func (p *Postgres) GetPlaylist(id uuid.UUID) (*Playlist, error) {
	var playlist Playlist
	query := `
SELECT *
FROM playlists
WHERE id=$1;	
`
	err := p.db.Get(&playlist, query, id)
	if err != nil {
		return nil, err
	}
	err = playlist.UnmarshalInput()
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

// GetPlaylists retrieves all the playlists associated with a given userID
func (p *Postgres) GetPlaylists(userID uuid.UUID) ([]Playlist, error) {
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

	for i := range playlists {
		err = playlists[i].UnmarshalInput()
		if err != nil {
			return nil, err
		}
	}

	return playlists, nil
}

func (p *Postgres) UpdatePlaylistStartBuild(id uuid.UUID) error {
	query := `
UPDATE playlists SET
	building=TRUE
WHERE id=$1;
	`
	_, err := p.db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePlaylistGoodBuild updates a playlist entry after a successful build of the playlist
func (p *Postgres) UpdatePlaylistGoodBuild(id uuid.UUID, spotifyID string) error {
	query := `
UPDATE playlists SET
	spotify_id=$1,
	last_built_at=$2,
	failure_msg=NULL,
	building=FALSE,
	current=TRUE
WHERE id=$3;
`
	_, err := p.db.Exec(query, spotifyID, time.Now(), id)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePlaylistBadBuild updates a playlist entry after a failed build of the playlist
func (p *Postgres) UpdatePlaylistBadBuild(id uuid.UUID, failureMsg string) error {
	query := `
UPDATE playlists SET
	last_built_at=$1,
	failure_msg=$2,
	building=FALSE
WHERE id=$3;
`
	_, err := p.db.Exec(query, time.Now(), failureMsg, id)
	if err != nil {
		return err
	}
	return nil
}

// DeletePlaylist deletes the playlist entry matching the given id
func (p *Postgres) DeletePlaylist(id uuid.UUID) error {
	query := `
DELETE FROM playlists
WHERE id=$1;
`
	_, err := p.db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func (p *Postgres) UpdatePlaylistBadDelete(id uuid.UUID, failureMsg string) error {
	query := `
UPDATE playlists SET
	failure_msg=$1,
WHERE id=$2;
`
	_, err := p.db.Exec(query, failureMsg, id)
	if err != nil {
		return err
	}
	return nil
}

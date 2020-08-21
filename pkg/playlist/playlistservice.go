package playlist

// import (
// 	"encoding/json"
// 	"errors"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/jmoiron/sqlx"
// )

// type PlaylistService struct {
// 	db *sqlx.DB
// }

// func New(db *sqlx.DB) *PlaylistService {
// 	return &PlaylistService{
// 		db: db,
// 	}
// }

// // TODO remove cyclic dependency by moving input/output to DB
// type INPUTODOREMOVE struct {
// }

// // CreatePlaylist inserts a new playlist into the DB
// func (p *PlaylistService) CreatePlaylist(userID uuid.UUID, input INPUTODOREMOVE, name, description string, public bool) error {
// 	b, err := json.Marshal(&input)
// 	if err != nil {
// 		return err
// 	}
// 	inputJSON := string(b)

// 	query := `
// INSERT INTO playlists (
// 	user_id,
// 	input,
// 	name,
// 	description,
// 	public,
// )
// VALUES (
// 	$1,
// 	$2,
// 	$3,
// 	$4,
// 	$5,
// );
// `
// 	_, err = p.db.Exec(query, userID, inputJSON, name, description, public)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (p *PlaylistService) UpdatePlaylist(id uuid.UUID, name string, userID uuid.UUID) error {
// 	return errors.New("not implemented")
// }

// // GetPlaylist returns the playlist with a given id
// func (p *PlaylistService) GetPlaylist(id uuid.UUID) (*Playlist, error) {
// 	var playlist Playlist
// 	query := `
// SELECT *
// FROM playlists
// WHERE id=$1;
// `
// 	err := p.db.Get(&playlist, query, id)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &playlist, nil
// }

// // GetPlaylists retrieves all the playlists associated with a given userID
// func (p *PlaylistService) GetPlaylists(userID uuid.UUID) ([]Playlist, error) {
// 	playlists := []Playlist{}
// 	query := `
// SELECT *
// FROM playlists
// WHERE user_id=$1;
// `
// 	err := p.db.Select(&playlists, query, userID)
// 	if err != nil {
// 		return playlists, err
// 	}
// 	return playlists, nil
// }

// // UpdatePlaylistGoodBuild updates a playlist entry after a successful build of the playlist
// func (p *PlaylistService) UpdatePlaylistGoodBuild(id uuid.UUID, spotifyID string) error {
// 	query := `
// UPDATE playlists SET
// 	spotify_id=$1,
// 	last_built_at=$2,
// 	failure_msg=NULL
// WHERE id=$3;
// `
// 	_, err := p.db.Exec(query, spotifyID, time.Now(), id)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// // UpdatePlaylistBadBuild updates a playlist entry after a failed build of the playlist
// func (p *PlaylistService) UpdatePlaylistBadBuild(id uuid.UUID, failureMsg string) error {
// 	query := `
// UPDATE playlists SET
// 	last_built_at=$1,
// 	failure_msg=$2
// WHERE id=$3;
// `
// 	_, err := p.db.Exec(query, time.Now(), failureMsg, id)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

package tmpl

import "github.com/calebschoepp/playlist-rotator/pkg/playlist"

type Home struct {
	Playlists []playlist.Playlist
}

type Login struct {
	SpotifyAuthURL string
}

type NewPlaylist struct {
	Name  string
	Saved bool
}

package tmpl

import (
	"html/template"
	"net/http"
	"os"

	"github.com/calebschoepp/playlist-rotator/pkg/playlist"
)

type TmplServicer interface {
	TmplHome(w http.ResponseWriter, data Home) error
	TmplLogin(w http.ResponseWriter, data Login) error
	TmplNewPlaylist(w http.ResponseWriter, data NewPlaylist) error
}

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

type TmplService struct {
	templates *template.Template
}

func New() (*TmplService, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	templates, err := template.ParseGlob(pwd + "/pkg/tmpl/*.html")
	if err != nil {
		return nil, err
	}

	return &TmplService{
		templates: templates,
	}, nil
}

func (t *TmplService) TmplHome(w http.ResponseWriter, data Home) error {
	return t.renderTemplate(w, "home", data)
}

func (t *TmplService) TmplLogin(w http.ResponseWriter, data Login) error {
	return t.renderTemplate(w, "login", data)
}

func (t *TmplService) TmplNewPlaylist(w http.ResponseWriter, data NewPlaylist) error {
	return t.renderTemplate(w, "new-playlist", data)
}

func (t *TmplService) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) error {
	err := t.templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

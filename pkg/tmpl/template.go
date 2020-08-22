package tmpl

import (
	"html/template"
	"net/http"
	"os"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
)

// Templater provides methods for templating HTML web pages
type Templater interface {
	TmplHome(w http.ResponseWriter, data Home) error
	TmplLogin(w http.ResponseWriter, data Login) error
	TmplPlaylist(w http.ResponseWriter, data Playlist) error
}

// Home is the data required to template '/'
type Home struct {
	Playlists []store.Playlist
}

// Login is the data required to template '/login'
type Login struct {
	SpotifyAuthURL string
}

// Playlist is the data required to template '/playlist/{playlistID}'
type Playlist struct {
	IsNew bool

	Name  string
	Saved bool
}

// TemplateService is the concrete implmentation of Templater backed by html/template
type TemplateService struct {
	templates *template.Template
}

// New returns a pointer to a TemplateService
func New() (*TemplateService, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	templates, err := template.ParseGlob(pwd + "/pkg/tmpl/*.gohtml")
	if err != nil {
		return nil, err
	}

	return &TemplateService{
		templates: templates,
	}, nil
}

// TmplHome templates '/'
func (t *TemplateService) TmplHome(w http.ResponseWriter, data Home) error {
	return t.renderTemplate(w, "home", data)
}

// TmplLogin templates '/login'
func (t *TemplateService) TmplLogin(w http.ResponseWriter, data Login) error {
	return t.renderTemplate(w, "login", data)
}

// TmplPlaylist templates '/playlist/{playlistID}'
func (t *TemplateService) TmplPlaylist(w http.ResponseWriter, data Playlist) error {
	return t.renderTemplate(w, "playlist", data)
}

func (t *TemplateService) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) error {
	err := t.templates.ExecuteTemplate(w, tmpl+".gohtml", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

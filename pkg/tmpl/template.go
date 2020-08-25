package tmpl

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
)

// Templater provides methods for templating HTML web pages
type Templater interface {
	TmplHome(w http.ResponseWriter, data Home) error
	TmplLogin(w http.ResponseWriter, data Login) error
	TmplPlaylist(w http.ResponseWriter, data Playlist) error
	TmplTrackSource(w http.ResponseWriter, data TrackSource) error
}

// Home is the data required to template '/'
type Home struct {
	Playlists []PlaylistInfo
}

type PlaylistInfo struct {
	store.Playlist
	TotalSongs       int
	BuildTagSrc      string
	ScheduleBlurb    string
	ScheduleSentence string
}

// Login is the data required to template '/login'
type Login struct {
	SpotifyAuthURL string
}

// Playlist is the data required to template '/playlist/{playlistID}'
type Playlist struct {
	IsNew bool

	Name        string
	Description string
	Schedule    store.Schedule
	Public      bool

	Sources          []store.TrackSource
	PotentialSources []PotentialSource
}

type PotentialSource struct {
	Name string
	ID   string
	Type store.TrackSourceType
}

// TrackSource is the data required to template '/playlist/{playlistID}/source/type/{type}/name/{name}/id/{id}'
type TrackSource struct {
	Source store.TrackSource
}

// TemplateService is the concrete implmentation of Templater backed by html/template
type TemplateService struct {
	templates *template.Template
}

// New returns a pointer to a TemplateService
func New() (*TemplateService, error) {
	funcMap := template.FuncMap{
		"unixTime": func() string { return fmt.Sprintf("%v", time.Now().Unix()) },
	}

	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	templates, err := template.New("main").Funcs(funcMap).ParseGlob(pwd + "/pkg/tmpl/*/*.gohtml")
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

// TmplTrackSource templates '/playlist/{playlistID}/source/type/{type}/name/{name}/id/{id}'
func (t *TemplateService) TmplTrackSource(w http.ResponseWriter, data TrackSource) error {
	return t.renderTemplate(w, "track-source", data)
}

func (t *TemplateService) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) error {
	err := t.templates.ExecuteTemplate(w, tmpl+".gohtml", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

package tmpl

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"go.uber.org/zap"
)

// Templater provides methods for templating HTML web pages
type Templater interface {
	TmplHome(w http.ResponseWriter, data Home)
	TmplDashboard(w http.ResponseWriter, data Dashboard)
	TmplPlaylist(w http.ResponseWriter, data Playlist)
	TmplTrackSource(w http.ResponseWriter, data TrackSource)
	TmplMobile(w http.ResponseWriter)
	TmplHelp(w http.ResponseWriter)
}

// Home is the data required to template '/' and `/login`
type Home struct {
	SpotifyAuthURL string
	Env            string
}

// Dashboard is the data required to template `/dashboard`
type Dashboard struct {
	Playlists []PlaylistInfo
	Env       string
}

// PlaylistInfo is a playlist wrapped with extra metadata
type PlaylistInfo struct {
	store.Playlist
	TotalSongs       int
	BuildTagSrc      string
	ScheduleBlurb    string
	ScheduleSentence string
	ImageURL         string
	FailureBlurb     string
}

// Playlist is the data required to template '/playlist/{playlistID}'
type Playlist struct {
	IsNew bool

	Name           string
	NameErr        string
	Description    string
	DescriptionErr string
	Schedule       store.Schedule
	Public         bool

	Sources          []TrackSource
	SourcesErr       string
	PotentialSources []PotentialSource

	Env string
}

// PotentialSource is the data for a playlists potential source of tracks
type PotentialSource struct {
	Name     string
	ID       string
	Type     store.TrackSourceType
	ImageURL string
}

// TrackSource is the data required to template '/playlist/{playlistID}/source/type/{type}/name/{name}/id/{id}'
type TrackSource struct {
	store.TrackSource
	CountString string
	CountErr    string
}

// Help is the data required to template '/help'
type Help struct {
	Env string
}

// Mobile is the data required to template '/mobile'
type Mobile struct {
	Env string
}

// TemplateService is the concrete implmentation of Templater backed by html/template
type TemplateService struct {
	templates *template.Template
	log       *zap.SugaredLogger
	env       string
}

// New returns a pointer to a TemplateService
func New(log *zap.SugaredLogger, env string) (*TemplateService, error) {
	funcMap := template.FuncMap{
		"unixTime": func() string { return fmt.Sprintf("%v", time.Now().Unix()) },
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
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
		log:       log,
		env:       env,
	}, nil
}

// TmplHome templates '/'
func (t *TemplateService) TmplHome(w http.ResponseWriter, data Home) {
	data.Env = t.env
	t.renderTemplate(w, "home", data)
}

// TmplDashboard templates '/'
func (t *TemplateService) TmplDashboard(w http.ResponseWriter, data Dashboard) {
	data.Env = t.env
	t.renderTemplate(w, "dashboard", data)
}

// TmplPlaylist templates '/playlist/{playlistID}'
func (t *TemplateService) TmplPlaylist(w http.ResponseWriter, data Playlist) {
	data.Env = t.env
	t.renderTemplate(w, "playlist", data)
}

// TmplTrackSource templates '/playlist/{playlistID}/source/type/{type}/name/{name}/id/{id}'
func (t *TemplateService) TmplTrackSource(w http.ResponseWriter, data TrackSource) {
	t.renderTemplate(w, "track-source", data)
}

// TmplMobile templates `/mobile`
func (t *TemplateService) TmplMobile(w http.ResponseWriter) {
	data := Mobile{}
	data.Env = t.env
	t.renderTemplate(w, "mobile", data)
}

// TmplHelp templates `/help`
func (t *TemplateService) TmplHelp(w http.ResponseWriter) {
	data := Help{}
	data.Env = t.env
	t.renderTemplate(w, "help", data)
}

func (t *TemplateService) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := t.templates.ExecuteTemplate(w, tmpl+".gohtml", data)
	if err != nil {
		t.log.Errorw("failed to render template", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

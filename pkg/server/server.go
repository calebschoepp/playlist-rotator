package server

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/playlist"
	"github.com/calebschoepp/playlist-rotator/web"

	"github.com/gorilla/mux"
	"github.com/zmb3/spotify"
)

type Server struct {
	Log         *log.Logger
	Config      *Config
	DB          *sql.DB
	Router      *mux.Router
	SpotifyAuth *spotify.Authenticator
	Templates   *template.Template
}

const stateCookieName = "oauthState"
const stateCookieExpiry = 30 * time.Minute

// New builds a new Server struct
func New(log *log.Logger, config *Config, db *sql.DB, router *mux.Router) (*Server, error) {
	// TODO make sure I use the correct and minimal scopes
	scopes := []string{
		spotify.ScopeUserReadPrivate,
		spotify.ScopePlaylistModifyPrivate,
		spotify.ScopeUserLibraryRead,
	}
	spotifyAuth := spotify.NewAuthenticator(fmt.Sprintf("%s%s:%d/callback", "http://", config.Host, config.Port), scopes...)
	spotifyAuth.SetAuthInfo(config.ClientID, config.ClientSecret)

	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	templates, err := template.ParseGlob(pwd + "/web/*.html")
	if err != nil {
		return nil, err
	}

	return &Server{
		Log:         log,
		Config:      config,
		DB:          db,
		Router:      router,
		SpotifyAuth: &spotifyAuth,
		Templates:   templates,
	}, nil
}

// SetupRoutes wires up the handlers to the appropriate routes
func (s *Server) SetupRoutes() {
	s.Router.Use(newSessionAuthMiddleware(s.DB, s.Log, []string{"/login", "/callback"}))
	s.Router.Path("/").Methods("GET").HandlerFunc(s.homePage)
	s.Router.Path("/login").Methods("GET").HandlerFunc(s.loginPage)
	s.Router.Path("/callback").Methods("GET").HandlerFunc(s.callbackPage)
	s.Router.Path("/new-playlist").Methods("GET").HandlerFunc(s.newPlaylistPage)
	s.Router.Path("/new-playlist").Methods("POST").HandlerFunc(s.newPlaylistForm)
}

// Run makes the Server start listening and serving on the configured addr
func (s *Server) Run() {
	s.Log.Printf("Listening on port %d", s.Config.Port)
	s.Log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.Config.Port), s.Router))
}

func (s *Server) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := s.Templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) homePage(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate(w, "home", web.Home{Playlists: []playlist.Playlist{playlist.Playlist{Name: "This is the name of a playlist"}}})
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	// State should be randomly generated and passed along with Oauth request
	// in cookie for security purposes
	state := randomState()
	cookie := http.Cookie{
		Name:    stateCookieName,
		Value:   state,
		Expires: time.Now().Add(stateCookieExpiry),
	}
	http.SetCookie(w, &cookie)

	spotifyAuthURL := s.SpotifyAuth.AuthURL(state)

	s.renderTemplate(w, "login", web.Login{SpotifyAuthURL: spotifyAuthURL})
}

func (s *Server) callbackPage(w http.ResponseWriter, r *http.Request) {
	// Get oauth2 tokens
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	token, err := s.SpotifyAuth.Token(stateCookie.Value, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println(token.AccessToken)
	log.Println(token.RefreshToken)

	sessionCookie := http.Cookie{
		Name:    sessionCookieName,
		Value:   "TODO_MAKE_THIS_A_RANDOM_SESSION_VALUE",
		Expires: time.Now().Add(stateCookieExpiry),
	}
	http.SetCookie(w, &sessionCookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) newPlaylistPage(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate(w, "new-playlist", web.NewPlaylist{Name: "", Saved: false})
}

func (s *Server) newPlaylistForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// TODO do something with form data

	s.renderTemplate(w, "new-playlist", web.NewPlaylist{Name: r.FormValue("playlistName"), Saved: true})
}

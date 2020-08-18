package server

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/calebschoepp/playlist-rotator/web"
	"golang.org/x/oauth2"

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

type Config struct {
	ClientID     string
	ClientSecret string
	Addr         string
}

const stateCookieName = "oauthState"
const stateCookieExpiry = 30 * time.Minute

var globalToken *oauth2.Token

// New builds a new Server struct
func New(log *log.Logger, config *Config, db *sql.DB, router *mux.Router) (*Server, error) {
	// TODO make sure I use the correct and minimal scopes
	scopes := []string{
		spotify.ScopeUserReadPrivate,
		spotify.ScopePlaylistModifyPrivate,
		spotify.ScopeUserLibraryRead,
	}
	spotifyAuth := spotify.NewAuthenticator("http://"+config.Addr+"/callback", scopes...)
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
	s.Router.HandleFunc("/", s.homePage).Methods("GET")
	s.Router.HandleFunc("/login", s.loginPage).Methods("GET")
	s.Router.HandleFunc("/callback", s.callbackPage).Methods("GET")
}

// Run makes the Server start listening and serving on the configured addr
func (s *Server) Run() {
	s.Log.Printf("Serving on %s", s.Config.Addr)
	s.Log.Fatal(http.ListenAndServe(s.Config.Addr, s.Router))
}

func (s *Server) renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := s.Templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) homePage(w http.ResponseWriter, r *http.Request) {
	if globalToken == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	client := s.SpotifyAuth.NewClient(globalToken)
	likedSongs, err := client.CurrentUsersTracks()
	if err != nil {
		// TODO handle error
	}
	fmt.Fprintf(w, "%d total liked songs", likedSongs.Total)
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

	token.
		log.Println(token.AccessToken)
	log.Println(token.RefreshToken)

	if globalToken == nil {
		log.Println("Setting token")
		globalToken = token
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

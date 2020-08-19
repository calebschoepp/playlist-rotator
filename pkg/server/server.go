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
	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
	"github.com/calebschoepp/playlist-rotator/pkg/user"
	"github.com/jmoiron/sqlx"

	"github.com/gorilla/mux"
	"github.com/zmb3/spotify"
)

type Server struct {
	Log         *log.Logger
	Config      *Config
	DB          *sqlx.DB
	Router      *mux.Router
	SpotifyAuth *spotify.Authenticator
	Templates   *template.Template
}

// TODO move these to config?
const stateCookieName = "oauthState"
const stateCookieExpiry = 30 * time.Minute
const sessionCookieName = "playlistRotatorSession"
const sessionCokkieExpiry = 30 * time.Second // TODO fine tune this

// New builds a new Server struct
func New(log *log.Logger, config *Config, db *sqlx.DB, router *mux.Router) (*Server, error) {
	// TODO make sure I use the correct and minimal scopes
	scopes := []string{
		spotify.ScopeUserReadPrivate,
		spotify.ScopePlaylistModifyPrivate,
		spotify.ScopeUserLibraryRead,
	}
	// TODO proabably a more idiomatic way to build redirectURL
	var redirectURL string
	if config.Host == "localhost" {
		redirectURL = fmt.Sprintf("%s%s:%d/callback", config.Protocol, config.Host, config.Port)
	} else {
		redirectURL = fmt.Sprintf("%s%s/callback", config.Protocol, config.Host)
	}
	spotifyAuth := spotify.NewAuthenticator(redirectURL, scopes...)
	spotifyAuth.SetAuthInfo(config.ClientID, config.ClientSecret)

	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	templates, err := template.ParseGlob(pwd + "/pkg/tmpl/*.html")
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
	s.renderTemplate(w, "home", tmpl.Home{Playlists: []playlist.Playlist{playlist.Playlist{Name: "This is the name of a playlist"}}})
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

	s.renderTemplate(w, "login", tmpl.Login{SpotifyAuthURL: spotifyAuthURL})
}

// TODO improve error handling
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

	// Generate new session token with expiry
	sessionToken := randomSessionToken()
	sessionExpiry := time.Now().Add(sessionCokkieExpiry)
	sessionCookie := http.Cookie{
		Name:    sessionCookieName,
		Value:   sessionToken,
		Expires: sessionExpiry,
	}

	// Get spotify ID
	client := s.SpotifyAuth.NewClient(token)
	privateUser, err := client.CurrentUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// TODO confirm this is the right spotify ID to be using
	spotifyID := privateUser.User.ID

	// Check if user already exists for the spotify ID
	var user user.User
	err = s.DB.Get(&user, "SELECT * FROM users WHERE spotify_id=$1", spotifyID)
	if err != nil && err != sql.ErrNoRows {
		// Something went wrong in DB lookup, error out
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if err == sql.ErrNoRows {
		// User does not exist for spotify ID yet, make one
		_, err = s.DB.Exec(`
		INSERT INTO users (
			spotify_id,
			session_token,
			session_expiry,
			playlists_built,
			access_token,
			refresh_token
			)
		VALUES ($1, $2, $3, $4, $5, $6);`,
			spotifyID, sessionToken, sessionExpiry, 0, token.AccessToken, token.RefreshToken)
		if err != nil {
			s.Log.Printf("failed to build new user: %v", err)
			http.Error(w, "failed to build new user", http.StatusInternalServerError)
			return
		}
	} else {
		// User already exists for spotify ID, update it
		_, err = s.DB.Exec(`
		UPDATE users SET
			session_token=$1,
			session_expiry=$2,
			access_token=$3,
			refresh_token=$4
		WHERE spotify_id=$5;`,
			sessionToken, sessionExpiry, token.AccessToken, token.RefreshToken, spotifyID)
		if err != nil {
			s.Log.Printf("failed to update existing user: %v", err)
			http.Error(w, "failed to update existing user", http.StatusInternalServerError)
			return
		}
	}

	// Set session cookie and redirect to home
	http.SetCookie(w, &sessionCookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) newPlaylistPage(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate(w, "new-playlist", tmpl.NewPlaylist{Name: "", Saved: false})
}

func (s *Server) newPlaylistForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// TODO do something with form data

	s.renderTemplate(w, "new-playlist", tmpl.NewPlaylist{Name: r.FormValue("playlistName"), Saved: true})
}

package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/playlist"
	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
	"github.com/calebschoepp/playlist-rotator/pkg/user"
	userPostgres "github.com/calebschoepp/playlist-rotator/pkg/user/postgres"
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
	UserService user.UserServicer
	TmplService tmpl.TmplServicer
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

	// Build UserService
	userService := userPostgres.New(db)

	// Build TmplService
	tmplService, err := tmpl.New()
	if err != nil {
		return nil, err
	}

	return &Server{
		Log:         log,
		Config:      config,
		DB:          db,
		Router:      router,
		SpotifyAuth: &spotifyAuth,
		UserService: userService,
		TmplService: tmplService,
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

func (s *Server) homePage(w http.ResponseWriter, r *http.Request) {
	s.TmplService.TmplHome(w, tmpl.Home{Playlists: []playlist.Playlist{playlist.Playlist{Name: "This is the name of a playlist"}}})
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

	s.TmplService.TmplLogin(w, tmpl.Login{SpotifyAuthURL: spotifyAuthURL})
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
	userExists, err := s.UserService.UserExists(spotifyID)
	if err != nil {
		// TODO handle error
	}
	if userExists {
		// Update user with new token and session data
		err = s.UserService.UpdateUser(
			spotifyID,
			sessionToken,
			sessionExpiry,
			*token,
		)
		if err != nil {
			// TODO handle error
		}
	} else {
		// Create a new user
		err = s.UserService.CreateUser(
			spotifyID,
			sessionToken,
			sessionExpiry,
			*token,
		)
		if err != nil {
			// TODO handle error
		}
	}

	// Set session cookie and redirect to home
	http.SetCookie(w, &sessionCookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) newPlaylistPage(w http.ResponseWriter, r *http.Request) {
	s.TmplService.TmplNewPlaylist(w, tmpl.NewPlaylist{Name: "", Saved: false})
}

func (s *Server) newPlaylistForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// TODO do something with form data

	s.TmplService.TmplNewPlaylist(w, tmpl.NewPlaylist{Name: r.FormValue("playlistName"), Saved: true})
}

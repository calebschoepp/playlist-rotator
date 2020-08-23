package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/calebschoepp/playlist-rotator/pkg/build"
	"github.com/calebschoepp/playlist-rotator/pkg/config"
	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/zmb3/spotify"
)

// Server congregates all of the services required to listen and serve HTTP requests
type Server struct {
	Log         *log.Logger
	Config      *config.Config
	Router      *mux.Router
	SpotifyAuth *spotify.Authenticator
	Store       store.Store
	Tmpl        tmpl.Templater
	Builder     build.Builder
}

// New builds a new Server struct
func New(log *log.Logger, config *config.Config, db *sqlx.DB, router *mux.Router) (*Server, error) {
	// Build spotifyAuth
	// TODO proabably a more idiomatic way to build redirectURL
	var redirectURL string
	if config.Host == "localhost" {
		redirectURL = fmt.Sprintf("%s%s:%d/callback", config.Protocol, config.Host, config.Port)
	} else {
		redirectURL = fmt.Sprintf("%s%s/callback", config.Protocol, config.Host)
	}
	// TODO make sure I use the correct and minimal scopes
	scopes := []string{
		spotify.ScopeUserReadPrivate,
		spotify.ScopePlaylistModifyPrivate,
		spotify.ScopeUserLibraryRead,
	}
	spotifyAuth := spotify.NewAuthenticator(redirectURL, scopes...)
	spotifyAuth.SetAuthInfo(config.ClientID, config.ClientSecret)

	// Build store
	store := store.New(db)

	// Build TmplService
	tmpl, err := tmpl.New()
	if err != nil {
		return nil, err
	}

	// Build builder
	builder := build.New(store, spotifyAuth)

	return &Server{
		Log:         log,
		Config:      config,
		Router:      router,
		SpotifyAuth: &spotifyAuth,
		Store:       store,
		Tmpl:        tmpl,
		Builder:     builder,
	}, nil
}

// SetupRoutes wires up the handlers to the appropriate routes
func (s *Server) SetupRoutes() {
	// Build middleware
	loggingMiddleware := newRequestLoggerMiddleware(s.Log)
	authMiddleware := newSessionAuthMiddleware(s.Store, s.Log, []string{`^\/login`, `^\/callback`, `^\/static\/.*`})

	// Serve static files
	s.Router.PathPrefix("/static/").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))

	// Serve routes
	s.Router.Use(loggingMiddleware, authMiddleware)
	s.Router.Path("/").Methods("GET").HandlerFunc(s.homePage)
	s.Router.Path("/login").Methods("GET").HandlerFunc(s.loginPage)
	s.Router.Path("/logout").Methods("GET").HandlerFunc(s.logoutPage)
	s.Router.Path("/callback").Methods("GET").HandlerFunc(s.callbackPage)
	s.Router.Path("/playlist/{playlistID}").Methods("GET").HandlerFunc(s.playlistPage)
	s.Router.Path("/playlist/{playlistID}").Methods("POST").HandlerFunc(s.playlistForm)
	s.Router.Path("/playlist/{playlistID}/source/type/{type}/name/{name}/id/{id}").Methods("GET").HandlerFunc(s.playlistTrackSourceAPI)
	s.Router.Path("/playlist/{playlistID}/build").Methods("POST").HandlerFunc(s.playlistBuild)
}

// Run makes the Server start listening and serving on the configured addr
func (s *Server) Run() {
	s.Log.Printf("Listening on port %d", s.Config.Port)
	s.Log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.Config.Port), s.Router))
}

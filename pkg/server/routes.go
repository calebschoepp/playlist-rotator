package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zmb3/spotify"
)

func (s *Server) homePage(w http.ResponseWriter, r *http.Request) {
	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		// TODO handle error
	}

	tmplData := tmpl.Home{}

	// Get playlists
	playlists, err := s.Store.GetPlaylists(*userID)
	if err != nil {
		s.Log.Printf("Error fetching playlists: %v", err)
		// TODO handle error
	}
	tmplData.Playlists = playlists

	s.Tmpl.TmplHome(w, tmplData)
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	// State should be randomly generated and passed along with Oauth request
	// in cookie for security purposes
	state := randomString(32)
	cookie := http.Cookie{
		Name:    stateCookieName,
		Value:   state,
		Expires: time.Now().Add(stateCookieExpiry),
	}
	http.SetCookie(w, &cookie)

	spotifyAuthURL := s.SpotifyAuth.AuthURL(state)

	s.Tmpl.TmplLogin(w, tmpl.Login{SpotifyAuthURL: spotifyAuthURL})
}

func (s *Server) logoutPage(w http.ResponseWriter, r *http.Request) {
	// Delete session cookie to logout
	expire := time.Now().Add(-7 * 24 * time.Hour)
	cookie := http.Cookie{
		Name:    sessionCookieName,
		Value:   "",
		MaxAge:  -1,
		Expires: expire,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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
	sessionToken := randomString(64)
	sessionExpiry := time.Now().Add(sessionCookieExpiry)
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
	userExists, err := s.Store.UserExists(spotifyID)
	if err != nil {
		// TODO handle error
		s.Log.Printf("Error checking if user exists: %v", err)
	}
	if userExists {
		// Update user with new token and session data
		err = s.Store.UpdateUser(
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
		err = s.Store.CreateUser(
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

func (s *Server) playlistPage(w http.ResponseWriter, r *http.Request) {
	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		s.Log.Println("Failed to get userID from context")
		// TODO handle error
	}

	// Get playlistID
	vars := mux.Vars(r)
	playlistID := vars["playlistID"]

	var tmplData tmpl.Playlist
	if playlistID != "new" {
		// Build up the form with the existing playlist data
		tmplData.IsNew = false

		pid, err := uuid.Parse(playlistID)
		if err != nil {
			// TODO handl error
			// Probably return generic error page
		}
		playlist, err := s.Store.GetPlaylist(pid)
		if err != nil {
			// TODO handl error
			// Probably return generic error page
		}

		tmplData.Name = playlist.Name
		tmplData.Description = playlist.Description
		tmplData.Public = playlist.Public
		tmplData.Schedule = playlist.Schedule

		var input store.Input
		err = json.Unmarshal([]byte(playlist.Input), &input)
		if err != nil {
			// TODO handle error
			// Probably return generic error page
		}
		tmplData.Sources = input.TrackSources
	} else {
		// New playlist so everything is empty
		tmplData.IsNew = true
	}

	// Regardless we gather the potential sources
	potentialSources, err := getPotentialSources(s.Store, s.SpotifyAuth, userID)
	if err != nil {
		// TODO handle error
		// Probably return generic error page
	}
	tmplData.PotentialSources = potentialSources
	s.Log.Println(potentialSources)

	s.Tmpl.TmplPlaylist(w, tmplData)
}

func (s *Server) playlistForm(w http.ResponseWriter, r *http.Request) {
	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		s.Log.Println("Failed to get userID from context")
		// TODO handle error
	}

	r.ParseForm()

	for k, v := range r.Form {
		fmt.Printf("%v: %v\n", k, strings.Join(v, ""))
	}

	// TODO validate name
	// TODO all of this
	// name := r.FormValue("name")
	// description := r.FormValue("description")
	// public := false

	// err := s.Store.CreatePlaylist(*userID, store.Input{}, name, description, public)
	// if err != nil {
	// 	s.Log.Printf("Failed to create new playlist: %v", err)
	// 	// TODO handle error
	// }

	s.Tmpl.TmplPlaylist(w, tmpl.Playlist{Name: r.FormValue("name")})
}

func (s *Server) playlistTrackSourceAPI(w http.ResponseWriter, r *http.Request) {
	// Get ids
	// TODO validate IDs
	vars := mux.Vars(r)
	encodedName := vars["name"]
	id := vars["id"]
	typeString := vars["type"]

	name, err := url.QueryUnescape(encodedName)
	if err != nil {
		// TODO handle error
	}

	source := store.TrackSource{}
	source.Count = 0
	source.Method = store.Random
	source.Name = name
	source.ID = spotify.ID(id)
	switch typeString {
	case string(store.LikedSongsSrc):
		source.Type = store.LikedSongsSrc
	case string(store.AlbumSrc):
		source.Type = store.AlbumSrc
	case string(store.PlaylistSrc):
		source.Type = store.PlaylistSrc
	}

	s.Tmpl.TmplTrackSource(w, tmpl.TrackSource{Source: source})
}

package server

import (
	"net/http"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/playlist"
	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
)

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

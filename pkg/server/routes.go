package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

		// TODO don't need anymore
		// var input store.Input
		// err = json.Unmarshal([]byte(playlist.Input), &input)
		// if err != nil {
		// 	// TODO handle error
		// 	// Probably return generic error page
		// }
		tmplData.Sources = playlist.Input.TrackSources
	} else {
		// New playlist so everything is empty
		tmplData.IsNew = true
	}

	s.Log.Println(tmplData.Description)
	s.Log.Println(tmplData.Name)
	s.Log.Println(tmplData.Sources)
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

	// Get playlistID
	vars := mux.Vars(r)
	playlistID := vars["playlistID"]

	r.ParseForm()

	// TODO can I extract some logic here somehow?
	// Iterate over all form values and build up a playlist record
	playlist := store.Playlist{}
	trackSources := map[string]*store.TrackSource{}
	for k, v := range r.Form {
		fmt.Printf("%v: %v\n", k, strings.Join(v, ""))
		if k == "name" {
			playlist.Name = strings.Join(v, "")
		} else if k == "description" {
			playlist.Description = strings.Join(v, "")
		} else if k == "access" {
			switch strings.Join(v, "") {
			case "public":
				playlist.Public = true
			case "private":
				playlist.Public = false
			}
		} else if k == "schedule" {
			switch strings.Join(v, "") {
			case string(store.Never):
				playlist.Schedule = store.Never
			case string(store.Daily):
				playlist.Schedule = store.Daily
			case string(store.Weekly):
				playlist.Schedule = store.Weekly
			case string(store.BiWeekly):
				playlist.Schedule = store.BiWeekly
			case string(store.Monthly):
				playlist.Schedule = store.Monthly
			}
		} else if strings.HasSuffix(k, "type") {
			parts := strings.Split(k, "::")
			id := parts[0]
			typ := strings.Join(v, "")
			var typEnum store.TrackSourceType
			switch typ {
			case string(store.AlbumSrc):
				typEnum = store.AlbumSrc
			case string(store.LikedSrc):
				typEnum = store.LikedSrc
			case string(store.PlaylistSrc):
				typEnum = store.PlaylistSrc
			}
			if ts, ok := trackSources[id]; ok {
				ts.Type = typEnum
			} else {
				trackSources[id] = &store.TrackSource{Type: typEnum}
			}
		} else if strings.HasSuffix(k, "id") {
			parts := strings.Split(k, "::")
			id := parts[0]
			idVal := strings.Join(v, "")
			if ts, ok := trackSources[id]; ok {
				ts.ID = spotify.ID(idVal)
			} else {
				trackSources[id] = &store.TrackSource{ID: spotify.ID(idVal)}
			}
		} else if strings.HasSuffix(k, "count") {
			parts := strings.Split(k, "::")
			id := parts[0]
			count := strings.Join(v, "")
			countVal, err := strconv.Atoi(count)
			if err != nil {
				// TODO handle error
			}
			if ts, ok := trackSources[id]; ok {
				ts.Count = countVal
			} else {
				trackSources[id] = &store.TrackSource{Count: countVal}
			}
		} else if strings.HasSuffix(k, "method") {
			parts := strings.Split(k, "::")
			id := parts[0]
			method := strings.Join(v, "")
			var methodEnum store.ExtractMethod
			switch method {
			case string(store.Randomly):
				methodEnum = store.Randomly
			case string(store.Latest):
				methodEnum = store.Latest
			}
			if ts, ok := trackSources[id]; ok {
				ts.Method = methodEnum
			} else {
				trackSources[id] = &store.TrackSource{Method: methodEnum}
			}
		} else if strings.HasSuffix(k, "name") {
			parts := strings.Split(k, "::")
			id := parts[0]
			name := strings.Join(v, "")
			if ts, ok := trackSources[id]; ok {
				ts.Name = name
			} else {
				trackSources[id] = &store.TrackSource{Name: name}
			}
		} else if k == "submit" {
			// TODO should I do something about this
			s.Log.Println("Just submit no worries for now")
		} else {
			// TODO log and potentially error on extraneous form input
			s.Log.Println("Form value did not match expected format")
		}
	}

	// TODO validate that everything on playlist that should be filled in is
	// Add input to playlist
	input := store.Input{}
	for _, ts := range trackSources {
		input.TrackSources = append(input.TrackSources, *ts)
	}
	// TODO don't need anymore
	// b, err := json.Marshal(&input)
	// if err != nil {
	// 	// TODO handle error
	// }
	// playlist.Input = string(b)
	playlist.Input = input

	s.Log.Printf("%v", playlist)

	// Move data into store
	if playlistID == "new" {
		err := s.Store.CreatePlaylist(
			*userID,
			input,
			playlist.Name,
			playlist.Description,
			playlist.Public,
			playlist.Schedule,
		)
		if err != nil {
			// TODO handle error
			s.Log.Printf("INSERT error: %v", err)
		}
	} else {
		pid, err := uuid.Parse(playlistID)
		if err != nil {
			// TODO handle error
			s.Log.Printf("ERROR HERE: %v", err)
		}
		err = s.Store.UpdatePlaylistConfig(pid, playlist)
		if err != nil {
			// TODO handle error
			s.Log.Printf("ERROR HERE: %v", err)
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
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
	source.Method = store.Latest
	source.Name = name
	source.ID = spotify.ID(id)
	switch typeString {
	case string(store.LikedSrc):
		source.Type = store.LikedSrc
	case string(store.AlbumSrc):
		source.Type = store.AlbumSrc
	case string(store.PlaylistSrc):
		source.Type = store.PlaylistSrc
	}

	s.Tmpl.TmplTrackSource(w, tmpl.TrackSource{Source: source})
}

func (s *Server) playlistBuild(w http.ResponseWriter, r *http.Request) {
	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		s.Log.Println("Failed to get userID from context")
		// TODO handle error
	}

	// Get playlistID
	vars := mux.Vars(r)
	pid := vars["playlistID"]
	playlistID, err := uuid.Parse(pid)
	if err != nil {
		// TODO handle error
	}

	s.Log.Printf("%v, %v", userID, playlistID)
	go s.Builder.BuildPlaylist(*userID, playlistID)
	w.WriteHeader(http.StatusAccepted)
}

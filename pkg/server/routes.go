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
	"golang.org/x/oauth2"
)

func (s *Server) homePage(w http.ResponseWriter, r *http.Request) {
	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		s.Log.Error("failed to get userID from context")
		http.Error(w, "failure authenticating", http.StatusForbidden)
		return
	}

	// Build spotify client
	user, err := s.Store.GetUserByID(*userID)
	if err != nil {
		s.Log.Errorw("failed to load user from db", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	token := oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenType:    user.TokenType,
		Expiry:       user.TokenExpiry,
	}
	client := s.SpotifyAuth.NewClient(&token)

	tmplData := tmpl.Home{}

	// Get playlists
	playlists, err := s.Store.GetPlaylists(*userID)
	if err != nil {
		s.Log.Errorw("failed to load playlists from db", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	for _, p := range playlists {
		// Total song count
		totalSongs := 0
		for _, ts := range p.Input.TrackSources {
			totalSongs += ts.Count
		}

		// Scheduling messages
		var scheduleBlurb string
		var scheduleSentence string
		format := "Scheduled to build at %s"
		layout := "3 PM on Monday, January 2"
		lastBuilt := p.LastBuiltAt
		if lastBuilt == nil {
			t := time.Now()
			lastBuilt = &t
		}
		schedule := p.Schedule
		switch schedule {
		case store.Never:
			scheduleBlurb = ""
			scheduleSentence = "You must manually build the playlist"
		case store.Daily:
			scheduleBlurb = "built daily"
			t := lastBuilt.AddDate(0, 0, 1)
			scheduleSentence = fmt.Sprintf(format, t.Format(layout))
		case store.Weekly:
			scheduleBlurb = "built weekly"
			t := lastBuilt.AddDate(0, 0, 7)
			scheduleSentence = fmt.Sprintf(format, t.Format(layout))
		case store.BiWeekly:
			scheduleBlurb = "built bi-weekly"
			t := lastBuilt.AddDate(0, 0, 14)
			scheduleSentence = fmt.Sprintf(format, t.Format(layout))
		case store.Monthly:
			scheduleBlurb = "built monthly"
			t := lastBuilt.AddDate(0, 1, 0)
			scheduleSentence = fmt.Sprintf(format, t.Format(layout))
		}

		// Build status
		var pillName string
		if p.Building {
			pillName = "building"
		} else if p.FailureMsg != nil {
			pillName = "failed"
		} else if p.Current {
			pillName = "built"
		} else {
			pillName = "not_built_yet"
		}
		buildTagSrc := fmt.Sprintf("/static/%s_pill.svg", pillName)

		// Playlist cover image
		imageURL := "/static/missing_cover_image.svg"
		if p.SpotifyID != nil {
			spotifyPlaylist, err := client.GetPlaylistOpt(spotify.ID(*p.SpotifyID), "images")
			if err != nil || len(spotifyPlaylist.Images) == 0 {
				s.Log.Warnw("failed to fetch cover image for playlist", "err", err.Error(), "spotifyID", *p.SpotifyID)
			}
			imageURL = spotifyPlaylist.Images[0].URL
		}

		// Source cover images
		for i := range p.Input.TrackSources {
			srcImageURL := "/static/missing_cover_image.svg"
			switch p.Input.TrackSources[i].Type {
			case store.LikedSrc:
				srcImageURL = "/static/liked_songs_cover.svg"
			case store.AlbumSrc:
				spotifyAlbum, err := client.GetAlbum(p.Input.TrackSources[i].ID)
				if err != nil || len(spotifyAlbum.Images) == 0 {
					s.Log.Warnw("failed to fetch cover image for album track source", "err", err.Error(), "spotifyID", p.Input.TrackSources[i].ID)
				}
				srcImageURL = spotifyAlbum.Images[0].URL
			case store.PlaylistSrc:
				spotifyPlaylist, err := client.GetPlaylistOpt(p.Input.TrackSources[i].ID, "images")
				if err != nil || len(spotifyPlaylist.Images) == 0 {
					s.Log.Warnw("failed to fetch cover image for playlist track source", "err", err.Error(), "spotifyID", p.Input.TrackSources[i].ID)

				}
				srcImageURL = spotifyPlaylist.Images[0].URL
			}
			p.Input.TrackSources[i].ImageURL = srcImageURL
		}

		pInfo := tmpl.PlaylistInfo{
			Playlist:         p,
			TotalSongs:       totalSongs,
			BuildTagSrc:      buildTagSrc,
			ScheduleBlurb:    scheduleBlurb,
			ScheduleSentence: scheduleSentence,
			ImageURL:         imageURL,
		}
		tmplData.Playlists = append(tmplData.Playlists, pInfo)
	}

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

func (s *Server) callbackPage(w http.ResponseWriter, r *http.Request) {
	// Get oauth2 tokens
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		s.Log.Errorw("failed to get state cookie for oauth2", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	token, err := s.SpotifyAuth.Token(stateCookie.Value, r)
	if err != nil {
		s.Log.Errorw("failed to build spotify auth", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// Generate new session token with expiry
	sessionToken := randomString(64)
	sessionExpiry := time.Now().Add(sessionCookieExpiry)
	sessionCookie := http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Expires:  sessionExpiry,
		HttpOnly: true,
	}

	// Get spotify ID
	client := s.SpotifyAuth.NewClient(token)
	privateUser, err := client.CurrentUser()
	if err != nil {
		s.Log.Errorw("failed to get current spotify userID", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	spotifyID := privateUser.User.ID

	// Check if user already exists for the spotify ID
	userExists, err := s.Store.UserExists(spotifyID)
	if err != nil {
		s.Log.Errorw("failed to check if user already exists in db", "err", err.Error(), "spotifyID", spotifyID)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
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
			s.Log.Errorw("failed to update user with new session data", "err", err.Error())
			http.Error(w, "server error", http.StatusInternalServerError)
			return
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
			s.Log.Errorw("failed to create new user", "err", err.Error())
			http.Error(w, "server error", http.StatusInternalServerError)
			return
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
		s.Log.Error("failed to build spotify auth")
		http.Error(w, "failure authenticating", http.StatusForbidden)
		return
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
			s.Log.Errorw("failed to parse playlist UUID", "err", err.Error(), "playlistID", playlistID)
			http.Error(w, "invalid playlistID", http.StatusInternalServerError)
			return
		}
		playlist, err := s.Store.GetPlaylist(pid)
		if err != nil {
			s.Log.Errorw("failed to get playlist from db", "err", err.Error())
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		tmplData.Name = playlist.Name
		tmplData.Description = playlist.Description
		tmplData.Public = playlist.Public
		tmplData.Schedule = playlist.Schedule

		// Build spotify client
		user, err := s.Store.GetUserByID(*userID)
		if err != nil {
			s.Log.Errorw("failed to get user from db", "err", err.Error())
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		token := oauth2.Token{
			AccessToken:  user.AccessToken,
			RefreshToken: user.RefreshToken,
			TokenType:    user.TokenType,
			Expiry:       user.TokenExpiry,
		}
		client := s.SpotifyAuth.NewClient(&token)

		// Source cover images
		for i := range playlist.Input.TrackSources {
			srcImageURL := "/static/missing_cover_image.svg"
			switch playlist.Input.TrackSources[i].Type {
			case store.LikedSrc:
				srcImageURL = "/static/liked_songs_cover.svg"
			case store.AlbumSrc:
				spotifyAlbum, err := client.GetAlbum(playlist.Input.TrackSources[i].ID)
				if err != nil || len(spotifyAlbum.Images) == 0 {
					s.Log.Warnw("failed to fetch album source cover image", "err", err.Error(), "spotifyID", playlist.Input.TrackSources[i].ID)
				}
				srcImageURL = spotifyAlbum.Images[0].URL
			case store.PlaylistSrc:
				spotifyPlaylist, err := client.GetPlaylistOpt(playlist.Input.TrackSources[i].ID, "images")
				if err != nil || len(spotifyPlaylist.Images) == 0 {
					s.Log.Warnw("failed to fetch playlist album source cover image", "err", err.Error(), "spotifyID", playlist.Input.TrackSources[i].ID)
				}
				srcImageURL = spotifyPlaylist.Images[0].URL
			}
			playlist.Input.TrackSources[i].ImageURL = srcImageURL
		}

		tmplData.Sources = playlist.Input.TrackSources
	} else {
		// New playlist so everything is empty
		tmplData.IsNew = true
	}

	// Regardless we gather the potential sources
	potentialSources, err := getPotentialSources(s.Store, s.SpotifyAuth, userID)
	if err != nil {
		s.Log.Errorw("failed to get potential track sources", "err", err.Error(), "userID", userID)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	tmplData.PotentialSources = potentialSources

	s.Tmpl.TmplPlaylist(w, tmplData)
}

func (s *Server) playlistForm(w http.ResponseWriter, r *http.Request) {
	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		s.Log.Error("failed to get userID from context")
		http.Error(w, "failure authenticating", http.StatusForbidden)
		return
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
				s.Log.Errorw("failed to parse count into integer", "err", err.Error())
				http.Error(w, "count must be an integer", http.StatusInternalServerError)
				return
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
			// Do nothing in this case
		} else {
			s.Log.Warnw("extraneous form input", "key", k, "value", strings.Join(v, ""))
		}
	}

	// TODO validate that everything on playlist that should be filled in is
	// Add input to playlist
	input := store.Input{}
	for _, ts := range trackSources {
		input.TrackSources = append(input.TrackSources, *ts)
	}
	playlist.Input = input

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
			s.Log.Errorw("failed to insert playlist into db", "err", err.Error())
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	} else {
		pid, err := uuid.Parse(playlistID)
		if err != nil {
			s.Log.Errorw("failed to parse playlistID as UUID", "err", err.Error(), "playlistID", playlistID)
			http.Error(w, "invalid playlistID", http.StatusInternalServerError)
			return
		}
		err = s.Store.UpdatePlaylistConfig(pid, playlist)
		if err != nil {
			s.Log.Errorw("failed to update playlist in db", "err", err.Error(), "playlistID", pid)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
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
		s.Log.Errorw("name is improperly url encoded", "err", err.Error(), "encodedName", encodedName)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		s.Log.Error("failed to get userID from context")
		http.Error(w, "failure authenticating", http.StatusForbidden)
		return
	}

	// TODO extract this into helper method
	// Build spotify client
	user, err := s.Store.GetUserByID(*userID)
	if err != nil {
		s.Log.Errorw("failed to get user from db", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	token := oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenType:    user.TokenType,
		Expiry:       user.TokenExpiry,
	}
	client := s.SpotifyAuth.NewClient(&token)

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

	// Get track source cover image
	srcImageURL := "/static/missing_cover_image.svg"
	switch source.Type {
	case store.LikedSrc:
		srcImageURL = "/static/liked_songs_cover.svg"
	case store.AlbumSrc:
		spotifyAlbum, err := client.GetAlbum(source.ID)
		if err != nil || len(spotifyAlbum.Images) == 0 {
			s.Log.Warnw("failed to fetch album cover image", "err", err.Error(), "spotifyID", source.ID)
		}
		srcImageURL = spotifyAlbum.Images[0].URL
	case store.PlaylistSrc:
		spotifyPlaylist, err := client.GetPlaylistOpt(source.ID, "images")
		if err != nil || len(spotifyPlaylist.Images) == 0 {
			s.Log.Warnw("failed to fetch playlist cover image", "err", err.Error(), "spotifyID", source.ID)
		}
		srcImageURL = spotifyPlaylist.Images[0].URL
	}
	source.ImageURL = srcImageURL

	s.Tmpl.TmplTrackSource(w, tmpl.TrackSource{Source: source})
}

func (s *Server) playlistBuild(w http.ResponseWriter, r *http.Request) {
	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		s.Log.Error("failed to get userID from context")
		http.Error(w, "failure authenticating", http.StatusForbidden)
		return
	}

	// Get playlistID
	vars := mux.Vars(r)
	pid := vars["playlistID"]
	playlistID, err := uuid.Parse(pid)
	if err != nil {
		s.Log.Errorw("failed to parse playlist as UUID", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	s.Log.Info("triggering background go routine to build playlist")
	go s.Builder.BuildPlaylist(*userID, playlistID)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) playlistDelete(w http.ResponseWriter, r *http.Request) {
	// Get userID
	userID := getUserID(r.Context())
	if userID == nil {
		s.Log.Error("failed to get userID from context")
		http.Error(w, "failure authenticating", http.StatusForbidden)
		return
	}

	// Get playlistID
	vars := mux.Vars(r)
	pid := vars["playlistID"]
	playlistID, err := uuid.Parse(pid)
	if err != nil {
		s.Log.Errorw("failed to parse playlist as UUID", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	s.Log.Info("triggering background go routine to delete playlist")
	go s.Builder.DeletePlaylist(*userID, playlistID)
	w.WriteHeader(http.StatusAccepted)
}

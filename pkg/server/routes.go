package server

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zmb3/spotify"
)

func (s *Server) homePage(w http.ResponseWriter, r *http.Request) {
	// State should be randomly generated and passed along with Oauth request
	// in cookie for security purposes
	state, err := generateRandomString(32)
	if err != nil {
		s.Log.Error("failed to generate random string for state")
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	cookie := http.Cookie{
		Name:    s.Config.StateCookieName,
		Value:   state,
		Expires: time.Now().Add(s.Config.StateCookieExpiry),
	}
	http.SetCookie(w, &cookie)

	spotifyAuthURL := s.Spotify.AuthURL(state)

	s.Tmpl.TmplHome(w, tmpl.Home{SpotifyAuthURL: spotifyAuthURL})
}

func (s *Server) logoutPage(w http.ResponseWriter, r *http.Request) {
	// Delete session cookie to logout
	expire := time.Now().Add(-7 * 24 * time.Hour)
	cookie := http.Cookie{
		Name:    s.Config.SessionCookieName,
		Value:   "",
		MaxAge:  -1,
		Expires: expire,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) callbackPage(w http.ResponseWriter, r *http.Request) {
	// Get oauth2 tokens
	stateCookie, err := r.Cookie(s.Config.StateCookieName)
	if err != nil {
		s.Log.Errorw("failed to get state cookie for oauth2", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	token, err := s.Spotify.Token(stateCookie.Value, r)
	if err != nil {
		s.Log.Errorw("failed to build spotify auth", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// Generate new session token with expiry
	sessionToken, err := generateRandomString(64)
	if err != nil {
		s.Log.Error("failed to generate random string for session token")
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	sessionExpiry := time.Now().Add(s.Config.SessionCookieExpiry)
	sessionCookie := http.Cookie{
		Name:     s.Config.SessionCookieName,
		Value:    sessionToken,
		Expires:  sessionExpiry,
		HttpOnly: true,
	}

	// Get spotify ID
	client := s.Spotify.NewClient(token)
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
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) dashboardPage(w http.ResponseWriter, r *http.Request) {
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
	client := s.Spotify.NewClient(&user.Token)

	tmplData := tmpl.Dashboard{}

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
		overrideSentence := false
		if lastBuilt == nil && p.Schedule != store.Never {
			t := time.Now()
			lastBuilt = &t
			overrideSentence = true
		}
		schedule := p.Schedule
		switch schedule {
		case store.Never:
			scheduleBlurb = ""
			scheduleSentence = "Click the build button to generate the playlist."
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

		if overrideSentence {
			scheduleSentence = "You need to click the build button once before it will build automatically."
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
				spotifyAlbum, err := client.GetAlbum(spotify.ID(p.Input.TrackSources[i].ID))
				if err != nil || len(spotifyAlbum.Images) == 0 {
					s.Log.Warnw("failed to fetch cover image for album track source", "err", err.Error(), "spotifyID", p.Input.TrackSources[i].ID)
				}
				srcImageURL = spotifyAlbum.Images[0].URL
			case store.PlaylistSrc:
				spotifyPlaylist, err := client.GetPlaylistOpt(spotify.ID(p.Input.TrackSources[i].ID), "images")
				if err != nil || len(spotifyPlaylist.Images) == 0 {
					s.Log.Warnw("failed to fetch cover image for playlist track source", "err", err.Error(), "spotifyID", p.Input.TrackSources[i].ID)

				}
				srcImageURL = spotifyPlaylist.Images[0].URL
			}
			p.Input.TrackSources[i].ImageURL = srcImageURL
		}

		var failureBlurb string
		if p.FailureMsg != nil {
			failureBlurb = *p.FailureMsg
		} else {
			failureBlurb = ""
		}

		pInfo := tmpl.PlaylistInfo{
			Playlist:         p,
			TotalSongs:       totalSongs,
			BuildTagSrc:      buildTagSrc,
			ScheduleBlurb:    scheduleBlurb,
			ScheduleSentence: scheduleSentence,
			ImageURL:         imageURL,
			FailureBlurb:     failureBlurb,
		}
		tmplData.Playlists = append(tmplData.Playlists, pInfo)
	}

	s.Tmpl.TmplDashboard(w, tmplData)
}

func (s *Server) helpPage(w http.ResponseWriter, r *http.Request) {
	s.Tmpl.TmplHelp(w)
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
		client := s.Spotify.NewClient(&user.Token)

		// Source cover images
		for i := range playlist.Input.TrackSources {
			srcImageURL := "/static/missing_cover_image.svg"
			switch playlist.Input.TrackSources[i].Type {
			case store.LikedSrc:
				srcImageURL = "/static/liked_songs_cover.svg"
			case store.AlbumSrc:
				spotifyAlbum, err := client.GetAlbum(spotify.ID(playlist.Input.TrackSources[i].ID))
				if err != nil || len(spotifyAlbum.Images) == 0 {
					s.Log.Warnw("failed to fetch album source cover image", "err", err.Error(), "spotifyID", playlist.Input.TrackSources[i].ID)
				}
				srcImageURL = spotifyAlbum.Images[0].URL
			case store.PlaylistSrc:
				spotifyPlaylist, err := client.GetPlaylistOpt(spotify.ID(playlist.Input.TrackSources[i].ID), "images")
				if err != nil || len(spotifyPlaylist.Images) == 0 {
					s.Log.Warnw("failed to fetch playlist album source cover image", "err", err.Error(), "spotifyID", playlist.Input.TrackSources[i].ID)
				}
				srcImageURL = spotifyPlaylist.Images[0].URL
			}
			playlist.Input.TrackSources[i].ImageURL = srcImageURL
		}

		var extraTrackSources []tmpl.TrackSource
		for _, ts := range playlist.Input.TrackSources {
			ets := tmpl.TrackSource{TrackSource: ts, CountErr: "", CountString: ""}
			extraTrackSources = append(extraTrackSources, ets)
		}
		tmplData.Sources = extraTrackSources
	} else {
		// New playlist so everything is empty
		tmplData.IsNew = true
	}

	// Regardless we gather the potential sources
	potentialSources, err := getPotentialSources(s.Store, s.Spotify, userID)
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

	playlistPtr, playlistTmplPtr, err := parsePlaylistForm(r.Form)
	if err != nil {
		s.Log.Errorw("failed to parse form", "err", err.Error())
		http.Error(w, "failed to parse form", http.StatusInternalServerError)
		return
	}
	if playlistTmplPtr != nil {
		s.Log.Info("parsed invalid form")
		playlistTmpl := *playlistTmplPtr
		ps, err := getPotentialSources(s.Store, s.Spotify, userID)
		if err != nil {
			s.Log.Errorw("failed to get potential sources", "err", err.Error())
			http.Error(w, "server error", http.StatusInternalServerError)
		}
		s.Log.Debugw("before", "ps", playlistTmpl.PotentialSources)
		playlistTmpl.PotentialSources = ps
		s.Log.Debugw("after", "ps", playlistTmpl.PotentialSources)
		s.Tmpl.TmplPlaylist(w, playlistTmpl)
		return
	}

	// Move data into store
	playlist := *playlistPtr
	if playlistID == "new" {
		err := s.Store.CreatePlaylist(
			*userID,
			playlist.Input,
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

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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

	// Build spotify client
	user, err := s.Store.GetUserByID(*userID)
	if err != nil {
		s.Log.Errorw("failed to get user from db", "err", err.Error())
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	client := s.Spotify.NewClient(&user.Token)

	source := store.TrackSource{}
	source.Count = 0
	source.Method = store.Latest
	source.Name = name
	source.ID = id
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
		spotifyAlbum, err := client.GetAlbum(spotify.ID(source.ID))
		if err != nil || len(spotifyAlbum.Images) == 0 {
			s.Log.Warnw("failed to fetch album cover image", "err", err.Error(), "spotifyID", source.ID)
		}
		srcImageURL = spotifyAlbum.Images[0].URL
	case store.PlaylistSrc:
		spotifyPlaylist, err := client.GetPlaylistOpt(spotify.ID(source.ID), "images")
		if err != nil || len(spotifyPlaylist.Images) == 0 {
			s.Log.Warnw("failed to fetch playlist cover image", "err", err.Error(), "spotifyID", source.ID)
		}
		srcImageURL = spotifyPlaylist.Images[0].URL
	}
	source.ImageURL = srcImageURL

	s.Tmpl.TmplTrackSource(w, tmpl.TrackSource{TrackSource: source, CountErr: "", CountString: ""})
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

func (s *Server) mobilePage(w http.ResponseWriter, r *http.Request) {
	s.Tmpl.TmplMobile(w)
}

package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/calebschoepp/playlist-rotator/pkg/motify"
	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"github.com/calebschoepp/playlist-rotator/pkg/tmpl"
	"github.com/google/uuid"
	zs "github.com/zmb3/spotify"
)

type ctxKey int

type playlistForm struct {
	name         string
	schedule     store.Schedule
	description  string
	public       bool
	trackSources map[string]*tmpl.ExtraTrackSource
}

const (
	userIDCtxKey ctxKey = iota
)

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
func generateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func getUserID(ctx context.Context) *uuid.UUID {
	userID, ok := ctx.Value(userIDCtxKey).(*uuid.UUID)
	if !ok {
		return nil
	}
	return userID
}

func getPotentialSources(s store.Store, spotify *motify.Spotify, userID *uuid.UUID) ([]tmpl.PotentialSource, error) {
	// Build spotify client
	user, err := s.GetUserByID(*userID)
	if err != nil {
		return nil, err
	}
	client := spotify.NewClient(&user.Token)

	// Add liked songs
	pss := []tmpl.PotentialSource{}
	pss = append(pss, tmpl.PotentialSource{Name: "Liked Songs", ID: "", Type: store.LikedSrc})

	// Find and add playlists
	limit := 50
	playlists, err := client.CurrentUsersPlaylistsOpt(&zs.Options{
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}
	// TODO check total to see if there is > 50 playlists, if so grab the rest
	for _, playlist := range playlists.Playlists {
		ps := tmpl.PotentialSource{
			Name: playlist.Name,
			ID:   string(playlist.ID),
			Type: store.PlaylistSrc,
		}
		pss = append(pss, ps)
	}

	// Find and add albums
	albums, err := client.CurrentUsersAlbumsOpt(&zs.Options{
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}
	// TODO check total to see if there is > 50 albums, if so grab the rest
	for _, album := range albums.Albums {
		ps := tmpl.PotentialSource{
			Name: album.FullAlbum.SimpleAlbum.Name,
			ID:   string(album.FullAlbum.SimpleAlbum.ID),
			Type: store.AlbumSrc,
		}
		pss = append(pss, ps)
	}

	return pss, nil
}

func parsePlaylistForm(values url.Values) (*store.Playlist, *tmpl.Playlist, error) {
	var data playlistForm
	data.trackSources = make(map[string]*tmpl.ExtraTrackSource)
	for k, v := range values {
		if k == "name" {
			data.name = strings.Join(v, "")
		} else if k == "description" {
			data.description = strings.Join(v, "")
		} else if k == "access" {
			switch strings.Join(v, "") {
			case "public":
				data.public = true
			case "private":
				data.public = false
			default:
				return nil, nil, fmt.Errorf("invalid access type: %v", strings.Join(v, ""))
			}
		} else if k == "schedule" {
			switch strings.Join(v, "") {
			case string(store.Never):
				data.schedule = store.Never
			case string(store.Daily):
				data.schedule = store.Daily
			case string(store.Weekly):
				data.schedule = store.Weekly
			case string(store.BiWeekly):
				data.schedule = store.BiWeekly
			case string(store.Monthly):
				data.schedule = store.Monthly
			default:
				return nil, nil, fmt.Errorf("invalid schedule type: %v", strings.Join(v, ""))
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
			default:
				return nil, nil, fmt.Errorf("invalid source type: %v", typ)
			}
			if ts, ok := data.trackSources[id]; ok {
				ts.Type = typEnum
			} else {
				data.trackSources[id] = &tmpl.ExtraTrackSource{TrackSource: store.TrackSource{Type: typEnum}}
			}
		} else if strings.HasSuffix(k, "id") {
			parts := strings.Split(k, "::")
			id := parts[0]
			idVal := strings.Join(v, "")
			if ts, ok := data.trackSources[id]; ok {
				ts.ID = idVal
			} else {
				data.trackSources[id] = &tmpl.ExtraTrackSource{TrackSource: store.TrackSource{ID: idVal}}
			}
		} else if strings.HasSuffix(k, "count") {
			parts := strings.Split(k, "::")
			id := parts[0]
			count := strings.Join(v, "")
			if ts, ok := data.trackSources[id]; ok {
				ts.CountString = count
			} else {
				data.trackSources[id] = &tmpl.ExtraTrackSource{CountString: count}
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
			default:
				return nil, nil, fmt.Errorf("invalid method type: %v", method)
			}
			if ts, ok := data.trackSources[id]; ok {
				ts.Method = methodEnum
			} else {
				data.trackSources[id] = &tmpl.ExtraTrackSource{TrackSource: store.TrackSource{Method: methodEnum}}
			}
		} else if strings.HasSuffix(k, "name") {
			parts := strings.Split(k, "::")
			id := parts[0]
			name := strings.Join(v, "")
			if ts, ok := data.trackSources[id]; ok {
				ts.Name = name
			} else {
				data.trackSources[id] = &tmpl.ExtraTrackSource{TrackSource: store.TrackSource{Name: name}}
			}
		} else if k == "submit" {
			// Do nothing in this case
		} else {
			return nil, nil, fmt.Errorf("extraneous form input: %v", k)

		}
	}

	// Validate all of the data parsed from the form
	invalid := false
	var tmplData tmpl.Playlist

	if len(data.name) == 0 {
		invalid = true
		tmplData.NameErr = "Name can't be empty."
	}
	if len(data.name) > 50 {
		invalid = true
		tmplData.NameErr = "Name is too long."
	}
	if !isASCII(data.name) {
		invalid = true
		tmplData.NameErr = "Name contains invalid characters."
	}
	if len(data.description) > 250 {
		invalid = true
		tmplData.DescriptionErr = "Description is too long."
	}
	if !isASCII(data.description) {
		invalid = true
		tmplData.DescriptionErr = "Description contains invalid characters."
	}
	if len(data.trackSources) == 0 {
		invalid = true
		tmplData.SourcesErr = "At least one source is required."
	}
	if len(data.trackSources) > 10 {
		invalid = true
		tmplData.SourcesErr = "Too many sources."
	}

	// TODO check that there are no duplicate track sources

	for _, fts := range data.trackSources {
		count, err := strconv.Atoi(fts.CountString)
		if err != nil {
			invalid = true
			fts.CountErr = "Count is not a number."
			fts.Count = 0
		}
		fts.Count = count
		if fts.Count < 0 {
			invalid = true
			fts.CountErr = "Count is negative."
		}
		if fts.Count > 10000 {
			invalid = true
			fts.CountErr = "Count is too big."
		}
		if len(fts.ID) == 0 {
			invalid = true
			return nil, nil, errors.New("empty ID on source")
		}
		if len(fts.Name) == 0 {
			invalid = true
			return nil, nil, errors.New("empty name on source")
		}
	}

	if invalid {
		tmplData.Name = data.name
		tmplData.Description = data.description
		tmplData.IsNew = false
		tmplData.Public = data.public
		tmplData.Schedule = data.schedule

		var srcs []tmpl.ExtraTrackSource
		for _, v := range data.trackSources {
			srcs = append(srcs, *v)
		}
		tmplData.Sources = srcs

		tmplData.PotentialSources = nil // TODO build this up

		return nil, &tmplData, nil
	}

	// Build up playlist
	var playlist store.Playlist
	playlist.Name = data.name
	playlist.Description = data.description
	playlist.Public = data.public
	playlist.Schedule = data.schedule

	// Add input to playlist
	input := store.Input{}
	for _, ets := range data.trackSources {
		ts := store.TrackSource{
			Name:     ets.Name,
			ID:       ets.ID,
			Type:     ets.Type,
			Count:    ets.Count,
			Method:   ets.Method,
			ImageURL: ets.ImageURL, // TODO where is this coming from... Need to embed in form?
		}
		input.TrackSources = append(input.TrackSources, ts)
	}
	playlist.Input = input
	return &playlist, nil, nil
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

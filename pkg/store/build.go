package store

import "github.com/zmb3/spotify"

// Output configures the user facing result of a newly built Spotify playlist
type Output struct {
	Name        string
	Description string
	Public      bool
}

// Input configures the sources used to generate a new Spotify playlist
type Input struct {
	TrackSources []TrackSource `json:"trackSources"`
}

// TrackSource represents a single source of tracks for a generated Spotify playlist
type TrackSource struct {
	Name     string          `json:"name"`
	ID       spotify.ID      `json:"id"` // TODO this should be a string
	Type     TrackSourceType `json:"type"`
	Count    int             `json:"count"`
	Method   ExtractMethod   `json:"method"`
	ImageURL string          // Not serialized and stored in DB, only used to display in UI
}

// StringifyMethod returns a string version of an extraction method
func (t TrackSource) StringifyMethod() string {
	switch t.Method {
	case Randomly:
		return string(Randomly)
	case Latest:
		return string(Latest)
	}
	return "Unknown method"
}

// ExtractMethod is the means by which the server pulls songs from a track source
type ExtractMethod string

const (
	// Randomly songs are chosen from the source
	Randomly ExtractMethod = "Randomly"
	// Latest songs are chosen from the source
	Latest = "Latest"
)

// Schedule is how often spotify playlists are automatically built
type Schedule string

const (
	// Never automatically build playlist
	Never Schedule = "Never"
	// Daily build the playlist
	Daily = "Daily"
	// Weekly build the playlist
	Weekly = "Weekly"
	// BiWeekly build the playlist
	BiWeekly = "Bi-Weekly"
	// Monthly build the playlist
	Monthly = "Monthly"
)

// TrackSourceType is an enumeration of the possible track sources for a playlist
type TrackSourceType string

const (
	// LikedSrc pulls tracks from Liked Songs
	LikedSrc TrackSourceType = "Liked"
	// AlbumSrc pulls tracks from an album
	AlbumSrc = "Album"
	// PlaylistSrc pulls tracks from a playlist
	PlaylistSrc = "Playlist"
)

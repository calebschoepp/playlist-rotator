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
	PlaylistInputs []PlaylistInput `json:"playlistInputs"`
}

// PlaylistInput represents a single source for a generated Spotify playlist
type PlaylistInput struct {
	PlaylistID spotify.ID    `json:"playlistID"`
	IsSaved    bool          `json:"isSaved"`
	Count      int           `json:"count"`
	Method     ExtractMethod `json:"method"`
}

type ExtractMethod string

const (
	// Random songs are chosen from the source
	Random ExtractMethod = "random"
	// Top songs are chosen from the source TODO change this to latest
	Top = "top"
)

type Schedule string

const (
	Never    Schedule = "Never"
	Daily             = "Daily"
	Weekly            = "Weekly"
	BiWeekly          = "Bi-Weekly"
	Monthly           = "Monthly"
)

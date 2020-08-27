package build

import (
	"sync"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
)

// BuildScheduledPlaylists builds all scheduled playlists whose deadlines have passed
func (s *Service) BuildScheduledPlaylists() {
	s.log.Info("starting build job")

	var wg sync.WaitGroup

	// Get playlists
	playlists, err := s.store.GetAllPlaylists()
	if err != nil {
		// This probably won't happen but if it does something is seriously wrong
		// Not much we can do here expcept log it and wait until the next time the job runs
		s.log.Errorw("failed to get all playlists from store", "err", err.Error())
		return
	}

	built := 0
	neverScheduled := 0
	neverManuallyBuilt := 0
	notDeadline := 0

	// For every playlist if the deadline has passed build it
	for i, p := range playlists {
		// Don't build playlists that are never scheduled
		if p.Schedule == store.Never {
			s.log.Infow("skip building playist that is never scheduled", "idx", i, "playlistID", p.ID)
			neverScheduled++
			continue
		}

		// Don't build playlists that haven't been built manually at least once
		if p.LastBuiltAt == nil {
			s.log.Infow("skip building playist that has never been manually built", "idx", i, "playlistID", p.ID)
			neverManuallyBuilt++
			continue
		}

		// Don't build playlists whose deadlines haven't passed yet
		deadline := *p.LastBuiltAt
		switch p.Schedule {
		case store.Never:
			// Shouldn't reach here
		case store.Daily:
			deadline = deadline.AddDate(0, 0, 1)
		case store.Weekly:
			deadline = deadline.AddDate(0, 0, 7)
		case store.BiWeekly:
			deadline = deadline.AddDate(0, 0, 14)
		case store.Monthly:
			deadline = deadline.AddDate(0, 1, 0)
		}
		if time.Now().Before(deadline) {
			s.log.Infow("skip building playist whose deadline hasn't passed", "idx", i, "playlistID", p.ID)
			notDeadline++
			continue
		}

		// By this point we know we want to build the playlist
		s.log.Infow("building playlist", "idx", i, "playlistID", p.ID)
		built++
		wg.Add(1)
		go func(playlist store.Playlist) {
			defer wg.Done()
			s.BuildPlaylist(playlist.UserID, playlist.ID)
		}(p)
	}

	wg.Wait()

	// Print out summary
	s.log.Infow(
		"Build summary",
		"total",
		len(playlists),
		"built",
		built,
		"neverScheduled",
		neverScheduled,
		"neverManuallyBuilt",
		neverManuallyBuilt,
		"notDeadline",
		notDeadline,
	)
}

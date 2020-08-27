package cmd

import (
	"sync"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/build"
	"github.com/calebschoepp/playlist-rotator/pkg/config"
	"github.com/calebschoepp/playlist-rotator/pkg/motify"
	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	// This driver here is okay because test cmd is ran seperate from serve cmd
	_ "github.com/lib/pq"
)

func init() {
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build any scheduled playlists with deadlines that have passed",
	Run: func(cmd *cobra.Command, args []string) {
		// Setup log
		logger, _ := zap.NewDevelopment()
		sugarLogger := logger.Sugar()

		// Setup config
		conf, err := config.New()
		if err != nil {
			sugarLogger.Fatalw("failed to build config", "err", err)
		}

		// Setup DB
		var db *sqlx.DB
		db, err = sqlx.Open("postgres", conf.DatabaseURL)
		if err != nil {
			sugarLogger.Fatalw("failed to setup db", "err", err)
		}

		// Setup store
		store := store.New(db)

		// Setup spotify auth
		spotify := motify.New("", conf.ClientID, conf.ClientSecret)

		// Setup build service
		buildService := build.New(store, spotify, sugarLogger)

		buildScheduledPlaylists(sugarLogger, store, buildService)
	},
}

// TODO this probably deserves to be a method on build service so it is more testable
func buildScheduledPlaylists(log *zap.SugaredLogger, s store.Store, bs build.Builder) {
	log.Info("starting build job")

	var wg sync.WaitGroup

	// Get playlists
	playlists, err := s.GetAllPlaylists()
	if err != nil {
		// This probably won't happen but if it does something is seriously wrong
		// Not much we can do here expcept log it and wait until the next time the job runs
		log.Errorw("failed to get all playlists from store", "err", err.Error())
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
			log.Infow("skip building playist that is never scheduled", "idx", i, "playlistID", p.ID)
			neverScheduled++
			continue
		}

		// Don't build playlists that haven't been built manually at least once
		if p.LastBuiltAt == nil {
			log.Infow("skip building playist that has never been manually built", "idx", i, "playlistID", p.ID)
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
			log.Infow("skip building playist whose deadline hasn't passed", "idx", i, "playlistID", p.ID)
			notDeadline++
			continue
		}

		// By this point we know we want to build the playlist
		log.Infow("building playlist", "idx", i, "playlistID", p.ID)
		built++
		wg.Add(1)
		go func(playlist store.Playlist) {
			defer wg.Done()
			bs.BuildPlaylist(playlist.UserID, playlist.ID)
		}(p)
	}

	wg.Wait()

	// Print out summary
	log.Infow(
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

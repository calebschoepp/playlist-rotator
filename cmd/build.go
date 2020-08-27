package cmd

import (
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

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build any scheduled playlists with matured deadlines",
	Run: func(cmd *cobra.Command, args []string) {
		// Setup log
		logger, _ := zap.NewDevelopment()
		sugarLogger := logger.Sugar()

		// Setup config
		conf, err := config.New()
		if err != nil {
			sugarLogger.Fatalw("failed to build config", "err", err)
		}

		sugarLogger.Infof("config %v", conf)

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

func init() {
	rootCmd.AddCommand(buildCmd)
}

func buildScheduledPlaylists(log *zap.SugaredLogger, store store.Store, bs build.Builder) {
	log.Info("Here we are")
}

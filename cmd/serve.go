package cmd

import (
	"github.com/calebschoepp/playlist-rotator/pkg/config"
	"github.com/calebschoepp/playlist-rotator/pkg/server"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Spin up the playlist-rotator HTTP server",
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

		// Setup router
		router := mux.NewRouter()

		// Setup server
		server, err := server.New(sugarLogger, conf, db, router)
		if err != nil {
			sugarLogger.Fatalw("failed to build server", "err", err)
		}
		server.SetupRoutes()

		// Start serving requests
		server.Run()
	},
}

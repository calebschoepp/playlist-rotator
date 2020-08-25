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

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Spin up the playlist-rotator HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		// Setup log
		logger, _ := zap.NewDevelopment()
		sugarLogger := logger.Sugar()

		// Setup config
		conf, err := config.NewConfig()
		if err != nil {
			sugarLogger.Fatalf("cmd: failed to build server: %v", err)
		}

		// Setup DB
		var db *sqlx.DB
		db, err = sqlx.Open("postgres", conf.DatabaseURL)
		if err != nil {
			sugarLogger.Fatalf("cmd: failed to setup db: %v", err)
		}

		// Setup router
		router := mux.NewRouter()

		// Setup server
		server, err := server.New(sugarLogger, conf, db, router)
		if err != nil {
			sugarLogger.Fatalf("cmd: failed to build server: %v", err)
		}
		server.SetupRoutes()

		// Start serving requests
		server.Run()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

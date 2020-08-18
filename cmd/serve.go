package cmd

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/gorilla/mux"

	"github.com/calebschoepp/playlist-rotator/pkg/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Spin up the playlist-rotator HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		// Setup config
		config := &server.Config{
			ClientID:     "",
			ClientSecret: "",
			Addr:         "localhost:8080",
			DatabaseURL:  os.Getenv("DATABASE_URL"),
		}

		// Setup DB
		var db *sql.DB
		db, err := sql.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatalf("cmd: failed to setup db: %v", err)
		}

		// Setup log
		log := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

		// Setup router
		router := mux.NewRouter()

		// Setup server
		server, err := server.New(log, config, db, router)
		if err != nil {
			log.Fatalf("cmd: failed to build server: %v", err)
		}
		server.SetupRoutes()

		// Start serving requests
		server.Run()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

package cmd

import (
	"database/sql"
	"log"
	"os"

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
		}

		// Setup DB
		var db sql.DB
		// db, err := sql.Open("postgres", "user=theUser dbname=theDbName sslmode=verify-full")
		// if err != nil {
		// 	panic(err)
		// }

		// Setup log
		log := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

		// Setup router
		router := mux.NewRouter()

		// Setup server
		server, err := server.New(log, config, &db, router)
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

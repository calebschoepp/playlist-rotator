package cmd

import (
	"fmt"
	"log"

	"github.com/calebschoepp/playlist-rotator/pkg/motify"
	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/calebschoepp/playlist-rotator/pkg/build"
	"github.com/calebschoepp/playlist-rotator/pkg/store"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the spotify playlyist building functionality",
	Run: func(cmd *cobra.Command, args []string) {
		// Setup DB
		var db *sqlx.DB
		db, err := sqlx.Open("postgres", "postgres://postgres:postgres@localhost:5432/playlist-rotator")
		if err != nil {
			log.Fatalf("cmd: failed to setup db: %v", err)
		}

		// Build store
		store := store.New(db)

		// Setup log
		logger, _ := zap.NewDevelopment()
		sugarLogger := logger.Sugar()

		// Build spotify auth
		spotify := motify.New("", "", "")

		buildService := build.New(store, spotify, sugarLogger)
		uid, _ := uuid.Parse("5a85837b-e8c9-4b72-9bad-adaf68edd488")
		pid, _ := uuid.Parse("bfe53ae7-2ee8-48a2-aed3-fae3c25ea1f7")
		buildService.BuildPlaylist(uid, pid)
		fmt.Println("Finished")
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}

package server

import (
	"database/sql"
	"log"
	"net/http"
)

const sessionCookieName = "playlistRotatorSession"

func newSessionAuthMiddleware(db *sql.DB, log *log.Logger, blacklist []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ignore blacklisted paths
			for _, path := range blacklist {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}
			// Get session cookie
			_, err := r.Cookie(sessionCookieName)
			if err != nil {
				log.Println("User not authenticated: redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// TODO
			// Verify session is not expired

			// Call the next handler in chain
			next.ServeHTTP(w, r)
		})
	}
}

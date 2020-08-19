package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

// TODO this key thing is gross
type contextKey int

const (
	userKey contextKey = iota
)

func newSessionAuthMiddleware(db *sqlx.DB, log *log.Logger, blacklist []string) func(next http.Handler) http.Handler {
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
			sessionCookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				log.Println("User not authenticated: no session: redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Get user
			var user User
			err = db.Get(&user, "SELECT * FROM users WHERE session_token=$1", sessionCookie.Value)
			if err != nil {
				log.Printf("%v", err)
				log.Println("User not authenticated: no matching user: redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Store user in context
			r.WithContext(context.WithValue(r.Context(), userKey, user))

			// Verify session is not expired
			if time.Now().Sub(user.SessionExpiry) > 0 {
				log.Println("User not authenticated: session expired: redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Call the next handler in chain
			next.ServeHTTP(w, r)
		})
	}
}

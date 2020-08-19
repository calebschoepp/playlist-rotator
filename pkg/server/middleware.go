package server

import (
	"log"
	"net/http"
	"time"

	"github.com/calebschoepp/playlist-rotator/pkg/user"
)

// TODO this key thing is gross
type contextKey int

const (
	userKey contextKey = iota
)

func newSessionAuthMiddleware(userService user.UserServicer, log *log.Logger, blacklist []string) func(next http.Handler) http.Handler {
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
				log.Println("User not authenticated: no session cookie: redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Get session expiry
			sessionExpiry, err := userService.GetSessionExpiry(sessionCookie.Value)
			if err != nil {
				// TODO better error handling here
				log.Printf("%v", err)
				log.Println("User not authenticated: no matching user: redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Verify session is not expired
			if time.Now().Sub(*sessionExpiry) > 0 {
				log.Println("User not authenticated: session expired: redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Call the next handler in chain
			next.ServeHTTP(w, r)
		})
	}
}

package server

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"go.uber.org/zap"

	"github.com/calebschoepp/playlist-rotator/pkg/store"
)

type contextKey int

const (
	userKey contextKey = iota
)

func newRequestLoggerMiddleware(log *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Call the next handler in chain
			next.ServeHTTP(w, r)

			// Log the request
			log.Infow("served request", "path", r.URL.Path, "method", r.Method)
		})
	}
}

func newSessionAuthMiddleware(store store.Store, log *zap.SugaredLogger, blacklist []string, sessionCookieName string) func(next http.Handler) http.Handler {
	// Cache the regex object of each route
	var blacklistRegexp []*regexp.Regexp
	for _, expr := range blacklist {
		regexp, err := regexp.Compile(expr)
		if err != nil {
			panic("Invalid regex expression for path blacklist")
		}
		blacklistRegexp = append(blacklistRegexp, regexp)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ignore blacklisted paths
			for _, regexp := range blacklistRegexp {
				if regexp.MatchString(r.URL.Path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Get session cookie
			sessionCookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				log.Info("user not authenticated: no session cookie")
				log.Info("redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Get session expiry
			sessionExpiry, err := store.GetSessionExpiry(sessionCookie.Value)
			if err != nil {
				log.Warnw("user not authenticated: no matching user", "err", err.Error())
				log.Info("redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Verify session is not expired
			if sessionExpiry.Before(time.Now()) {
				log.Info("user not authenticated: session expired")
				log.Info("redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Store userID in context
			userID, err := store.GetUserID(sessionCookie.Value)
			if err != nil {
				log.Errorw("something failed fetching userID", "err", err.Error())
				log.Info("redirecting to /login")
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			ctx := context.WithValue(r.Context(), userIDCtxKey, userID)

			// Call the next handler in chain
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

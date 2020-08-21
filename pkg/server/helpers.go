package server

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type ctxKey int

const (
	userIDCtxKey ctxKey = iota
)

// TODO move these to config?
const stateCookieName = "oauthState"
const stateCookieExpiry = 30 * time.Minute
const sessionCookieName = "playlistRotatorSession"
const sessionCookieExpiry = 30 * time.Minute // TODO fine tune this

// TODO use crypto/rand?
func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func getUserID(ctx context.Context) *uuid.UUID {
	userID, ok := ctx.Value(userIDCtxKey).(*uuid.UUID)
	if !ok {
		return nil
	}
	return userID
}

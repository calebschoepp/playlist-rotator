package server

import (
	"math/rand"
	"time"
)

// TODO make this random
func randomState() string {
	return randomString(32)
}

// TODO use crypto/rand?
func randomSessionToken() string {
	return randomString(64)
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

// TODO move these to config?
const stateCookieName = "oauthState"
const stateCookieExpiry = 30 * time.Minute
const sessionCookieName = "playlistRotatorSession"
const sessionCokkieExpiry = 30 * time.Second // TODO fine tune this

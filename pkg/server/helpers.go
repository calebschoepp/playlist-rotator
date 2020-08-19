package server

import (
	"context"
	"errors"
	"math/rand"
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

// TODO handle empty case properly
func getUser(ctx context.Context) (*User, error) {
	val := ctx.Value(userKey)
	user, ok := val.(User)
	if !ok {
		return nil, errors.New("stored user is of invalid type")
	}
	return &user, nil
}

package motify

import (
	"net/http"

	zs "github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

// TODO add some metrics to this that count the number of api calls
// TODO make this stuff an interface so that it is more testable

// Spotify authenticates and builds clients
type Spotify struct {
	auth zs.Authenticator
}

// New returns a new Spotify struct which can be used to authenticate and build clients
func New(redirectURL, clientID, clientSecret string) *Spotify {
	// TODO make sure I use the correct and minimal scopes
	scopes := []string{
		zs.ScopeUserReadPrivate,
		zs.ScopePlaylistReadPrivate,
		zs.ScopePlaylistModifyPrivate,
		zs.ScopeUserLibraryRead,
	}

	auth := zs.NewAuthenticator(redirectURL, scopes...)
	auth.SetAuthInfo(clientID, clientSecret)

	return &Spotify{
		auth: auth,
	}
}

// NewClient returns a Client that can be used to access Spotify APIs
func (s *Spotify) NewClient(token *oauth2.Token) Client {
	client := s.auth.NewClient(token)
	return newClient(client)
}

// AuthURL returns a url at which a user can authenticate
func (s *Spotify) AuthURL(state string) string {
	return s.auth.AuthURL(state)
}

// Token extracts an oauth2 token from a http.Request
func (s *Spotify) Token(state string, r *http.Request) (*oauth2.Token, error) {
	return s.auth.Token(state, r)
}

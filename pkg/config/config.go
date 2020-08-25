package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the settings for a Server
type Config struct {
	ClientID            string
	ClientSecret        string
	Protocol            string
	Host                string
	Port                int
	DatabaseURL         string
	StateCookieName     string
	StateCookieExpiry   time.Duration
	SessionCookieName   string
	SessionCookieExpiry time.Duration
}

// New returns a Config struct with sane defaults and env variable overrides
func New() (*Config, error) {
	config := Config{
		ClientID:            "",
		ClientSecret:        "",
		Protocol:            "http://",
		Host:                "localhost",
		Port:                8080,
		DatabaseURL:         "",
		StateCookieName:     "oauthState",
		StateCookieExpiry:   30 * time.Minute,
		SessionCookieName:   "session",
		SessionCookieExpiry: 60 * time.Minute,
	}

	if clientID, present := os.LookupEnv("CLIENT_ID"); present {
		config.ClientID = clientID
	}
	if clientSecret, present := os.LookupEnv("CLIENT_SECRET"); present {
		config.ClientSecret = clientSecret
	}
	if protocol, present := os.LookupEnv("PROTOCOL"); present {
		config.Protocol = protocol
	}
	if host, present := os.LookupEnv("HOST"); present {
		config.Host = host
	}
	if port, present := os.LookupEnv("PORT"); present {
		var err error
		config.Port, err = strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
	}
	if databaseURL, present := os.LookupEnv("DATABASE_URL"); present {
		config.DatabaseURL = databaseURL
	}
	if stateCookieName, present := os.LookupEnv("STATE_COOKIE_NAME"); present {
		config.StateCookieName = stateCookieName
	}
	if stateCookieExpiryString, present := os.LookupEnv("STATE_COOKIE_EXPIRY"); present {
		stateCookieExpiry, err := time.ParseDuration(stateCookieExpiryString)
		if err != nil {
			return nil, err
		}
		config.StateCookieExpiry = stateCookieExpiry
	}
	if sessionCookieName, present := os.LookupEnv("STATE_COOKIE_NAME"); present {
		config.SessionCookieName = sessionCookieName
	}
	if sessionCookieExpiryString, present := os.LookupEnv("STATE_COOKIE_EXPIRY"); present {
		sessionCookieExpiry, err := time.ParseDuration(sessionCookieExpiryString)
		if err != nil {
			return nil, err
		}
		config.SessionCookieExpiry = sessionCookieExpiry
	}

	return &config, nil
}

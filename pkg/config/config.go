package config

import (
	"os"
	"strconv"
)

// Config holds the settings for a Server
type Config struct {
	ClientID     string
	ClientSecret string
	Protocol     string
	Host         string
	Port         int
	DatabaseURL  string
}

// NewConfig returns a Config struct with sane defaults and env variable overrides
func NewConfig() (*Config, error) {
	config := Config{
		ClientID:     "",
		ClientSecret: "",
		Protocol:     "http://",
		Host:         "localhost",
		Port:         8080,
		DatabaseURL:  "",
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
		return nil, err
	}
	if databaseURL, present := os.LookupEnv("DATABASE_URL"); present {
		config.DatabaseURL = databaseURL
	}

	return &config, nil
}

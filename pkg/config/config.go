package config

import (
	"os"
	"strconv"
)

// TODO move this into its own package?

type Config struct {
	ClientID     string
	ClientSecret string
	Protocol     string
	Host         string
	Port         int
	DatabaseURL  string
}

func NewConfig() *Config {
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
		// TODO handle this error
		config.Port, _ = strconv.Atoi(port)
	}
	if databaseURL, present := os.LookupEnv("DATABASE_URL"); present {
		config.DatabaseURL = databaseURL
	}

	return &config
}

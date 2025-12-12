package config

import (
	"errors"
	"os"
)

// Config holds the application configuration
type Config struct {
	GoogleAPIKey string
	Port         string
	AppName      string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GOOGLE_API_KEY environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "agent-go-ag-ui"
	}

	return &Config{
		GoogleAPIKey: apiKey,
		Port:         port,
		AppName:      appName,
	}, nil
}

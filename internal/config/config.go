// Package config manages configuration for the podread CLI.
package config

import (
	"os"
	"path/filepath"
)

const (
	// DefaultBaseURL is the production API base URL.
	DefaultBaseURL = "https://podread.app"

	// EnvAPIURL is the environment variable for overriding the API base URL.
	EnvAPIURL = "PODREAD_API_URL"

	// configDirName is the subdirectory under the user's config home.
	configDirName = "podread"

	// tokenFileName is the name of the token file within the config directory.
	tokenFileName = "token"
)

// Dir returns the path to the podread config directory (~/.config/podread/).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", configDirName), nil
}

// TokenPath returns the full path to the token file.
func TokenPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, tokenFileName), nil
}

// BaseURL returns the API base URL. It checks the PODREAD_API_URL environment
// variable first, falling back to the default production URL.
func BaseURL() string {
	if url := os.Getenv(EnvAPIURL); url != "" {
		return url
	}
	return DefaultBaseURL
}

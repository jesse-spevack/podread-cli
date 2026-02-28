// Package config manages configuration for the podread CLI.
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	if u := os.Getenv(EnvAPIURL); u != "" {
		return u
	}
	return DefaultBaseURL
}

// ValidateBaseURL checks that the API base URL uses HTTPS, allowing HTTP only
// for localhost development. Returns an error if the URL is insecure.
func ValidateBaseURL() error {
	u := BaseURL()

	parsed, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid API URL %q: %w", u, err)
	}

	if parsed.Scheme == "https" {
		return nil
	}

	if parsed.Scheme == "http" {
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" {
			return nil
		}
	}

	return fmt.Errorf("API URL must use HTTPS (got %q); HTTP is only allowed for localhost", strings.TrimRight(u, "/"))
}

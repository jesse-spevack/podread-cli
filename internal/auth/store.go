// Package auth handles token storage for the podread CLI.
package auth

import (
	"errors"
	"os"
	"strings"

	"github.com/jspevack/podread-cli/internal/config"
)

// ErrNoToken is returned when no authentication token is stored.
var ErrNoToken = errors.New("no authentication token found")

// LoadToken reads the stored bearer token from disk.
// Returns ErrNoToken if no token file exists.
func LoadToken() (string, error) {
	path, err := config.TokenPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoToken
		}
		return "", err
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", ErrNoToken
	}
	return token, nil
}

// SaveToken writes the bearer token to disk, creating the config directory
// if necessary.
func SaveToken(token string) error {
	dir, err := config.Dir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path, err := config.TokenPath()
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(token+"\n"), 0600)
}

// DeleteToken removes the stored token file.
// Returns nil if the file does not exist (idempotent).
func DeleteToken() error {
	path, err := config.TokenPath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

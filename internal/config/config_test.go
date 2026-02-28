package config

import (
	"os"
	"strings"
	"testing"
)

func TestBaseURL_Default(t *testing.T) {
	t.Setenv(EnvAPIURL, "")

	got := BaseURL()
	if got != DefaultBaseURL {
		t.Errorf("BaseURL() = %q, want %q", got, DefaultBaseURL)
	}
}

func TestBaseURL_Override(t *testing.T) {
	t.Setenv(EnvAPIURL, "https://staging.podread.app")

	got := BaseURL()
	if got != "https://staging.podread.app" {
		t.Errorf("BaseURL() = %q, want %q", got, "https://staging.podread.app")
	}
}

func TestValidateBaseURL_HTTPS(t *testing.T) {
	t.Setenv(EnvAPIURL, "https://podread.app")

	if err := ValidateBaseURL(); err != nil {
		t.Errorf("HTTPS URL should be valid, got: %v", err)
	}
}

func TestValidateBaseURL_Localhost(t *testing.T) {
	tests := []struct {
		url  string
		ok   bool
	}{
		{"http://localhost:3000", true},
		{"http://127.0.0.1:3000", true},
		{"http://localhost", true},
		{"http://127.0.0.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Setenv(EnvAPIURL, tt.url)
			err := ValidateBaseURL()
			if tt.ok && err != nil {
				t.Errorf("expected %q to be valid, got: %v", tt.url, err)
			}
			if !tt.ok && err == nil {
				t.Errorf("expected %q to be invalid", tt.url)
			}
		})
	}
}

func TestValidateBaseURL_RejectsHTTP(t *testing.T) {
	tests := []string{
		"http://podread.app",
		"http://example.com",
		"http://192.168.1.1:3000",
	}

	for _, u := range tests {
		t.Run(u, func(t *testing.T) {
			t.Setenv(EnvAPIURL, u)
			err := ValidateBaseURL()
			if err == nil {
				t.Errorf("expected %q to be rejected", u)
			}
			if err != nil && !strings.Contains(err.Error(), "HTTPS") {
				t.Errorf("error should mention HTTPS, got: %v", err)
			}
		})
	}
}

func TestValidateBaseURL_DefaultIsValid(t *testing.T) {
	t.Setenv(EnvAPIURL, "")

	if err := ValidateBaseURL(); err != nil {
		t.Errorf("default URL should be valid, got: %v", err)
	}
}

func TestDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { os.Setenv("HOME", origHome) }()

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if !strings.HasSuffix(dir, ".config/podread") {
		t.Errorf("Dir() = %q, want suffix .config/podread", dir)
	}
}

func TestTokenPath(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { os.Setenv("HOME", origHome) }()

	path, err := TokenPath()
	if err != nil {
		t.Fatalf("TokenPath() error: %v", err)
	}
	if !strings.HasSuffix(path, ".config/podread/token") {
		t.Errorf("TokenPath() = %q, want suffix .config/podread/token", path)
	}
}

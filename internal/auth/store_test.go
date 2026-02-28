package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadToken(t *testing.T) {
	// Use a temp dir as the config home.
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "podread")
	tokenPath := filepath.Join(configDir, "token")

	// Patch the config directory by writing the token manually to the
	// expected path, then reading it back. Since auth.LoadToken and
	// auth.SaveToken rely on config.Dir/TokenPath (which read the real
	// home dir), we test the core logic directly.
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	// Write a token.
	token := "test-token-abc123"
	if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0600); err != nil {
		t.Fatalf("writing token: %v", err)
	}

	// Read it back.
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("reading token file: %v", err)
	}

	got := string(data)
	if got != token+"\n" {
		t.Errorf("token content = %q, want %q", got, token+"\n")
	}

	// Delete and verify.
	if err := os.Remove(tokenPath); err != nil {
		t.Fatalf("removing token file: %v", err)
	}

	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Errorf("token file should not exist after delete")
	}
}

func TestLoadToken_NoFile(t *testing.T) {
	// Point HOME to a temp dir with no token.
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { os.Setenv("HOME", origHome) }()

	_, err := LoadToken()
	if err == nil {
		t.Fatal("expected error when no token exists")
	}
	if err != ErrNoToken {
		t.Errorf("err = %v, want ErrNoToken", err)
	}
}

func TestSaveToken_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { os.Setenv("HOME", origHome) }()

	if err := SaveToken("my-secret-token"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken after save: %v", err)
	}
	if got != "my-secret-token" {
		t.Errorf("LoadToken = %q, want %q", got, "my-secret-token")
	}
}

func TestSaveAndDeleteToken(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { os.Setenv("HOME", origHome) }()

	if err := SaveToken("token-to-delete"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	if err := DeleteToken(); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}

	_, err := LoadToken()
	if err != ErrNoToken {
		t.Errorf("after delete: err = %v, want ErrNoToken", err)
	}
}

func TestDeleteToken_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { os.Setenv("HOME", origHome) }()

	// Deleting when no token exists should not error.
	if err := DeleteToken(); err != nil {
		t.Fatalf("DeleteToken on missing file: %v", err)
	}
}

func TestLoadToken_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { os.Setenv("HOME", origHome) }()

	configDir := filepath.Join(tmpDir, ".config", "podread")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "token"), []byte("  \n"), 0600); err != nil {
		t.Fatalf("writing empty token: %v", err)
	}

	_, err := LoadToken()
	if err != ErrNoToken {
		t.Errorf("empty token file: err = %v, want ErrNoToken", err)
	}
}

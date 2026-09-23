package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jspevack/podread-cli/internal/auth"
	"github.com/jspevack/podread-cli/internal/config"
	"github.com/spf13/cobra"
)

// serveJSON points the CLI at a server that answers every request with body.
func serveJSON(t *testing.T, body string) {
	t.Helper()
	serveStatus(t, http.StatusOK, body)
}

// serveStatus points the CLI at a server that answers every request with status and body.
func serveStatus(t *testing.T, status int, body string) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	t.Setenv("HOME", t.TempDir())
	t.Setenv(config.EnvAPIURL, server.URL)
	if err := auth.SaveToken("test-token"); err != nil {
		t.Fatalf("saving the token: %v", err)
	}
}

// captureOutput collects what a command writes to stdout.
func captureOutput(t *testing.T, cmd *cobra.Command) *bytes.Buffer {
	t.Helper()

	var out bytes.Buffer
	cmd.SetOut(&out)
	t.Cleanup(func() { cmd.SetOut(nil) })
	return &out
}

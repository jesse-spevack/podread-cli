// Package cmd contains the cobra commands for the podread CLI.
package cmd

import (
	"os"

	"github.com/jspevack/podread-cli/internal/api"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "podread",
	Short:   "CLI for podread.app — text to speech to your personal podcast feed",
	Long: `podread is a command-line interface for podread.app.
Turn text into podcast episodes delivered to your personal RSS feed.`,
	Version: api.Version,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

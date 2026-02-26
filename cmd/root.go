// Package cmd contains the cobra commands for the podread CLI.
package cmd

import (
	"os"

	"github.com/jspevack/podread-cli/internal/api"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "podread",
	Short:   "CLI for podread.app — AI-powered podcast summaries and transcripts",
	Long: `podread is a command-line interface for podread.app.
Create podcast episode summaries and transcripts from your terminal.`,
	Version: api.Version,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

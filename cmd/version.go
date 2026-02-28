package cmd

import (
	"fmt"

	"github.com/jspevack/podread-cli/internal/api"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of podread",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("podread %s (commit: %s, built: %s)\n", api.Version, api.Commit, api.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

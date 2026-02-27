package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(feedCmd)
}

// feedResponse is the response from GET /api/v1/feed.
type feedResponse struct {
	FeedURL string `json:"feed_url"`
}

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Show your RSS feed URL",
	Long:  `Display the personal RSS feed URL for your podread.app account.`,
	RunE:  runFeed,
}

func runFeed(cmd *cobra.Command, args []string) error {
	client, err := authenticatedClient()
	if err != nil {
		return err
	}

	var resp feedResponse
	if err := client.Get("/api/v1/feed", &resp); err != nil {
		return fmt.Errorf("fetching feed URL: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), resp.FeedURL)
	return nil
}

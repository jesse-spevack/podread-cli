package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/jspevack/podread-cli/internal/api"
	"github.com/jspevack/podread-cli/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(episodeCmd)
	episodeCmd.AddCommand(episodeCreateCmd)
	episodeCmd.AddCommand(episodeStatusCmd)
	episodeCmd.AddCommand(episodeListCmd)
	episodeCmd.AddCommand(episodeDeleteCmd)

	// episode create flags
	episodeCreateCmd.Flags().String("url", "", "URL to convert to audio")
	episodeCreateCmd.Flags().String("text", "", "Text to convert to audio")
	episodeCreateCmd.Flags().Bool("stdin", false, "Read text from stdin")
	episodeCreateCmd.Flags().String("title", "", "Episode title")
	episodeCreateCmd.Flags().String("voice", "", "Voice to use (see 'podread voices')")
	episodeCreateCmd.Flags().Bool("no-wait", false, "Do not wait for processing to complete")
	episodeCreateCmd.Flags().Int("timeout", 600, "Maximum seconds to wait for processing")
	episodeCreateCmd.Flags().Bool("json", false, "Output as JSON")

	// episode status flags
	episodeStatusCmd.Flags().Bool("json", false, "Output as JSON")

	// episode list flags
	episodeListCmd.Flags().Int("limit", 10, "Maximum number of episodes to list")
	episodeListCmd.Flags().Bool("json", false, "Output as JSON")
}

// --- types ---

// episodeCreateRequest is the request body for POST /api/v1/episodes.
type episodeCreateRequest struct {
	SourceType string `json:"source_type"`
	Text       string `json:"text,omitempty"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	Voice      string `json:"voice,omitempty"`
}

// episodeResponse is the response from the episodes API.
type episodeResponse struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	SourceType      string `json:"source_type,omitempty"`
	SourceURL       string `json:"source_url,omitempty"`
	DurationSeconds int    `json:"duration_seconds,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// episodeListResponse is the response from GET /api/v1/episodes.
type episodeListResponse struct {
	Episodes []episodeResponse `json:"episodes"`
}

// --- parent command ---

var episodeCmd = &cobra.Command{
	Use:     "episode",
	Aliases: []string{"ep"},
	Short:   "Manage episodes",
	Long:    `Create, list, and manage your podcast episodes.`,
}

// --- episode create ---

var episodeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new episode from text or a URL",
	Long: `Create a new podcast episode by providing text or a URL.

Exactly one of --url, --text, or --stdin must be provided.

By default, the command waits for processing to complete, printing progress
updates to stderr and the final result to stdout. Use --no-wait to return
immediately after the episode is created.`,
	RunE: runEpisodeCreate,
}

func runEpisodeCreate(cmd *cobra.Command, args []string) error {
	urlFlag, _ := cmd.Flags().GetString("url")
	textFlag, _ := cmd.Flags().GetString("text")
	stdinFlag, _ := cmd.Flags().GetBool("stdin")
	titleFlag, _ := cmd.Flags().GetString("title")
	voiceFlag, _ := cmd.Flags().GetString("voice")
	noWaitFlag, _ := cmd.Flags().GetBool("no-wait")
	timeoutFlag, _ := cmd.Flags().GetInt("timeout")
	jsonFlag, _ := cmd.Flags().GetBool("json")

	// Determine wait behavior.
	shouldWait := !noWaitFlag

	// Validate exactly one source is provided.
	sourceCount := 0
	if urlFlag != "" {
		sourceCount++
	}
	if textFlag != "" {
		sourceCount++
	}
	if stdinFlag {
		sourceCount++
	}
	if sourceCount != 1 {
		return fmt.Errorf("exactly one of --url, --text, or --stdin must be provided")
	}

	// Build the request.
	reqBody := episodeCreateRequest{
		Title: titleFlag,
		Voice: voiceFlag,
	}

	if urlFlag != "" {
		reqBody.SourceType = "url"
		reqBody.URL = urlFlag
	} else {
		reqBody.SourceType = "text"
		if stdinFlag {
			const maxStdinSize = 1024 * 1024 // 1 MB
			data, err := io.ReadAll(io.LimitReader(os.Stdin, maxStdinSize))
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			reqBody.Text = string(data)
			if reqBody.Text == "" {
				return fmt.Errorf("stdin was empty, provide text to convert")
			}
		} else {
			reqBody.Text = textFlag
		}
	}

	client, err := authenticatedClient()
	if err != nil {
		return err
	}

	var ep episodeResponse
	if err := client.Post("/api/v1/episodes", reqBody, &ep); err != nil {
		return fmt.Errorf("creating episode: %w", err)
	}

	if !shouldWait {
		return printEpisode(cmd, ep, jsonFlag)
	}

	// Poll until complete or failed.
	fmt.Fprintf(cmd.ErrOrStderr(), "Processing episode %s...\n", ep.ID)

	deadline := time.Now().Add(time.Duration(timeoutFlag) * time.Second)
	consecutiveErrors := 0
	const maxConsecutiveErrors = 5

	lastStatus := ep.Status
	for ep.Status != "complete" && ep.Status != "failed" {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %d seconds waiting for processing; check status with: podread episode status %s", timeoutFlag, ep.ID)
		}
		time.Sleep(3 * time.Second)

		if err := client.Get("/api/v1/episodes/"+url.PathEscape(ep.ID), &ep); err != nil {
			var apiErr *api.APIError
			if errors.As(err, &apiErr) {
				// Server returned an error status — this is not transient, bail.
				return fmt.Errorf("checking episode status: %w", err)
			}
			// Network/transient error — log and retry.
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				return fmt.Errorf("checking episode status after %d consecutive errors: %w", maxConsecutiveErrors, err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: connection error, retrying... (%d/%d)\n", consecutiveErrors, maxConsecutiveErrors)
			continue
		}
		consecutiveErrors = 0 // Reset on success

		if ep.Status != lastStatus {
			fmt.Fprintf(cmd.ErrOrStderr(), "Status: %s\n", ep.Status)
			lastStatus = ep.Status
		}
	}

	if ep.Status == "failed" {
		msg := "episode processing failed"
		if ep.ErrorMessage != "" {
			msg += ": " + ep.ErrorMessage
		}
		return fmt.Errorf(msg)
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "Done!")
	return printEpisode(cmd, ep, jsonFlag)
}

// --- episode status ---

var episodeStatusCmd = &cobra.Command{
	Use:   "status <id>",
	Short: "Show the processing status of an episode",
	Long:  `Display the current processing status and details of an episode.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEpisodeStatus,
}

func runEpisodeStatus(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	client, err := authenticatedClient()
	if err != nil {
		return err
	}

	var ep episodeResponse
	if err := client.Get("/api/v1/episodes/"+url.PathEscape(args[0]), &ep); err != nil {
		return fmt.Errorf("fetching episode: %w", err)
	}

	return printEpisode(cmd, ep, jsonFlag)
}

// --- episode list ---

var episodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your episodes",
	Long:  `Display a list of your podcast episodes.`,
	RunE:  runEpisodeList,
}

func runEpisodeList(cmd *cobra.Command, args []string) error {
	limitFlag, _ := cmd.Flags().GetInt("limit")
	jsonFlag, _ := cmd.Flags().GetBool("json")

	client, err := authenticatedClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/episodes?limit=%d", limitFlag)

	var listResp episodeListResponse
	if err := client.Get(path, &listResp); err != nil {
		return fmt.Errorf("listing episodes: %w", err)
	}

	if jsonFlag {
		data, err := json.MarshalIndent(listResp.Episodes, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	if len(listResp.Episodes) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No episodes found")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s  %-12s  %s\n", "ID", "STATUS", "TITLE")
	for _, ep := range listResp.Episodes {
		title := ep.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s  %-12s  %s\n", ep.ID, ep.Status, title)
	}

	return nil
}

// --- episode delete ---

var episodeDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an episode",
	Long:  `Permanently delete an episode.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEpisodeDelete,
}

func runEpisodeDelete(cmd *cobra.Command, args []string) error {
	client, err := authenticatedClient()
	if err != nil {
		return err
	}

	if err := client.Delete("/api/v1/episodes/" + url.PathEscape(args[0])); err != nil {
		return fmt.Errorf("deleting episode: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Deleted episode %s\n", args[0])
	return nil
}

// --- helpers ---

// authenticatedClient loads the stored token and returns an API client.
// It prints a helpful message and exits if not logged in.
func authenticatedClient() (*api.Client, error) {
	token, err := auth.LoadToken()
	if err != nil {
		if errors.Is(err, auth.ErrNoToken) {
			return nil, fmt.Errorf("not logged in, run 'podread auth login' first")
		}
		return nil, fmt.Errorf("reading token: %w", err)
	}
	return api.NewClient(token), nil
}

// printEpisode outputs an episode in human-friendly or JSON format.
func printEpisode(cmd *cobra.Command, ep episodeResponse, asJSON bool) error {
	if asJSON {
		data, err := json.MarshalIndent(ep, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	title := ep.Title
	if title == "" {
		title = "(untitled)"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "ID:     %s\n", ep.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "Title:  %s\n", title)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", ep.Status)

	if ep.DurationSeconds > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Duration: %ds\n", ep.DurationSeconds)
	}
	if ep.ErrorMessage != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Error:  %s\n", ep.ErrorMessage)
	}

	return nil
}

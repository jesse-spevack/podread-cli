package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(voicesCmd)

	voicesCmd.Flags().Bool("json", false, "Output as JSON")
}

// voiceResponse is a single voice from the API.
type voiceResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Accent string `json:"accent,omitempty"`
	Gender string `json:"gender,omitempty"`
}

// voicesListResponse is the response from GET /api/v1/voices.
type voicesListResponse struct {
	Voices []voiceResponse `json:"voices"`
}

var voicesCmd = &cobra.Command{
	Use:   "voices",
	Short: "List available voices",
	Long:  `Display the available voices for text-to-speech synthesis.`,
	RunE:  runVoices,
}

func runVoices(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	client, err := authenticatedClient()
	if err != nil {
		return err
	}

	var resp voicesListResponse
	if err := client.Get("/api/v1/voices", &resp); err != nil {
		return fmt.Errorf("listing voices: %w", err)
	}

	if jsonFlag {
		data, err := json.MarshalIndent(resp.Voices, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	if len(resp.Voices) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No voices available")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%-12s %-10s  %-10s  %s\n", "ID", "NAME", "ACCENT", "GENDER")
	for _, v := range resp.Voices {
		line := fmt.Sprintf("%-12s %-10s", v.ID, v.Name)
		if v.Accent != "" {
			line += fmt.Sprintf("  %-10s", v.Accent)
		}
		if v.Gender != "" {
			line += fmt.Sprintf("  %s", v.Gender)
		}
		fmt.Fprintln(cmd.OutOrStdout(), line)
	}

	return nil
}

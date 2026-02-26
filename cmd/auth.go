package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jspevack/podread-cli/internal/api"
	"github.com/jspevack/podread-cli/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with podread.app",
	Long:  `Log in, log out, and check your authentication status with podread.app.`,
}

// --- login ---

// deviceCodeResponse is the response from POST /api/v1/auth/device-codes.
type deviceCodeResponse struct {
	Code            string `json:"code"`
	DeviceCode      string `json:"device_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresAt       string `json:"expires_at"`
	Interval        int    `json:"interval"`
}

// deviceTokenRequest is the request body for POST /api/v1/auth/device-tokens.
type deviceTokenRequest struct {
	DeviceCode string `json:"device_code"`
}

// deviceTokenResponse is the response from POST /api/v1/auth/device-tokens.
type deviceTokenResponse struct {
	Status    string `json:"status"`
	Token     string `json:"token"`
	UserEmail string `json:"user_email"`
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to podread.app using a device code",
	Long: `Authenticate with podread.app using the device code flow.
Opens a verification URL and waits for you to enter the code in your browser.`,
	RunE: runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	client := api.NewClient("")

	// Step 1: Request a device code.
	var codeResp deviceCodeResponse
	if err := client.Post("/api/v1/auth/device-codes", nil, &codeResp); err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}

	// Step 2: Display the code and URL to the user.
	fmt.Fprintf(cmd.OutOrStdout(), "Open this URL: %s\n", codeResp.VerificationURL)
	fmt.Fprintf(cmd.OutOrStdout(), "Enter code: %s\n", codeResp.Code)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Waiting for authorization...")

	// Step 3: Poll for the token.
	interval := time.Duration(codeResp.Interval) * time.Second
	if interval < 1*time.Second {
		interval = 5 * time.Second
	}

	expiresAt, err := time.Parse(time.RFC3339, codeResp.ExpiresAt)
	if err != nil {
		// If we can't parse the expiry, use a reasonable default.
		expiresAt = time.Now().Add(15 * time.Minute)
	}

	tokenReq := deviceTokenRequest{DeviceCode: codeResp.DeviceCode}

	for {
		if time.Now().After(expiresAt) {
			return fmt.Errorf("device code expired, please run 'podread auth login' again")
		}

		time.Sleep(interval)

		var tokenResp deviceTokenResponse
		if err := client.Post("/api/v1/auth/device-tokens", tokenReq, &tokenResp); err != nil {
			// API errors during polling are not fatal — the server may return
			// an error status while the code is still pending.
			var apiErr *api.APIError
			if errors.As(err, &apiErr) {
				continue
			}
			return fmt.Errorf("polling for token: %w", err)
		}

		if tokenResp.Status == "pending" {
			continue
		}

		if tokenResp.Token != "" {
			if err := auth.SaveToken(tokenResp.Token); err != nil {
				return fmt.Errorf("saving token: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s\n", tokenResp.UserEmail)
			return nil
		}
	}
}

// --- logout ---

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of podread.app",
	Long:  `Remove the stored authentication token.`,
	RunE:  runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	if err := auth.DeleteToken(); err != nil {
		return fmt.Errorf("deleting token: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Logged out")
	return nil
}

// --- status ---

// authStatusResponse is the response from GET /api/v1/auth/status.
type authStatusResponse struct {
	Email string `json:"email"`
	Tier  string `json:"tier"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Long:  `Check whether you are logged in and display your account information.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	token, err := auth.LoadToken()
	if err != nil {
		if errors.Is(err, auth.ErrNoToken) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Not logged in")
			os.Exit(1)
		}
		return fmt.Errorf("reading token: %w", err)
	}

	client := api.NewClient(token)
	var statusResp authStatusResponse
	if err := client.Get("/api/v1/auth/status", &statusResp); err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Session expired, run 'podread auth login'")
			os.Exit(1)
		}
		return fmt.Errorf("checking auth status: %w", err)
	}

	msg := fmt.Sprintf("Logged in as %s", statusResp.Email)
	if statusResp.Tier != "" {
		msg += fmt.Sprintf(" (%s)", statusResp.Tier)
	}
	fmt.Fprintln(cmd.OutOrStdout(), msg)
	return nil
}

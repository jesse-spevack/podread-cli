package cmd

import (
	"errors"
	"fmt"
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

// deviceCodeResponse is the response from POST /api/v1/auth/device_codes.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	VerificationURL string `json:"verification_url"`
	UserCode        string `json:"user_code"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// deviceTokenRequest is the request body for POST /api/v1/auth/device_tokens.
type deviceTokenRequest struct {
	DeviceCode string `json:"device_code"`
}

// deviceTokenResponse is the response from POST /api/v1/auth/device_tokens.
type deviceTokenResponse struct {
	AccessToken string `json:"access_token"`
	UserEmail   string `json:"user_email"`
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
	if err := client.Post("/api/v1/auth/device_codes", nil, &codeResp); err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}

	// Step 2: Display the code and URL to the user.
	fmt.Fprintf(cmd.OutOrStdout(), "Open this URL: %s\n", codeResp.VerificationURL)
	fmt.Fprintf(cmd.OutOrStdout(), "Enter code: %s\n", codeResp.UserCode)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Waiting for authorization...")

	// Step 3: Poll for the token.
	interval := time.Duration(codeResp.Interval) * time.Second
	if interval < 1*time.Second {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(codeResp.ExpiresIn) * time.Second)

	tokenReq := deviceTokenRequest{DeviceCode: codeResp.DeviceCode}
	consecutiveErrors := 0
	const maxConsecutiveErrors = 5

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("device code expired, please run 'podread auth login' again")
		}

		time.Sleep(interval)

		var tokenResp deviceTokenResponse
		if err := client.Post("/api/v1/auth/device_tokens", tokenReq, &tokenResp); err != nil {
			var apiErr *api.APIError
			if errors.As(err, &apiErr) {
				if apiErr.StatusCode == 400 && apiErr.Message == "authorization_pending" {
					consecutiveErrors = 0
					continue
				}
				if apiErr.StatusCode == 429 {
					fmt.Fprintln(cmd.ErrOrStderr(), "Rate limited, backing off...")
					time.Sleep(interval)
					consecutiveErrors = 0
					continue
				}
				return fmt.Errorf("polling for token: %w", err)
			}
			// Network/transient error — retry.
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				return fmt.Errorf("polling for token after %d consecutive errors: %w", maxConsecutiveErrors, err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Connection error, retrying... (%d/%d)\n", consecutiveErrors, maxConsecutiveErrors)
			continue
		}
		consecutiveErrors = 0

		if tokenResp.AccessToken != "" {
			if err := auth.SaveToken(tokenResp.AccessToken); err != nil {
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
			return fmt.Errorf("not logged in")
		}
		return fmt.Errorf("reading token: %w", err)
	}

	client := api.NewClient(token)
	var statusResp authStatusResponse
	if err := client.Get("/api/v1/auth/status", &statusResp); err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403) {
			return fmt.Errorf("session expired, run 'podread auth login'")
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

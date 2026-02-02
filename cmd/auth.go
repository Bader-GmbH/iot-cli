package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/Bader-GmbH/iot-cli/internal/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the platform",
	Long:  `Manage authentication with the Bader IoT Platform.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the platform",
	Long: `Authenticate with the Bader IoT Platform.

This will open your browser to complete authentication.
Once you've logged in, the CLI will automatically receive your credentials.`,
	RunE: runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and clear stored credentials",
	RunE:  runLogout,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	// Login flags
	loginCmd.Flags().Duration("timeout", 5*time.Minute, "Timeout for authentication")
}

func runLogin(cmd *cobra.Command, args []string) error {
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.iot.bader.solutions"
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create session auth client
	sessionAuth := auth.NewSessionAuth(apiURL)

	fmt.Println("Creating authentication session...")

	// Create a new CLI session
	session, err := sessionAuth.CreateSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	fmt.Println()
	fmt.Println("Opening browser to complete authentication...")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n", session.LoginURL)
	fmt.Println()

	// Open browser
	if err := openBrowser(session.LoginURL); err != nil {
		fmt.Printf("Warning: Could not open browser: %v\n", err)
	}

	fmt.Println("Waiting for authentication...")

	// Poll for completion
	status, err := sessionAuth.PollSession(ctx, session.SessionID, 2*time.Second, timeout)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Store credentials
	tokenStore, err := auth.NewTokenStore()
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	creds := &auth.Credentials{
		AccessToken:  status.AccessToken,
		IDToken:      status.IDToken,
		RefreshToken: status.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(status.ExpiresIn) * time.Second),
	}

	if err := tokenStore.Save(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	// Reload to get parsed user info
	creds, _ = tokenStore.Load()

	fmt.Println()
	fmt.Println("✓ Successfully logged in!")
	if creds.Email != "" {
		fmt.Printf("  Email:    %s\n", creds.Email)
	}
	if creds.TenantID != "" {
		fmt.Printf("  Tenant:   %s\n", creds.TenantID)
	}
	fmt.Println()

	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	tokenStore, err := auth.NewTokenStore()
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	if err := tokenStore.Delete(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	fmt.Println("✓ Logged out successfully")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	tokenStore, err := auth.NewTokenStore()
	if err != nil {
		return fmt.Errorf("failed to initialize token store: %w", err)
	}

	creds, err := tokenStore.Load()
	if err != nil {
		fmt.Println("Not logged in")
		fmt.Println()
		fmt.Println("Run 'iot auth login' to authenticate.")
		return nil
	}

	// Check expiration
	expired := time.Now().After(creds.ExpiresAt)

	if IsJSON() {
		return outputJSON(map[string]interface{}{
			"loggedIn":  !expired,
			"email":     creds.Email,
			"tenantId":  creds.TenantID,
			"expiresAt": creds.ExpiresAt,
		})
	}

	if expired {
		fmt.Println("Session expired")
		fmt.Println()
		fmt.Println("Run 'iot auth login' to re-authenticate.")
		return nil
	}

	fmt.Println("✓ Logged in")
	if creds.Email != "" {
		fmt.Printf("  Email:      %s\n", creds.Email)
	}
	if creds.TenantID != "" {
		fmt.Printf("  Tenant:     %s\n", creds.TenantID)
	}
	fmt.Printf("  Expires:    %s\n", creds.ExpiresAt.Format(time.RFC3339))
	fmt.Println()

	return nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

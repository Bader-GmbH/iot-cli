package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/Bader-GmbH/iot-cli/internal/api"
	"github.com/Bader-GmbH/iot-cli/internal/terminal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var sshCmd = &cobra.Command{
	Use:   "ssh <device>",
	Short: "Open a terminal session to a device",
	Long: `Open an interactive terminal session to a remote device.

The device can be specified by name or ID. The device must be online
and approved for the connection to succeed.

Examples:
  iot ssh my-device          # Connect by device name
  iot ssh abc123             # Connect by device ID
  iot ssh ec2-instance       # Connect to EC2 instance`,
	Args: cobra.ExactArgs(1),
	RunE: runSSH,
}

func init() {
	rootCmd.AddCommand(sshCmd)
}

func runSSH(cmd *cobra.Command, args []string) error {
	deviceID := args[0]

	// Create API client
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.iot.bader.solutions"
	}

	client, err := api.NewClient(apiURL)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Check if device is online
	if !IsQuiet() {
		fmt.Fprintf(os.Stderr, "Connecting to %s...\n", deviceID)
	}

	// Create terminal session
	session, err := client.CreateTerminalSession(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Get credentials for WebSocket auth
	accessToken, tenantID, err := client.GetCredentials()
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Connect to WebSocket
	termSession, err := terminal.Connect(ctx, client.GetBaseURL(), session.SessionID, accessToken, tenantID)
	if err != nil {
		// Clean up the session on error
		_ = client.CloseTerminalSession(ctx, session.SessionID)
		return fmt.Errorf("failed to connect: %w", err)
	}

	if !IsQuiet() {
		fmt.Fprintf(os.Stderr, "Connected. Press Ctrl+D to exit.\n\n")
	}

	// Run the terminal session (blocks until closed)
	if err := termSession.Run(); err != nil {
		return fmt.Errorf("session error: %w", err)
	}

	// Clean up
	_ = client.CloseTerminalSession(ctx, session.SessionID)

	if !IsQuiet() {
		fmt.Fprintf(os.Stderr, "\nConnection closed.\n")
	}

	return nil
}

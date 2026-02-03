package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Bader-GmbH/iot-cli/internal/config"
	"github.com/Bader-GmbH/iot-cli/internal/update"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update iot-cli to the latest version",
	Long: `Check for and install the latest version of iot-cli.

This command will:
  1. Check GitHub releases for the latest version
  2. Download the appropriate binary for your system
  3. Replace the current binary with the new one

Examples:
  iot update           # Update to latest version
  iot update --check   # Only check, don't install`,
	RunE: runUpdate,
}

var (
	checkOnly bool
)

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, don't install")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	checker := update.NewChecker(Version, configDir)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	info, err := checker.CheckForUpdate(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !info.IsUpdateAvailable() {
		fmt.Println("You're already running the latest version!")
		return nil
	}

	fmt.Printf("\nNew version available: %s -> %s\n", info.CurrentVersion, info.LatestVersion)
	fmt.Printf("Released: %s\n", info.PublishedAt.Format("2006-01-02"))

	if info.ReleaseNotes != "" {
		fmt.Println("\nRelease notes:")
		// Print first 500 chars of release notes
		notes := info.ReleaseNotes
		if len(notes) > 500 {
			notes = notes[:500] + "..."
		}
		fmt.Println(notes)
	}

	if checkOnly {
		fmt.Printf("\nRun 'iot update' to install the new version.\n")
		return nil
	}

	if info.DownloadURL == "" {
		fmt.Printf("\nNo binary available for your platform.\n")
		fmt.Printf("Visit %s to download manually.\n", info.ReleaseURL)
		return nil
	}

	fmt.Println("\nDownloading update...")

	// Download with progress
	tmpPath, err := checker.DownloadUpdate(ctx, info.DownloadURL, func(downloaded, total int64) {
		if total > 0 {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rDownloading... %.1f%%", percent)
		} else {
			fmt.Printf("\rDownloading... %d bytes", downloaded)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tmpPath) // Clean up on error

	fmt.Println("\nInstalling update...")

	if err := update.ApplyUpdate(tmpPath); err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}

	// Clear the cache so next run doesn't show update notification
	_ = checker.ClearCache()

	fmt.Printf("\nSuccessfully updated to version %s!\n", info.LatestVersion)

	return nil
}

// CheckForUpdateInBackground checks for updates and prints a notice if available
// This is called on startup and should not block or fail loudly
func CheckForUpdateInBackground() {
	// Don't check for dev builds or when running in quiet mode
	if Version == "dev" || IsQuiet() {
		return
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return
	}

	checker := update.NewChecker(Version, configDir)

	// Use a short timeout for background check
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	info, err := checker.CheckForUpdateCached(ctx)
	if err != nil || info == nil {
		return
	}

	// Print update notice to stderr so it doesn't interfere with command output
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "A new version of iot-cli is available: %s -> %s\n", info.CurrentVersion, info.LatestVersion)
	fmt.Fprintf(os.Stderr, "Run 'iot update' to upgrade.\n")
	fmt.Fprintf(os.Stderr, "\n")
}
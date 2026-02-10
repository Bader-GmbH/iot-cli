package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/Bader-GmbH/iot-cli/internal/api"
	"github.com/Bader-GmbH/iot-cli/internal/file"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var getCmd = &cobra.Command{
	Use:   "get <device>:<remote-path> [local-path]",
	Short: "Download files from a device",
	Long: `Download files or directories from a device to your local machine.

The remote path uses the format device:path where device is the device ID or name,
and path is an absolute path on the device.

If no local path is specified, files are downloaded to the current directory.
If the local path ends with /, it's treated as a directory.

Examples:
  iot get device-1:/var/log/app.log           # Download to ./app.log
  iot get device-1:/var/log/app.log ./logs/   # Download to ./logs/app.log
  iot get device-1:/etc/myapp/ -r             # Download directory recursively
  iot get device-1:/var/log/app.log --limit 1M  # Limit to 1 MB/s`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().BoolP("recursive", "r", false, "Download directories recursively")
	getCmd.Flags().StringP("limit", "l", "", "Bandwidth limit (e.g., 1M, 500K)")
	getCmd.Flags().Bool("progress", true, "Show progress bar")
	getCmd.Flags().BoolP("force", "f", false, "Overwrite existing files without prompt")
	getCmd.Flags().Bool("dry-run", false, "Show what would be downloaded without actually downloading")
}

func runGet(cmd *cobra.Command, args []string) error {
	// Parse source (device:path)
	source := args[0]
	if !file.IsRemotePath(source) {
		return fmt.Errorf("invalid source %q: expected format device:path", source)
	}

	remote, err := file.ParseRemotePath(source)
	if err != nil {
		return err
	}

	// Parse optional destination
	dest := ""
	if len(args) > 1 {
		dest = args[1]
	}

	// Parse flags
	recursive, _ := cmd.Flags().GetBool("recursive")
	limitStr, _ := cmd.Flags().GetString("limit")
	showProgress, _ := cmd.Flags().GetBool("progress")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	limit, err := file.ParseBandwidthLimit(limitStr)
	if err != nil {
		return err
	}

	// Check if stdout is a terminal for progress bar
	if !isTerminal() {
		showProgress = false
	}

	opts := file.TransferOptions{
		Recursive:    recursive,
		Limit:        limit,
		Quiet:        IsQuiet(),
		Force:        force,
		DryRun:       dryRun,
		ShowProgress: showProgress,
	}

	// Create API client
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.iot.bader.solutions"
	}

	client, err := api.NewClient(apiURL)
	if err != nil {
		return err
	}

	// Print header
	if !IsQuiet() && !dryRun {
		fmt.Printf("Downloading from %s...\n", remote.DeviceID)
	}

	// Execute download
	ctx := context.Background()
	result, err := file.Download(ctx, client, remote.DeviceID, remote.Path, dest, opts)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Print summary
	if !IsQuiet() {
		if dryRun {
			fmt.Printf("\nDry run complete. Would transfer %d file(s).\n", result.FilesTransferred)
		} else {
			fmt.Printf("\nDownloaded %d file(s), %s total\n",
				result.FilesTransferred, file.FormatBytes(result.BytesTransferred))
		}
	}

	// Report errors if any
	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nWarnings (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
	}

	return nil
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

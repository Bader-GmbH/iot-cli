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

var putCmd = &cobra.Command{
	Use:   "put <local-path>... <device>:<remote-path>",
	Short: "Upload files to a device",
	Long: `Upload files or directories from your local machine to a device.

The remote path uses the format device:path where device is the device ID or name,
and path is an absolute path on the device.

If the remote path ends with /, files are uploaded into that directory.
Multiple local files can be specified, and they will all be uploaded to the destination.

Examples:
  iot put ./script.sh device-1:/opt/           # Upload to /opt/script.sh
  iot put ./config/ device-1:/etc/myapp/ -r    # Upload directory recursively
  iot put ./a.txt ./b.txt device-1:/tmp/       # Upload multiple files
  iot put ./data.tar.gz device-1:/tmp/ --limit 500K  # Limit to 500 KB/s`,
	Args: cobra.MinimumNArgs(2),
	RunE: runPut,
}

func init() {
	rootCmd.AddCommand(putCmd)

	putCmd.Flags().BoolP("recursive", "r", false, "Upload directories recursively")
	putCmd.Flags().StringP("limit", "l", "", "Bandwidth limit (e.g., 1M, 500K)")
	putCmd.Flags().Bool("progress", true, "Show progress bar")
	putCmd.Flags().BoolP("force", "f", false, "Overwrite existing files without prompt")
	putCmd.Flags().Bool("dry-run", false, "Show what would be uploaded without actually uploading")
}

func runPut(cmd *cobra.Command, args []string) error {
	// Last argument is the destination (device:path)
	dest := args[len(args)-1]
	localPaths := args[:len(args)-1]

	if !file.IsRemotePath(dest) {
		return fmt.Errorf("invalid destination %q: expected format device:path", dest)
	}

	remote, err := file.ParseRemotePath(dest)
	if err != nil {
		return err
	}

	// Validate local paths exist
	for _, p := range localPaths {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("local path %q not found: %w", p, err)
		}
	}

	// If multiple files, destination must be a directory
	if len(localPaths) > 1 && !file.IsDirectory(remote.Path) {
		return fmt.Errorf("when uploading multiple files, destination must end with / (directory)")
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
		fmt.Printf("Uploading to %s...\n", remote.DeviceID)
	}

	// Execute upload
	ctx := context.Background()
	result, err := file.Upload(ctx, client, localPaths, remote.DeviceID, remote.Path, opts)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	// Print summary
	if !IsQuiet() {
		if dryRun {
			fmt.Printf("\nDry run complete. Would transfer %d file(s).\n", result.FilesTransferred)
		} else {
			fmt.Printf("\nUploaded %d file(s), %s total\n",
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

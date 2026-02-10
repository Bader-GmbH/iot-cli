package cmd

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/Bader-GmbH/iot-cli/internal/config"
	"github.com/Bader-GmbH/iot-cli/internal/update"
	"github.com/spf13/cobra"
)

// Version information (set at build time)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("iot version %s\n", Version)
		if Commit != "unknown" {
			fmt.Printf("  commit:  %s\n", Commit)
		}
		if BuildDate != "unknown" {
			fmt.Printf("  built:   %s\n", BuildDate)
		}
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

		// Check for latest version (non-blocking, with timeout)
		if Version != "dev" {
			checkLatestVersion()
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func checkLatestVersion() {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return
	}

	checker := update.NewChecker(Version, configDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := checker.CheckForUpdate(ctx)
	if err != nil {
		return
	}

	if info.IsUpdateAvailable() {
		fmt.Printf("\n  latest:  %s (run 'iot update' to upgrade)\n", info.LatestVersion)
	} else {
		fmt.Printf("\n  latest:  %s (up to date)\n", info.LatestVersion)
	}
}

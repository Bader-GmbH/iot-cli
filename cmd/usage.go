package cmd

import (
	"context"
	"fmt"

	"github.com/Bader-GmbH/iot-cli/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show usage statistics",
	Long: `Display current data transfer usage for your account.

Examples:
  iot usage              Show current month usage
  iot usage --history    Show usage history
  iot usage --json       Output as JSON`,
	RunE: runUsage,
}

var usageHistoryFlag bool

func init() {
	rootCmd.AddCommand(usageCmd)
	usageCmd.Flags().BoolVar(&usageHistoryFlag, "history", false, "Show usage history")
}

func runUsage(cmd *cobra.Command, args []string) error {
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.iot.bader.solutions"
	}

	client, err := api.NewClient(apiURL)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if usageHistoryFlag {
		return showUsageHistory(ctx, client)
	}

	return showCurrentUsage(ctx, client)
}

func showCurrentUsage(ctx context.Context, client *api.Client) error {
	usage, err := client.GetUsage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	if IsJSON() {
		return outputJSON(usage)
	}

	fmt.Printf("Usage Stats (%s)\n\n", usage.YearMonth)
	fmt.Printf("  Data Transferred:  %s\n", usage.BytesTransferredFormatted)
	fmt.Printf("  Raw Bytes:         %d\n", usage.BytesTransferred)
	fmt.Println()

	return nil
}

func showUsageHistory(ctx context.Context, client *api.Client) error {
	history, err := client.GetUsageHistory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get usage history: %w", err)
	}

	if IsJSON() {
		return outputJSON(history)
	}

	if len(history) == 0 {
		fmt.Println("No usage history found")
		return nil
	}

	fmt.Println("Usage History")
	fmt.Println()

	for _, h := range history {
		fmt.Printf("  %s:  %s\n", h.YearMonth, h.BytesTransferredFormatted)
	}
	fmt.Println()

	return nil
}

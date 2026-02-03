package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Bader-GmbH/iot-cli/internal/api"
	"github.com/Bader-GmbH/iot-cli/internal/output"
	"github.com/Bader-GmbH/iot-cli/pkg/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deviceCmd = &cobra.Command{
	Use:     "device",
	Aliases: []string{"d"},
	Short:   "Manage devices",
	Long:    `List, view, and manage IoT devices in your fleet.`,
}

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices",
	Long: `List all devices in your fleet.

Examples:
  iot device list              List all devices
  iot device list --status online   Filter by status
  iot device list --json       Output as JSON`,
	RunE: runDeviceList,
}

var deviceGetCmd = &cobra.Command{
	Use:   "get <device-id>",
	Short: "Get device details",
	Long: `Get detailed information about a specific device.

Examples:
  iot device get abc123
  iot d get abc123 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDeviceGet,
}

func init() {
	rootCmd.AddCommand(deviceCmd)
	deviceCmd.AddCommand(deviceListCmd)
	deviceCmd.AddCommand(deviceGetCmd)

	// List flags
	deviceListCmd.Flags().String("status", "", "Filter by status (online, offline)")
	deviceListCmd.Flags().String("group", "", "Filter by group name")
}

func runDeviceList(cmd *cobra.Command, args []string) error {
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.iot.bader.solutions"
	}

	client, err := api.NewClient(apiURL)
	if err != nil {
		return err
	}

	ctx := context.Background()
	devices, err := client.ListDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	// Apply filters
	statusFilter, _ := cmd.Flags().GetString("status")
	groupFilter, _ := cmd.Flags().GetString("group")

	if statusFilter != "" || groupFilter != "" {
		devices = filterDevices(devices, statusFilter, groupFilter)
	}

	// Output
	if IsJSON() {
		return outputJSON(devices)
	}

	if len(devices) == 0 {
		fmt.Println("No devices found")
		return nil
	}

	// Table output
	headers := []string{"NAME", "STATUS", "GROUP", "LAST SEEN"}
	var rows [][]string

	for _, d := range devices {
		status := output.StatusIcon(d.Online) + " " + d.OnlineStatus()
		group := ""
		if d.GroupName != nil {
			group = *d.GroupName
		}
		rows = append(rows, []string{
			d.Name,
			status,
			group,
			d.LastSeenString(),
		})
	}

	output.Table(headers, rows)
	return nil
}

func runDeviceGet(cmd *cobra.Command, args []string) error {
	deviceID := args[0]

	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.iot.bader.solutions"
	}

	client, err := api.NewClient(apiURL)
	if err != nil {
		return err
	}

	ctx := context.Background()
	device, err := client.GetDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	if IsJSON() {
		return outputJSON(device)
	}

	// Pretty print device details
	fmt.Printf("Device: %s\n", device.Name)
	fmt.Printf("  ID:          %s\n", device.ID)
	fmt.Printf("  Status:      %s %s\n", output.StatusIcon(device.Online), device.OnlineStatus())
	fmt.Printf("  Approval:    %s\n", device.Status)
	if device.GroupName != nil {
		fmt.Printf("  Group:       %s\n", *device.GroupName)
	}
	fmt.Printf("  Last Seen:   %s\n", device.LastSeenString())
	fmt.Println()

	return nil
}

func filterDevices(devices []models.Device, status, group string) []models.Device {
	var filtered []models.Device

	for _, d := range devices {
		// Status filter
		if status != "" {
			if status == "online" && !d.Online {
				continue
			}
			if status == "offline" && d.Online {
				continue
			}
		}

		// Group filter
		if group != "" {
			if d.GroupName == nil || !strings.EqualFold(*d.GroupName, group) {
				continue
			}
		}

		filtered = append(filtered, d)
	}

	return filtered
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

package cmd

import (
	"fmt"
	"os"

	"github.com/Bader-GmbH/iot-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile        string
	jsonOutputFlag bool
	yamlOutputFlag bool
	quiet          bool
	verbose        bool
)

var rootCmd = &cobra.Command{
	Use:   "iot",
	Short: "Bader IoT Platform CLI",
	Long: `Manage your IoT device fleet from the command line.

The Bader IoT CLI provides terminal access to your devices,
fleet management, and AI-powered diagnostics.

Get started:
  iot auth login     Authenticate with the platform
  iot device list    List all devices
  iot device ssh     SSH into a device`,
	SilenceUsage: true,
}

func Execute() error {
	// Check for updates in background after command completes
	defer CheckForUpdateInBackground()

	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/iot/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&jsonOutputFlag, "json", "j", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&yamlOutputFlag, "yaml", "y", false, "Output in YAML format")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output for debugging")

	// Bind flags to viper (errors only occur if flag doesn't exist, which is a programmer error)
	_ = viper.BindPFlag("output.json", rootCmd.PersistentFlags().Lookup("json"))
	_ = viper.BindPFlag("output.yaml", rootCmd.PersistentFlags().Lookup("yaml"))
	_ = viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := config.GetConfigDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not determine config directory:", err)
			return
		}

		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("IOT")
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

// Helper functions for commands to use
func IsJSON() bool {
	return viper.GetBool("output.json")
}

func IsYAML() bool {
	return viper.GetBool("output.yaml")
}

func IsQuiet() bool {
	return viper.GetBool("quiet")
}

func IsVerbose() bool {
	return viper.GetBool("verbose")
}

package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	AppName = "iot"
)

// Default configuration values
var Defaults = struct {
	APIURL        string
	AuthURL       string
	CognitoRegion string
	ClientID      string
}{
	APIURL:        "https://api.iot.bader.solutions",
	AuthURL:       "https://auth.iot.bader.solutions",
	CognitoRegion: "eu-central-1",
	ClientID:      "", // Will be set based on Cognito app client
}

// GetConfigDir returns the configuration directory path based on OS
func GetConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support/iot
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", AppName)
	case "windows":
		// Windows: %APPDATA%\iot
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		configDir = filepath.Join(appData, AppName)
	default:
		// Linux and others: ~/.config/iot (XDG Base Directory)
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			xdgConfig = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(xdgConfig, AppName)
	}

	return configDir, nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return configDir, nil
}

// GetCredentialsPath returns the path to the credentials file
func GetCredentialsPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "credentials.json"), nil
}

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// GitHubOwner is the GitHub organization/user
	GitHubOwner = "Bader-GmbH"
	// GitHubRepo is the repository name
	GitHubRepo = "iot-cli"
	// CacheFileName is the name of the update cache file
	CacheFileName = "update-check.json"
	// CheckInterval is how often to check for updates
	CheckInterval = 24 * time.Hour
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	PublishedAt time.Time      `json:"published_at"`
	Body        string         `json:"body"`
	Assets      []GitHubAsset  `json:"assets"`
	HTMLURL     string         `json:"html_url"`
}

// GitHubAsset represents a release asset
type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// UpdateCache stores the last update check result
type UpdateCache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
	CurrentVersion string   `json:"current_version"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	DownloadURL    string
	ReleaseNotes   string
	PublishedAt    time.Time
}

// Checker handles update checking
type Checker struct {
	currentVersion string
	cacheDir       string
	httpClient     *http.Client
}

// NewChecker creates a new update checker
func NewChecker(currentVersion, cacheDir string) *Checker {
	return &Checker{
		currentVersion: currentVersion,
		cacheDir:       cacheDir,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckForUpdate checks if a newer version is available
func (c *Checker) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	release, err := c.fetchLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(c.currentVersion, "v")

	// Find the right asset for this platform
	assetName := c.getAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	info := &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		ReleaseURL:     release.HTMLURL,
		DownloadURL:    downloadURL,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
	}

	return info, nil
}

// IsUpdateAvailable returns true if a newer version is available
func (info *UpdateInfo) IsUpdateAvailable() bool {
	return CompareVersions(info.LatestVersion, info.CurrentVersion) > 0
}

// CheckForUpdateCached checks for updates using a cache
// Returns nil if checked recently and no update available
func (c *Checker) CheckForUpdateCached(ctx context.Context) (*UpdateInfo, error) {
	cache, err := c.loadCache()
	if err == nil && cache != nil {
		// Check if cache is still valid
		if time.Since(cache.LastCheck) < CheckInterval {
			// Cache is fresh, check if there's an update
			if CompareVersions(cache.LatestVersion, c.currentVersion) > 0 {
				return &UpdateInfo{
					CurrentVersion: c.currentVersion,
					LatestVersion:  cache.LatestVersion,
				}, nil
			}
			return nil, nil // No update, cache is fresh
		}
	}

	// Cache expired or doesn't exist, check for real
	info, err := c.CheckForUpdate(ctx)
	if err != nil {
		return nil, err
	}

	// Save to cache
	_ = c.saveCache(&UpdateCache{
		LastCheck:      time.Now(),
		LatestVersion:  info.LatestVersion,
		CurrentVersion: c.currentVersion,
	})

	if info.IsUpdateAvailable() {
		return info, nil
	}

	return nil, nil
}

// fetchLatestRelease fetches the latest release from GitHub
func (c *Checker) fetchLatestRelease(ctx context.Context) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("iot-cli/%s", c.currentVersion))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// getAssetName returns the expected asset name for the current platform
func (c *Checker) getAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Normalize architecture names
	switch arch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	case "386":
		arch = "386"
	}

	// Asset naming convention: iot-cli-{os}-{arch}[.exe]
	name := fmt.Sprintf("iot-cli-%s-%s", os, arch)
	if os == "windows" {
		name += ".exe"
	}

	return name
}

// DownloadUpdate downloads the update to a temporary file
func (c *Checker) DownloadUpdate(ctx context.Context, downloadURL string, progress func(downloaded, total int64)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("iot-cli/%s", c.currentVersion))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "iot-cli-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Download with progress
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := tmpFile.Write(buf[:n]); werr != nil {
				os.Remove(tmpFile.Name())
				return "", fmt.Errorf("failed to write update: %w", werr)
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, resp.ContentLength)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tmpFile.Name())
			return "", fmt.Errorf("download interrupted: %w", err)
		}
	}

	return tmpFile.Name(), nil
}

// ApplyUpdate replaces the current binary with the downloaded update
func ApplyUpdate(downloadedPath string) error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Make downloaded file executable
	if err := os.Chmod(downloadedPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Backup current binary
	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(downloadedPath, execPath); err != nil {
		// Try to restore backup
		_ = os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Remove backup
	_ = os.Remove(backupPath)

	return nil
}

// loadCache loads the update cache from disk
func (c *Checker) loadCache() (*UpdateCache, error) {
	cachePath := filepath.Join(c.cacheDir, CacheFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache UpdateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// saveCache saves the update cache to disk
func (c *Checker) saveCache(cache *UpdateCache) error {
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	cachePath := filepath.Join(c.cacheDir, CacheFileName)
	return os.WriteFile(cachePath, data, 0644)
}

// ClearCache removes the update cache
func (c *Checker) ClearCache() error {
	cachePath := filepath.Join(c.cacheDir, CacheFileName)
	return os.Remove(cachePath)
}

// CompareVersions compares two version strings
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func CompareVersions(v1, v2 string) int {
	// Strip 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Handle "dev" version
	if v1 == "dev" {
		return -1 // dev is always older
	}
	if v2 == "dev" {
		return 1 // anything is newer than dev
	}

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			_, _ = fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			_, _ = fmt.Sscanf(parts2[i], "%d", &n2)
		}

		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	return 0
}

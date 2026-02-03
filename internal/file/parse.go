package file

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RemotePath represents a parsed device:path reference
type RemotePath struct {
	DeviceID string
	Path     string
}

// ParseRemotePath parses a string in the format "device:path"
// Returns the device ID and remote path
func ParseRemotePath(s string) (*RemotePath, error) {
	// Find the colon separator
	idx := strings.Index(s, ":")
	if idx == -1 {
		return nil, fmt.Errorf("invalid remote path %q: expected format device:path", s)
	}

	deviceID := s[:idx]
	path := s[idx+1:]

	if deviceID == "" {
		return nil, fmt.Errorf("invalid remote path %q: device ID cannot be empty", s)
	}

	if path == "" {
		return nil, fmt.Errorf("invalid remote path %q: path cannot be empty", s)
	}

	// Remote paths should be absolute
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid remote path %q: path must be absolute (start with /)", s)
	}

	return &RemotePath{
		DeviceID: deviceID,
		Path:     path,
	}, nil
}

// IsRemotePath checks if a string looks like a remote path (contains :)
func IsRemotePath(s string) bool {
	// Must contain : but not be a Windows drive letter (C:\)
	idx := strings.Index(s, ":")
	if idx == -1 {
		return false
	}

	// Check if it's a Windows drive letter (single letter before colon)
	if idx == 1 && len(s) > 2 && s[2] == '\\' {
		return false
	}

	return true
}

// BaseName returns the base name of a path (local or remote)
func BaseName(path string) string {
	// Handle trailing slash for directories
	path = strings.TrimSuffix(path, "/")
	return filepath.Base(path)
}

// IsDirectory checks if a path ends with / (indicating a directory)
func IsDirectory(path string) bool {
	return strings.HasSuffix(path, "/")
}

// JoinRemotePath joins path components for a remote path
func JoinRemotePath(base, name string) string {
	if strings.HasSuffix(base, "/") {
		return base + name
	}
	return base + "/" + name
}

// ResolveLocalDestination determines the local destination path for a download
// If dest is empty, uses current directory with the source filename
// If dest is a directory (ends with /), appends the source filename
func ResolveLocalDestination(remotePath, localDest string) string {
	baseName := BaseName(remotePath)

	if localDest == "" {
		return "./" + baseName
	}

	if IsDirectory(localDest) || strings.HasSuffix(localDest, string(filepath.Separator)) {
		return filepath.Join(localDest, baseName)
	}

	return localDest
}

// ResolveRemoteDestination determines the remote destination path for an upload
// If the remote path ends with /, appends the local filename
func ResolveRemoteDestination(localPath, remotePath string) string {
	if IsDirectory(remotePath) {
		return JoinRemotePath(remotePath, BaseName(localPath))
	}
	return remotePath
}

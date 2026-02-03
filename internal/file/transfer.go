package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bader-GmbH/iot-cli/internal/api"
)

// TransferOptions configures file transfer behavior
type TransferOptions struct {
	Recursive    bool
	Limit        int64 // bytes per second, 0 = unlimited
	Quiet        bool
	Force        bool
	DryRun       bool
	ShowProgress bool
}

// TransferResult contains the result of a transfer operation
type TransferResult struct {
	FilesTransferred int
	BytesTransferred int64
	Errors           []error
}

// Download downloads a file or directory from a device to local
func Download(ctx context.Context, client *api.Client, deviceID, remotePath, localPath string, opts TransferOptions) (*TransferResult, error) {
	// Check device is online
	if err := client.CheckDeviceOnline(ctx, deviceID); err != nil {
		return nil, err
	}

	// Get info about the remote path
	info, err := client.StatFile(ctx, deviceID, remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat remote path: %w", err)
	}

	result := &TransferResult{}

	if info.IsDirectory {
		if !opts.Recursive {
			return nil, fmt.Errorf("%s is a directory, use -r flag for recursive download", remotePath)
		}
		err = downloadDirectory(ctx, client, deviceID, remotePath, localPath, opts, result)
	} else {
		err = downloadFile(ctx, client, deviceID, remotePath, localPath, info.Size, opts, result)
	}

	return result, err
}

// downloadFile downloads a single file
func downloadFile(ctx context.Context, client *api.Client, deviceID, remotePath, localPath string, size int64, opts TransferOptions, result *TransferResult) error {
	// Resolve local destination
	localPath = ResolveLocalDestination(remotePath, localPath)

	if opts.DryRun {
		fmt.Printf("Would download: %s -> %s (%s)\n", remotePath, localPath, FormatBytes(size))
		return nil
	}

	// Check if file exists
	if !opts.Force {
		if _, err := os.Stat(localPath); err == nil {
			return fmt.Errorf("file %s already exists, use --force to overwrite", localPath)
		}
	}

	// Create parent directory if needed
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Download file
	reader, contentLength, err := client.DownloadFile(ctx, deviceID, remotePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Use content length if available, otherwise use the size from stat
	if contentLength > 0 {
		size = contentLength
	}

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", localPath, err)
	}
	defer file.Close()

	// Apply throttling
	var src io.Reader = reader
	if opts.Limit > 0 {
		src = NewThrottledReader(reader, opts.Limit)
	}

	// Apply progress reporting
	var dst io.Writer = file
	if opts.ShowProgress && !opts.Quiet {
		progress := NewProgressWriter(file, size, BaseName(remotePath), opts.Quiet)
		dst = progress
		defer progress.Finish()
	}

	// Copy data
	written, err := CopyWithContext(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	result.FilesTransferred++
	result.BytesTransferred += written

	if opts.Quiet {
		// Print minimal output
	} else if !opts.ShowProgress {
		fmt.Printf("  %s  %s\n", BaseName(remotePath), FormatBytes(written))
	}

	return nil
}

// downloadDirectory recursively downloads a directory
func downloadDirectory(ctx context.Context, client *api.Client, deviceID, remotePath, localPath string, opts TransferOptions, result *TransferResult) error {
	// List remote directory
	files, err := client.ListFiles(ctx, deviceID, remotePath)
	if err != nil {
		return fmt.Errorf("failed to list directory: %w", err)
	}

	// Resolve local directory path
	if localPath == "" {
		localPath = "./" + BaseName(remotePath)
	} else if IsDirectory(localPath) {
		localPath = filepath.Join(localPath, BaseName(remotePath))
	}

	if opts.DryRun {
		fmt.Printf("Would create directory: %s\n", localPath)
	} else {
		if err := os.MkdirAll(localPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", localPath, err)
		}
	}

	// Download each file/subdirectory
	for _, f := range files {
		remoteFilePath := JoinRemotePath(remotePath, f.Name)
		localFilePath := filepath.Join(localPath, f.Name)

		if f.IsDirectory {
			if err := downloadDirectory(ctx, client, deviceID, remoteFilePath, localFilePath, opts, result); err != nil {
				if opts.Force {
					result.Errors = append(result.Errors, err)
					continue
				}
				return err
			}
		} else {
			if err := downloadFile(ctx, client, deviceID, remoteFilePath, localFilePath, f.Size, opts, result); err != nil {
				if opts.Force {
					result.Errors = append(result.Errors, err)
					continue
				}
				return err
			}
		}
	}

	return nil
}

// Upload uploads a file or directory from local to a device
func Upload(ctx context.Context, client *api.Client, localPaths []string, deviceID, remotePath string, opts TransferOptions) (*TransferResult, error) {
	// Check device is online
	if err := client.CheckDeviceOnline(ctx, deviceID); err != nil {
		return nil, err
	}

	result := &TransferResult{}

	for _, localPath := range localPaths {
		info, err := os.Stat(localPath)
		if err != nil {
			return result, fmt.Errorf("failed to stat local path %s: %w", localPath, err)
		}

		if info.IsDir() {
			if !opts.Recursive {
				return result, fmt.Errorf("%s is a directory, use -r flag for recursive upload", localPath)
			}
			if err := uploadDirectory(ctx, client, localPath, deviceID, remotePath, opts, result); err != nil {
				return result, err
			}
		} else {
			destPath := ResolveRemoteDestination(localPath, remotePath)
			if err := uploadFile(ctx, client, localPath, deviceID, destPath, info.Size(), opts, result); err != nil {
				return result, err
			}
		}
	}

	return result, nil
}

// uploadFile uploads a single file
func uploadFile(ctx context.Context, client *api.Client, localPath, deviceID, remotePath string, size int64, opts TransferOptions, result *TransferResult) error {
	if opts.DryRun {
		fmt.Printf("Would upload: %s -> %s:%s (%s)\n", localPath, deviceID, remotePath, FormatBytes(size))
		return nil
	}

	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer file.Close()

	// Apply throttling
	var src io.Reader = file
	if opts.Limit > 0 {
		src = NewThrottledReader(file, opts.Limit)
	}

	// Apply progress reporting
	if opts.ShowProgress && !opts.Quiet {
		progress := NewProgressReader(src, size, BaseName(localPath), opts.Quiet)
		src = progress
		defer progress.Finish()
	}

	// Read all content (needed for multipart upload)
	// For large files, this should be streamed, but the current API uses multipart
	content, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Upload
	if err := client.UploadFile(ctx, deviceID, remotePath, strings.NewReader(string(content)), int64(len(content))); err != nil {
		return err
	}

	result.FilesTransferred++
	result.BytesTransferred += size

	if opts.Quiet {
		// Print minimal output
	} else if !opts.ShowProgress {
		fmt.Printf("  %s  %s  -> %s\n", BaseName(localPath), FormatBytes(size), remotePath)
	}

	return nil
}

// uploadDirectory recursively uploads a directory
func uploadDirectory(ctx context.Context, client *api.Client, localPath, deviceID, remotePath string, opts TransferOptions, result *TransferResult) error {
	// Resolve remote directory path
	destPath := remotePath
	if IsDirectory(remotePath) {
		destPath = JoinRemotePath(remotePath, BaseName(localPath))
	}

	if opts.DryRun {
		fmt.Printf("Would create directory: %s:%s\n", deviceID, destPath)
	} else {
		// Create remote directory
		if err := client.MkdirOnDevice(ctx, deviceID, destPath); err != nil {
			// Directory might already exist, continue
			if !strings.Contains(err.Error(), "exists") {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}
	}

	// Read local directory
	entries, err := os.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", localPath, err)
	}

	// Upload each file/subdirectory
	for _, entry := range entries {
		localFilePath := filepath.Join(localPath, entry.Name())
		remoteFilePath := JoinRemotePath(destPath, entry.Name())

		if entry.IsDir() {
			if err := uploadDirectory(ctx, client, localFilePath, deviceID, remoteFilePath, opts, result); err != nil {
				if opts.Force {
					result.Errors = append(result.Errors, err)
					continue
				}
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				if opts.Force {
					result.Errors = append(result.Errors, err)
					continue
				}
				return err
			}
			if err := uploadFile(ctx, client, localFilePath, deviceID, remoteFilePath, info.Size(), opts, result); err != nil {
				if opts.Force {
					result.Errors = append(result.Errors, err)
					continue
				}
				return err
			}
		}
	}

	return nil
}

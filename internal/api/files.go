package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// FileInfo represents metadata about a file on a device
type FileInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	IsDirectory bool   `json:"isDirectory"`
	Mode        string `json:"mode"`
	ModTime     int64  `json:"modTime"`
}

// ListFiles lists files in a directory on a device
func (c *Client) ListFiles(ctx context.Context, deviceID, path string) ([]FileInfo, error) {
	endpoint := fmt.Sprintf("/api/devices/%s/files/list?path=%s", deviceID, url.QueryEscape(path))

	var files []FileInfo
	if err := c.Get(ctx, endpoint, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// StatFile gets metadata for a file on a device
func (c *Client) StatFile(ctx context.Context, deviceID, path string) (*FileInfo, error) {
	endpoint := fmt.Sprintf("/api/devices/%s/files/stat?path=%s", deviceID, url.QueryEscape(path))

	var info FileInfo
	if err := c.Get(ctx, endpoint, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// DownloadFile downloads a file from a device
// Returns a reader for the file content and the file size
func (c *Client) DownloadFile(ctx context.Context, deviceID, path string) (io.ReadCloser, int64, error) {
	endpoint := fmt.Sprintf("/api/devices/%s/files/download?path=%s", deviceID, url.QueryEscape(path))

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("unauthorized: please run 'iot auth login'")
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("file not found: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, 0, fmt.Errorf("download failed (status %d): %s", resp.StatusCode, string(body))
	}

	return resp.Body, resp.ContentLength, nil
}

// UploadFile uploads a file to a device
func (c *Client) UploadFile(ctx context.Context, deviceID, path string, content io.Reader, size int64) error {
	endpoint := fmt.Sprintf("/api/devices/%s/files/upload?path=%s", deviceID, url.QueryEscape(path))

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", "upload")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, content); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	resp, err := c.doMultipartRequest(ctx, "POST", endpoint, &buf, writer.FormDataContentType())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: please run 'iot auth login'")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// MkdirOnDevice creates a directory on a device
func (c *Client) MkdirOnDevice(ctx context.Context, deviceID, path string) error {
	endpoint := fmt.Sprintf("/api/devices/%s/files/mkdir?path=%s", deviceID, url.QueryEscape(path))

	resp, err := c.doRequest(ctx, "POST", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: please run 'iot auth login'")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mkdir failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// doMultipartRequest performs an authenticated multipart request
func (c *Client) doMultipartRequest(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	accessToken, err := c.tokenStore.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	tenantID, err := c.tokenStore.GetTenantID()
	if err != nil {
		return nil, fmt.Errorf("tenant ID not found: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "iot-cli/1.0")
	req.Header.Set("X-Client-Type", "cli")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// CheckDeviceOnline checks if a device is online and returns an error if not
func (c *Client) CheckDeviceOnline(ctx context.Context, deviceID string) error {
	endpoint := fmt.Sprintf("/api/devices/%s", deviceID)

	var device struct {
		Online bool   `json:"online"`
		Name   string `json:"name"`
	}

	if err := c.Get(ctx, endpoint, &device); err != nil {
		return err
	}

	if !device.Online {
		return fmt.Errorf("device %q is offline", device.Name)
	}

	return nil
}

// parseAPIError attempts to parse an API error response
func parseAPIError(body []byte) string {
	var errResp struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Message != "" {
			return errResp.Message
		}
		if errResp.Error != "" {
			return errResp.Error
		}
	}

	return string(body)
}

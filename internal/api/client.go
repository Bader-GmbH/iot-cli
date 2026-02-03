package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Bader-GmbH/iot-cli/internal/auth"
)

// Client is the API client for the Bader IoT Platform
type Client struct {
	baseURL    string
	httpClient *http.Client
	tokenStore *auth.TokenStore
}

// NewClient creates a new API client
func NewClient(baseURL string) (*Client, error) {
	tokenStore, err := auth.NewTokenStore()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token store: %w", err)
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tokenStore: tokenStore,
	}, nil
}

// doRequest performs an authenticated HTTP request
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	// Get access token
	accessToken, err := c.tokenStore.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	// Get tenant ID
	tenantID, err := c.tokenStore.GetTenantID()
	if err != nil {
		return nil, fmt.Errorf("tenant ID not found: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "iot-cli/1.0")
	req.Header.Set("X-Client-Type", "cli")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// GetCredentials returns the access token and tenant ID for WebSocket auth
func (c *Client) GetCredentials() (accessToken, tenantID string, err error) {
	accessToken, err = c.tokenStore.GetAccessToken()
	if err != nil {
		return "", "", err
	}
	tenantID, err = c.tokenStore.GetTenantID()
	if err != nil {
		return "", "", err
	}
	return accessToken, tenantID, nil
}

// GetBaseURL returns the API base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// Get performs an authenticated GET request
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: please run 'iot auth login'")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

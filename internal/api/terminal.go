package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TerminalSession represents a terminal session response
type TerminalSession struct {
	SessionID    string `json:"sessionId"`
	DeviceID     string `json:"deviceId"`
	WebsocketURL string `json:"websocketUrl"`
	Status       string `json:"status"`
}

// CreateTerminalSession creates a new terminal session for a device
func (c *Client) CreateTerminalSession(ctx context.Context, deviceID string) (*TerminalSession, error) {
	path := fmt.Sprintf("/api/terminal/devices/%s/sessions", deviceID)

	resp, err := c.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: please run 'iot auth login'")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// Try to parse error message
		var errResp map[string]interface{}
		if json.Unmarshal(body, &errResp) == nil {
			if msg, ok := errResp["message"].(string); ok {
				return nil, fmt.Errorf("%s", msg)
			}
			if msg, ok := errResp["error"].(string); ok {
				return nil, fmt.Errorf("%s", msg)
			}
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var session TerminalSession
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &session, nil
}

// CloseTerminalSession closes a terminal session
func (c *Client) CloseTerminalSession(ctx context.Context, sessionID string) error {
	path := fmt.Sprintf("/api/terminal/sessions/%s", sessionID)

	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

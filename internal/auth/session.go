package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CLISession represents a pending CLI authentication session
type CLISession struct {
	SessionID string `json:"sessionId"`
	LoginURL  string `json:"loginUrl"`
	ExpiresAt string `json:"expiresAt"`
}

// CLISessionStatus represents the status of a CLI session
type CLISessionStatus struct {
	Status       string `json:"status"` // "pending", "completed", "expired"
	AccessToken  string `json:"accessToken,omitempty"`
	IDToken      string `json:"idToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresIn    int    `json:"expiresIn,omitempty"`
}

// SessionAuth handles the browser-based authentication flow
type SessionAuth struct {
	apiURL     string
	httpClient *http.Client
}

// NewSessionAuth creates a new SessionAuth instance
func NewSessionAuth(apiURL string) *SessionAuth {
	return &SessionAuth{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateSession creates a new CLI authentication session
func (s *SessionAuth) CreateSession(ctx context.Context) (*CLISession, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", s.apiURL+"/api/auth/cli-session", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create session: server returned %d", resp.StatusCode)
	}

	var session CLISession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &session, nil
}

// PollSession polls for the session status until completed or timeout
func (s *SessionAuth) PollSession(ctx context.Context, sessionID string, interval time.Duration, timeout time.Duration) (*CLISessionStatus, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		status, err := s.GetSessionStatus(ctx, sessionID)
		if err != nil {
			// Log error but continue polling
			time.Sleep(interval)
			continue
		}

		switch status.Status {
		case "completed":
			return status, nil
		case "expired":
			return nil, fmt.Errorf("session expired")
		case "pending":
			// Continue polling
			time.Sleep(interval)
		default:
			return nil, fmt.Errorf("unknown session status: %s", status.Status)
		}
	}

	return nil, fmt.Errorf("authentication timed out")
}

// GetSessionStatus gets the current status of a CLI session
func (s *SessionAuth) GetSessionStatus(ctx context.Context, sessionID string) (*CLISessionStatus, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.apiURL+"/api/auth/cli-session/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get session status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("session not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get session status: server returned %d", resp.StatusCode)
	}

	var status CLISessionStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}
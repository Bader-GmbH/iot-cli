package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bader-GmbH/iot-cli/internal/config"
)

// Credentials represents stored authentication credentials
type Credentials struct {
	AccessToken  string    `json:"accessToken"`
	IDToken      string    `json:"idToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Email        string    `json:"email,omitempty"`
	TenantID     string    `json:"tenantId,omitempty"`
}

// TokenStore handles credential storage and retrieval
type TokenStore struct {
	credentialsPath string
}

// NewTokenStore creates a new TokenStore
func NewTokenStore() (*TokenStore, error) {
	credPath, err := config.GetCredentialsPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials path: %w", err)
	}

	return &TokenStore{
		credentialsPath: credPath,
	}, nil
}

// Save stores credentials to disk
func (t *TokenStore) Save(creds *Credentials) error {
	// Ensure directory exists
	dir := filepath.Dir(t.credentialsPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Extract user info from ID token
	if creds.IDToken != "" {
		claims, err := parseJWTClaims(creds.IDToken)
		if err == nil {
			if email, ok := claims["email"].(string); ok {
				creds.Email = email
			}
			if tenantID, ok := claims["custom:tenant_id"].(string); ok {
				creds.TenantID = tenantID
			}
		}
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(t.credentialsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// Load retrieves stored credentials
func (t *TokenStore) Load() (*Credentials, error) {
	data, err := os.ReadFile(t.credentialsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in (run 'iot auth login')")
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// Delete removes stored credentials
func (t *TokenStore) Delete() error {
	err := os.Remove(t.credentialsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}
	return nil
}

// IsLoggedIn checks if valid credentials exist
func (t *TokenStore) IsLoggedIn() bool {
	creds, err := t.Load()
	if err != nil {
		return false
	}

	// Check if tokens are expired
	if time.Now().After(creds.ExpiresAt) {
		// Could try to refresh here, but for simplicity just return false
		return false
	}

	return true
}

// GetAccessToken returns the current access token, refreshing if needed
func (t *TokenStore) GetAccessToken() (string, error) {
	creds, err := t.Load()
	if err != nil {
		return "", err
	}

	// Check if token is expired (with 5 minute buffer)
	if time.Now().Add(5 * time.Minute).After(creds.ExpiresAt) {
		// TODO: Implement token refresh
		return "", fmt.Errorf("token expired, please login again")
	}

	return creds.AccessToken, nil
}

// GetTenantID returns the tenant ID from stored credentials
func (t *TokenStore) GetTenantID() (string, error) {
	creds, err := t.Load()
	if err != nil {
		return "", err
	}

	if creds.TenantID == "" {
		return "", fmt.Errorf("tenant ID not found in credentials")
	}

	return creds.TenantID, nil
}

// parseJWTClaims extracts claims from a JWT token without verification
func parseJWTClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	return claims, nil
}

package rancher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// TokenInfo represents the token information returned by Rancher API
type TokenInfo struct {
	ExpiresAt string `json:"expiresAt"`
	TTL       int64  `json:"ttl"`
	Expired   bool   `json:"expired"`
	Created   string `json:"created"`
	Enabled   bool   `json:"enabled"`
}

// GetTokenExpiration queries Rancher API for token expiration info
// Returns the expiration time of the token, or zero time if token never expires
func (c *Client) GetTokenExpiration(token string) (time.Time, error) {
	// 1. Parse token to extract token name
	// Token format: <token-name>:<secret-key>
	// Example: kubeconfig-u-abc123xyz:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	if token == "" {
		return time.Time{}, fmt.Errorf("invalid token format: token cannot be empty")
	}
	
	parts := strings.Split(token, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return time.Time{}, fmt.Errorf("invalid token format: expected <token-name>:<secret-key>")
	}
	tokenName := parts[0]

	// 2. Query Rancher API
	url := fmt.Sprintf("%s/v3/tokens/%s", c.BaseURL, tokenName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	body, respCode, err := doRequest(c.httpClient, req)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query token info: %w", err)
	}

	if respCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("failed to get token info, status %d: %s", respCode, string(body))
	}

	// 3. Parse response
	var tokenInfo TokenInfo
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse token info: %w", err)
	}

	// 4. Handle never-expiring tokens (TTL = 0)
	// Rancher tokens with TTL = 0 never expire
	if tokenInfo.TTL == 0 {
		// Return zero time to indicate token never expires
		return time.Time{}, nil
	}

	// 5. Parse expiration time
	expiresAt, err := time.Parse(time.RFC3339, tokenInfo.ExpiresAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse expiration time: %w", err)
	}

	return expiresAt, nil
}

// ShouldRefreshToken checks if token needs refresh based on expiration time and threshold
// Returns true if token should be refreshed, false otherwise
// Parameters:
//   - expiresAt: Token expiration time (zero time means never expires)
//   - thresholdDays: Refresh threshold in days before expiration
func ShouldRefreshToken(expiresAt time.Time, thresholdDays int) bool {
	// Token never expires (zero time)
	if expiresAt.IsZero() {
		return false
	}

	// Calculate threshold duration
	threshold := time.Duration(thresholdDays) * 24 * time.Hour

	// Check if token expires within the threshold period
	// time.Until returns negative duration if time has passed
	return time.Until(expiresAt) <= threshold
}

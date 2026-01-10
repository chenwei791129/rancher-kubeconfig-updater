package rancher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
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
	req.Header.Set("Authorization", "Bearer "+c.token)

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

// RegenerationReason represents the reason for token regeneration decision
type RegenerationReason string

const (
	// ReasonForceRefreshEnabled indicates force refresh flag is enabled
	ReasonForceRefreshEnabled RegenerationReason = "force_refresh_enabled"
	// ReasonNoExistingToken indicates no token exists in kubeconfig
	ReasonNoExistingToken RegenerationReason = "no_existing_token"
	// ReasonExpiresSoon indicates token expires within threshold
	ReasonExpiresSoon RegenerationReason = "expires_soon"
	// ReasonStillValid indicates token is still valid beyond threshold
	ReasonStillValid RegenerationReason = "still_valid"
	// ReasonNeverExpires indicates token never expires
	ReasonNeverExpires RegenerationReason = "never_expires"
	// ReasonNeverExpiresButRefreshRequired indicates never-expiring token that needs refresh (should not happen)
	ReasonNeverExpiresButRefreshRequired RegenerationReason = "never_expires_but_refresh_required"
	// ReasonExpirationCheckFailed indicates failed to check token expiration
	ReasonExpirationCheckFailed RegenerationReason = "expiration_check_failed"
)

// TokenRegenerationDecision represents the decision and context for token regeneration
type TokenRegenerationDecision struct {
	ShouldRegenerate bool
	Reason           RegenerationReason
	ExpiresAt        time.Time
	DaysUntilExpiry  float64
}

// DetermineTokenRegeneration decides whether a token should be regenerated
// Returns a decision with reason for logging purposes
// Parameters:
//   - client: Rancher client for API calls
//   - currentToken: Current token from kubeconfig (empty if none exists)
//   - forceRefresh: Whether to bypass expiration checks
//   - thresholdDays: Refresh threshold in days before expiration
//   - clusterName: Cluster name for logging context
func (c *Client) DetermineTokenRegeneration(currentToken string, forceRefresh bool, thresholdDays int, clusterName string) TokenRegenerationDecision {
	// Force refresh overrides all other checks
	if forceRefresh {
		return TokenRegenerationDecision{
			ShouldRegenerate: true,
			Reason:           ReasonForceRefreshEnabled,
		}
	}

	// No current token means we need to generate one
	if currentToken == "" {
		return TokenRegenerationDecision{
			ShouldRegenerate: true,
			Reason:           ReasonNoExistingToken,
		}
	}

	// Check token expiration
	expiresAt, err := c.GetTokenExpiration(currentToken)
	if err != nil {
		// If we can't check expiration, regenerate to be safe
		c.logger.Warn("Failed to check token expiration, will regenerate for safety",
			zap.String("cluster", clusterName),
			zap.Error(err))
		return TokenRegenerationDecision{
			ShouldRegenerate: true,
			Reason:           ReasonExpirationCheckFailed,
		}
	}

	// Check if token needs refresh based on expiration and threshold
	shouldRefresh := ShouldRefreshToken(expiresAt, thresholdDays)

	if !shouldRefresh {
		// Token is still valid
		if expiresAt.IsZero() {
			return TokenRegenerationDecision{
				ShouldRegenerate: false,
				Reason:           ReasonNeverExpires,
				ExpiresAt:        expiresAt,
			}
		}
		return TokenRegenerationDecision{
			ShouldRegenerate: false,
			Reason:           ReasonStillValid,
			ExpiresAt:        expiresAt,
			DaysUntilExpiry:  time.Until(expiresAt).Hours() / 24,
		}
	}

	// Token needs refresh
	if expiresAt.IsZero() {
		// This should never happen based on ShouldRefreshToken logic,
		// but keep for defensive programming
		return TokenRegenerationDecision{
			ShouldRegenerate: true,
			Reason:           ReasonNeverExpiresButRefreshRequired,
			ExpiresAt:        expiresAt,
		}
	}

	return TokenRegenerationDecision{
		ShouldRegenerate: true,
		Reason:           ReasonExpiresSoon,
		ExpiresAt:        expiresAt,
		DaysUntilExpiry:  time.Until(expiresAt).Hours() / 24,
	}
}

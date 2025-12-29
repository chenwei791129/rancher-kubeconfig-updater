// Package kubeconfig provides functionality for managing Kubernetes configuration files.
package kubeconfig

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims represents the claims section of a JWT token
type JWTClaims struct {
	Exp int64 `json:"exp"` // Expiration time (Unix timestamp)
	Iat int64 `json:"iat"` // Issued at time (Unix timestamp)
}

// ParseTokenExpiration parses the expiration time from a Rancher token.
// Token format: <token-name>:<jwt-token>
// Returns the expiration time and any error encountered during parsing.
func ParseTokenExpiration(token string) (time.Time, error) {
	if token == "" {
		return time.Time{}, fmt.Errorf("token is empty")
	}

	// Split token by colon to separate token name from JWT
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid token format: expected '<token-name>:<jwt-token>'")
	}

	jwtToken := parts[1]

	// Parse JWT (format: header.payload.signature)
	jwtParts := strings.Split(jwtToken, ".")
	if len(jwtParts) != 3 {
		return time.Time{}, fmt.Errorf("invalid JWT format: expected 3 parts (header.payload.signature)")
	}

	// Base64 decode the payload (middle part)
	payload, err := base64.RawURLEncoding.DecodeString(jwtParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse JSON claims
	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	// Check if expiration claim exists
	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("JWT token missing expiration claim")
	}

	// Convert Unix timestamp to time.Time
	expirationTime := time.Unix(claims.Exp, 0)

	return expirationTime, nil
}

// ShouldRefreshToken checks if a token should be refreshed based on expiration.
// It returns true if:
//   - token is empty
//   - token cannot be parsed
//   - token is expired
//   - token is expiring within the threshold
//
// thresholdDays: number of days before expiration to trigger refresh (e.g., 30)
func ShouldRefreshToken(token string, thresholdDays int) (bool, error) {
	// No token exists, need to generate
	if token == "" {
		return true, nil
	}

	// Parse token to get expiration time
	expiresAt, err := ParseTokenExpiration(token)
	if err != nil {
		// Cannot parse token, refresh for safety
		return true, fmt.Errorf("cannot parse token: %w", err)
	}

	// Calculate threshold duration
	threshold := time.Duration(thresholdDays) * 24 * time.Hour

	// Calculate time until expiration
	timeUntilExpiry := time.Until(expiresAt)

	// Refresh if expired or expiring within threshold (inclusive)
	shouldRefresh := timeUntilExpiry <= threshold

	return shouldRefresh, nil
}

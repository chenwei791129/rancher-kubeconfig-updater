package kubeconfig

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Helper function to create a test JWT token with custom expiration
func createTestJWT(expiration time.Time) string {
	claims := JWTClaims{
		Exp: expiration.Unix(),
		Iat: time.Now().Unix(),
	}

	payload, _ := json.Marshal(claims)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	// Create a simple JWT structure (header.payload.signature)
	// We don't need valid signature for testing parsing
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))

	jwtToken := fmt.Sprintf("%s.%s.%s", header, encodedPayload, signature)
	return fmt.Sprintf("token-abc123:%s", jwtToken)
}

// TestParseTokenExpiration tests the ParseTokenExpiration function
func TestParseTokenExpiration(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectError bool
		checkTime   bool
	}{
		{
			name:        "valid token with future expiration",
			token:       createTestJWT(time.Now().Add(24 * time.Hour)),
			expectError: false,
			checkTime:   true,
		},
		{
			name:        "valid token with past expiration",
			token:       createTestJWT(time.Now().Add(-24 * time.Hour)),
			expectError: false,
			checkTime:   true,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
			checkTime:   false,
		},
		{
			name:        "token without colon separator",
			token:       "invalid-token-format",
			expectError: true,
			checkTime:   false,
		},
		{
			name:        "token with too many colons",
			token:       "token:name:extra:jwt-part",
			expectError: true,
			checkTime:   false,
		},
		{
			name:        "token with invalid JWT format (missing parts)",
			token:       "token-name:header.payload",
			expectError: true,
			checkTime:   false,
		},
		{
			name:        "token with invalid base64 payload",
			token:       "token-name:header.!!!invalid!!!.signature",
			expectError: true,
			checkTime:   false,
		},
		{
			name:        "token with invalid JSON in payload",
			token:       fmt.Sprintf("token-name:header.%s.signature", base64.RawURLEncoding.EncodeToString([]byte("not-json"))),
			expectError: true,
			checkTime:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expiresAt, err := ParseTokenExpiration(tt.token)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseTokenExpiration() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ParseTokenExpiration() unexpected error: %v", err)
				}
				if tt.checkTime && expiresAt.IsZero() {
					t.Errorf("ParseTokenExpiration() returned zero time")
				}
			}
		})
	}
}

// TestParseTokenExpiration_ExpirationTime tests that expiration time is correctly parsed
func TestParseTokenExpiration_ExpirationTime(t *testing.T) {
	expectedExpiration := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	token := createTestJWT(expectedExpiration)

	actualExpiration, err := ParseTokenExpiration(token)
	if err != nil {
		t.Fatalf("ParseTokenExpiration() error: %v", err)
	}

	// Check if times are equal (within 1 second tolerance for any precision issues)
	timeDiff := actualExpiration.Sub(expectedExpiration)
	if timeDiff < -time.Second || timeDiff > time.Second {
		t.Errorf("Expected expiration %v, got %v (diff: %v)", expectedExpiration, actualExpiration, timeDiff)
	}
}

// TestParseTokenExpiration_MissingExpClaim tests token without exp claim
func TestParseTokenExpiration_MissingExpClaim(t *testing.T) {
	// Create token without exp claim
	claims := map[string]interface{}{
		"iat": time.Now().Unix(),
		"sub": "user123",
	}
	payload, _ := json.Marshal(claims)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	token := fmt.Sprintf("token-name:%s.%s.%s", header, encodedPayload, signature)

	_, err := ParseTokenExpiration(token)
	if err == nil {
		t.Error("ParseTokenExpiration() should return error for token without exp claim")
	}
	if !strings.Contains(err.Error(), "missing expiration claim") {
		t.Errorf("Expected 'missing expiration claim' error, got: %v", err)
	}
}

// TestShouldRefreshToken tests the ShouldRefreshToken function
func TestShouldRefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		thresholdDays  int
		expectRefresh  bool
		expectError    bool
		errorSubstring string
	}{
		{
			name:          "empty token should refresh",
			token:         "",
			thresholdDays: 30,
			expectRefresh: true,
			expectError:   false,
		},
		{
			name:          "token expiring in 10 days with 30-day threshold should refresh",
			token:         createTestJWT(time.Now().Add(10 * 24 * time.Hour)),
			thresholdDays: 30,
			expectRefresh: true,
			expectError:   false,
		},
		{
			name:          "token expiring in 45 days with 30-day threshold should not refresh",
			token:         createTestJWT(time.Now().Add(45 * 24 * time.Hour)),
			thresholdDays: 30,
			expectRefresh: false,
			expectError:   false,
		},
		{
			name:          "expired token should refresh",
			token:         createTestJWT(time.Now().Add(-24 * time.Hour)),
			thresholdDays: 30,
			expectRefresh: true,
			expectError:   false,
		},
		{
			name:          "token expiring in exactly 30 days with 30-day threshold should not refresh",
			token:         createTestJWT(time.Now().Add(30*24*time.Hour + 1*time.Hour)),
			thresholdDays: 30,
			expectRefresh: false,
			expectError:   false,
		},
		{
			name:          "token expiring in 5 days with 7-day threshold should refresh",
			token:         createTestJWT(time.Now().Add(5 * 24 * time.Hour)),
			thresholdDays: 7,
			expectRefresh: true,
			expectError:   false,
		},
		{
			name:          "token expiring in 10 days with 7-day threshold should not refresh",
			token:         createTestJWT(time.Now().Add(10 * 24 * time.Hour)),
			thresholdDays: 7,
			expectRefresh: false,
			expectError:   false,
		},
		{
			name:           "invalid token format should refresh with error",
			token:          "invalid-format",
			thresholdDays:  30,
			expectRefresh:  true,
			expectError:    true,
			errorSubstring: "cannot parse token",
		},
		{
			name:           "malformed JWT should refresh with error",
			token:          "token-name:not.a.valid.jwt",
			thresholdDays:  30,
			expectRefresh:  true,
			expectError:    true,
			errorSubstring: "cannot parse token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRefresh, err := ShouldRefreshToken(tt.token, tt.thresholdDays)

			if tt.expectError {
				if err == nil {
					t.Errorf("ShouldRefreshToken() expected error but got none")
				} else if tt.errorSubstring != "" && !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorSubstring, err)
				}
			} else {
				if err != nil {
					t.Errorf("ShouldRefreshToken() unexpected error: %v", err)
				}
			}

			if shouldRefresh != tt.expectRefresh {
				t.Errorf("ShouldRefreshToken() expected refresh=%v, got %v", tt.expectRefresh, shouldRefresh)
			}
		})
	}
}

// TestShouldRefreshToken_BoundaryConditions tests edge cases
func TestShouldRefreshToken_BoundaryConditions(t *testing.T) {
	// Test token expiring in exactly 1 hour with 0-day threshold
	tokenIn1Hour := createTestJWT(time.Now().Add(1 * time.Hour))
	shouldRefresh, err := ShouldRefreshToken(tokenIn1Hour, 0)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if shouldRefresh {
		t.Error("Token expiring in 1 hour with 0-day threshold should not refresh")
	}

	// Test token expiring in 1 minute with 0-day threshold
	tokenIn1Min := createTestJWT(time.Now().Add(1 * time.Minute))
	shouldRefresh, err = ShouldRefreshToken(tokenIn1Min, 0)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if shouldRefresh {
		t.Error("Token expiring in 1 minute with 0-day threshold should not refresh")
	}

	// Test token expired 1 second ago with any threshold
	tokenExpired1SecAgo := createTestJWT(time.Now().Add(-1 * time.Second))
	shouldRefresh, err = ShouldRefreshToken(tokenExpired1SecAgo, 30)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !shouldRefresh {
		t.Error("Expired token should always refresh")
	}
}

// TestShouldRefreshToken_DifferentThresholds tests various threshold values
func TestShouldRefreshToken_DifferentThresholds(t *testing.T) {
	thresholds := []struct {
		days             int
		tokenValidDays   int
		expectedRefresh  bool
	}{
		{days: 1, tokenValidDays: 2, expectedRefresh: false},
		{days: 1, tokenValidDays: 1, expectedRefresh: true},  // Boundary: token expires exactly at threshold (should refresh due to <= comparison)
		{days: 7, tokenValidDays: 10, expectedRefresh: false},
		{days: 7, tokenValidDays: 5, expectedRefresh: true},
		{days: 30, tokenValidDays: 60, expectedRefresh: false},
		{days: 30, tokenValidDays: 15, expectedRefresh: true},
		{days: 90, tokenValidDays: 100, expectedRefresh: false},
		{days: 90, tokenValidDays: 60, expectedRefresh: true},
	}

	for _, th := range thresholds {
		t.Run(fmt.Sprintf("threshold_%dd_token_%dd", th.days, th.tokenValidDays), func(t *testing.T) {
			token := createTestJWT(time.Now().Add(time.Duration(th.tokenValidDays) * 24 * time.Hour))
			shouldRefresh, err := ShouldRefreshToken(token, th.days)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if shouldRefresh != th.expectedRefresh {
				t.Errorf("For threshold %d days and token valid for %d days, expected refresh=%v, got %v",
					th.days, th.tokenValidDays, th.expectedRefresh, shouldRefresh)
			}
		})
	}
}

// TestRealWorldRancherToken tests with a realistic Rancher token structure
func TestRealWorldRancherToken(t *testing.T) {
	// Simulate a realistic Rancher token structure
	// Format: token-xxxxx:yyy...yyy (JWT)
	expirationTime := time.Now().Add(30 * 24 * time.Hour) // 30 days from now
	
	claims := JWTClaims{
		Exp: expirationTime.Unix(),
		Iat: time.Now().Unix(),
	}

	payload, _ := json.Marshal(claims)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	// Simulate Rancher JWT structure
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("signature-data-here"))
	
	rancherToken := fmt.Sprintf("token-x7b9c:%s.%s.%s", header, encodedPayload, signature)

	// Test parsing
	parsedExp, err := ParseTokenExpiration(rancherToken)
	if err != nil {
		t.Fatalf("Failed to parse realistic Rancher token: %v", err)
	}

	// Check expiration is approximately correct (within 1 second)
	timeDiff := parsedExp.Sub(expirationTime)
	if timeDiff < -time.Second || timeDiff > time.Second {
		t.Errorf("Parsed expiration time mismatch: expected %v, got %v", expirationTime, parsedExp)
	}

	// Test refresh logic
	shouldRefresh, err := ShouldRefreshToken(rancherToken, 30)
	if err != nil {
		t.Fatalf("Failed to check refresh status: %v", err)
	}

	// Token expires in exactly 30 days, threshold is 30 days
	// At or within threshold, should refresh (using <= comparison)
	if !shouldRefresh {
		t.Error("Token expiring in exactly 30 days with 30-day threshold should refresh")
	}

	// Test with smaller threshold
	shouldRefresh, err = ShouldRefreshToken(rancherToken, 15)
	if err != nil {
		t.Fatalf("Failed to check refresh status: %v", err)
	}
	if shouldRefresh {
		t.Error("Token expiring in 30 days with 15-day threshold should not refresh")
	}
}

package rancher

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestGetTokenExpiration_Success tests successfully retrieving token expiration
func TestGetTokenExpiration_Success(t *testing.T) {
	// Mock response with expiration date 30 days from now
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	mockResponse := `{
		"expiresAt": "` + expiresAt + `",
		"expired": false,
		"ttl": 2592000000,
		"created": "2024-01-01T00:00:00Z",
		"enabled": true
	}`

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Verify request
			assert.Equal(t, "/v3/tokens/kubeconfig-u-abc123", req.URL.Path)
			assert.Equal(t, "Bearer kubeconfig-u-abc123:secretkey123", req.Header.Get("Authorization"))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			}, nil
		},
	}

	logger := zap.NewNop()
	client := &Client{
		token:      "test-token",
		httpClient: mockClient,
		BaseURL:    "https://rancher.example.com",
		logger:     logger,
	}

	// Test with valid token format
	token := "kubeconfig-u-abc123:secretkey123"
	expiration, err := client.GetTokenExpiration(token)

	assert.NoError(t, err)
	assert.False(t, expiration.IsZero())
	
	// Verify expiration is approximately 30 days from now (with 1 minute tolerance)
	expectedExpiration, _ := time.Parse(time.RFC3339, expiresAt)
	assert.WithinDuration(t, expectedExpiration, expiration, time.Minute)
}

// TestGetTokenExpiration_NeverExpires tests handling of never-expiring tokens
func TestGetTokenExpiration_NeverExpires(t *testing.T) {
	// Mock response with TTL = 0 (never expires)
	mockResponse := `{
		"expiresAt": "",
		"expired": false,
		"ttl": 0,
		"created": "2024-01-01T00:00:00Z",
		"enabled": true
	}`

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			}, nil
		},
	}

	logger := zap.NewNop()
	client := &Client{
		token:      "test-token",
		httpClient: mockClient,
		BaseURL:    "https://rancher.example.com",
		logger:     logger,
	}

	token := "kubeconfig-u-abc123:secretkey123"
	expiration, err := client.GetTokenExpiration(token)

	assert.NoError(t, err)
	assert.True(t, expiration.IsZero(), "Expected zero time for never-expiring token")
}

// TestGetTokenExpiration_InvalidTokenFormat tests error handling for invalid token format
func TestGetTokenExpiration_InvalidTokenFormat(t *testing.T) {
	// Create mock client that should never be called for invalid formats
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Fatal("HTTP request should not be made for invalid token format")
			return nil, nil
		},
	}

	logger := zap.NewNop()
	client := &Client{
		token:      "test-token",
		httpClient: mockClient,
		BaseURL:    "https://rancher.example.com",
		logger:     logger,
	}

	tests := []struct {
		name  string
		token string
	}{
		{"missing colon", "invalid-token-format"},
		{"empty token", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetTokenExpiration(tt.token)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid token format")
		})
	}
}

// TestGetTokenExpiration_APIError tests API error handling
func TestGetTokenExpiration_APIError(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectedErr  string
	}{
		{
			name:         "unauthorized",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "unauthorized"}`,
			expectedErr:  "failed to get token info",
		},
		{
			name:         "not found",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "token not found"}`,
			expectedErr:  "failed to get token info",
		},
		{
			name:         "internal server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal error"}`,
			expectedErr:  "failed to get token info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
					}, nil
				},
			}

			logger := zap.NewNop()
			client := &Client{
				token:      "test-token",
				httpClient: mockClient,
				BaseURL:    "https://rancher.example.com",
				logger:     logger,
			}

			token := "kubeconfig-u-abc123:secretkey123"
			_, err := client.GetTokenExpiration(token)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// TestGetTokenExpiration_InvalidJSON tests handling of invalid JSON response
func TestGetTokenExpiration_InvalidJSON(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
			}, nil
		},
	}

	logger := zap.NewNop()
	client := &Client{
		token:      "test-token",
		httpClient: mockClient,
		BaseURL:    "https://rancher.example.com",
		logger:     logger,
	}

	token := "kubeconfig-u-abc123:secretkey123"
	_, err := client.GetTokenExpiration(token)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token info")
}

// TestShouldRefreshToken tests token refresh decision logic
func TestShouldRefreshToken(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		expiresAt     time.Time
		thresholdDays int
		expected      bool
		description   string
	}{
		{
			name:          "never expires",
			expiresAt:     time.Time{}, // zero time
			thresholdDays: 30,
			expected:      false,
			description:   "Token never expires (zero time)",
		},
		{
			name:          "expired already",
			expiresAt:     now.Add(-1 * time.Hour),
			thresholdDays: 30,
			expected:      true,
			description:   "Token already expired",
		},
		{
			name:          "expires within threshold",
			expiresAt:     now.Add(15 * 24 * time.Hour), // 15 days from now
			thresholdDays: 30,
			expected:      true,
			description:   "Token expires within 30-day threshold",
		},
		{
			name:          "expires outside threshold",
			expiresAt:     now.Add(60 * 24 * time.Hour), // 60 days from now
			thresholdDays: 30,
			expected:      false,
			description:   "Token expires outside 30-day threshold",
		},
		{
			name:          "expires exactly at threshold",
			expiresAt:     now.Add(30 * 24 * time.Hour), // exactly 30 days
			thresholdDays: 30,
			expected:      true,
			description:   "Token expires exactly at threshold boundary",
		},
		{
			name:          "zero threshold",
			expiresAt:     now.Add(1 * time.Hour),
			thresholdDays: 0,
			expected:      false,
			description:   "Zero threshold - only expired tokens should refresh",
		},
		{
			name:          "large threshold",
			expiresAt:     now.Add(100 * 24 * time.Hour),
			thresholdDays: 365,
			expected:      true,
			description:   "Large threshold (1 year) should trigger refresh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRefreshToken(tt.expiresAt, tt.thresholdDays)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestShouldRefreshToken_EdgeCases tests edge cases for token refresh logic
func TestShouldRefreshToken_EdgeCases(t *testing.T) {
	now := time.Now()

	// Test with negative threshold (invalid but should still work)
	result := ShouldRefreshToken(now.Add(10*24*time.Hour), -5)
	assert.False(t, result, "Negative threshold should not trigger refresh for valid token")

	// Test with very large expiration date
	futureDate := now.Add(10 * 365 * 24 * time.Hour) // ~10 years
	result = ShouldRefreshToken(futureDate, 30)
	assert.False(t, result, "Token expiring in far future should not need refresh")
}

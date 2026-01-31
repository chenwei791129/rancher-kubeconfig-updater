package rancher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockRancherServer is a complete mock implementation of Rancher API server.
// It simulates the essential Rancher API endpoints for testing purposes.
// This approach is inspired by the rancher/apiserver project architecture.
type MockRancherServer struct {
	server *httptest.Server

	// Internal state
	mu              sync.RWMutex
	users           map[string]mockUser
	clusters        []Cluster
	tokens          map[string]mockToken
	kubeconfigToken string

	// For tracking API calls
	apiCalls []apiCall
}

// mockUser represents a user in the mock server
type mockUser struct {
	Username string
	Password string
	AuthType AuthType
}

// mockToken represents a token in the mock server
type mockToken struct {
	Name      string
	Token     string
	TTL       int64
	ExpiresAt time.Time
	Expired   bool
	Enabled   bool
	Created   time.Time
}

// apiCall represents a recorded API call for verification
type apiCall struct {
	Method   string
	Path     string
	Query    string
	Headers  http.Header
	Body     string
	Response int
}

// MockRancherServerOption configures the mock server
type MockRancherServerOption func(*MockRancherServer)

// WithMockUser adds a user to the mock server
func WithMockUser(username, password string, authType AuthType) MockRancherServerOption {
	return func(s *MockRancherServer) {
		s.users[username] = mockUser{
			Username: username,
			Password: password,
			AuthType: authType,
		}
	}
}

// WithMockClusters sets the clusters for the mock server
func WithMockClusters(clusters []Cluster) MockRancherServerOption {
	return func(s *MockRancherServer) {
		s.clusters = clusters
	}
}

// WithMockToken adds a token to the mock server
func WithMockToken(name, tokenValue string, ttl int64, expiresAt time.Time) MockRancherServerOption {
	return func(s *MockRancherServer) {
		s.tokens[name] = mockToken{
			Name:      name,
			Token:     tokenValue,
			TTL:       ttl,
			ExpiresAt: expiresAt,
			Expired:   time.Now().After(expiresAt) && ttl > 0,
			Enabled:   true,
			Created:   time.Now().Add(-24 * time.Hour),
		}
	}
}

// WithKubeconfigToken sets the token returned in kubeconfig generation
func WithKubeconfigToken(token string) MockRancherServerOption {
	return func(s *MockRancherServer) {
		s.kubeconfigToken = token
	}
}

// NewMockRancherServer creates a new mock Rancher server
func NewMockRancherServer(opts ...MockRancherServerOption) *MockRancherServer {
	s := &MockRancherServer{
		users:           make(map[string]mockUser),
		clusters:        []Cluster{},
		tokens:          make(map[string]mockToken),
		kubeconfigToken: "default-kubeconfig-token:secret123",
		apiCalls:        []apiCall{},
	}

	for _, opt := range opts {
		opt(s)
	}

	s.server = httptest.NewServer(http.HandlerFunc(s.handleRequest))
	return s
}

// URL returns the server URL
func (s *MockRancherServer) URL() string {
	return s.server.URL
}

// Client returns an HTTP client configured for the test server
func (s *MockRancherServer) Client() *http.Client {
	return s.server.Client()
}

// Close shuts down the mock server
func (s *MockRancherServer) Close() {
	s.server.Close()
}

// GetAPICalls returns all recorded API calls
func (s *MockRancherServer) GetAPICalls() []apiCall {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]apiCall{}, s.apiCalls...)
}

// recordCall records an API call for later verification
func (s *MockRancherServer) recordCall(method, path, query string, headers http.Header, body string, response int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiCalls = append(s.apiCalls, apiCall{
		Method:   method,
		Path:     path,
		Query:    query,
		Headers:  headers,
		Body:     body,
		Response: response,
	})
}

// handleRequest is the main request handler for the mock server
func (s *MockRancherServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	action := r.URL.Query().Get("action")

	// Route to appropriate handler based on path and action
	switch {
	// Authentication endpoints (POST only, matching production behavior)
	case strings.Contains(path, "/v3-public/localProviders/local") && action == "login" && r.Method == "POST":
		s.handleLocalLogin(w, r)
	case strings.Contains(path, "/v3-public/openLdapProviders/openldap") && action == "login" && r.Method == "POST":
		s.handleLDAPLogin(w, r)

	// Cluster endpoints
	case path == "/v3/clusters" && r.Method == "GET":
		s.handleListClusters(w, r)
	case strings.HasPrefix(path, "/v3/clusters/") && action == "generateKubeconfig" && r.Method == "POST":
		s.handleGenerateKubeconfig(w, r)

	// Token endpoints
	case strings.HasPrefix(path, "/v3/tokens/") && r.Method == "GET":
		s.handleGetToken(w, r)

	default:
		s.recordCall(r.Method, path, r.URL.RawQuery, r.Header, "", http.StatusNotFound)
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// handleLocalLogin handles local authentication
func (s *MockRancherServer) handleLocalLogin(w http.ResponseWriter, r *http.Request) {
	s.handleLogin(w, r, AuthTypeLocal)
}

// handleLDAPLogin handles LDAP authentication
func (s *MockRancherServer) handleLDAPLogin(w http.ResponseWriter, r *http.Request) {
	s.handleLogin(w, r, AuthTypeLDAP)
}

// handleLogin is the common login handler
func (s *MockRancherServer) handleLogin(w http.ResponseWriter, r *http.Request, authType AuthType) {
	// Read and preserve the request body for recording
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, "", http.StatusBadRequest)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	bodyStr := string(bodyBytes)

	// Restore the body for decoding
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req struct {
		Username     string `json:"username"`
		Password     string `json:"password"`
		ResponseType string `json:"responseType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, bodyStr, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify user credentials
	user, exists := s.users[req.Username]
	if !exists || user.Password != req.Password || user.AuthType != authType {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, bodyStr, http.StatusUnauthorized)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid credentials"}`))
		return
	}

	// Generate token response
	token := fmt.Sprintf("token-%s-%d", req.Username, time.Now().UnixNano())
	response := map[string]string{"token": token}
	respBytes, _ := json.Marshal(response)

	s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, bodyStr, http.StatusCreated)
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// handleListClusters handles the list clusters endpoint
func (s *MockRancherServer) handleListClusters(w http.ResponseWriter, r *http.Request) {
	// Verify authorization header
	if !s.verifyAuth(r) {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, "", http.StatusUnauthorized)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
		return
	}

	response := struct {
		Data []Cluster `json:"data"`
	}{
		Data: s.clusters,
	}

	respBytes, _ := json.Marshal(response)
	s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, "", http.StatusOK)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// handleGenerateKubeconfig handles the generate kubeconfig endpoint
func (s *MockRancherServer) handleGenerateKubeconfig(w http.ResponseWriter, r *http.Request) {
	// Verify authorization header
	if !s.verifyAuth(r) {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, "", http.StatusUnauthorized)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
		return
	}

	// Extract cluster ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, "", http.StatusBadRequest)
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	clusterID := parts[3]

	// Verify cluster exists
	found := false
	var clusterName string
	for _, c := range s.clusters {
		if c.ID == clusterID {
			found = true
			clusterName = c.Name
			break
		}
	}
	if !found {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, "", http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "cluster not found"}`))
		return
	}

	// Generate kubeconfig YAML
	kubeconfig := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: %s
  name: %s
current-context: %s
kind: Config
users:
- name: %s
  user:
    token: %s
`, s.server.URL, clusterName, clusterName, clusterName, clusterName, clusterName, clusterName, s.kubeconfigToken)

	response := struct {
		Config string `json:"config"`
	}{
		Config: kubeconfig,
	}

	respBytes, _ := json.Marshal(response)
	s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, clusterID, http.StatusOK)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// handleGetToken handles the get token endpoint
func (s *MockRancherServer) handleGetToken(w http.ResponseWriter, r *http.Request) {
	// Verify authorization header
	if !s.verifyAuth(r) {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, "", http.StatusUnauthorized)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
		return
	}

	// Extract token name from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, "", http.StatusBadRequest)
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	tokenName := parts[3]

	// Find token
	token, exists := s.tokens[tokenName]
	if !exists {
		s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, tokenName, http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "token not found"}`))
		return
	}

	// Build response
	var expiresAtStr string
	if token.TTL > 0 {
		expiresAtStr = token.ExpiresAt.Format(time.RFC3339)
	}

	response := struct {
		ExpiresAt string `json:"expiresAt"`
		TTL       int64  `json:"ttl"`
		Expired   bool   `json:"expired"`
		Created   string `json:"created"`
		Enabled   bool   `json:"enabled"`
	}{
		ExpiresAt: expiresAtStr,
		TTL:       token.TTL,
		Expired:   token.Expired,
		Created:   token.Created.Format(time.RFC3339),
		Enabled:   token.Enabled,
	}

	respBytes, _ := json.Marshal(response)
	s.recordCall(r.Method, r.URL.Path, r.URL.RawQuery, r.Header, tokenName, http.StatusOK)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// verifyAuth checks the authorization header
func (s *MockRancherServer) verifyAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	return strings.HasPrefix(auth, "Bearer ")
}

// =============================================================================
// Test Cases using MockRancherServer
// =============================================================================

// TestMockRancherServer_LocalAuthentication tests local auth via mock server
func TestMockRancherServer_LocalAuthentication(t *testing.T) {
	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password123", AuthTypeLocal),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password123",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotEmpty(t, client.token)

	// Verify API call was made correctly
	calls := mockServer.GetAPICalls()
	assert.Len(t, calls, 1)
	assert.Equal(t, "POST", calls[0].Method)
	assert.Contains(t, calls[0].Path, "/v3-public/localProviders/local")
	assert.Equal(t, http.StatusCreated, calls[0].Response)
}

// TestMockRancherServer_LDAPAuthentication tests LDAP auth via mock server
func TestMockRancherServer_LDAPAuthentication(t *testing.T) {
	mockServer := NewMockRancherServer(
		WithMockUser("ldapuser", "ldappass", AuthTypeLDAP),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"ldapuser",
		"ldappass",
		AuthTypeLDAP,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotEmpty(t, client.token)
}

// TestMockRancherServer_AuthenticationFailure tests auth failure scenarios
func TestMockRancherServer_AuthenticationFailure(t *testing.T) {
	mockServer := NewMockRancherServer(
		WithMockUser("admin", "correctpassword", AuthTypeLocal),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	tests := []struct {
		name        string
		username    string
		password    string
		authType    AuthType
		expectError string
	}{
		{
			name:        "wrong password",
			username:    "admin",
			password:    "wrongpassword",
			authType:    AuthTypeLocal,
			expectError: "login failed",
		},
		{
			name:        "wrong user",
			username:    "wronguser",
			password:    "correctpassword",
			authType:    AuthTypeLocal,
			expectError: "login failed",
		},
		{
			name:        "wrong auth type",
			username:    "admin",
			password:    "correctpassword",
			authType:    AuthTypeLDAP,
			expectError: "login failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(
				mockServer.URL(),
				tt.username,
				tt.password,
				tt.authType,
				logger,
				false,
				WithHTTPClient(mockServer.Client()),
			)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

// TestMockRancherServer_ListClusters tests listing clusters via mock server
func TestMockRancherServer_ListClusters(t *testing.T) {
	expectedClusters := []Cluster{
		{ID: "c-m-abc123", Name: "production"},
		{ID: "c-m-def456", Name: "staging"},
		{ID: "c-m-ghi789", Name: "development"},
	}

	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password", AuthTypeLocal),
		WithMockClusters(expectedClusters),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)

	clusters, err := client.ListClusters()

	assert.NoError(t, err)
	assert.Len(t, clusters, 3)

	for i, expected := range expectedClusters {
		assert.Equal(t, expected.ID, clusters[i].ID)
		assert.Equal(t, expected.Name, clusters[i].Name)
	}
}

// TestMockRancherServer_GetClusterToken tests getting cluster token via mock server
func TestMockRancherServer_GetClusterToken(t *testing.T) {
	clusters := []Cluster{
		{ID: "c-m-prod", Name: "production"},
	}
	expectedToken := "kubeconfig-user-abc:secretkey123456"

	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password", AuthTypeLocal),
		WithMockClusters(clusters),
		WithKubeconfigToken(expectedToken),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)

	token := client.GetClusterToken("c-m-prod")

	assert.Equal(t, expectedToken, token)
}

// TestMockRancherServer_GetClusterToken_NotFound tests token retrieval for non-existent cluster
func TestMockRancherServer_GetClusterToken_NotFound(t *testing.T) {
	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password", AuthTypeLocal),
		WithMockClusters([]Cluster{}), // No clusters
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)

	token := client.GetClusterToken("non-existent-cluster")

	assert.Empty(t, token)
}

// TestMockRancherServer_GetTokenExpiration tests token expiration check via mock server
func TestMockRancherServer_GetTokenExpiration(t *testing.T) {
	futureExpiry := time.Now().Add(30 * 24 * time.Hour)

	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password", AuthTypeLocal),
		WithMockToken("kubeconfig-user-abc", "kubeconfig-user-abc:secret", 2592000000, futureExpiry),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)

	expiration, err := client.GetTokenExpiration("kubeconfig-user-abc:secret")

	assert.NoError(t, err)
	assert.WithinDuration(t, futureExpiry, expiration, time.Second)
}

// TestMockRancherServer_GetTokenExpiration_NeverExpires tests never-expiring tokens
func TestMockRancherServer_GetTokenExpiration_NeverExpires(t *testing.T) {
	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password", AuthTypeLocal),
		WithMockToken("kubeconfig-user-abc", "kubeconfig-user-abc:secret", 0, time.Time{}), // TTL=0 means never expires
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)

	expiration, err := client.GetTokenExpiration("kubeconfig-user-abc:secret")

	assert.NoError(t, err)
	assert.True(t, expiration.IsZero(), "Expected zero time for never-expiring token")
}

// TestMockRancherServer_GetTokenExpiration_NotFound tests token not found scenario
func TestMockRancherServer_GetTokenExpiration_NotFound(t *testing.T) {
	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password", AuthTypeLocal),
		// No tokens configured
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)

	_, err = client.GetTokenExpiration("non-existent-token:secret")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get token info")
}

// TestMockRancherServer_DetermineTokenRegeneration tests the full token regeneration flow
func TestMockRancherServer_DetermineTokenRegeneration(t *testing.T) {
	// Token expires in 15 days (within 30-day threshold)
	soonExpiry := time.Now().Add(15 * 24 * time.Hour)
	// Token expires in 60 days (outside 30-day threshold)
	laterExpiry := time.Now().Add(60 * 24 * time.Hour)

	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password", AuthTypeLocal),
		WithMockToken("kubeconfig-soon", "kubeconfig-soon:secret", 1296000000, soonExpiry),
		WithMockToken("kubeconfig-later", "kubeconfig-later:secret", 5184000000, laterExpiry),
		WithMockToken("kubeconfig-forever", "kubeconfig-forever:secret", 0, time.Time{}),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		token          string
		forceRefresh   bool
		thresholdDays  int
		expectedRegen  bool
		expectedReason RegenerationReason
	}{
		{
			name:           "token expires soon",
			token:          "kubeconfig-soon:secret",
			forceRefresh:   false,
			thresholdDays:  30,
			expectedRegen:  true,
			expectedReason: ReasonExpiresSoon,
		},
		{
			name:           "token still valid",
			token:          "kubeconfig-later:secret",
			forceRefresh:   false,
			thresholdDays:  30,
			expectedRegen:  false,
			expectedReason: ReasonStillValid,
		},
		{
			name:           "token never expires",
			token:          "kubeconfig-forever:secret",
			forceRefresh:   false,
			thresholdDays:  30,
			expectedRegen:  false,
			expectedReason: ReasonNeverExpires,
		},
		{
			name:           "force refresh overrides",
			token:          "kubeconfig-later:secret",
			forceRefresh:   true,
			thresholdDays:  30,
			expectedRegen:  true,
			expectedReason: ReasonForceRefreshEnabled,
		},
		{
			name:           "no existing token",
			token:          "",
			forceRefresh:   false,
			thresholdDays:  30,
			expectedRegen:  true,
			expectedReason: ReasonNoExistingToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := client.DetermineTokenRegeneration(tt.token, tt.forceRefresh, tt.thresholdDays, "test-cluster")

			assert.Equal(t, tt.expectedRegen, decision.ShouldRegenerate, "ShouldRegenerate mismatch")
			assert.Equal(t, tt.expectedReason, decision.Reason, "Reason mismatch")
		})
	}
}

// TestMockRancherServer_FullWorkflow tests a complete workflow using mock server
func TestMockRancherServer_FullWorkflow(t *testing.T) {
	// Setup mock server with complete configuration
	clusters := []Cluster{
		{ID: "c-m-prod", Name: "production"},
		{ID: "c-m-stage", Name: "staging"},
	}
	futureExpiry := time.Now().Add(60 * 24 * time.Hour)

	mockServer := NewMockRancherServer(
		WithMockUser("admin", "securepass", AuthTypeLocal),
		WithMockClusters(clusters),
		WithMockToken("kubeconfig-admin", "kubeconfig-admin:secret123", 5184000000, futureExpiry),
		WithKubeconfigToken("kubeconfig-admin:secret123"),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	// Step 1: Authenticate
	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"securepass",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Step 2: List clusters
	listedClusters, err := client.ListClusters()
	assert.NoError(t, err)
	assert.Len(t, listedClusters, 2)

	// Step 3: Get kubeconfig token for each cluster
	for _, cluster := range listedClusters {
		token := client.GetClusterToken(cluster.ID)
		assert.NotEmpty(t, token, "Expected token for cluster %s", cluster.Name)
	}

	// Step 4: Check token expiration
	expiration, err := client.GetTokenExpiration("kubeconfig-admin:secret123")
	assert.NoError(t, err)
	assert.False(t, expiration.IsZero())

	// Step 5: Determine if regeneration is needed
	decision := client.DetermineTokenRegeneration("kubeconfig-admin:secret123", false, 30, "production")
	assert.False(t, decision.ShouldRegenerate)
	assert.Equal(t, ReasonStillValid, decision.Reason)

	// Verify all API calls were recorded
	calls := mockServer.GetAPICalls()
	assert.GreaterOrEqual(t, len(calls), 5, "Expected at least 5 API calls")
}

// TestMockRancherServer_ConcurrentAccess tests concurrent access to mock server
func TestMockRancherServer_ConcurrentAccess(t *testing.T) {
	clusters := []Cluster{
		{ID: "c-m-cluster1", Name: "cluster1"},
		{ID: "c-m-cluster2", Name: "cluster2"},
		{ID: "c-m-cluster3", Name: "cluster3"},
	}

	mockServer := NewMockRancherServer(
		WithMockUser("admin", "password", AuthTypeLocal),
		WithMockClusters(clusters),
	)
	defer mockServer.Close()

	logger := zap.NewNop()

	client, err := NewClient(
		mockServer.URL(),
		"admin",
		"password",
		AuthTypeLocal,
		logger,
		false,
		WithHTTPClient(mockServer.Client()),
	)
	assert.NoError(t, err)

	// Make concurrent requests
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.ListClusters()
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

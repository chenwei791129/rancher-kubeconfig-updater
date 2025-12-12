package rancher

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockHTTPClient implements HTTPClient interface for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// TestListClusters_Success tests successfully retrieving cluster list
func TestListClusters_Success(t *testing.T) {
	// Prepare mock response
	mockResponse := `{
		"data": [
			{"id": "c-m-12345", "name": "production"},
			{"id": "c-m-67890", "name": "staging"}
		]
	}`

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Verify request
			assert.Equal(t, "/v3/clusters", req.URL.Path)
			assert.Equal(t, "Bearer test-token-123", req.Header.Get("Authorization"))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			}, nil
		},
	}

	// Create test client
	logger := zap.NewNop()
	client := &Client{
		token:      "test-token-123",
		httpClient: mockClient,
		BaseURL:    "https://rancher.example.com",
		logger:     logger,
	}

	// Execute test
	clusters, err := client.ListClusters()

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, clusters, 2)
	assert.Equal(t, "c-m-12345", clusters[0].ID)
	assert.Equal(t, "production", clusters[0].Name)
	assert.Equal(t, "c-m-67890", clusters[1].ID)
	assert.Equal(t, "staging", clusters[1].Name)
}

// TestListClusters_APIError tests API error handling
func TestListClusters_APIError(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": "unauthorized"}`)),
			}, nil
		},
	}

	logger := zap.NewNop()
	client := &Client{
		token:      "invalid-token",
		httpClient: mockClient,
		BaseURL:    "https://rancher.example.com",
		logger:     logger,
	}

	clusters, err := client.ListClusters()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list clusters")
	assert.Empty(t, clusters)
}

// TestNewClient_WithHTTPTest performs contract testing using httptest
func TestNewClient_WithHTTPTest(t *testing.T) {
	// Create fake Rancher API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify login request contract
		assert.Equal(t, "/v3-public/localProviders/local", r.URL.Path)
		assert.Equal(t, "login", r.URL.Query().Get("action"))
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Respond with contract-compliant data
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"token": "test-token-from-server"}`))
	}))
	defer server.Close()

	logger := zap.NewNop()

	// Create client using test server
	client, err := NewClient(
		server.URL,
		"testuser",
		"testpass",
		AuthTypeLocal,
		logger,
		WithHTTPClient(server.Client()), // Inject test HTTP client
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "test-token-from-server", client.token)
}

// TestGetClusterToken_Success tests retrieving cluster token
func TestGetClusterToken_Success(t *testing.T) {
	// Create mock response matching Rancher API response format
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.Path, "/v3/clusters/c-m-12345")
			assert.Equal(t, "generateKubeconfig", req.URL.Query().Get("action"))

			// Define YAML kubeconfig (better readability)
			kubeconfig := `apiVersion: v1
clusters:
- cluster:
    server: https://rancher.example.com
  name: prod
contexts:
- context:
    cluster: prod
    user: prod
  name: prod
current-context: prod
kind: Config
users:
- name: prod
  user:
    token: kubeconfig-token-xyz123
`
			// Wrap YAML in JSON response
			type response struct {
				Config string `json:"config"`
			}
			resp := response{Config: kubeconfig}
			jsonBytes, _ := json.Marshal(resp)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(jsonBytes)),
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

	token := client.GetClusterToken("c-m-12345")

	assert.Equal(t, "kubeconfig-token-xyz123", token)
}

// TestGetRancherToken_Local tests Local authentication
func TestGetRancherToken_Local(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Local login endpoint
		assert.Contains(t, r.URL.Path, "/v3-public/localProviders/local")
		assert.Equal(t, "login", r.URL.Query().Get("action"))

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"token": "local-token-123"}`))
	}))
	defer server.Close()

	token, err := getRancherToken(
		server.URL,
		"localuser",
		"localpass",
		AuthTypeLocal,
		server.Client(),
	)

	assert.NoError(t, err)
	assert.Equal(t, "local-token-123", token)
}

// TestGetRancherToken_LDAP tests LDAP authentication
func TestGetRancherToken_LDAP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify LDAP login endpoint
		assert.Contains(t, r.URL.Path, "/v3-public/openLdapProviders/openldap")
		assert.Equal(t, "login", r.URL.Query().Get("action"))

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"token": "ldap-token-abc"}`))
	}))
	defer server.Close()

	token, err := getRancherToken(
		server.URL,
		"ldapuser",
		"ldappass",
		AuthTypeLDAP,
		server.Client(),
	)

	assert.NoError(t, err)
	assert.Equal(t, "ldap-token-abc", token)
}

// TestGetRancherToken_InvalidAuthType tests invalid authentication type
func TestGetRancherToken_InvalidAuthType(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Fatal("Should not send HTTP request")
			return nil, nil
		},
	}

	token, err := getRancherToken(
		"https://rancher.example.com",
		"user",
		"pass",
		AuthType("invalid"),
		mockClient,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid auth type")
	assert.Empty(t, token)
}

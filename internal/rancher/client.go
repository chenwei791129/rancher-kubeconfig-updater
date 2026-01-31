// Package rancher provides client functionality for interacting with Rancher API.
package rancher

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// HTTPClient 介面用於抽象化 HTTP 呼叫，使其可測試
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// createTransport creates an HTTP transport with the specified TLS configuration
func createTransport(insecureSkipVerify bool) *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}
}

type Client struct {
	token      string
	httpClient HTTPClient
	BaseURL    string
	logger     *zap.Logger
}

type Cluster struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Clusters []Cluster

// ClientOption 用於配置 Client
type ClientOption func(*Client)

// WithHTTPClient 允許注入自定義的 HTTPClient（用於測試）
func WithHTTPClient(client HTTPClient) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

func NewClient(baseurl, username, password string, authType AuthType, logger *zap.Logger, insecureSkipVerify bool, opts ...ClientOption) (*Client, error) {
	// Create HTTP client with TLS configuration
	transport := createTransport(insecureSkipVerify)
	client := &Client{
		httpClient: &http.Client{Transport: transport},
		BaseURL:    baseurl,
		logger:     logger,
	}

	// Log warning if TLS verification is disabled
	if insecureSkipVerify {
		logger.Warn("⚠️  WARNING: TLS certificate verification is disabled!")
		logger.Warn("⚠️  This is insecure and should only be used in development/test environments.")
		logger.Warn("⚠️  Your connection may be vulnerable to man-in-the-middle attacks.")
	}

	// Apply client options (allows injecting mock client for testing)
	// Note: If WithHTTPClient is used, it will override the transport configuration above.
	// This is intentional for testing purposes where custom HTTP clients (e.g., httptest.Server.Client())
	// need to be injected. In production, WithHTTPClient should not be used.
	for _, opt := range opts {
		opt(client)
	}

	// Obtain authentication token
	token, err := getRancherToken(baseurl, username, password, authType, client.httpClient)
	if err != nil {
		return nil, err
	}

	client.token = token
	logger.Debug("Successfully authenticated with Rancher API")

	return client, nil
}

func (c *Client) ListClusters() (Clusters, error) {
	var clusters Clusters
	type getClustersResponse struct {
		Data []Cluster `json:"data"`
	}

	url := fmt.Sprintf("%s/v3/clusters", c.BaseURL)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)

	body, respCode, err := doRequest(c.httpClient, req)
	if err != nil {
		return clusters, err
	}

	if respCode != http.StatusOK {
		return clusters, fmt.Errorf("failed to list clusters, status %d: %s", respCode, string(body))
	}

	var result getClustersResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return clusters, fmt.Errorf("failed to parse response: %w", err)
	}

	clusters = append(clusters, result.Data...)

	return clusters, nil
}

// GetClusterKubeconfig retrieves the full kubeconfig for a cluster from Rancher API.
// The returned *api.Config includes the primary Rancher proxy context and any
// Downstream Directly contexts if the cluster has them configured.
func (c *Client) GetClusterKubeconfig(clusterID string) (*api.Config, error) {
	type getClusterKubeconfigResponse struct {
		Config string `json:"config"`
	}

	url := fmt.Sprintf("%s/v3/clusters/%s?action=generateKubeconfig", c.BaseURL, clusterID)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)

	body, respCode, err := doRequest(c.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	if respCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get kubeconfig, status %d: %s", respCode, string(body))
	}

	var result getClusterKubeconfigResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig response: %w", err)
	}

	// Parse the kubeconfig YAML using client-go
	kubeconfig, err := clientcmd.Load([]byte(result.Config))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig YAML: %w", err)
	}

	return kubeconfig, nil
}

// GetClusterToken retrieves only the token from a cluster's kubeconfig.
// This is a convenience method that calls GetClusterKubeconfig and extracts the token.
// Returns empty string if the token cannot be retrieved.
func (c *Client) GetClusterToken(clusterID string) string {
	kubeconfig, err := c.GetClusterKubeconfig(clusterID)
	if err != nil {
		return ""
	}

	// Extract token from the first user in the kubeconfig
	for _, authInfo := range kubeconfig.AuthInfos {
		if authInfo.Token != "" {
			return authInfo.Token
		}
	}

	return ""
}

func doRequest(client HTTPClient, req *http.Request) ([]byte, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	return body, resp.StatusCode, nil
}

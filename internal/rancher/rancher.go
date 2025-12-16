package rancher

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// HTTPClient 介面用於抽象化 HTTP 呼叫，使其可測試
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type AuthType string

const (
	AuthTypeLDAP  AuthType = "ldap"
	AuthTypeLocal AuthType = "local"
)

const (
	LDAP_LOGIN_URL  = "/v3-public/openLdapProviders/openldap?action=login"
	LOCAL_LOGIN_URL = "/v3-public/localProviders/local?action=login"
)

var (
	insecure = false
	tr       *http.Transport
)

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

func init() {
	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
}

// ClientOption 用於配置 Client
type ClientOption func(*Client)

// WithHTTPClient 允許注入自定義的 HTTPClient（用於測試）
func WithHTTPClient(client HTTPClient) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

func NewClient(baseurl, username, password string, authType AuthType, logger *zap.Logger, opts ...ClientOption) (*Client, error) {
	// 預設使用標準 HTTP client
	client := &Client{
		httpClient: &http.Client{Transport: tr},
		BaseURL:    baseurl,
		logger:     logger,
	}

	// 套用選項（可注入 mock client）
	for _, opt := range opts {
		opt(client)
	}

	// 取得 token
	token, err := getRancherToken(baseurl, username, password, authType, client.httpClient)
	if err != nil {
		return nil, err
	}

	client.token = token
	logger.Debug("Successfully authenticated with Rancher API")

	return client, nil
}

// GET /v3/clusters
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

func (c *Client) GetClusterToken(clusterId string) string {
	type KubeConfigToken struct {
		Token string `yaml:"token"`
	}

	type KubeConfigUser struct {
		User KubeConfigToken `yaml:"user"`
	}

	type Kubeconfig struct {
		Users []KubeConfigUser `yaml:"users"`
	}

	type getClusterTokenResponse struct {
		Config string `json:"config"`
	}
	url := fmt.Sprintf("%s/v3/clusters/%s?action=generateKubeconfig", c.BaseURL, clusterId)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)

	body, respCode, err := doRequest(c.httpClient, req)
	if err != nil || respCode != http.StatusOK {
		return ""
	}

	var result getClusterTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	// fmt.Printf("[debug] config: %s", result.Config)

	var kubeconfig Kubeconfig
	if err := yaml.Unmarshal([]byte(result.Config), &kubeconfig); err != nil {
		return ""
	}

	return kubeconfig.Users[0].User.Token
}

// POST /v3-public/openLdapProviders/openldap?action=login or /v3-public/localProviders/local?action=login
func getRancherToken(baseurl, username, password string, authType AuthType, httpClient HTTPClient) (string, error) {
	type loginResponse struct {
		Token string `json:"token"`
	}

	// Prepare login request body
	body := map[string]string{
		"username":     username,
		"password":     password,
		"responseType": "json",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Select login URL based on auth type
	var loginURL string
	switch authType {
	case AuthTypeLDAP:
		loginURL = LDAP_LOGIN_URL
	case AuthTypeLocal:
		loginURL = LOCAL_LOGIN_URL
	default:
		return "", fmt.Errorf("invalid auth type: %s", authType)
	}

	url := fmt.Sprintf("%s%s", baseurl, loginURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	respBody, respCode, err := doRequest(httpClient, req)
	if err != nil {
		return "", err
	}

	if respCode != http.StatusCreated {
		return "", fmt.Errorf("login failed with status %d: %s", respCode, string(respBody))
	}

	var result loginResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Token == "" {
		return "", fmt.Errorf("token not found in response")
	}

	return result.Token, nil
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

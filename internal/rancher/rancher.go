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

var (
	insecure = false
	tr       *http.Transport
)

type Client struct {
	token   string
	client  *http.Client
	BaseURL string
	logger  *zap.Logger
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

func NewClient(baseurl, username, password string, logger *zap.Logger) (*Client, error) {
	token, err := getRancherToken(baseurl, username, password)
	if err != nil {
		return nil, err
	}

	logger.Debug("Successfully authenticated with Rancher API")

	return &Client{
		token:   token,
		client:  &http.Client{Transport: tr},
		BaseURL: baseurl,
		logger:  logger,
	}, nil
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

	body, respCode, err := doRequest(c.client, req)
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

	body, respCode, err := doRequest(c.client, req)
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

// POST /v3-public/openLdapProviders/openldap?action=login
func getRancherToken(baseurl, username, password string) (string, error) {
	type loginResponse struct {
		Token string `json:"token"`
	}

	httpClient := &http.Client{Transport: tr}

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

	url := fmt.Sprintf("%s/v3-public/openLdapProviders/openldap?action=login", baseurl)

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

func doRequest(client *http.Client, req *http.Request) ([]byte, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	return body, resp.StatusCode, nil
}

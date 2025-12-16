package rancher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type AuthType string

const (
	AuthTypeLDAP  AuthType = "ldap"
	AuthTypeLocal AuthType = "local"
)

const (
	LDAP_LOGIN_URL  = "/v3-public/openLdapProviders/openldap?action=login"
	LOCAL_LOGIN_URL = "/v3-public/localProviders/local?action=login"
)

// getRancherToken authenticates with Rancher and returns an API token
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

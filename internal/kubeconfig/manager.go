// Package kubeconfig provides functionality for managing Kubernetes configuration files.
package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// LoadKubeconfig loads a kubeconfig file using the following precedence order:
//  1. Explicit path parameter (if provided) - highest priority
//  2. KUBECONFIG environment variable (if set) - respects multiple files
//  3. Default location: ~/.kube/config - fallback
//
// When KUBECONFIG contains multiple files:
//   - For reading: uses the first existing file
//   - If no files exist: returns an empty config
//
// This implementation uses client-go's ClientConfigLoadingRules to ensure
// compatibility with kubectl and other Kubernetes tools.
func LoadKubeconfig(path string) (*api.Config, error) {
	// Use client-go's ClientConfigLoadingRules to respect KUBECONFIG and handle all edge cases.
	// This ensures compatibility with other client-go based tools and kubectl for all common scenarios.
	// Note: The behavior for multiple non-existent files in KUBECONFIG may differ slightly from
	// kubectl's PathOptions, but this edge case is rare and the common cases (single file,
	// multiple files with at least one existing) behave identically.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	// If an explicit path is provided, use it; otherwise, use client-go's default logic
	if path != "" {
		expandedPath, err := expandPath(path)
		if err != nil {
			return nil, fmt.Errorf("failed to expand path %q: %w", path, err)
		}
		loadingRules.ExplicitPath = expandedPath
	}
	
	// Get the actual file path we'll use (respects KUBECONFIG, precedence, etc.)
	targetPath := loadingRules.GetDefaultFilename()

	// Check if file exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// If file doesn't exist, return a new empty kubeconfig structure
		return api.NewConfig(), nil
	}

	// Load kubeconfig using client-go
	config, err := clientcmd.LoadFromFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig file: %w", err)
	}

	return config, nil
}

func UpdateTokenByName(c *api.Config, clusterID, clusterName, token, rancherURL string, autoCreate bool, logger *zap.Logger) error {
	// Check if user already exists
	if authInfo, exists := c.AuthInfos[clusterName]; exists {
		authInfo.Token = token
		return nil
	}

	// If auto-create is enabled, create new cluster, context, and user entries
	if autoCreate {
		// Initialize maps if nil
		if c.Clusters == nil {
			c.Clusters = make(map[string]*api.Cluster)
		}
		if c.Contexts == nil {
			c.Contexts = make(map[string]*api.Context)
		}
		if c.AuthInfos == nil {
			c.AuthInfos = make(map[string]*api.AuthInfo)
		}

		// Create new cluster entry with correct server URL using cluster ID
		// Remove trailing slash from rancherURL to avoid double slashes
		cleanURL := strings.TrimSuffix(rancherURL, "/")
		c.Clusters[clusterName] = &api.Cluster{
			Server: cleanURL + "/k8s/clusters/" + clusterID,
		}

		// Create new context entry
		c.Contexts[clusterName] = &api.Context{
			Cluster:  clusterName,
			AuthInfo: clusterName,
		}

		// Create new user entry
		c.AuthInfos[clusterName] = &api.AuthInfo{
			Token: token,
		}

		logger.Info("Created new kubeconfig entry for cluster: " + clusterName)
		return nil
	}

	logger.Warn("Cluster not found in kubeconfig, skipping: " + clusterName)
	return fmt.Errorf("user %s not found in kubeconfig", clusterName)
}

// SaveKubeconfig saves a kubeconfig file using the following precedence order:
//  1. Explicit path parameter (if provided) - highest priority
//  2. KUBECONFIG environment variable (if set) - handles multiple files
//  3. Default location: ~/.kube/config - fallback
//
// When KUBECONFIG contains multiple files:
//   - If any file exists: writes to the first existing file
//   - If no files exist: writes to the first file in the list
//
// The file is saved with secure permissions (0600 on Unix systems) and a backup
// is created if the file already exists.
//
// This implementation uses client-go's ClientConfigLoadingRules to ensure
// compatibility with kubectl and other Kubernetes tools.
func SaveKubeconfig(c *api.Config, path string, logger *zap.Logger) error {
	// Use client-go's loading rules to respect KUBECONFIG and handle all edge cases
	// This follows kubectl behavior exactly for write operations
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	// If an explicit path is provided, use it; otherwise, use client-go's default logic
	if path != "" {
		expandedPath, err := expandPath(path)
		if err != nil {
			return fmt.Errorf("failed to expand path %q: %w", path, err)
		}
		loadingRules.ExplicitPath = expandedPath
	}
	
	// Get the actual file path we'll use (respects KUBECONFIG, precedence, etc.)
	targetPath := loadingRules.GetDefaultFilename()

	// 2. Ensure directory exists with platform-appropriate permissions
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, getSecureDirMode()); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 3. Create backup if file exists (fail if backup fails)
	backupPath, err := createBackup(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Log backup path if a backup was created
	if backupPath != "" && logger != nil {
		logger.Info("Created backup of kubeconfig file: " + backupPath)
	}

	// 4. Write kubeconfig using client-go
	if err := clientcmd.WriteToFile(*c, targetPath); err != nil {
		return fmt.Errorf("failed to write kubeconfig file: %w", err)
	}

	// 5. Set secure file permissions (client-go might not set them correctly on all platforms)
	if err := os.Chmod(targetPath, getSecureFileMode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// getSecureFileMode returns the appropriate file mode for secure kubeconfig files
// Windows ignores Unix permissions, so we use default values there
func getSecureFileMode() os.FileMode {
	if runtime.GOOS == "windows" {
		// Windows will ignore Unix permissions, use default value
		return 0666
	}
	return 0600 // Unix: owner read/write only
}

// getSecureDirMode returns the appropriate directory mode for secure kubeconfig directories
func getSecureDirMode() os.FileMode {
	if runtime.GOOS == "windows" {
		return 0777
	}
	return 0700 // Unix: owner read/write/execute only
}

// GetDefaultKubeconfigPath returns the default kubeconfig path for the current platform
func GetDefaultKubeconfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".kube", "config"), nil
}

// expandPath expands the given path, handling various path formats across platforms
func expandPath(path string) (string, error) {
	// Handle empty path - use default
	if path == "" {
		return GetDefaultKubeconfigPath()
	}

	// Handle ~ prefix (Unix-style)
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home dir: %w", err)
		}

		if path == "~" {
			return homeDir, nil
		}

		var remainingPath string
		// Remove ~/ or ~\ (support both separators)
		if len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
			remainingPath = path[2:]
		} else {
			remainingPath = path[1:]
		}

		// Normalize path separators: replace backslashes with forward slashes,
		// then convert to OS-specific separators
		remainingPath = strings.ReplaceAll(remainingPath, "\\", "/")
		remainingPath = filepath.FromSlash(remainingPath)
		return filepath.Join(homeDir, remainingPath), nil
	}

	// Clean path (normalize separators)
	return filepath.Clean(path), nil
}

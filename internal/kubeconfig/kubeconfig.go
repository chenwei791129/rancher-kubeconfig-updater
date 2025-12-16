package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Kubeconfig struct {
	APIVersion     string          `yaml:"apiVersion"`
	Kind           string          `yaml:"kind"`
	Clusters       []ConfigCluster `yaml:"clusters"`
	Contexts       []ConfigContext `yaml:"contexts"`
	CurrentContext string          `yaml:"current-context"`
	Users          []ConfigUser    `yaml:"users"`
}

type ConfigCluster struct {
	Name    string         `yaml:"name"`
	Cluster map[string]any `yaml:"cluster"`
}

type ConfigContext struct {
	Name    string         `yaml:"name"`
	Context map[string]any `yaml:"context"`
}

type ConfigUser struct {
	Name string `yaml:"name"`
	User User   `yaml:"user"`
}

type User struct {
	Token string `yaml:"token"`
}

func LoadKubeconfig(path string) (Kubeconfig, error) {
	var config Kubeconfig

	expandedPath, err := expandPath(path)
	if err != nil {
		return config, fmt.Errorf("failed to expand path: %w", err)
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		// If file doesn't exist, return a new empty kubeconfig structure
		if os.IsNotExist(err) {
			return Kubeconfig{
				APIVersion: "v1",
				Kind:       "Config",
				Clusters:   []ConfigCluster{},
				Contexts:   []ConfigContext{},
				Users:      []ConfigUser{},
			}, nil
		}
		return config, fmt.Errorf("failed to read kubeconfig file: %w", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse kubeconfig YAML: %w", err)
	}

	return config, nil
}

func (c *Kubeconfig) UpdateTokenByName(clusterID, clusterName, token, rancherURL string, autoCreate bool, logger *zap.Logger) error {
	// Check if user already exists
	for i, user := range c.Users {
		if user.Name == clusterName {
			c.Users[i].User.Token = token
			return nil
		}
	}

	// If auto-create is enabled, create new cluster, context, and user entries
	if autoCreate {
		// Create new cluster entry with correct server URL using cluster ID
		// Remove trailing slash from rancherURL to avoid double slashes
		cleanURL := strings.TrimSuffix(rancherURL, "/")
		newCluster := ConfigCluster{
			Name: clusterName,
			Cluster: map[string]any{
				"server": cleanURL + "/k8s/clusters/" + clusterID,
			},
		}
		c.Clusters = append(c.Clusters, newCluster)

		// Create new context entry
		newContext := ConfigContext{
			Name: clusterName,
			Context: map[string]any{
				"cluster": clusterName,
				"user":    clusterName,
			},
		}
		c.Contexts = append(c.Contexts, newContext)

		// Create new user entry
		newUser := ConfigUser{
			Name: clusterName,
			User: User{
				Token: token,
			},
		}
		c.Users = append(c.Users, newUser)

		logger.Info("Created new kubeconfig entry for cluster: " + clusterName)
		return nil
	}

	logger.Warn("Cluster not found in kubeconfig, skipping: " + clusterName)
	return fmt.Errorf("user %s not found in kubeconfig", clusterName)
}

func (c *Kubeconfig) SaveKubeconfig(path string) error {
	// 1. Expand path
	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// 2. Ensure directory exists with platform-appropriate permissions
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, getSecureDirMode()); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 3. Marshal data
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal kubeconfig to YAML: %w", err)
	}

	// 4. Create backup if file exists (fail if backup fails)
	if err := createBackup(expandedPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// 5. Atomic write with platform-appropriate permissions
	if err := atomicWriteFile(expandedPath, data, getSecureFileMode()); err != nil {
		return fmt.Errorf("failed to write kubeconfig file: %w", err)
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

// atomicWriteFile writes data to a file atomically by writing to a temp file first,
// then renaming it to the target path. This ensures the file is never in a partially written state.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".kubeconfig.tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup of temp file on failure
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic operation: rename temp file to target path
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// createBackup creates a backup of the file at the given path.
// The backup filename includes a microsecond-precision timestamp to ensure uniqueness.
// If the file doesn't exist or backup fails, it logs a warning but doesn't stop the operation.
func createBackup(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil // New file, no backup needed
	}
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory: %s", path)
	}

	// Read original file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read original file: %w", err)
	}

	// Backup filename: unique with microsecond timestamp
	backupPath := fmt.Sprintf("%s.backup.%s", path,
		time.Now().Format("20060102-150405.000000"))

	// Write backup using atomic write with platform-appropriate permissions
	return atomicWriteFile(backupPath, data, getSecureFileMode())
}

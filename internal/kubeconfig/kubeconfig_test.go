package kubeconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd/api"
)

// TestExpandPath tests the expandPath function with various path formats
func TestExpandPath(t *testing.T) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() error = %v", err)
	}
	pathSeparator := string(os.PathSeparator)
	defaultPath, err := GetDefaultKubeconfigPath()
	if err != nil {
		t.Fatalf("GetDefaultKubeconfigPath() error = %v", err)
	}
	tests := []struct {
		name    string
		input   string
		expect  string
		wantErr bool
	}{
		{"tilde only", "~", userHomeDir, false},
		{"tilde with slash", "~/.kube/config", filepath.FromSlash(userHomeDir + "/.kube/config"), false},
		{"tilde with backslash", "~\\.kube\\config", userHomeDir + pathSeparator + ".kube" + pathSeparator + "config", false},
		{"absolute path unix", "/home/user/.kube/config", filepath.FromSlash("/home/user/.kube/config"), false},
		{"absolute path windows", "C:\\Users\\user\\.kube\\config", "C:\\Users\\user\\.kube\\config", false},
		{"relative path", ".kube/config", filepath.FromSlash(".kube/config"), false},
		{"empty path", "", defaultPath, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPath(tt.input)
			t.Logf("expandPath() input = %v, result = %v, error = %v", tt.input, result, err)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if result == "" && !tt.wantErr {
				t.Error("expandPath() returned empty string")
			}
			if result != tt.expect {
				t.Errorf("expandPath() expected %v, got %v", tt.expect, result)
			}
		})
	}
}

// TestGetDefaultKubeconfigPath tests the GetDefaultKubeconfigPath function
func TestGetDefaultKubeconfigPath(t *testing.T) {
	path, err := GetDefaultKubeconfigPath()
	if err != nil {
		t.Fatalf("GetDefaultKubeconfigPath() error = %v", err)
	}

	if path == "" {
		t.Error("GetDefaultKubeconfigPath() returned empty string")
	}

	// Check path contains .kube and config
	if !strings.Contains(path, ".kube") {
		t.Error("Path should contain .kube directory")
	}
	if !strings.HasSuffix(path, "config") {
		t.Error("Path should end with 'config'")
	}
}

// TestGetSecureFileMode tests the getSecureFileMode function
func TestGetSecureFileMode(t *testing.T) {
	mode := getSecureFileMode()

	if runtime.GOOS == "windows" {
		if mode != 0666 {
			t.Errorf("On Windows, expected mode 0666, got %o", mode)
		}
	} else {
		if mode != 0600 {
			t.Errorf("On Unix, expected mode 0600, got %o", mode)
		}
	}
}

// TestGetSecureDirMode tests the getSecureDirMode function
func TestGetSecureDirMode(t *testing.T) {
	mode := getSecureDirMode()

	if runtime.GOOS == "windows" {
		if mode != 0777 {
			t.Errorf("On Windows, expected mode 0777, got %o", mode)
		}
	} else {
		if mode != 0700 {
			t.Errorf("On Unix, expected mode 0700, got %o", mode)
		}
	}
}

// ============================================================================
// Test Helper Functions
// ============================================================================

// createTestLogger creates a no-op logger for testing
func createTestLogger() *zap.Logger {
	return zap.NewNop()
}

// createTestKubeconfigContent returns a valid test kubeconfig YAML string
func createTestKubeconfigContent() string {
	return `apiVersion: v1
kind: Config
clusters:
- name: test-cluster
  cluster:
    server: https://rancher.example.com/k8s/clusters/c-test123
contexts:
- name: test-cluster
  context:
    cluster: test-cluster
    user: test-cluster
current-context: test-cluster
users:
- name: test-cluster
  user:
    token: test-token-123
`
}

// createTestKubeconfig creates a test Kubeconfig structure
func createTestKubeconfig() *api.Config {
	config := api.NewConfig()
	config.Clusters["test-cluster"] = &api.Cluster{
		Server: "https://rancher.example.com/k8s/clusters/c-test123",
	}
	config.Contexts["test-cluster"] = &api.Context{
		Cluster:  "test-cluster",
		AuthInfo: "test-cluster",
	}
	config.CurrentContext = "test-cluster"
	config.AuthInfos["test-cluster"] = &api.AuthInfo{
		Token: "test-token-123",
	}
	return config
}

// ============================================================================
// LoadKubeconfig() Tests
// ============================================================================

// TestLoadKubeconfig_ValidFile tests loading a valid kubeconfig file
func TestLoadKubeconfig_ValidFile(t *testing.T) {
	// Create temp file with test content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")
	content := createTestKubeconfigContent()

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load kubeconfig
	config, err := LoadKubeconfig(testFile)
	if err != nil {
		t.Fatalf("LoadKubeconfig() error = %v", err)
	}

	// Verify structure
	if len(config.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(config.Clusters))
	}
	if len(config.Contexts) != 1 {
		t.Errorf("Expected 1 context, got %d", len(config.Contexts))
	}
	if len(config.AuthInfos) != 1 {
		t.Errorf("Expected 1 user, got %d", len(config.AuthInfos))
	}

	// Verify cluster details
	if config.Clusters["test-cluster"] == nil {
		t.Error("Expected cluster test-cluster to exist")
	} else if config.Clusters["test-cluster"].Server != "https://rancher.example.com/k8s/clusters/c-test123" {
		t.Errorf("Expected server URL, got %s", config.Clusters["test-cluster"].Server)
	}

	// Verify user details
	if config.AuthInfos["test-cluster"] == nil {
		t.Error("Expected user test-cluster to exist")
	} else if config.AuthInfos["test-cluster"].Token != "test-token-123" {
		t.Errorf("Expected token test-token-123, got %s", config.AuthInfos["test-cluster"].Token)
	}
}

// TestLoadKubeconfig_FileNotExist tests loading a non-existent file
func TestLoadKubeconfig_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "does-not-exist")

	config, err := LoadKubeconfig(nonExistentFile)
	if err != nil {
		t.Fatalf("LoadKubeconfig() should not return error for non-existent file, got: %v", err)
	}

	// Should return empty but valid structure
	if config.Clusters == nil {
		t.Error("Expected non-nil Clusters map")
	}
	if config.Contexts == nil {
		t.Error("Expected non-nil Contexts map")
	}
	if config.AuthInfos == nil {
		t.Error("Expected non-nil AuthInfos map")
	}
}

// TestLoadKubeconfig_InvalidYAML tests loading a file with invalid YAML
func TestLoadKubeconfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid")
	invalidContent := "this is not: valid: yaml: content::"

	if err := os.WriteFile(testFile, []byte(invalidContent), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadKubeconfig(testFile)
	if err == nil {
		t.Error("LoadKubeconfig() should return error for invalid YAML")
	}
}

// TestLoadKubeconfig_EmptyPath tests loading with empty path
func TestLoadKubeconfig_EmptyPath(t *testing.T) {
	// Empty path should use default path
	_, err := LoadKubeconfig("")
	// We don't care if it succeeds or fails (file may not exist)
	// Just verify it attempted to use the default path
	if err != nil {
		defaultPath, _ := GetDefaultKubeconfigPath()
		// Error message should reference the default path or be a "not exist" error
		if !strings.Contains(err.Error(), defaultPath) && !os.IsNotExist(err) && !strings.Contains(err.Error(), "no such file") {
			t.Logf("Note: LoadKubeconfig with empty path returned: %v", err)
		}
	}
}

// ============================================================================
// SaveKubeconfig Tests
// ============================================================================

// TestSaveKubeconfig_Success tests successful save operation
func TestSaveKubeconfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")

	config := createTestKubeconfig()

	err := SaveKubeconfig(config, testFile, nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("SaveKubeconfig() did not create file")
	}

	// Verify file permissions
	info, _ := os.Stat(testFile)
	mode := info.Mode().Perm()
	expectedMode := getSecureFileMode()
	if runtime.GOOS != "windows" && mode != expectedMode {
		t.Errorf("Expected file mode %o, got %o", expectedMode, mode)
	}

	// Load and verify content
	loaded, err := LoadKubeconfig(testFile)
	if err != nil {
		t.Fatalf("Failed to load saved file: %v", err)
	}
	if len(loaded.AuthInfos) != 1 || loaded.AuthInfos["test-cluster"].Token != "test-token-123" {
		t.Error("Saved content doesn't match original")
	}
}

// TestSaveKubeconfig_AutoCreateDirectory tests automatic directory creation
func TestSaveKubeconfig_AutoCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dirs", "config")

	config := createTestKubeconfig()

	err := SaveKubeconfig(config, nestedPath, nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Error("SaveKubeconfig() did not create file in nested directory")
	}

	// Verify directory permissions
	dirPath := filepath.Dir(nestedPath)
	dirInfo, _ := os.Stat(dirPath)
	dirMode := dirInfo.Mode().Perm()
	expectedDirMode := getSecureDirMode()
	if runtime.GOOS != "windows" && dirMode != expectedDirMode {
		t.Errorf("Expected directory mode %o, got %o", expectedDirMode, dirMode)
	}
}

// TestSaveKubeconfig_BackupCreation tests backup file creation
func TestSaveKubeconfig_BackupCreation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")

	// Create initial file
	initialConfig := createTestKubeconfig()
	initialConfig.AuthInfos["test-cluster"].Token = "old-token"
	if err := SaveKubeconfig(initialConfig, testFile, nil); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Save updated config
	updatedConfig := createTestKubeconfig()
	updatedConfig.AuthInfos["test-cluster"].Token = "new-token"
	if err := SaveKubeconfig(updatedConfig, testFile, nil); err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify backup file exists
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	backupFound := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "config.backup.") {
			backupFound = true

			// Load backup and verify it has old token
			backupPath := filepath.Join(tmpDir, entry.Name())
			backupConfig, err := LoadKubeconfig(backupPath)
			if err != nil {
				t.Errorf("Failed to load backup: %v", err)
			}
			if backupConfig.AuthInfos["test-cluster"].Token != "old-token" {
				t.Errorf("Backup should have old-token, got %s", backupConfig.AuthInfos["test-cluster"].Token)
			}
			break
		}
	}

	if !backupFound {
		t.Error("Backup file was not created")
	}

	// Verify main file has new token
	mainConfig, _ := LoadKubeconfig(testFile)
	if mainConfig.AuthInfos["test-cluster"].Token != "new-token" {
		t.Errorf("Main file should have new-token, got %s", mainConfig.AuthInfos["test-cluster"].Token)
	}
}

// TestSaveKubeconfig_YAMLSerialization tests YAML serialization correctness
func TestSaveKubeconfig_YAMLSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")

	// Create complex config
	config := api.NewConfig()
	config.Clusters["cluster-1"] = &api.Cluster{
		Server:                   "https://server1.example.com",
		CertificateAuthorityData: []byte("base64data"),
	}
	config.Clusters["cluster-2"] = &api.Cluster{
		Server: "https://server2.example.com",
	}
	config.Contexts["context-1"] = &api.Context{
		Cluster:  "cluster-1",
		AuthInfo: "user-1",
	}
	config.CurrentContext = "context-1"
	config.AuthInfos["user-1"] = &api.AuthInfo{
		Token: "token-1",
	}

	if err := SaveKubeconfig(config, testFile, nil); err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Load and verify all fields
	loaded, err := LoadKubeconfig(testFile)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if len(loaded.Clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(loaded.Clusters))
	}
	if loaded.CurrentContext != "context-1" {
		t.Errorf("Expected current-context context-1, got %s", loaded.CurrentContext)
	}
}

// ============================================================================
// UpdateTokenByName Tests
// ============================================================================

// TestUpdateTokenByName_ExistingUser tests updating an existing user's token
func TestUpdateTokenByName_ExistingUser(t *testing.T) {
	config := createTestKubeconfig()
	logger := createTestLogger()

	err := UpdateTokenByName(config, "c-test123", "test-cluster", "new-token-456", "https://rancher.example.com", false, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Verify token was updated
	if config.AuthInfos["test-cluster"].Token != "new-token-456" {
		t.Errorf("Expected token new-token-456, got %s", config.AuthInfos["test-cluster"].Token)
	}

	// Verify other fields unchanged
	if len(config.Clusters) != 1 {
		t.Error("Clusters should not change")
	}
	if len(config.Contexts) != 1 {
		t.Error("Contexts should not change")
	}
}

// TestUpdateTokenByName_AutoCreateTrue tests auto-creation of new user
func TestUpdateTokenByName_AutoCreateTrue(t *testing.T) {
	config := api.NewConfig()
	logger := createTestLogger()

	err := UpdateTokenByName(config, "c-newcluster", "new-cluster", "new-token", "https://rancher.example.com", true, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Verify cluster was created
	if len(config.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(config.Clusters))
	}
	if config.Clusters["new-cluster"] == nil {
		t.Error("Expected cluster new-cluster to exist")
	}
	expectedServer := "https://rancher.example.com/k8s/clusters/c-newcluster"
	if config.Clusters["new-cluster"].Server != expectedServer {
		t.Errorf("Expected server %s, got %v", expectedServer, config.Clusters["new-cluster"].Server)
	}

	// Verify context was created
	if len(config.Contexts) != 1 {
		t.Fatalf("Expected 1 context, got %d", len(config.Contexts))
	}
	if config.Contexts["new-cluster"] == nil {
		t.Error("Expected context new-cluster to exist")
	}

	// Verify user was created
	if len(config.AuthInfos) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(config.AuthInfos))
	}
	if config.AuthInfos["new-cluster"] == nil {
		t.Error("Expected user new-cluster to exist")
	}
	if config.AuthInfos["new-cluster"].Token != "new-token" {
		t.Errorf("Expected token new-token, got %s", config.AuthInfos["new-cluster"].Token)
	}
}

// TestUpdateTokenByName_AutoCreateFalse tests error when user doesn't exist
func TestUpdateTokenByName_AutoCreateFalse(t *testing.T) {
	config := api.NewConfig()
	logger := createTestLogger()

	err := UpdateTokenByName(config, "c-test", "nonexistent", "token", "https://rancher.example.com", false, logger)
	if err == nil {
		t.Error("UpdateTokenByName() should return error when autoCreate=false and user doesn't exist")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestUpdateTokenByName_RancherURLFormatting tests various Rancher URL formats
func TestUpdateTokenByName_RancherURLFormatting(t *testing.T) {
	tests := []struct {
		name        string
		rancherURL  string
		clusterID   string
		expectedURL string
	}{
		{
			name:        "URL without trailing slash",
			rancherURL:  "https://rancher.example.com",
			clusterID:   "c-abc123",
			expectedURL: "https://rancher.example.com/k8s/clusters/c-abc123",
		},
		{
			name:        "URL with trailing slash",
			rancherURL:  "https://rancher.example.com/",
			clusterID:   "c-abc123",
			expectedURL: "https://rancher.example.com/k8s/clusters/c-abc123",
		},
		{
			name:        "HTTP URL",
			rancherURL:  "http://rancher.local",
			clusterID:   "c-test",
			expectedURL: "http://rancher.local/k8s/clusters/c-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := api.NewConfig()
			logger := createTestLogger()

			err := UpdateTokenByName(config, tt.clusterID, "test", "token", tt.rancherURL, true, logger)
			if err != nil {
				t.Fatalf("UpdateTokenByName() error = %v", err)
			}

			if config.Clusters["test"].Server != tt.expectedURL {
				t.Errorf("Expected server %s, got %v", tt.expectedURL, config.Clusters["test"].Server)
			}
		})
	}
}

// TestUpdateTokenByName_SpecialCharacters tests cluster names with special characters
func TestUpdateTokenByName_SpecialCharacters(t *testing.T) {
	specialNames := []string{
		"cluster-with-dashes",
		"cluster_with_underscores",
		"cluster.with.dots",
		"cluster123",
	}

	for _, name := range specialNames {
		t.Run(name, func(t *testing.T) {
			config := api.NewConfig()
			logger := createTestLogger()

			err := UpdateTokenByName(config, "c-test", name, "token", "https://rancher.example.com", true, logger)
			if err != nil {
				t.Fatalf("UpdateTokenByName() failed for name %s: %v", name, err)
			}

			if config.AuthInfos[name] == nil {
				t.Errorf("Expected user name %s to exist", name)
			}
		})
	}
}

// ============================================================================
// createBackup Tests
// ============================================================================

// TestCreateBackup_Success tests successful backup creation
func TestCreateBackup_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")
	originalContent := []byte("original content")

	// Create original file
	if err := os.WriteFile(testFile, originalContent, 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	backupPath, err := createBackup(testFile)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}
	if backupPath == "" {
		t.Fatal("createBackup() should return backup path")
	}

	// Find backup file
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	backupFound := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "config.backup.") {
			backupFound = true

			// Verify backup content matches original
			backupPath := filepath.Join(tmpDir, entry.Name())
			backupContent, err := os.ReadFile(backupPath)
			if err != nil {
				t.Fatalf("Failed to read backup: %v", err)
			}
			if string(backupContent) != string(originalContent) {
				t.Errorf("Backup content doesn't match original")
			}
			break
		}
	}

	if !backupFound {
		t.Error("Backup file was not created")
	}
}

// TestCreateBackup_FileNotExist tests backup when file doesn't exist
func TestCreateBackup_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "does-not-exist")

	// Should not return error for non-existent file
	backupPath, err := createBackup(nonExistentFile)
	if err != nil {
		t.Errorf("createBackup() should not error for non-existent file, got: %v", err)
	}
	if backupPath != "" {
		t.Error("createBackup() should return empty path for non-existent file")
	}

	// Verify no backup was created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), "backup") {
			t.Error("Backup file should not be created for non-existent file")
		}
	}
}

// TestCreateBackup_FilenameFormat tests backup filename format with timestamp
func TestCreateBackup_FilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")

	// Create original file
	if err := os.WriteFile(testFile, []byte("content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	backupPath, err := createBackup(testFile)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}
	if backupPath == "" {
		t.Fatal("createBackup() should return backup path")
	}

	// Find and verify backup filename format
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "config.backup.") {
			// Verify filename format: config.backup.YYYYMMDD-HHMMSS.mmmmmm
			name := entry.Name()

			// Extract timestamp part
			parts := strings.Split(name, ".")
			if len(parts) < 3 {
				t.Errorf("Backup filename should have timestamp: %s", name)
			}

			// Verify timestamp format (basic check)
			timestamp := parts[2]
			if len(timestamp) < 15 { // YYYYMMDD-HHMMSS minimum
				t.Errorf("Timestamp format incorrect: %s", timestamp)
			}
			return
		}
	}

	t.Error("Backup file not found")
}

// TestCreateBackup_Directory tests error when trying to backup a directory
func TestCreateBackup_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")

	// Create directory
	if err := os.Mkdir(subDir, 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Should return error for directory
	backupPath, err := createBackup(subDir)
	if err == nil {
		t.Error("createBackup() should return error for directory")
	}
	if backupPath != "" {
		t.Error("createBackup() should return empty path on error")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("Expected directory error, got: %v", err)
	}
}

// TestCreateBackup_Permissions tests backup with different file permissions
func TestCreateBackup_Permissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission tests not applicable on Windows")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")

	// Create file with specific permissions
	if err := os.WriteFile(testFile, []byte("content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	backupPath, err := createBackup(testFile)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}
	if backupPath == "" {
		t.Fatal("createBackup() should return backup path")
	}

	// Find backup and verify it has secure permissions
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "config.backup.") {
			backupPath := filepath.Join(tmpDir, entry.Name())
			info, err := os.Stat(backupPath)
			if err != nil {
				t.Fatalf("Failed to stat backup: %v", err)
			}

			mode := info.Mode().Perm()
			expectedMode := getSecureFileMode()
			if mode != expectedMode {
				t.Errorf("Expected backup permissions %o, got %o", expectedMode, mode)
			}
			return
		}
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

// TestIntegration_CompleteUpdateFlow tests the complete flow
func TestIntegration_CompleteUpdateFlow(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kube", "config")
	logger := createTestLogger()

	// Step 1: Load non-existent config (should return empty structure)
	config, err := LoadKubeconfig(configPath)
	if err != nil {
		t.Fatalf("LoadKubeconfig() error = %v", err)
	}

	if len(config.AuthInfos) != 0 {
		t.Error("New config should have no users")
	}

	// Step 2: Update token with autoCreate
	err = UpdateTokenByName(config, "c-test123", "test-cluster", "token-123", "https://rancher.example.com", true, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Verify structure was created
	if len(config.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(config.Clusters))
	}
	if len(config.AuthInfos) != 1 {
		t.Errorf("Expected 1 user, got %d", len(config.AuthInfos))
	}

	// Step 3: Save config
	err = SaveKubeconfig(config, configPath, nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Step 4: Reload and verify
	reloaded, err := LoadKubeconfig(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if len(reloaded.AuthInfos) != 1 {
		t.Errorf("Expected 1 user after reload, got %d", len(reloaded.AuthInfos))
	}
	if reloaded.AuthInfos["test-cluster"].Token != "token-123" {
		t.Errorf("Expected token token-123, got %s", reloaded.AuthInfos["test-cluster"].Token)
	}

	// Step 5: Update token again
	err = UpdateTokenByName(reloaded, "c-test123", "test-cluster", "token-456", "https://rancher.example.com", false, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error on second update: %v", err)
	}

	// Step 6: Save again (should create backup)
	err = SaveKubeconfig(reloaded, configPath, nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error on second save: %v", err)
	}

	// Verify backup was created
	kubedir := filepath.Dir(configPath)
	entries, err := os.ReadDir(kubedir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	backupFound := false
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "backup") {
			backupFound = true
			break
		}
	}
	if !backupFound {
		t.Error("Backup should be created on second save")
	}

	// Step 7: Verify final state
	final, err := LoadKubeconfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load final config: %v", err)
	}
	if final.AuthInfos["test-cluster"].Token != "token-456" {
		t.Errorf("Expected final token token-456, got %s", final.AuthInfos["test-cluster"].Token)
	}
}

// TestIntegration_FirstTimeUse tests first-time use scenario
func TestIntegration_FirstTimeUse(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "new", "config")
	logger := createTestLogger()

	// Load non-existent file
	config, err := LoadKubeconfig(configPath)
	if err != nil {
		t.Fatalf("LoadKubeconfig() should not error for non-existent file: %v", err)
	}

	// Add first cluster
	err = UpdateTokenByName(config, "c-first", "first-cluster", "token-1", "https://rancher.example.com", true, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Add second cluster
	err = UpdateTokenByName(config, "c-second", "second-cluster", "token-2", "https://rancher.example.com", true, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Verify both clusters exist
	if len(config.Clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(config.Clusters))
	}
	if len(config.AuthInfos) != 2 {
		t.Errorf("Expected 2 users, got %d", len(config.AuthInfos))
	}

	// Save
	err = SaveKubeconfig(config, configPath, nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify file structure is correct
	reloaded, err := LoadKubeconfig(configPath)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	if len(reloaded.Clusters) != 2 {
		t.Errorf("Expected 2 clusters after reload, got %d", len(reloaded.Clusters))
	}
}

// TestIntegration_MultipleUpdates tests multiple sequential updates
func TestIntegration_MultipleUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	logger := createTestLogger()

	config := createTestKubeconfig()

	// Perform multiple updates
	updates := []struct {
		token string
	}{
		{"token-v1"},
		{"token-v2"},
		{"token-v3"},
	}

	for i, update := range updates {
		err := UpdateTokenByName(config, "c-test123", "test-cluster", update.token, "https://rancher.example.com", false, logger)
		if err != nil {
			t.Fatalf("Update %d failed: %v", i, err)
		}

		// Verify token was updated
		if config.AuthInfos["test-cluster"].Token != update.token {
			t.Errorf("Update %d: expected token %s, got %s", i, update.token, config.AuthInfos["test-cluster"].Token)
		}
	}

	// Save and verify final state
	if err := SaveKubeconfig(config, configPath, nil); err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	final, _ := LoadKubeconfig(configPath)
	if final.AuthInfos["test-cluster"].Token != "token-v3" {
		t.Errorf("Expected final token token-v3, got %s", final.AuthInfos["test-cluster"].Token)
	}
}

// TestSaveKubeconfig_WithLogger tests that backup file path is logged when logger is provided
func TestSaveKubeconfig_WithLogger(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")

	// Create initial file
	initialConfig := createTestKubeconfig()
	if err := SaveKubeconfig(initialConfig, testFile, nil); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Save again with a logger to trigger backup
	updatedConfig := createTestKubeconfig()
	updatedConfig.AuthInfos["test-cluster"].Token = "updated-token"

	// Create a logger to verify the backup path is logged
	logger := createTestLogger()
	if err := SaveKubeconfig(updatedConfig, testFile, logger); err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify backup file was created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	backupFound := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "config.backup.") {
			backupFound = true
			break
		}
	}

	if !backupFound {
		t.Error("Backup file was not created")
	}
}

// ============================================================================
// KUBECONFIG Environment Variable Tests
// ============================================================================

// TestLoadKubeconfig_WithKUBECONFIG_SingleFile tests loading with KUBECONFIG env var pointing to a single file
func TestLoadKubeconfig_WithKUBECONFIG_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigFile := filepath.Join(tmpDir, "my-config")

	// Create a test kubeconfig file
	content := createTestKubeconfigContent()
	if err := os.WriteFile(kubeconfigFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Set KUBECONFIG environment variable
	t.Setenv("KUBECONFIG", kubeconfigFile)

	// Load with empty path (should use KUBECONFIG)
	config, err := LoadKubeconfig("")
	if err != nil {
		t.Fatalf("LoadKubeconfig() error = %v", err)
	}

	// Verify structure
	if len(config.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(config.Clusters))
	}
	if config.AuthInfos["test-cluster"] == nil {
		t.Error("Expected user test-cluster to exist")
	}
}

// TestLoadKubeconfig_WithKUBECONFIG_MultipleFiles tests loading with KUBECONFIG pointing to multiple files
func TestLoadKubeconfig_WithKUBECONFIG_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config1")
	file2 := filepath.Join(tmpDir, "config2")

	// Create first config file
	if err := os.WriteFile(file1, []byte(createTestKubeconfigContent()), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create second config file (doesn't exist yet)
	// According to kubectl behavior, when multiple files are specified:
	// - For reading: merge all files
	// - For writing: use first existing file, or last file if none exist

	// Set KUBECONFIG with multiple files (OS-specific path separator)
	separator := string(os.PathListSeparator)
	t.Setenv("KUBECONFIG", file1+separator+file2)

	// Load with empty path (should use first file from KUBECONFIG)
	config, err := LoadKubeconfig("")
	if err != nil {
		t.Fatalf("LoadKubeconfig() error = %v", err)
	}

	// Verify structure loaded from first file
	if len(config.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(config.Clusters))
	}
}

// TestSaveKubeconfig_WithKUBECONFIG_SingleFile tests saving with KUBECONFIG env var
func TestSaveKubeconfig_WithKUBECONFIG_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigFile := filepath.Join(tmpDir, "my-config")

	// Set KUBECONFIG environment variable
	t.Setenv("KUBECONFIG", kubeconfigFile)

	config := createTestKubeconfig()

	// Save with empty path (should use KUBECONFIG)
	err := SaveKubeconfig(config, "", nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify file was created at KUBECONFIG location
	if _, err := os.Stat(kubeconfigFile); os.IsNotExist(err) {
		t.Error("SaveKubeconfig() did not create file at KUBECONFIG location")
	}

	// Verify content
	loaded, err := LoadKubeconfig("")
	if err != nil {
		t.Fatalf("Failed to load saved file: %v", err)
	}
	if loaded.AuthInfos["test-cluster"].Token != "test-token-123" {
		t.Error("Saved content doesn't match original")
	}
}

// TestSaveKubeconfig_WithKUBECONFIG_MultipleFiles_FirstExists tests save behavior with multiple files
func TestSaveKubeconfig_WithKUBECONFIG_MultipleFiles_FirstExists(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config1")
	file2 := filepath.Join(tmpDir, "config2")

	// Create first file (existing)
	if err := os.WriteFile(file1, []byte(createTestKubeconfigContent()), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Set KUBECONFIG with multiple files
	separator := string(os.PathListSeparator)
	t.Setenv("KUBECONFIG", file1+separator+file2)

	config := createTestKubeconfig()
	config.AuthInfos["test-cluster"].Token = "new-token"

	// Save with empty path (should use first existing file)
	err := SaveKubeconfig(config, "", nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify first file was updated
	loaded, err := LoadKubeconfig(file1)
	if err != nil {
		t.Fatalf("Failed to load file1: %v", err)
	}
	if loaded.AuthInfos["test-cluster"].Token != "new-token" {
		t.Error("First file should be updated")
	}

	// Verify second file was not created
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("Second file should not be created when first file exists")
	}
}

// TestSaveKubeconfig_WithKUBECONFIG_MultipleFiles_NoneExist tests save when no files exist
func TestSaveKubeconfig_WithKUBECONFIG_MultipleFiles_NoneExist(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config1")
	file2 := filepath.Join(tmpDir, "config2")

	// Neither file exists yet

	// Set KUBECONFIG with multiple files
	separator := string(os.PathListSeparator)
	t.Setenv("KUBECONFIG", file1+separator+file2)

	config := createTestKubeconfig()

	// Save with empty path
	err := SaveKubeconfig(config, "", nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Note: ClientConfigLoadingRules.GetDefaultFilename() returns the first file when none exist.
	// This differs from kubectl's PathOptions.GetDefaultFilename() which returns the last file.
	// We use ClientConfigLoadingRules because:
	// 1. It's the core client-go API that handles all file loading/precedence logic
	// 2. It ensures consistency with other client-go based tools
	// 3. For the primary use case (single file in KUBECONFIG), both behave identically
	// 4. When multiple files exist, the behavior is the same (uses first existing file)
	// The difference only affects the edge case of multiple non-existent files in KUBECONFIG.
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Error("First file should be created when no files exist (client-go ClientConfigLoadingRules behavior)")
	}

	// Verify second file was not created (differs from kubectl PathOptions)
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("Second file should not be created (client-go uses first file, not last)")
	}
}

// TestLoadKubeconfig_ExplicitPathOverridesKUBECONFIG tests that explicit path takes precedence
func TestLoadKubeconfig_ExplicitPathOverridesKUBECONFIG(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigFile := filepath.Join(tmpDir, "env-config")
	explicitFile := filepath.Join(tmpDir, "explicit-config")

	// Create both files with different content
	content1 := createTestKubeconfigContent()
	if err := os.WriteFile(kubeconfigFile, []byte(content1), 0600); err != nil {
		t.Fatalf("Failed to write env config: %v", err)
	}

	// Create explicit file with different token
	config2 := createTestKubeconfig()
	config2.AuthInfos["test-cluster"].Token = "explicit-token"
	if err := SaveKubeconfig(config2, explicitFile, nil); err != nil {
		t.Fatalf("Failed to create explicit config: %v", err)
	}

	// Set KUBECONFIG environment variable
	t.Setenv("KUBECONFIG", kubeconfigFile)

	// Load with explicit path (should ignore KUBECONFIG)
	config, err := LoadKubeconfig(explicitFile)
	if err != nil {
		t.Fatalf("LoadKubeconfig() error = %v", err)
	}

	// Verify we loaded from explicit file, not KUBECONFIG
	if config.AuthInfos["test-cluster"].Token != "explicit-token" {
		t.Errorf("Expected explicit-token, got %s", config.AuthInfos["test-cluster"].Token)
	}
}

// TestSaveKubeconfig_ExplicitPathOverridesKUBECONFIG tests that explicit path takes precedence for saving
func TestSaveKubeconfig_ExplicitPathOverridesKUBECONFIG(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigFile := filepath.Join(tmpDir, "env-config")
	explicitFile := filepath.Join(tmpDir, "explicit-config")

	// Set KUBECONFIG environment variable
	t.Setenv("KUBECONFIG", kubeconfigFile)

	config := createTestKubeconfig()

	// Save with explicit path (should ignore KUBECONFIG)
	err := SaveKubeconfig(config, explicitFile, nil)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify file was created at explicit location
	if _, err := os.Stat(explicitFile); os.IsNotExist(err) {
		t.Error("SaveKubeconfig() should create file at explicit path")
	}

	// Verify KUBECONFIG file was not created
	if _, err := os.Stat(kubeconfigFile); !os.IsNotExist(err) {
		t.Error("SaveKubeconfig() should not create file at KUBECONFIG location when explicit path provided")
	}
}

// TestLoadKubeconfig_NoKUBECONFIG_UsesDefault tests default behavior when KUBECONFIG is not set
func TestLoadKubeconfig_NoKUBECONFIG_UsesDefault(t *testing.T) {
	// Unset KUBECONFIG
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Failed to unset KUBECONFIG: %v", err)
	}
	t.Cleanup(func() {
		// Restore original value after test
		if original := os.Getenv("KUBECONFIG"); original != "" {
			_ = os.Setenv("KUBECONFIG", original)
		}
	})

	// Load with empty path (should use default ~/.kube/config)
	config, err := LoadKubeconfig("")
	if err != nil {
		t.Fatalf("LoadKubeconfig() error = %v", err)
	}

	// Should return empty config if default doesn't exist, or loaded config if it does
	if config == nil {
		t.Error("LoadKubeconfig() should return a config structure")
	}
}

// TestLoadKubeconfig_EmptyKUBECONFIG_UsesDefault tests default behavior when KUBECONFIG is set to empty string
func TestLoadKubeconfig_EmptyKUBECONFIG_UsesDefault(t *testing.T) {
	// Set KUBECONFIG to empty string
	t.Setenv("KUBECONFIG", "")

	// Load with empty path (should use default ~/.kube/config)
	config, err := LoadKubeconfig("")
	if err != nil {
		t.Fatalf("LoadKubeconfig() error = %v", err)
	}

	// Should return empty config if default doesn't exist, or loaded config if it does
	if config == nil {
		t.Error("LoadKubeconfig() should return a config structure")
	}
}

// ============================================================================
// MergeKubeconfig Tests
// ============================================================================

// createTestSourceKubeconfig creates a source kubeconfig with direct contexts
func createTestSourceKubeconfig() *api.Config {
	config := api.NewConfig()

	// Primary cluster
	config.Clusters["demo-cluster"] = &api.Cluster{
		Server: "https://rancher.example.com/k8s/clusters/c-m-demo",
	}

	// Direct clusters
	config.Clusters["demo-cluster-node01"] = &api.Cluster{
		Server:                   "https://192.168.1.101:6443",
		CertificateAuthorityData: []byte("test-ca-data"),
	}
	config.Clusters["demo-cluster-node02"] = &api.Cluster{
		Server:                   "https://192.168.1.102:6443",
		CertificateAuthorityData: []byte("test-ca-data"),
	}

	// Primary context
	config.Contexts["demo-cluster"] = &api.Context{
		Cluster:  "demo-cluster",
		AuthInfo: "demo-cluster",
	}

	// Direct contexts
	config.Contexts["demo-cluster-node01"] = &api.Context{
		Cluster:  "demo-cluster-node01",
		AuthInfo: "demo-cluster",
	}
	config.Contexts["demo-cluster-node02"] = &api.Context{
		Cluster:  "demo-cluster-node02",
		AuthInfo: "demo-cluster",
	}

	// User (shared by all contexts)
	config.AuthInfos["demo-cluster"] = &api.AuthInfo{
		Token: "kubeconfig-user:demo-token",
	}

	config.CurrentContext = "demo-cluster"

	return config
}

// TestMergeKubeconfig_WithDirectlyEnabled tests merging all contexts
func TestMergeKubeconfig_WithDirectlyEnabled(t *testing.T) {
	target := api.NewConfig()
	source := createTestSourceKubeconfig()

	MergeKubeconfig(target, source, "demo-cluster", true)

	// Verify all clusters were merged
	if len(target.Clusters) != 3 {
		t.Errorf("Expected 3 clusters, got %d", len(target.Clusters))
	}
	if target.Clusters["demo-cluster"] == nil {
		t.Error("Primary cluster should be merged")
	}
	if target.Clusters["demo-cluster-node01"] == nil {
		t.Error("Direct cluster node01 should be merged")
	}
	if target.Clusters["demo-cluster-node02"] == nil {
		t.Error("Direct cluster node02 should be merged")
	}

	// Verify all contexts were merged
	if len(target.Contexts) != 3 {
		t.Errorf("Expected 3 contexts, got %d", len(target.Contexts))
	}

	// Verify authInfo was merged
	if len(target.AuthInfos) != 1 {
		t.Errorf("Expected 1 authInfo (shared), got %d", len(target.AuthInfos))
	}
	if target.AuthInfos["demo-cluster"] == nil {
		t.Error("AuthInfo should be merged")
	}
	if target.AuthInfos["demo-cluster"].Token != "kubeconfig-user:demo-token" {
		t.Errorf("Expected token kubeconfig-user:demo-token, got %s", target.AuthInfos["demo-cluster"].Token)
	}
}

// TestMergeKubeconfig_WithDirectlyDisabled tests merging only primary context
func TestMergeKubeconfig_WithDirectlyDisabled(t *testing.T) {
	target := api.NewConfig()
	source := createTestSourceKubeconfig()

	MergeKubeconfig(target, source, "demo-cluster", false)

	// Verify only primary cluster was merged
	if len(target.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(target.Clusters))
	}
	if target.Clusters["demo-cluster"] == nil {
		t.Error("Primary cluster should be merged")
	}
	if target.Clusters["demo-cluster-node01"] != nil {
		t.Error("Direct cluster node01 should NOT be merged")
	}
	if target.Clusters["demo-cluster-node02"] != nil {
		t.Error("Direct cluster node02 should NOT be merged")
	}

	// Verify only primary context was merged
	if len(target.Contexts) != 1 {
		t.Errorf("Expected 1 context, got %d", len(target.Contexts))
	}
	if target.Contexts["demo-cluster"] == nil {
		t.Error("Primary context should be merged")
	}

	// Verify authInfo was merged
	if len(target.AuthInfos) != 1 {
		t.Errorf("Expected 1 authInfo, got %d", len(target.AuthInfos))
	}
}

// TestMergeKubeconfig_OverwriteExisting tests overwrite behavior
func TestMergeKubeconfig_OverwriteExisting(t *testing.T) {
	// Create target with existing entries
	target := api.NewConfig()
	target.Clusters["demo-cluster"] = &api.Cluster{
		Server: "https://old-server.example.com",
	}
	target.Contexts["demo-cluster"] = &api.Context{
		Cluster:  "demo-cluster",
		AuthInfo: "demo-cluster",
	}
	target.AuthInfos["demo-cluster"] = &api.AuthInfo{
		Token: "old-token",
	}

	// Create source with new values
	source := createTestSourceKubeconfig()

	MergeKubeconfig(target, source, "demo-cluster", false)

	// Verify values were overwritten
	if target.Clusters["demo-cluster"].Server != "https://rancher.example.com/k8s/clusters/c-m-demo" {
		t.Errorf("Cluster server should be overwritten, got %s", target.Clusters["demo-cluster"].Server)
	}
	if target.AuthInfos["demo-cluster"].Token != "kubeconfig-user:demo-token" {
		t.Errorf("Token should be overwritten, got %s", target.AuthInfos["demo-cluster"].Token)
	}
}

// TestMergeKubeconfig_PreservesOtherEntries tests that other entries are preserved
func TestMergeKubeconfig_PreservesOtherEntries(t *testing.T) {
	// Create target with existing entries from different cluster
	target := api.NewConfig()
	target.Clusters["other-cluster"] = &api.Cluster{
		Server: "https://other-server.example.com",
	}
	target.Contexts["other-cluster"] = &api.Context{
		Cluster:  "other-cluster",
		AuthInfo: "other-cluster",
	}
	target.AuthInfos["other-cluster"] = &api.AuthInfo{
		Token: "other-token",
	}

	source := createTestSourceKubeconfig()

	MergeKubeconfig(target, source, "demo-cluster", true)

	// Verify other entries are preserved
	if target.Clusters["other-cluster"] == nil {
		t.Error("Other cluster should be preserved")
	}
	if target.Clusters["other-cluster"].Server != "https://other-server.example.com" {
		t.Error("Other cluster server should not change")
	}
	if target.AuthInfos["other-cluster"] == nil {
		t.Error("Other authInfo should be preserved")
	}
	if target.AuthInfos["other-cluster"].Token != "other-token" {
		t.Error("Other token should not change")
	}

	// Verify new entries are added
	if target.Clusters["demo-cluster"] == nil {
		t.Error("Demo cluster should be added")
	}

	// Total should be 4 clusters (1 other + 3 from source)
	if len(target.Clusters) != 4 {
		t.Errorf("Expected 4 clusters, got %d", len(target.Clusters))
	}
}

// TestMergeKubeconfig_NilMaps tests handling of nil maps in target
func TestMergeKubeconfig_NilMaps(t *testing.T) {
	// Create target with nil maps (as returned by api.NewConfig() doesn't initialize maps as nil)
	target := &api.Config{
		Clusters:  nil,
		Contexts:  nil,
		AuthInfos: nil,
	}

	source := createTestSourceKubeconfig()

	// Should not panic
	MergeKubeconfig(target, source, "demo-cluster", true)

	// Verify maps were initialized and entries added
	if target.Clusters == nil {
		t.Error("Clusters map should be initialized")
	}
	if target.Contexts == nil {
		t.Error("Contexts map should be initialized")
	}
	if target.AuthInfos == nil {
		t.Error("AuthInfos map should be initialized")
	}

	if len(target.Clusters) != 3 {
		t.Errorf("Expected 3 clusters, got %d", len(target.Clusters))
	}
}

// TestMergeKubeconfig_EmptySource tests handling of empty source
func TestMergeKubeconfig_EmptySource(t *testing.T) {
	target := createTestKubeconfig()
	source := api.NewConfig()

	originalClusters := len(target.Clusters)

	MergeKubeconfig(target, source, "nonexistent", true)

	// Target should be unchanged
	if len(target.Clusters) != originalClusters {
		t.Errorf("Target clusters should be unchanged, expected %d, got %d", originalClusters, len(target.Clusters))
	}
}

// TestMergeKubeconfig_DirectContextPatternMatching tests correct pattern matching for direct contexts
func TestMergeKubeconfig_DirectContextPatternMatching(t *testing.T) {
	target := api.NewConfig()

	// Create source with various context names
	source := api.NewConfig()
	source.Clusters["prod"] = &api.Cluster{Server: "https://prod.example.com"}
	source.Clusters["prod-node1"] = &api.Cluster{Server: "https://prod-node1.example.com"}
	source.Clusters["production"] = &api.Cluster{Server: "https://production.example.com"} // Should NOT match
	source.Clusters["prod-"] = &api.Cluster{Server: "https://prod-.example.com"}           // Edge case

	source.Contexts["prod"] = &api.Context{Cluster: "prod", AuthInfo: "prod"}
	source.Contexts["prod-node1"] = &api.Context{Cluster: "prod-node1", AuthInfo: "prod"}
	source.Contexts["production"] = &api.Context{Cluster: "production", AuthInfo: "production"} // Different user
	source.Contexts["prod-"] = &api.Context{Cluster: "prod-", AuthInfo: "prod"}

	source.AuthInfos["prod"] = &api.AuthInfo{Token: "prod-token"}
	source.AuthInfos["production"] = &api.AuthInfo{Token: "production-token"}

	MergeKubeconfig(target, source, "prod", true)

	// Should match: prod, prod-node1, prod-
	// Should NOT match: production (doesn't start with "prod-")
	if target.Contexts["prod"] == nil {
		t.Error("Primary context 'prod' should be merged")
	}
	if target.Contexts["prod-node1"] == nil {
		t.Error("Direct context 'prod-node1' should be merged")
	}
	if target.Contexts["prod-"] == nil {
		t.Error("Direct context 'prod-' should be merged")
	}
	if target.Contexts["production"] != nil {
		t.Error("Context 'production' should NOT be merged (doesn't match pattern)")
	}

	// Verify correct count
	if len(target.Contexts) != 3 {
		t.Errorf("Expected 3 contexts, got %d", len(target.Contexts))
	}
}

// TestExtractTokenFromKubeconfig tests the ExtractTokenFromKubeconfig function
func TestExtractTokenFromKubeconfig(t *testing.T) {
	tests := []struct {
		name          string
		kubeconfig    *api.Config
		expectedToken string
		expectedOK    bool
	}{
		{
			name:          "nil kubeconfig",
			kubeconfig:    nil,
			expectedToken: "",
			expectedOK:    false,
		},
		{
			name: "empty CurrentContext",
			kubeconfig: &api.Config{
				CurrentContext: "",
				Contexts:       map[string]*api.Context{"test": {AuthInfo: "test"}},
				AuthInfos:      map[string]*api.AuthInfo{"test": {Token: "test-token"}},
			},
			expectedToken: "",
			expectedOK:    false,
		},
		{
			name: "CurrentContext not found in Contexts",
			kubeconfig: &api.Config{
				CurrentContext: "missing",
				Contexts:       map[string]*api.Context{"test": {AuthInfo: "test"}},
				AuthInfos:      map[string]*api.AuthInfo{"test": {Token: "test-token"}},
			},
			expectedToken: "",
			expectedOK:    false,
		},
		{
			name: "nil Context entry",
			kubeconfig: &api.Config{
				CurrentContext: "test",
				Contexts:       map[string]*api.Context{"test": nil},
				AuthInfos:      map[string]*api.AuthInfo{"test": {Token: "test-token"}},
			},
			expectedToken: "",
			expectedOK:    false,
		},
		{
			name: "empty AuthInfo in Context",
			kubeconfig: &api.Config{
				CurrentContext: "test",
				Contexts:       map[string]*api.Context{"test": {AuthInfo: ""}},
				AuthInfos:      map[string]*api.AuthInfo{"test": {Token: "test-token"}},
			},
			expectedToken: "",
			expectedOK:    false,
		},
		{
			name: "AuthInfo not found",
			kubeconfig: &api.Config{
				CurrentContext: "test",
				Contexts:       map[string]*api.Context{"test": {AuthInfo: "missing"}},
				AuthInfos:      map[string]*api.AuthInfo{"test": {Token: "test-token"}},
			},
			expectedToken: "",
			expectedOK:    false,
		},
		{
			name: "nil AuthInfo entry",
			kubeconfig: &api.Config{
				CurrentContext: "test",
				Contexts:       map[string]*api.Context{"test": {AuthInfo: "test"}},
				AuthInfos:      map[string]*api.AuthInfo{"test": nil},
			},
			expectedToken: "",
			expectedOK:    false,
		},
		{
			name: "empty token",
			kubeconfig: &api.Config{
				CurrentContext: "test",
				Contexts:       map[string]*api.Context{"test": {AuthInfo: "test"}},
				AuthInfos:      map[string]*api.AuthInfo{"test": {Token: ""}},
			},
			expectedToken: "",
			expectedOK:    false,
		},
		{
			name: "successful extraction",
			kubeconfig: &api.Config{
				CurrentContext: "production",
				Contexts:       map[string]*api.Context{"production": {AuthInfo: "production-user"}},
				AuthInfos:      map[string]*api.AuthInfo{"production-user": {Token: "kubeconfig-user:abc123"}},
			},
			expectedToken: "kubeconfig-user:abc123",
			expectedOK:    true,
		},
		{
			name: "multiple contexts - extracts from CurrentContext",
			kubeconfig: &api.Config{
				CurrentContext: "staging",
				Contexts: map[string]*api.Context{
					"production": {AuthInfo: "prod-user"},
					"staging":    {AuthInfo: "staging-user"},
					"dev":        {AuthInfo: "dev-user"},
				},
				AuthInfos: map[string]*api.AuthInfo{
					"prod-user":    {Token: "prod-token"},
					"staging-user": {Token: "staging-token"},
					"dev-user":     {Token: "dev-token"},
				},
			},
			expectedToken: "staging-token",
			expectedOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := ExtractTokenFromKubeconfig(tt.kubeconfig)
			if token != tt.expectedToken {
				t.Errorf("ExtractTokenFromKubeconfig() token = %v, want %v", token, tt.expectedToken)
			}
			if ok != tt.expectedOK {
				t.Errorf("ExtractTokenFromKubeconfig() ok = %v, want %v", ok, tt.expectedOK)
			}
		})
	}
}

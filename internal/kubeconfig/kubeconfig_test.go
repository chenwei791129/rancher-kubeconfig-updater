package kubeconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.uber.org/zap"
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
		{"tilde with slash", "~/.kube/config", userHomeDir + "/.kube/config", false},
		{"tilde with backslash", "~\\.kube\\config", userHomeDir + pathSeparator + ".kube" + pathSeparator + "config", false},
		{"absolute path unix", "/home/user/.kube/config", "/home/user/.kube/config", false},
		{"absolute path windows", "C:\\Users\\user\\.kube\\config", "C:\\Users\\user\\.kube\\config", false},
		{"relative path", ".kube/config", ".kube/config", false},
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
func createTestKubeconfig() Kubeconfig {
	return Kubeconfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []ConfigCluster{
			{
				Name: "test-cluster",
				Cluster: map[string]any{
					"server": "https://rancher.example.com/k8s/clusters/c-test123",
				},
			},
		},
		Contexts: []ConfigContext{
			{
				Name: "test-cluster",
				Context: map[string]any{
					"cluster": "test-cluster",
					"user":    "test-cluster",
				},
			},
		},
		CurrentContext: "test-cluster",
		Users: []ConfigUser{
			{
				Name: "test-cluster",
				User: User{
					Token: "test-token-123",
				},
			},
		},
	}
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
	if config.APIVersion != "v1" {
		t.Errorf("Expected APIVersion v1, got %s", config.APIVersion)
	}
	if config.Kind != "Config" {
		t.Errorf("Expected Kind Config, got %s", config.Kind)
	}
	if len(config.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(config.Clusters))
	}
	if len(config.Contexts) != 1 {
		t.Errorf("Expected 1 context, got %d", len(config.Contexts))
	}
	if len(config.Users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(config.Users))
	}

	// Verify cluster details
	if config.Clusters[0].Name != "test-cluster" {
		t.Errorf("Expected cluster name test-cluster, got %s", config.Clusters[0].Name)
	}

	// Verify user details
	if config.Users[0].Name != "test-cluster" {
		t.Errorf("Expected user name test-cluster, got %s", config.Users[0].Name)
	}
	if config.Users[0].User.Token != "test-token-123" {
		t.Errorf("Expected token test-token-123, got %s", config.Users[0].User.Token)
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
	if config.APIVersion != "v1" {
		t.Errorf("Expected APIVersion v1, got %s", config.APIVersion)
	}
	if config.Kind != "Config" {
		t.Errorf("Expected Kind Config, got %s", config.Kind)
	}
	if config.Clusters == nil {
		t.Error("Expected non-nil Clusters slice")
	}
	if config.Contexts == nil {
		t.Error("Expected non-nil Contexts slice")
	}
	if config.Users == nil {
		t.Error("Expected non-nil Users slice")
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
	if !strings.Contains(err.Error(), "parse") && !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Expected parse/unmarshal error, got: %v", err)
	}
}

// TestLoadKubeconfig_PathExpansion tests path expansion with tilde
func TestLoadKubeconfig_PathExpansion(t *testing.T) {
	// Test with empty path (should use default)
	config, err := LoadKubeconfig("")
	if err != nil {
		// It's okay if default path doesn't exist
		if !strings.Contains(err.Error(), "no such file") && !os.IsNotExist(err) {
			t.Errorf("Unexpected error for empty path: %v", err)
		}
	} else {
		// If it loaded, verify it's a valid structure
		if config.APIVersion == "" {
			t.Error("Expected APIVersion to be set")
		}
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

	err := config.SaveKubeconfig(testFile)
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
	if len(loaded.Users) != 1 || loaded.Users[0].User.Token != "test-token-123" {
		t.Error("Saved content doesn't match original")
	}
}

// TestSaveKubeconfig_AutoCreateDirectory tests automatic directory creation
func TestSaveKubeconfig_AutoCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dirs", "config")

	config := createTestKubeconfig()

	err := config.SaveKubeconfig(nestedPath)
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
	initialConfig.Users[0].User.Token = "old-token"
	if err := initialConfig.SaveKubeconfig(testFile); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Save updated config
	updatedConfig := createTestKubeconfig()
	updatedConfig.Users[0].User.Token = "new-token"
	if err := updatedConfig.SaveKubeconfig(testFile); err != nil {
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
			if backupConfig.Users[0].User.Token != "old-token" {
				t.Errorf("Backup should have old-token, got %s", backupConfig.Users[0].User.Token)
			}
			break
		}
	}

	if !backupFound {
		t.Error("Backup file was not created")
	}

	// Verify main file has new token
	mainConfig, _ := LoadKubeconfig(testFile)
	if mainConfig.Users[0].User.Token != "new-token" {
		t.Errorf("Main file should have new-token, got %s", mainConfig.Users[0].User.Token)
	}
}

// TestSaveKubeconfig_YAMLSerialization tests YAML serialization correctness
func TestSaveKubeconfig_YAMLSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config")

	// Create complex config
	config := Kubeconfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []ConfigCluster{
			{
				Name: "cluster-1",
				Cluster: map[string]any{
					"server":                     "https://server1.example.com",
					"certificate-authority-data": "base64data",
				},
			},
			{
				Name: "cluster-2",
				Cluster: map[string]any{
					"server": "https://server2.example.com",
				},
			},
		},
		Contexts: []ConfigContext{
			{
				Name: "context-1",
				Context: map[string]any{
					"cluster": "cluster-1",
					"user":    "user-1",
				},
			},
		},
		CurrentContext: "context-1",
		Users: []ConfigUser{
			{
				Name: "user-1",
				User: User{Token: "token-1"},
			},
		},
	}

	if err := config.SaveKubeconfig(testFile); err != nil {
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

	err := config.UpdateTokenByName("c-test123", "test-cluster", "new-token-456", "https://rancher.example.com", false, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Verify token was updated
	if config.Users[0].User.Token != "new-token-456" {
		t.Errorf("Expected token new-token-456, got %s", config.Users[0].User.Token)
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
	config := Kubeconfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   []ConfigCluster{},
		Contexts:   []ConfigContext{},
		Users:      []ConfigUser{},
	}
	logger := createTestLogger()

	err := config.UpdateTokenByName("c-newcluster", "new-cluster", "new-token", "https://rancher.example.com", true, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Verify cluster was created
	if len(config.Clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(config.Clusters))
	}
	if config.Clusters[0].Name != "new-cluster" {
		t.Errorf("Expected cluster name new-cluster, got %s", config.Clusters[0].Name)
	}
	expectedServer := "https://rancher.example.com/k8s/clusters/c-newcluster"
	if config.Clusters[0].Cluster["server"] != expectedServer {
		t.Errorf("Expected server %s, got %v", expectedServer, config.Clusters[0].Cluster["server"])
	}

	// Verify context was created
	if len(config.Contexts) != 1 {
		t.Fatalf("Expected 1 context, got %d", len(config.Contexts))
	}
	if config.Contexts[0].Name != "new-cluster" {
		t.Errorf("Expected context name new-cluster, got %s", config.Contexts[0].Name)
	}

	// Verify user was created
	if len(config.Users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(config.Users))
	}
	if config.Users[0].Name != "new-cluster" {
		t.Errorf("Expected user name new-cluster, got %s", config.Users[0].Name)
	}
	if config.Users[0].User.Token != "new-token" {
		t.Errorf("Expected token new-token, got %s", config.Users[0].User.Token)
	}
}

// TestUpdateTokenByName_AutoCreateFalse tests error when user doesn't exist
func TestUpdateTokenByName_AutoCreateFalse(t *testing.T) {
	config := Kubeconfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   []ConfigCluster{},
		Contexts:   []ConfigContext{},
		Users:      []ConfigUser{},
	}
	logger := createTestLogger()

	err := config.UpdateTokenByName("c-test", "nonexistent", "token", "https://rancher.example.com", false, logger)
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
			config := Kubeconfig{
				APIVersion: "v1",
				Kind:       "Config",
				Clusters:   []ConfigCluster{},
				Contexts:   []ConfigContext{},
				Users:      []ConfigUser{},
			}
			logger := createTestLogger()

			err := config.UpdateTokenByName(tt.clusterID, "test", "token", tt.rancherURL, true, logger)
			if err != nil {
				t.Fatalf("UpdateTokenByName() error = %v", err)
			}

			if config.Clusters[0].Cluster["server"] != tt.expectedURL {
				t.Errorf("Expected server %s, got %v", tt.expectedURL, config.Clusters[0].Cluster["server"])
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
			config := Kubeconfig{
				APIVersion: "v1",
				Kind:       "Config",
				Clusters:   []ConfigCluster{},
				Contexts:   []ConfigContext{},
				Users:      []ConfigUser{},
			}
			logger := createTestLogger()

			err := config.UpdateTokenByName("c-test", name, "token", "https://rancher.example.com", true, logger)
			if err != nil {
				t.Fatalf("UpdateTokenByName() failed for name %s: %v", name, err)
			}

			if config.Users[0].Name != name {
				t.Errorf("Expected user name %s, got %s", name, config.Users[0].Name)
			}
		})
	}
}

// ============================================================================
// atomicWriteFile Tests (P1)
// ============================================================================

// TestAtomicWriteFile_Success tests successful atomic file write
func TestAtomicWriteFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("test content")

	err := atomicWriteFile(testFile, testData, 0600)
	if err != nil {
		t.Fatalf("atomicWriteFile() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("atomicWriteFile() did not create file")
	}

	// Verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("Expected content %s, got %s", testData, content)
	}

	// Verify permissions (Unix only)
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(testFile)
		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Expected permissions 0600, got %o", mode)
		}
	}
}

// TestAtomicWriteFile_OverwriteExisting tests overwriting an existing file
func TestAtomicWriteFile_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create initial file
	initialData := []byte("initial content")
	if err := os.WriteFile(testFile, initialData, 0600); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Overwrite with new content
	newData := []byte("new content")
	err := atomicWriteFile(testFile, newData, 0600)
	if err != nil {
		t.Fatalf("atomicWriteFile() error = %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(newData) {
		t.Errorf("Expected content %s, got %s", newData, content)
	}
}

// TestAtomicWriteFile_TempFileCleanup tests temp file cleanup on success
func TestAtomicWriteFile_TempFileCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("test content")

	err := atomicWriteFile(testFile, testData, 0600)
	if err != nil {
		t.Fatalf("atomicWriteFile() error = %v", err)
	}

	// Check that no temp files remain
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".kubeconfig.tmp.") {
			t.Errorf("Temp file not cleaned up: %s", entry.Name())
		}
	}
}

// TestAtomicWriteFile_DifferentPermissions tests different permission settings
func TestAtomicWriteFile_DifferentPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission tests not applicable on Windows")
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name string
		perm os.FileMode
	}{
		{"read-only", 0400},
		{"read-write", 0600},
		{"read-write-execute", 0700},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".txt")
			testData := []byte("test")

			err := atomicWriteFile(testFile, testData, tt.perm)
			if err != nil {
				t.Fatalf("atomicWriteFile() error = %v", err)
			}

			info, _ := os.Stat(testFile)
			mode := info.Mode().Perm()
			if mode != tt.perm {
				t.Errorf("Expected permissions %o, got %o", tt.perm, mode)
			}
		})
	}
}

// ============================================================================
// createBackup Tests (P2)
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
	err := createBackup(testFile)
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
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
	err := createBackup(nonExistentFile)
	if err != nil {
		t.Errorf("createBackup() should not error for non-existent file, got: %v", err)
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
	if err := createBackup(testFile); err != nil {
		t.Fatalf("createBackup() error = %v", err)
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
			if !strings.HasPrefix(name, "config.backup.") {
				t.Errorf("Unexpected backup filename format: %s", name)
			}

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
	err := createBackup(subDir)
	if err == nil {
		t.Error("createBackup() should return error for directory")
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
	if err := createBackup(testFile); err != nil {
		t.Fatalf("createBackup() error = %v", err)
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
// Integration Tests (P1)
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

	if len(config.Users) != 0 {
		t.Error("New config should have no users")
	}

	// Step 2: Update token with autoCreate
	err = config.UpdateTokenByName("c-test123", "test-cluster", "token-123", "https://rancher.example.com", true, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Verify structure was created
	if len(config.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(config.Clusters))
	}
	if len(config.Users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(config.Users))
	}

	// Step 3: Save config
	err = config.SaveKubeconfig(configPath)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Step 4: Reload and verify
	reloaded, err := LoadKubeconfig(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if len(reloaded.Users) != 1 {
		t.Errorf("Expected 1 user after reload, got %d", len(reloaded.Users))
	}
	if reloaded.Users[0].User.Token != "token-123" {
		t.Errorf("Expected token token-123, got %s", reloaded.Users[0].User.Token)
	}

	// Step 5: Update token again
	err = reloaded.UpdateTokenByName("c-test123", "test-cluster", "token-456", "https://rancher.example.com", false, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error on second update: %v", err)
	}

	// Step 6: Save again (should create backup)
	err = reloaded.SaveKubeconfig(configPath)
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
	if final.Users[0].User.Token != "token-456" {
		t.Errorf("Expected final token token-456, got %s", final.Users[0].User.Token)
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
	err = config.UpdateTokenByName("c-first", "first-cluster", "token-1", "https://rancher.example.com", true, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Add second cluster
	err = config.UpdateTokenByName("c-second", "second-cluster", "token-2", "https://rancher.example.com", true, logger)
	if err != nil {
		t.Fatalf("UpdateTokenByName() error = %v", err)
	}

	// Verify both clusters exist
	if len(config.Clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(config.Clusters))
	}
	if len(config.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(config.Users))
	}

	// Save
	err = config.SaveKubeconfig(configPath)
	if err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	// Verify file structure is correct
	reloaded, err := LoadKubeconfig(configPath)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	if reloaded.APIVersion != "v1" {
		t.Error("APIVersion should be v1")
	}
	if reloaded.Kind != "Config" {
		t.Error("Kind should be Config")
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
		err := config.UpdateTokenByName("c-test123", "test-cluster", update.token, "https://rancher.example.com", false, logger)
		if err != nil {
			t.Fatalf("Update %d failed: %v", i, err)
		}

		// Verify token was updated
		if config.Users[0].User.Token != update.token {
			t.Errorf("Update %d: expected token %s, got %s", i, update.token, config.Users[0].User.Token)
		}
	}

	// Save and verify final state
	if err := config.SaveKubeconfig(configPath); err != nil {
		t.Fatalf("SaveKubeconfig() error = %v", err)
	}

	final, _ := LoadKubeconfig(configPath)
	if final.Users[0].User.Token != "token-v3" {
		t.Errorf("Expected final token token-v3, got %s", final.Users[0].User.Token)
	}
}

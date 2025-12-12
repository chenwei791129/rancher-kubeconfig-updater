package kubeconfig

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// TestExpandPath tests the expandPath function with various path formats
func TestExpandPath(t *testing.T) {
	userHomeDir, _ := os.UserHomeDir()
	pathSeparator := string(os.PathSeparator)
	defaultPath, _ := GetDefaultKubeconfigPath()

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

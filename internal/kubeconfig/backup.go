package kubeconfig

import (
	"fmt"
	"os"
	"time"
)

// createBackup creates a backup of the file at the given path.
// The backup filename includes a microsecond-precision timestamp to ensure uniqueness.
// If the file doesn't exist or backup fails, it logs a warning but doesn't stop the operation.
// Returns the backup file path and any error that occurred.
func createBackup(path string) (string, error) {
	// Check if file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", nil // New file, no backup needed
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory: %s", path)
	}

	// Read original file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %w", err)
	}

	// Backup filename: unique with microsecond timestamp
	backupPath := fmt.Sprintf("%s.backup.%s", path,
		time.Now().Format("20060102-150405.000000"))

	// Write backup with platform-appropriate permissions
	if err := os.WriteFile(backupPath, data, getSecureFileMode()); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

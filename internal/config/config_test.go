package config

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestGetBool_FlagSet tests when flag is explicitly set
func TestGetBool_FlagSet(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Set the flag
	err := cmd.Flags().Set("test-flag", "true")
	assert.NoError(t, err)

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.True(t, result)
}

// TestGetBool_EnvVarTrue tests when environment variable is "true"
func TestGetBool_EnvVarTrue(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Set environment variable
	os.Setenv("TEST_ENV", "true")
	defer os.Unsetenv("TEST_ENV")

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.True(t, result)
}

// TestGetBool_EnvVar1 tests when environment variable is "1"
func TestGetBool_EnvVar1(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Set environment variable to "1"
	os.Setenv("TEST_ENV", "1")
	defer os.Unsetenv("TEST_ENV")

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.True(t, result)
}

// TestGetBool_EnvVarFalse tests when environment variable is "false"
func TestGetBool_EnvVarFalse(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Set environment variable to "false"
	os.Setenv("TEST_ENV", "false")
	defer os.Unsetenv("TEST_ENV")

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.False(t, result)
}

// TestGetBool_EnvVarEmpty tests when environment variable is empty
func TestGetBool_EnvVarEmpty(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Ensure environment variable is not set
	os.Unsetenv("TEST_ENV")

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.False(t, result)
}

// TestGetBool_FlagOverridesEnv tests that flag takes precedence over environment variable
func TestGetBool_FlagOverridesEnv(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Set environment variable to true
	os.Setenv("TEST_ENV", "true")
	defer os.Unsetenv("TEST_ENV")

	// Set flag to false
	err := cmd.Flags().Set("test-flag", "false")
	assert.NoError(t, err)

	// Flag should override environment variable
	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.False(t, result)
}

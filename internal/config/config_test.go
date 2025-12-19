package config

import (
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
	t.Setenv("TEST_ENV", "true")

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.True(t, result)
}

// TestGetBool_EnvVar1 tests when environment variable is "1"
func TestGetBool_EnvVar1(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Set environment variable to "1"
	t.Setenv("TEST_ENV", "1")

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.True(t, result)
}

// TestGetBool_EnvVarFalse tests when environment variable is "false"
func TestGetBool_EnvVarFalse(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Set environment variable to "false"
	t.Setenv("TEST_ENV", "false")

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.False(t, result)
}

// TestGetBool_EnvVarEmpty tests when environment variable is empty
func TestGetBool_EnvVarEmpty(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Ensure environment variable is not set (empty string)
	t.Setenv("TEST_ENV", "")

	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.False(t, result)
}

// TestGetBool_FlagOverridesEnv tests that flag takes precedence over environment variable
func TestGetBool_FlagOverridesEnv(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "test flag")

	// Set environment variable to true
	t.Setenv("TEST_ENV", "true")

	// Set flag to false
	err := cmd.Flags().Set("test-flag", "false")
	assert.NoError(t, err)

	// Flag should override environment variable
	result := GetBool(cmd, "test-flag", "TEST_ENV")
	assert.False(t, result)
}

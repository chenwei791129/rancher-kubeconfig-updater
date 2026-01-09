package config

import (
	"strconv"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestGetBool_FlagSet tests when flag is explicitly set
func TestGetBool_FlagSet(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		expected  bool
	}{
		{
			name:      "FlagTrue",
			flagValue: "true",
			expected:  true,
		},
		{
			name:      "FlagFalse",
			flagValue: "false",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("test-flag", false, "test flag")

			// Empty environment variable
			t.Setenv("TEST_ENV", "")

			// Set the flag
			err := cmd.Flags().Set("test-flag", tt.flagValue)
			assert.NoError(t, err)

			result := GetBool(cmd, "test-flag", "TEST_ENV")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetBool_EnvVar tests environment variable handling with different values
func TestGetBool_EnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "EnvVarTrue",
			envValue: "true",
			expected: true,
		},
		{
			name:     "EnvVarTrue",
			envValue: "True",
			expected: true,
		},
		{
			name:     "EnvVarTrue",
			envValue: "TRUE",
			expected: true,
		},
		{
			name:     "EnvVar1",
			envValue: "1",
			expected: true,
		},
		{
			name:     "EnvVarFalse",
			envValue: "false",
			expected: false,
		},
		{
			name:     "EnvVarFalse",
			envValue: "False",
			expected: false,
		},
		{
			name:     "EnvVarFalse",
			envValue: "FALSE",
			expected: false,
		},
		{
			name:     "EnvVar0",
			envValue: "0",
			expected: false,
		},
		{
			name:     "EnvVarEmpty",
			envValue: "",
			expected: false,
		},
		{
			name:     "EnvVarEmpty",
			envValue: " ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("test-flag", false, "test flag")

			// Set environment variable
			t.Setenv("TEST_ENV", tt.envValue)

			result := GetBool(cmd, "test-flag", "TEST_ENV")
			t.Logf("EnvValue: '%s' => Result: %v", tt.envValue, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetBool_FlagOverridesEnv tests that flag takes precedence over environment variable
func TestGetBool_FlagOverridesEnv(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		flagValue string
		expected  bool
	}{
		{
			name:      "EnvTrue_FlagFalse",
			envValue:  "true",
			flagValue: "false",
			expected:  false,
		},
		{
			name:      "EnvFalse_FlagTrue",
			envValue:  "false",
			flagValue: "true",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("test-flag", false, "test flag")

			// Set environment variable
			t.Setenv("TEST_ENV", tt.envValue)

			// Set flag
			err := cmd.Flags().Set("test-flag", tt.flagValue)
			assert.NoError(t, err)

			// Flag should override environment variable
			result := GetBool(cmd, "test-flag", "TEST_ENV")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetInt_FlagSet tests when flag is explicitly set
func TestGetInt_FlagSet(t *testing.T) {
	tests := []struct {
		name      string
		flagValue int
		expected  int
	}{
		{
			name:      "FlagPositive",
			flagValue: 42,
			expected:  42,
		},
		{
			name:      "FlagZero",
			flagValue: 0,
			expected:  0,
		},
		{
			name:      "FlagNegative",
			flagValue: -5,
			expected:  -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Int("test-flag", 100, "test flag")

			// Empty environment variable
			t.Setenv("TEST_ENV", "")

			// Set the flag
			err := cmd.Flags().Set("test-flag", strconv.Itoa(tt.flagValue))
			assert.NoError(t, err)

			result := GetInt(cmd, "test-flag", "TEST_ENV")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetInt_EnvVar tests environment variable handling
func TestGetInt_EnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "EnvVarPositive",
			envValue: "30",
			expected: 30,
		},
		{
			name:     "EnvVarZero",
			envValue: "0",
			expected: 0,
		},
		{
			name:     "EnvVarNegative",
			envValue: "-10",
			expected: -10,
		},
		{
			name:     "EnvVarEmpty",
			envValue: "",
			expected: 100, // should return default
		},
		{
			name:     "EnvVarInvalid",
			envValue: "invalid",
			expected: 100, // should return default on parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Int("test-flag", 100, "test flag")

			// Set environment variable
			t.Setenv("TEST_ENV", tt.envValue)

			result := GetInt(cmd, "test-flag", "TEST_ENV")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetInt_FlagOverridesEnv tests that flag takes precedence over environment variable
func TestGetInt_FlagOverridesEnv(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("test-flag", 100, "test flag")

	// Set environment variable
	t.Setenv("TEST_ENV", "50")

	// Set flag
	err := cmd.Flags().Set("test-flag", "25")
	assert.NoError(t, err)

	// Flag should override environment variable
	result := GetInt(cmd, "test-flag", "TEST_ENV")
	assert.Equal(t, 25, result)
}


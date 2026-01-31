package cmd

import (
	"rancher-kubeconfig-updater/internal/rancher"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestFilterClusters_SingleClusterByName tests filtering by a single cluster name
func TestFilterClusters_SingleClusterByName(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
		{ID: "c-m-11111", Name: "development"},
	}

	filtered := filterClusters(clusters, "production", logger)

	assert.Len(t, filtered, 1)
	assert.Equal(t, "production", filtered[0].Name)
	assert.Equal(t, "c-m-12345", filtered[0].ID)
}

// TestFilterClusters_SingleClusterByID tests filtering by a single cluster ID
func TestFilterClusters_SingleClusterByID(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
		{ID: "c-m-11111", Name: "development"},
	}

	filtered := filterClusters(clusters, "c-m-67890", logger)

	assert.Len(t, filtered, 1)
	assert.Equal(t, "staging", filtered[0].Name)
	assert.Equal(t, "c-m-67890", filtered[0].ID)
}

// TestFilterClusters_MultipleClusters tests filtering by multiple comma-separated clusters
func TestFilterClusters_MultipleClusters(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
		{ID: "c-m-11111", Name: "development"},
		{ID: "c-m-22222", Name: "testing"},
	}

	filtered := filterClusters(clusters, "production,staging,development", logger)

	assert.Len(t, filtered, 3)
	assert.Equal(t, "production", filtered[0].Name)
	assert.Equal(t, "staging", filtered[1].Name)
	assert.Equal(t, "development", filtered[2].Name)
}

// TestFilterClusters_CaseInsensitive tests case-insensitive matching
func TestFilterClusters_CaseInsensitive(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "Production"},
		{ID: "c-m-67890", Name: "Staging"},
	}

	// Test various case combinations
	filtered1 := filterClusters(clusters, "PRODUCTION", logger)
	assert.Len(t, filtered1, 1)
	assert.Equal(t, "Production", filtered1[0].Name)

	filtered2 := filterClusters(clusters, "production", logger)
	assert.Len(t, filtered2, 1)
	assert.Equal(t, "Production", filtered2[0].Name)

	filtered3 := filterClusters(clusters, "ProDucTion", logger)
	assert.Len(t, filtered3, 1)
	assert.Equal(t, "Production", filtered3[0].Name)
}

// TestFilterClusters_WithWhitespace tests handling of whitespace in comma-separated list
func TestFilterClusters_WithWhitespace(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
		{ID: "c-m-11111", Name: "development"},
	}

	// Test various whitespace scenarios
	filtered := filterClusters(clusters, " production , staging , development ", logger)

	assert.Len(t, filtered, 3)
	assert.Equal(t, "production", filtered[0].Name)
	assert.Equal(t, "staging", filtered[1].Name)
	assert.Equal(t, "development", filtered[2].Name)
}

// TestFilterClusters_EmptyString tests handling of empty strings in the list
func TestFilterClusters_EmptyString(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
	}

	// Test with empty strings mixed in
	filtered := filterClusters(clusters, "production,,staging,", logger)

	assert.Len(t, filtered, 2)
	assert.Equal(t, "production", filtered[0].Name)
	assert.Equal(t, "staging", filtered[1].Name)
}

// TestFilterClusters_NoMatch tests when no clusters match the filter
func TestFilterClusters_NoMatch(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
	}

	filtered := filterClusters(clusters, "nonexistent", logger)

	assert.Len(t, filtered, 0)
}

// TestFilterClusters_MixedNameAndID tests filtering with both names and IDs
func TestFilterClusters_MixedNameAndID(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
		{ID: "c-m-11111", Name: "development"},
	}

	// Mix of name and ID
	filtered := filterClusters(clusters, "production,c-m-67890", logger)

	assert.Len(t, filtered, 2)
	assert.Equal(t, "production", filtered[0].Name)
	assert.Equal(t, "staging", filtered[1].Name)
}

// TestFilterClusters_AllWhitespace tests when filter is only whitespace
func TestFilterClusters_AllWhitespace(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
	}

	// Should return all clusters when no valid filter provided
	filtered := filterClusters(clusters, "   ,  ,  ", logger)

	assert.Len(t, filtered, 2)
}

// TestFilterClusters_PartialMatch tests that partial matches are not accepted
func TestFilterClusters_PartialMatch(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "production-east"},
	}

	// "prod" should not match "production" or "production-east"
	filtered := filterClusters(clusters, "prod", logger)

	assert.Len(t, filtered, 0)
}

// TestFilterClusters_DuplicateInFilter tests when the same cluster is specified multiple times
func TestFilterClusters_DuplicateInFilter(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
	}

	// Same cluster specified multiple times
	filtered := filterClusters(clusters, "production,production,PRODUCTION", logger)

	// Should only return the cluster once
	assert.Len(t, filtered, 1)
	assert.Equal(t, "production", filtered[0].Name)
}

// TestFilterClusters_BothNameAndIDMatch tests when both name and ID of a cluster are in the filter
func TestFilterClusters_BothNameAndIDMatch(t *testing.T) {
	logger := zap.NewNop()
	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
	}

	// Filter contains both the cluster name and ID
	filtered := filterClusters(clusters, "production,c-m-12345", logger)

	// Should only return the cluster once, not twice
	assert.Len(t, filtered, 1)
	assert.Equal(t, "production", filtered[0].Name)
	assert.Equal(t, "c-m-12345", filtered[0].ID)
}

// TestFilterClusters_BothNameAndIDMatch_NoFalseWarning verifies that when both the name
// and ID of a cluster are specified in the filter, no "not found" warning should be logged
// for either the name or the ID. This test exposes a defect in the current implementation
// where only the first matched filter is recorded, causing the second one to be reported
// as "not found".
func TestFilterClusters_BothNameAndIDMatch_NoFalseWarning(t *testing.T) {
	// Create a logger with observer to capture log output
	observedZapCore, observedLogs := observer.New(zap.WarnLevel)
	logger := zap.New(observedZapCore)

	clusters := rancher.Clusters{
		{ID: "c-m-12345", Name: "production"},
		{ID: "c-m-67890", Name: "staging"},
	}

	// Filter contains both the cluster name and ID
	filtered := filterClusters(clusters, "production,c-m-12345,staging", logger)

	// Should return 2 unique clusters
	assert.Len(t, filtered, 2)

	// Check that no "not found" warnings were logged
	// Since both "production" and "c-m-12345" refer to the same cluster,
	// neither should be reported as "not found"
	warnLogs := observedLogs.FilterMessage("Specified cluster not found in Rancher").All()

	// This assertion will FAIL with the current implementation, exposing the defect
	// Expected: 0 warnings (both "production" and "c-m-12345" should be recognized)
	// Actual: 1 warning (one of them will be reported as "not found")
	assert.Equal(t, 0, len(warnLogs), "Expected no 'not found' warnings when both name and ID are specified for the same cluster")

	// Additionally verify no warning contains "production" or "c-m-12345"
	for _, log := range warnLogs {
		for _, field := range log.Context {
			if field.Key == "cluster" {
				clusterValue := field.String
				assert.NotContains(t, []string{"production", "c-m-12345"}, clusterValue,
					"Cluster '%s' was incorrectly reported as not found", clusterValue)
			}
		}
	}
}

// TestConfigFlag_FlagRegistered tests that the --config/-c flag is properly registered
func TestConfigFlag_FlagRegistered(t *testing.T) {
	cmd := NewRootCmd()

	// Test that the flag exists
	configFlag := cmd.Flags().Lookup("config")
	assert.NotNil(t, configFlag, "config flag should be registered")

	// Test that the short flag exists
	assert.Equal(t, "c", configFlag.Shorthand, "config flag should have 'c' as shorthand")

	// Test default value
	assert.Equal(t, "", configFlag.DefValue, "config flag should have empty string as default")

	// Test usage text
	assert.Contains(t, configFlag.Usage, "kubeconfig", "config flag usage should mention kubeconfig")
}

// TestConfigFlag_AcceptsValue tests that the --config flag accepts a value
func TestConfigFlag_AcceptsValue(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "LongFlag",
			args:     []string{"--config", "/path/to/config"},
			expected: "/path/to/config",
		},
		{
			name:     "ShortFlag",
			args:     []string{"-c", "/custom/kubeconfig"},
			expected: "/custom/kubeconfig",
		},
		{
			name:     "TildePath",
			args:     []string{"--config", "~/my-kubeconfig"},
			expected: "~/my-kubeconfig",
		},
		{
			name:     "RelativePath",
			args:     []string{"-c", "./configs/dev"},
			expected: "./configs/dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			cmd.SetArgs(tt.args)

			// We can't actually run the command without a full Rancher setup,
			// but we can parse the flags to verify they're set correctly
			err := cmd.ParseFlags(tt.args)
			assert.NoError(t, err, "parsing flags should not error")

			configValue, err := cmd.Flags().GetString("config")
			assert.NoError(t, err, "getting config flag value should not error")
			assert.Equal(t, tt.expected, configValue, "config flag should have the expected value")
		})
	}
}

// TestConfigFlag_CombinedWithOtherFlags tests that --config works with other flags
func TestConfigFlag_CombinedWithOtherFlags(t *testing.T) {
	cmd := NewRootCmd()
	args := []string{
		"--config", "/tmp/test-kubeconfig",
		"--auto-create",
		"--auth-type", "ldap",
		"--cluster", "prod,staging",
	}

	err := cmd.ParseFlags(args)
	assert.NoError(t, err, "parsing combined flags should not error")

	// Verify config flag
	configValue, _ := cmd.Flags().GetString("config")
	assert.Equal(t, "/tmp/test-kubeconfig", configValue)

	// Verify other flags are still working
	autoCreateValue, _ := cmd.Flags().GetBool("auto-create")
	assert.True(t, autoCreateValue)

	authTypeValue, _ := cmd.Flags().GetString("auth-type")
	assert.Equal(t, "ldap", authTypeValue)

	clusterValue, _ := cmd.Flags().GetString("cluster")
	assert.Equal(t, "prod,staging", clusterValue)
}

// TestNewRootCmd_ConfigFlagInitialization tests that configPath variable is properly initialized
func TestNewRootCmd_ConfigFlagInitialization(t *testing.T) {
	// Reset configPath to ensure clean state
	configPath = ""

	cmd := NewRootCmd()
	args := []string{"--config", "/test/path"}

	err := cmd.ParseFlags(args)
	assert.NoError(t, err)

	// After parsing, the global configPath variable should be set
	assert.Equal(t, "/test/path", configPath)
}

// TestDryRunFlag_FlagRegistered tests that the --dry-run flag is properly registered
func TestDryRunFlag_FlagRegistered(t *testing.T) {
	cmd := NewRootCmd()

	// Test that the flag exists
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	assert.NotNil(t, dryRunFlag, "dry-run flag should be registered")

	// Test default value is false
	assert.Equal(t, "false", dryRunFlag.DefValue, "dry-run flag should default to false")

	// Test usage text
	assert.Contains(t, dryRunFlag.Usage, "Preview", "dry-run flag usage should mention Preview")
}

// TestDryRunFlag_AcceptsValue tests that the --dry-run flag accepts a boolean value
func TestDryRunFlag_AcceptsValue(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "DryRunEnabled",
			args:     []string{"--dry-run"},
			expected: true,
		},
		{
			name:     "DryRunExplicitTrue",
			args:     []string{"--dry-run=true"},
			expected: true,
		},
		{
			name:     "DryRunExplicitFalse",
			args:     []string{"--dry-run=false"},
			expected: false,
		},
		{
			name:     "DryRunNotSpecified",
			args:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			assert.NoError(t, err, "parsing flags should not error")

			dryRunValue, err := cmd.Flags().GetBool("dry-run")
			assert.NoError(t, err, "getting dry-run flag value should not error")
			assert.Equal(t, tt.expected, dryRunValue, "dry-run flag should have the expected value")
		})
	}
}

// TestDryRunFlag_CombinedWithOtherFlags tests that --dry-run works with other flags
func TestDryRunFlag_CombinedWithOtherFlags(t *testing.T) {
	cmd := NewRootCmd()
	args := []string{
		"--dry-run",
		"--config", "/tmp/test-kubeconfig",
		"--auto-create",
		"--cluster", "prod,staging",
		"--force-refresh",
	}

	err := cmd.ParseFlags(args)
	assert.NoError(t, err, "parsing combined flags should not error")

	// Verify dry-run flag
	dryRunValue, _ := cmd.Flags().GetBool("dry-run")
	assert.True(t, dryRunValue)

	// Verify other flags are still working
	configValue, _ := cmd.Flags().GetString("config")
	assert.Equal(t, "/tmp/test-kubeconfig", configValue)

	autoCreateValue, _ := cmd.Flags().GetBool("auto-create")
	assert.True(t, autoCreateValue)

	clusterValue, _ := cmd.Flags().GetString("cluster")
	assert.Equal(t, "prod,staging", clusterValue)

	forceRefreshValue, _ := cmd.Flags().GetBool("force-refresh")
	assert.True(t, forceRefreshValue)
}

// TestNewRootCmd_DryRunFlagInitialization tests that dryRun variable is properly initialized
func TestNewRootCmd_DryRunFlagInitialization(t *testing.T) {
	// Reset dryRun to ensure clean state
	dryRun = false

	cmd := NewRootCmd()
	args := []string{"--dry-run"}

	err := cmd.ParseFlags(args)
	assert.NoError(t, err)

	// After parsing, the global dryRun variable should be set
	assert.True(t, dryRun)
}

// TestWithDirectlyFlag_FlagRegistered tests that the --with-directly flag is properly registered
func TestWithDirectlyFlag_FlagRegistered(t *testing.T) {
	cmd := NewRootCmd()

	// Test that the flag exists
	withDirectlyFlag := cmd.Flags().Lookup("with-directly")
	assert.NotNil(t, withDirectlyFlag, "with-directly flag should be registered")

	// Test default value is false
	assert.Equal(t, "false", withDirectlyFlag.DefValue, "with-directly flag should default to false")

	// Test usage text
	assert.Contains(t, withDirectlyFlag.Usage, "Downstream Directly", "with-directly flag usage should mention Downstream Directly")
}

// TestWithDirectlyFlag_AcceptsValue tests that the --with-directly flag accepts a boolean value
func TestWithDirectlyFlag_AcceptsValue(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "WithDirectlyEnabled",
			args:     []string{"--with-directly"},
			expected: true,
		},
		{
			name:     "WithDirectlyExplicitTrue",
			args:     []string{"--with-directly=true"},
			expected: true,
		},
		{
			name:     "WithDirectlyExplicitFalse",
			args:     []string{"--with-directly=false"},
			expected: false,
		},
		{
			name:     "WithDirectlyNotSpecified",
			args:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			assert.NoError(t, err, "parsing flags should not error")

			withDirectlyValue, err := cmd.Flags().GetBool("with-directly")
			assert.NoError(t, err, "getting with-directly flag value should not error")
			assert.Equal(t, tt.expected, withDirectlyValue, "with-directly flag should have the expected value")
		})
	}
}

// TestWithDirectlyFlag_CombinedWithOtherFlags tests that --with-directly works with other flags
func TestWithDirectlyFlag_CombinedWithOtherFlags(t *testing.T) {
	cmd := NewRootCmd()
	args := []string{
		"--with-directly",
		"--config", "/tmp/test-kubeconfig",
		"--auto-create",
		"--cluster", "prod,staging",
		"--dry-run",
	}

	err := cmd.ParseFlags(args)
	assert.NoError(t, err, "parsing combined flags should not error")

	// Verify with-directly flag
	withDirectlyValue, _ := cmd.Flags().GetBool("with-directly")
	assert.True(t, withDirectlyValue)

	// Verify other flags are still working
	configValue, _ := cmd.Flags().GetString("config")
	assert.Equal(t, "/tmp/test-kubeconfig", configValue)

	autoCreateValue, _ := cmd.Flags().GetBool("auto-create")
	assert.True(t, autoCreateValue)

	clusterValue, _ := cmd.Flags().GetString("cluster")
	assert.Equal(t, "prod,staging", clusterValue)

	dryRunValue, _ := cmd.Flags().GetBool("dry-run")
	assert.True(t, dryRunValue)
}

// TestNewRootCmd_WithDirectlyFlagInitialization tests that withDirectly variable is properly initialized
func TestNewRootCmd_WithDirectlyFlagInitialization(t *testing.T) {
	// Reset withDirectly to ensure clean state
	withDirectly = false

	cmd := NewRootCmd()
	args := []string{"--with-directly"}

	err := cmd.ParseFlags(args)
	assert.NoError(t, err)

	// After parsing, the global withDirectly variable should be set
	assert.True(t, withDirectly)
}

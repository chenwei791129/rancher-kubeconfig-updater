package cmd

import (
	"rancher-kubeconfig-updater/internal/rancher"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

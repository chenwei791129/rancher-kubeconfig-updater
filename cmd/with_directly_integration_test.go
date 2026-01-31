package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

// TestCountDirectContexts tests the countDirectContexts helper function
func TestCountDirectContexts(t *testing.T) {
	tests := []struct {
		name        string
		contexts    map[string]*api.Context
		clusterName string
		expected    int
	}{
		{
			name: "no direct contexts",
			contexts: map[string]*api.Context{
				"demo-cluster": {},
			},
			clusterName: "demo-cluster",
			expected:    0,
		},
		{
			name: "with direct contexts",
			contexts: map[string]*api.Context{
				"demo-cluster":        {},
				"demo-cluster-node01": {},
				"demo-cluster-node02": {},
				"demo-cluster-node03": {},
			},
			clusterName: "demo-cluster",
			expected:    3,
		},
		{
			name: "mixed clusters",
			contexts: map[string]*api.Context{
				"prod":         {},
				"prod-node01":  {},
				"staging":      {},
				"staging-fqdn": {},
			},
			clusterName: "prod",
			expected:    1,
		},
		{
			name: "no matching direct contexts",
			contexts: map[string]*api.Context{
				"demo-cluster": {},
				"other-node01": {},
				"another-fqdn": {},
			},
			clusterName: "demo-cluster",
			expected:    0,
		},
		{
			name: "pattern edge case - similar prefix",
			contexts: map[string]*api.Context{
				"prod":       {},
				"prod-node":  {},
				"production": {}, // Should NOT match
			},
			clusterName: "prod",
			expected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &api.Config{
				Contexts: tt.contexts,
			}
			result := countDirectContexts(cfg, tt.clusterName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCountDirectContexts_NilConfig tests handling of nil contexts map
func TestCountDirectContexts_NilConfig(t *testing.T) {
	cfg := &api.Config{
		Contexts: nil,
	}
	result := countDirectContexts(cfg, "demo")
	assert.Equal(t, 0, result)
}

// TestCountDirectContexts_EmptyConfig tests handling of empty contexts map
func TestCountDirectContexts_EmptyConfig(t *testing.T) {
	cfg := &api.Config{
		Contexts: make(map[string]*api.Context),
	}
	result := countDirectContexts(cfg, "demo")
	assert.Equal(t, 0, result)
}

# Design: Add --with-directly Flag for Downstream Directly Context Import

**Issue**: [#25](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/25)
**Date**: 2025-01-31
**Status**: Approved

## Overview

Add a new `--with-directly` flag that enables importing contexts using the "Downstream Directly" connection method provided by Rancher. This allows cluster operations to continue even when the Rancher server is unavailable.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Context handling | Merge mode | Include all direct contexts alongside proxy context |
| Refactoring strategy | Modify existing function | Change `GetClusterToken()` to return `*api.Config` |
| Conflict handling | Overwrite mode | Directly overwrite existing entries with same name |
| Configuration | Flag + Environment variable | Support `--with-directly` and `WITH_DIRECTLY` env var |

## Architecture

### Data Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│ cmd/root.go                                                             │
│ ├─ Parse --with-directly flag / WITH_DIRECTLY env                       │
│ └─ For each cluster:                                                    │
│    ├─ client.GetClusterKubeconfig(clusterID)  ──────────────────┐       │
│    └─ kubeconfig.MergeKubeconfig(target, source, name, withDir) │       │
└─────────────────────────────────────────────────────────────────┼───────┘
                                                                  │
┌─────────────────────────────────────────────────────────────────┼───────┐
│ internal/rancher/client.go                                      │       │
│ GetClusterKubeconfig(clusterID) → *api.Config                   │       │
│ ├─ POST /v3/clusters/{id}?action=generateKubeconfig             │       │
│ ├─ Parse YAML response into *api.Config                         │       │
│ └─ Return complete kubeconfig (includes direct contexts)  ◄─────┘       │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│ internal/kubeconfig/manager.go                                          │
│ MergeKubeconfig(target, source *api.Config, clusterName, withDirectly)  │
│ ├─ Always merge primary context (exact match on clusterName)            │
│ └─ If withDirectly=true, also merge contexts prefixed with clusterName- │
└─────────────────────────────────────────────────────────────────────────┘
```

### Kubeconfig Structure (Rancher Response)

Example from `generateKubeconfig` API when Downstream Directly is enabled:

```yaml
apiVersion: v1
kind: Config
clusters:
- name: "demo-cluster"                    # Primary (Rancher proxy)
  cluster:
    server: "https://rancher.example.com/k8s/clusters/c-m-xxxxx"
- name: "demo-cluster-node01"             # Direct access
  cluster:
    server: "https://192.168.1.101:6443"
    certificate-authority-data: "LS0tLS..."
- name: "demo-cluster-node02"             # Direct access
  cluster:
    server: "https://192.168.1.102:6443"
    certificate-authority-data: "LS0tLS..."

users:
- name: "demo-cluster"                    # Shared user for all contexts
  user:
    token: "kubeconfig-user:xxxxx"

contexts:
- name: "demo-cluster"
  context: {cluster: "demo-cluster", user: "demo-cluster"}
- name: "demo-cluster-node01"
  context: {cluster: "demo-cluster-node01", user: "demo-cluster"}
- name: "demo-cluster-node02"
  context: {cluster: "demo-cluster-node02", user: "demo-cluster"}
```

**Key observations:**
- Direct context naming: `{clusterName}-{nodeHostname}`
- All contexts share the same user (token)
- Direct clusters have `certificate-authority-data`, primary does not
- Node hostname comes from actual machine names

## API Changes

### GetClusterKubeconfig (Refactored from GetClusterToken)

**File**: `internal/rancher/client.go`

```go
// GetClusterKubeconfig retrieves the full kubeconfig for a cluster
// including Downstream Directly contexts if available.
func (c *Client) GetClusterKubeconfig(clusterID string) (*api.Config, error) {
    // POST /v3/clusters/{clusterID}?action=generateKubeconfig
    // Parse response.config (YAML) into *api.Config
    // Return the complete kubeconfig structure
}
```

### MergeKubeconfig (New Function)

**File**: `internal/kubeconfig/manager.go`

```go
// MergeKubeconfig merges source kubeconfig into target.
// When withDirectly is true, includes all contexts (proxy + direct).
// When withDirectly is false, only includes the primary proxy context.
// Existing entries with the same name are overwritten.
func MergeKubeconfig(target, source *api.Config, clusterName string, withDirectly bool) {
    // 1. Always merge primary context (matches clusterName exactly)
    //    - target.Clusters[clusterName] = source.Clusters[clusterName]
    //    - target.Contexts[clusterName] = source.Contexts[clusterName]
    //    - target.AuthInfos[clusterName] = source.AuthInfos[clusterName]

    // 2. If withDirectly is true, also merge direct contexts:
    //    - Contexts with prefix: {clusterName}-
    //    - Their associated clusters and authInfos
}
```

## CLI Changes

### New Flag

**File**: `cmd/root.go`

```go
var withDirectly bool

rootCmd.Flags().BoolVar(&withDirectly, "with-directly", false,
    "Include Downstream Directly contexts for direct cluster access")
```

### Configuration Priority

```
Flag (--with-directly) > Environment (WITH_DIRECTLY) > Default (false)
```

## Mock Server Extension

### New Data Structures

**File**: `internal/rancher/rancher_mock_test.go`

```go
// MockDirectNode represents a node for Downstream Directly access
type MockDirectNode struct {
    Hostname string // e.g., "node01", "master-1"
    Server   string // e.g., "192.168.1.101:6443" or "k8s.internal.local:6443"
}

// MockClusterConfig holds cluster configuration for mock responses
type MockClusterConfig struct {
    ID              string
    Name            string
    DirectNodes     []MockDirectNode // Nodes for direct access
    CACert          string           // CA certificate for direct clusters
}

// Option to configure clusters with Downstream Directly
func WithClusterDirectly(clusterID, clusterName string, nodes []MockDirectNode, caCert string) MockServerOption
```

### Mock Response Example

```yaml
clusters:
- name: "demo-cluster"
  cluster:
    server: "https://rancher.example.com/k8s/clusters/c-m-abcd1234"
- name: "demo-cluster-node01"
  cluster:
    server: "https://192.168.1.101:6443"
    certificate-authority-data: "bW9jay1jYS1jZXJ0LWRhdGEtZm9yLXRlc3Rpbmc="
- name: "demo-cluster-node02"
  cluster:
    server: "https://192.168.1.102:6443"
    certificate-authority-data: "bW9jay1jYS1jZXJ0LWRhdGEtZm9yLXRlc3Rpbmc="

users:
- name: "demo-cluster"
  user:
    token: "kubeconfig-user:mock-token-xxxxx"

contexts:
- name: "demo-cluster"
  context: {cluster: "demo-cluster", user: "demo-cluster"}
- name: "demo-cluster-node01"
  context: {cluster: "demo-cluster-node01", user: "demo-cluster"}
- name: "demo-cluster-node02"
  context: {cluster: "demo-cluster-node02", user: "demo-cluster"}
```

## Test Plan

### Unit Tests

**File**: `internal/rancher/client_test.go`
- `TestGetClusterKubeconfig_WithDirectly` - Verify returned config includes all contexts
- `TestGetClusterKubeconfig_WithoutDirectly` - Verify only primary context when no direct nodes

**File**: `internal/kubeconfig/manager_test.go`
- `TestMergeKubeconfig_WithDirectlyEnabled` - Verify all contexts merged
- `TestMergeKubeconfig_WithDirectlyDisabled` - Verify only primary context merged
- `TestMergeKubeconfig_OverwriteExisting` - Verify overwrite behavior

### Integration Tests

**File**: `cmd/with_directly_integration_test.go`
- `TestWithDirectlyFlag_Enabled` - End-to-end with mock server
- `TestWithDirectlyFlag_Disabled` - Verify direct contexts excluded
- `TestWithDirectlyFlag_EnvVar` - Verify environment variable works

## Implementation Plan

| Phase | Description | Files |
|-------|-------------|-------|
| 1 | Mock Server extension | `internal/rancher/rancher_mock_test.go` |
| 2 | API layer refactoring | `internal/rancher/client.go` |
| 3 | Kubeconfig merge logic | `internal/kubeconfig/manager.go` |
| 4 | CLI integration | `cmd/root.go` |
| 5 | Tests | `*_test.go` files |

## File Changes Summary

| File | Change Type |
|------|-------------|
| `internal/rancher/rancher_mock_test.go` | Modify |
| `internal/rancher/client.go` | Modify |
| `internal/kubeconfig/manager.go` | Modify |
| `cmd/root.go` | Modify |
| `internal/rancher/client_test.go` | Add/Modify |
| `internal/kubeconfig/manager_test.go` | Add/Modify |
| `cmd/with_directly_integration_test.go` | Add |

## References

- [Rancher: Authenticating Directly with a Downstream Cluster](https://ranchermanager.docs.rancher.com/how-to-guides/new-user-guides/manage-clusters/access-clusters/use-kubectl-and-kubeconfig#authenticating-directly-with-a-downstream-cluster)
- [Issue #25 Research Comment](https://github.com/chenwei791129/rancher-kubeconfig-updater/issues/25#issuecomment-3675670020)

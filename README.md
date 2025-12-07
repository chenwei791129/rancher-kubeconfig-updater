# Rancher Kubeconfig Updater

A command-line tool that automatically updates kubeconfig tokens for all clusters managed by Rancher. This tool helps maintain up-to-date credentials for your Kubernetes clusters without manual intervention.

## Features

- 🔄 Automatically fetch and update tokens for all Rancher-managed clusters
- 🆕 Auto-create kubeconfig entries for new clusters (optional)
- 📝 Structured logging with configurable output
- 💾 Automatic backup of kubeconfig before modifications
- ⚡ Atomic file writes to prevent corruption
- 🔒 Secure credential handling via environment variables

## Prerequisites

- Go 1.25 or later
- Access to a Rancher instance
- Valid Rancher credentials (username and password)

## Installation

### From Source

```bash
git clone <repository-url>
cd rancher-kubeconfig-updater
go build
```

### Binary Release

Download the latest release from the releases page and add it to your PATH.

## Configuration

Create a `.env` file in the project root or set environment variables:

```bash
RANCHER_URL=https://rancher.example.com
RANCHER_USERNAME=your-username
RANCHER_PASSWORD=your-password
```

You can use the provided `.env.example` as a template:

```bash
cp .env.example .env
# Edit .env with your actual credentials
```

## Usage

### Basic Usage

Update tokens for all existing clusters in your kubeconfig:

```bash
./rancher-kubeconfig-updater
```

### Auto-Create Mode

Automatically create kubeconfig entries for clusters that don't exist in your config:

```bash
./rancher-kubeconfig-updater --auto-create
# or use the short form
./rancher-kubeconfig-updater -a
```

### Command Line Options

- `--auto-create`, `-a`: Automatically create kubeconfig entries for clusters not found in the config

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `RANCHER_URL` | Rancher server URL (e.g., `https://rancher.example.com`) | Yes |
| `RANCHER_USERNAME` | Rancher username for authentication | Yes |
| `RANCHER_PASSWORD` | Rancher password for authentication | Yes |

## How It Works

1. **Authentication**: Connects to Rancher using provided credentials
2. **Cluster Discovery**: Retrieves list of all managed clusters
3. **Token Generation**: Generates fresh kubeconfig tokens for each cluster
4. **Config Update**: Updates or creates entries in `~/.kube/config`
5. **Backup**: Creates timestamped backups before any modifications
6. **Atomic Write**: Safely writes the updated configuration

## Output Examples

### Normal Mode (Existing Clusters)

```log
2025-12-08T10:30:00.123+0800 | INFO | Successfully updated kubeconfig token for cluster: production-cluster
2025-12-08T10:30:00.456+0800 | INFO | Successfully updated kubeconfig token for cluster: staging-cluster
2025-12-08T10:30:00.789+0800 | WARN | Cluster not found in kubeconfig, skipping: new-cluster
2025-12-08T10:30:01.012+0800 | INFO | All cluster tokens have been updated successfully
```

### Auto-Create Mode

```log
2025-12-08T10:30:00.123+0800 | INFO | Successfully updated kubeconfig token for cluster: production-cluster
2025-12-08T10:30:00.456+0800 | INFO | Created new kubeconfig entry for cluster: new-cluster
2025-12-08T10:30:00.789+0800 | INFO | All cluster tokens have been updated successfully
```

### First Run (New Config File)

```log
2025-12-08T10:30:00.123+0800 | INFO | Creating new kubeconfig file at ~/.kube/config
2025-12-08T10:30:00.456+0800 | INFO | Created new kubeconfig entry for cluster: cluster-1
2025-12-08T10:30:00.789+0800 | INFO | Created new kubeconfig entry for cluster: cluster-2
2025-12-08T10:30:01.012+0800 | INFO | All cluster tokens have been updated successfully
```

## File Structure

```
~/.kube/
├── config                          # Main kubeconfig file
└── config.backup.20251208-103000.000000  # Automatic backup
```

## Security Considerations

- ✅ Credentials are loaded from environment variables only
- ✅ No sensitive information is hardcoded
- ✅ Tokens are handled in memory only
- ✅ Backup files preserve original permissions (0600)
- ✅ New kubeconfig files are created with secure permissions (0600)
- ⚠️ Ensure your `.env` file is not committed to version control

## Troubleshooting

### Authentication Failed

```log
ERROR | Failed to authenticate with Rancher
```

**Solution**: Verify your `RANCHER_URL`, `RANCHER_USERNAME`, and `RANCHER_PASSWORD` are correct.

### Cluster Not Found

```log
WARN | Cluster not found in kubeconfig, skipping: cluster-name
```

**Solution**: Use the `--auto-create` flag to automatically create entries for new clusters.

### Permission Denied

```log
ERROR | Failed to save kubeconfig file | {"error": "permission denied"}
```

**Solution**: Ensure you have write permissions to `~/.kube/config` and the `~/.kube` directory.

## Development

### Build

```bash
go build
```

### Run Tests

```bash
go test ./...
```

### Dependencies

- [uber-go/zap](https://github.com/uber-go/zap) - Structured logging
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing
- [joho/godotenv](https://github.com/joho/godotenv) - Environment variable loading

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions, please open an issue in the repository.

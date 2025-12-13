# Rancher Kubeconfig Updater

![Visitors](https://api.visitorbadge.io/api/visitors?path=https%3A%2F%2Fgithub.com%2Fchenwei791129%2Francher-kubeconfig-updater&label=visitors&countColor=%230c7ebe&style=flat&labelStyle=none)
![License](https://img.shields.io/github/license/chenwei791129/rancher-kubeconfig-updater)
![GitHub Repo stars](https://img.shields.io/github/stars/chenwei791129/rancher-kubeconfig-updater)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/chenwei791129/rancher-kubeconfig-updater/go-test.yml?event=pull_request&label=pr-tests)

A command-line tool to update kubeconfig tokens for Rancher-managed Kubernetes clusters.

## Features

- Update kubeconfig tokens for all Rancher-managed clusters
- Auto-create kubeconfig entries for new clusters (optional)
- Backup kubeconfig before modifications

## Installation

### Linux / macOS

```bash
# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

# Download the latest release
curl -LO "https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/latest/download/rancher-kubeconfig-updater-${OS}-${ARCH}"

# Rename to simpler name
mv "rancher-kubeconfig-updater-${OS}-${ARCH}" rancher-kubeconfig-updater

# Make it executable
chmod +x rancher-kubeconfig-updater

# Move to PATH (optional)
sudo mv rancher-kubeconfig-updater /usr/local/bin/
```

### Windows

```powershell
# Download the latest release
curl.exe -LO https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/latest/download/rancher-kubeconfig-updater-windows-amd64.exe

# Rename to simpler name
ren rancher-kubeconfig-updater-windows-amd64.exe rancher-kubeconfig-updater.exe

# Move to a directory in your PATH (optional)
move rancher-kubeconfig-updater.exe C:\Windows\System32\
```

## Configuration

Set environment variables:

```bash
export RANCHER_URL=https://rancher.example.com
export RANCHER_USERNAME=your-username
export RANCHER_PASSWORD=your-password
export RANCHER_AUTH_TYPE=local  # Optional: "local" or "ldap" (default: local)
```

Or use a `.env` file:

```bash
cp .env.example .env
# Edit .env with your credentials
```

### Authentication Types

The tool supports two authentication methods:

- **local** (default) - Use Rancher local authentication
- **ldap** - Use LDAP authentication

To use LDAP authentication, set:

```bash
export RANCHER_AUTH_TYPE=ldap
```

## Usage

Update tokens for existing clusters:

```bash
./rancher-kubeconfig-updater
```

Auto-create entries for new clusters:

```bash
./rancher-kubeconfig-updater -a
# or
./rancher-kubeconfig-updater --auto-create
```

Use LDAP authentication:

```bash
./rancher-kubeconfig-updater --auth-type ldap
```

## Flags

```
Flags:
      --auth-type string        Authentication type: 'local' or 'ldap' (default: from RANCHER_AUTH_TYPE env or 'local')
  -a, --auto-create             Automatically create kubeconfig entries for clusters not found in the config
  -h, --help                    help for rancher-kubeconfig-updater
  -p, --password string[="-"]   Rancher Password
  -u, --user string             Rancher Username
```

**Note**: Command line flags take precedence over environment variables.

## License

MIT License - see [LICENSE](LICENSE) file for details.

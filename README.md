# Rancher Kubeconfig Updater

![Visitors](https://api.visitorbadge.io/api/visitors?path=https%3A%2F%2Fgithub.com%2Fchenwei791129%2Francher-kubeconfig-updater&label=visitors&countColor=%230c7ebe&style=flat&labelStyle=none)
![License](https://img.shields.io/github/license/chenwei791129/rancher-kubeconfig-updater)
![GitHub Repo stars](https://img.shields.io/github/stars/chenwei791129/rancher-kubeconfig-updater)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/chenwei791129/rancher-kubeconfig-updater/go-test.yml?event=pull_request&label=pr-tests)

A command-line tool to update kubeconfig tokens for Rancher-managed Kubernetes clusters.

## Features

- Bulk-update kubeconfig tokens for all Rancher-managed clusters
- Smart refresh: skip tokens still valid beyond a configurable threshold (handles never-expiring `TTL=0` tokens)
- Dry-run mode previews changes without touching kubeconfig
- Optionally auto-create kubeconfig entries for newly discovered clusters
- Backs up kubeconfig before modifications
- Supports self-signed certificates via TLS skip flag (dev/test only)

## Installation

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.sh | sh
```

| Variable      | Default             | Description                              |
| ------------- | ------------------- | ---------------------------------------- |
| `VERSION`     | `latest`            | Release tag to install (e.g., `v1.4.0`). |
| `INSTALL_DIR` | `$HOME/.local/bin`  | Target install directory.                |

### Windows

```powershell
irm https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.ps1 | iex
```

| Variable      | Default                       | Description                              |
| ------------- | ----------------------------- | ---------------------------------------- |
| `VERSION`     | `latest`                      | Release tag to install (e.g., `v1.4.0`). |
| `INSTALL_DIR` | `$env:USERPROFILE\.local\bin` | Target install directory.                |

## Configuration

Configure non-sensitive settings via environment variables in your shell:

```bash
export RANCHER_URL=https://rancher.example.com
export RANCHER_USERNAME=your-username
export RANCHER_AUTH_TYPE=local  # "local" (default) or "ldap"
```

Then run the tool with `-p` to enter your password interactively:

```bash
rancher-kubeconfig-updater -p
```

### Supported Environment Variables

| Variable                           | Description                                              |
| ---------------------------------- | -------------------------------------------------------- |
| `RANCHER_URL`                      | Rancher server URL.                                      |
| `RANCHER_USERNAME`                 | Rancher username.                                        |
| `RANCHER_PASSWORD`                 | Rancher password. Prefer `-p` for interactive input.     |
| `RANCHER_AUTH_TYPE`                | `local` (default) or `ldap`.                             |
| `RANCHER_INSECURE_SKIP_TLS_VERIFY` | Skip TLS verification (insecure; dev/test only).         |
| `TOKEN_THRESHOLD_DAYS`             | Token expiration threshold in days (default: `30`).      |
| `FORCE_REFRESH`                    | Force regeneration regardless of expiration.             |
| `DRY_RUN`                          | Preview changes without modifying kubeconfig.            |

Command-line flags take precedence over environment variables.

## Usage

```bash
# Update tokens for existing clusters (interactive password)
rancher-kubeconfig-updater -p

# Auto-create kubeconfig entries for newly discovered clusters
rancher-kubeconfig-updater -p -a

# Preview changes without modifying kubeconfig
rancher-kubeconfig-updater -p --dry-run

# Target a specific kubeconfig file and a subset of clusters
rancher-kubeconfig-updater -p -c ~/my-kubeconfig --cluster prod,staging

# Use LDAP authentication
rancher-kubeconfig-updater -p --auth-type ldap
```

If `RANCHER_PASSWORD` is already set in the environment, the `-p` flag can be omitted.

## Flags

```
Flags:
      --auth-type string           Authentication type: 'local' or 'ldap' (default: from RANCHER_AUTH_TYPE env or 'local')
  -a, --auto-create                Automatically create kubeconfig entries for clusters not found in the config
      --cluster string             Comma-separated list of cluster names or IDs to update
  -c, --config string              Path to kubeconfig file (default: ~/.kube/config)
      --dry-run                    Preview changes without modifying kubeconfig
      --force-refresh              Bypass expiration checks and force regeneration
  -h, --help                       help for rancher-kubeconfig-updater
      --insecure-skip-tls-verify   Skip TLS certificate verification (insecure, use only for development/testing)
  -p, --password string[="-"]      Rancher Password
      --threshold-days int         Expiration threshold in days (default: 30)
  -u, --user string                Rancher Username
```

### Notes

- `-p` prompts for the password interactively without echoing it. Pass `-p=<password>` to provide the value inline (less secure).
- `--cluster` accepts a comma-separated list of cluster **names or IDs**, case-insensitive; whitespace is trimmed and unknown entries are logged as warnings.
- `-c` accepts `~` and relative paths; defaults to `~/.kube/config`.
- Command-line flags take precedence over environment variables.

## Token Expiration Checking

Before regenerating, the tool queries each token's expiration via the Rancher API. Tokens still valid beyond `--threshold-days` (default: 30) are skipped, as are never-expiring tokens (`TTL=0`). If expiration cannot be determined, the tool regenerates the token to stay fail-safe. Use `--force-refresh` to bypass these checks entirely.

Example output:

```
INFO | Token is still valid, skipping regeneration | cluster=production | expiresAt=2024-03-15 10:30:00 | daysUntilExpiration=45.2
INFO | Token expires soon, regenerating | cluster=staging | expiresAt=2024-02-10 15:20:00 | daysUntilExpiration=15.8
INFO | Token never expires, skipping regeneration | cluster=development
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

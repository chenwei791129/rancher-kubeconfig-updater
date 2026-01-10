# Rancher Kubeconfig Updater

![Visitors](https://api.visitorbadge.io/api/visitors?path=https%3A%2F%2Fgithub.com%2Fchenwei791129%2Francher-kubeconfig-updater&label=visitors&countColor=%230c7ebe&style=flat&labelStyle=none)
![License](https://img.shields.io/github/license/chenwei791129/rancher-kubeconfig-updater)
![GitHub Repo stars](https://img.shields.io/github/stars/chenwei791129/rancher-kubeconfig-updater)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/chenwei791129/rancher-kubeconfig-updater/go-test.yml?event=pull_request&label=pr-tests)

A command-line tool to update kubeconfig tokens for Rancher-managed Kubernetes clusters.

## Features

- Update kubeconfig tokens for all Rancher-managed clusters
- **Smart token refresh**: Check token expiration before regenerating (skip unnecessary updates)
- **Dry-run mode**: Preview changes without modifying kubeconfig (safe testing and validation)
- Auto-create kubeconfig entries for new clusters (optional)
- Backup kubeconfig before modifications
- Skip TLS certificate verification for development/testing environments with self-signed certificates
- Configurable expiration threshold for token refresh
- Support for never-expiring tokens (TTL = 0)

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

### Recommended: Use Command-Line Password Input (Most Secure)

> [!IMPORTANT]
> **Security Best Practice**: For better security, avoid storing your password in files or environment variables. Instead, use the `-p` flag to enter your password interactively at runtime.

Configure only the non-sensitive settings in a `.env` file:

```bash
# Rancher Configuration
RANCHER_URL=https://rancher.example.com
RANCHER_USERNAME=your-username

# Auth type, defaults to "local", can be "ldap" or "local"
RANCHER_AUTH_TYPE=local

# Optional: Skip TLS certificate verification (see Security Considerations below)
# RANCHER_INSECURE_SKIP_TLS_VERIFY=false
```

**Steps:**
1. Create a new file named `.env` in your project directory
2. Copy the template above into the file
3. Replace the placeholder values:
   - `https://rancher.example.com` → Your Rancher server URL
   - `your-username` → Your Rancher username
   - `local` → Keep as `local` for Rancher local auth, or change to `ldap` for LDAP authentication
4. Run the tool with the `-p` flag to enter your password securely:
   ```bash
   ./rancher-kubeconfig-updater -p
   ```
   The tool will prompt you to enter your password, which won't be displayed on screen or stored anywhere.

### Alternative: Store Password (Less Secure)

> [!WARNING]
> **Security Risk**: Storing passwords in `.env` files or environment variables can lead to accidental exposure through version control, logs, or process listings. Only use this method in secure, isolated environments.

If you need to store the password (e.g., for automation in secure CI/CD pipelines), you can add it to your configuration:

**Option 1: Using a `.env` File**

Add the password to your `.env` file:

```bash
# Rancher Configuration
RANCHER_URL=https://rancher.example.com
RANCHER_USERNAME=your-username
RANCHER_PASSWORD=your-password  # ⚠️ Security risk

# Auth type, defaults to "local", can be "ldap" or "local"
RANCHER_AUTH_TYPE=local
```

> [!CAUTION]
> If you store passwords in `.env`, ensure the file is:
> - Added to `.gitignore` to prevent committing to version control
> - Protected with appropriate file permissions (e.g., `chmod 600 .env`)
> - Never shared or exposed in logs

**Option 2: Environment Variables**

Set environment variables in your shell:

```bash
export RANCHER_URL=https://rancher.example.com
export RANCHER_USERNAME=your-username
export RANCHER_PASSWORD=your-password  # ⚠️ Security risk
export RANCHER_AUTH_TYPE=local  # Optional: "local" or "ldap" (default: local)
```

### Authentication Types

The tool supports two authentication methods:

- **local** (default) - Use Rancher local authentication
- **ldap** - Use LDAP authentication

You can specify the authentication type in the `.env` file or via command line flag (see [Flags](#flags) section).

## Usage

### Recommended: Interactive Password Entry

Update tokens with password prompt (most secure):

```bash
./rancher-kubeconfig-updater -p
```

Auto-create entries for new clusters with password prompt:

```bash
./rancher-kubeconfig-updater -a -p
# or
./rancher-kubeconfig-updater --auto-create -p
```

Use LDAP authentication with password prompt:

```bash
./rancher-kubeconfig-updater --auth-type ldap -p
```

Use a custom kubeconfig file:

```bash
# Use custom kubeconfig file
./rancher-kubeconfig-updater -c /path/to/custom-kubeconfig -p

# Short form with tilde expansion
./rancher-kubeconfig-updater -c ~/my-configs/dev-kubeconfig -p

# Combined with other flags
./rancher-kubeconfig-updater -c ./test-config -a --auth-type ldap -p
```

### Alternative: Using Stored Credentials

If you have configured `RANCHER_PASSWORD` in your `.env` file or environment variables:

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

### Flag Details

- **`-p, --password`**: Prompt for password interactively (recommended for security)
  ```bash
  ./rancher-kubeconfig-updater -p
  # You will be prompted: "Enter Rancher Password:"
  # Your password input will be hidden for security
  ```
  > [!TIP]
  > Using `-p` ensures your password is never stored in files, environment variables, or shell history.

- **`-u, --user`**: Override the username from environment variables or `.env` file
  ```bash
  ./rancher-kubeconfig-updater -u admin -p
  ```

- **`-a, --auto-create`**: Automatically create kubeconfig entries for new clusters discovered in Rancher

- **`--auth-type`**: Specify authentication method (`local` or `ldap`)

- **`-c, --config`**: Specify a custom kubeconfig file path instead of using the default `~/.kube/config`
  ```bash
  # Use custom kubeconfig file
  ./rancher-kubeconfig-updater --config /path/to/custom-kubeconfig -p
  
  # Short form
  ./rancher-kubeconfig-updater -c ~/my-configs/dev-kubeconfig -p
  
  # Combined with other flags
  ./rancher-kubeconfig-updater -c ~/kubeconfig-test -a --auth-type ldap -p
  ```
  > [!TIP]
  > - Path expansion (e.g., `~` for home directory) is supported
  > - Relative paths are supported
  > - If not specified, defaults to `~/.kube/config`

- **`--cluster`**: Update tokens for specific clusters only (instead of all clusters)
  ```bash
  # Update a single cluster
  ./rancher-kubeconfig-updater --cluster production -p
  
  # Update multiple clusters (comma-separated)
  ./rancher-kubeconfig-updater --cluster prod,staging,dev -p
  
  # Use cluster IDs instead of names
  ./rancher-kubeconfig-updater --cluster c-m-12345,c-m-67890 -p
  
  # Mix cluster names and IDs
  ./rancher-kubeconfig-updater --cluster production,c-m-67890 -p
  
  # Combined with other flags
  ./rancher-kubeconfig-updater --cluster prod,staging -a --auth-type ldap -p
  ```
  > [!TIP]
  > - Cluster matching is **case-insensitive**
  > - Both cluster **names** and **IDs** are supported
  > - The tool will log a warning if a specified cluster is not found
  > - Whitespace around cluster names is automatically trimmed

- **`--dry-run`**: Preview changes without actually modifying the kubeconfig file
  ```bash
  # Preview token updates for all clusters
  ./rancher-kubeconfig-updater --dry-run -p
  
  # Preview with specific clusters
  ./rancher-kubeconfig-updater --dry-run --cluster prod,staging -p
  
  # Preview with auto-create enabled
  ./rancher-kubeconfig-updater --dry-run --auto-create -p
  
  # Preview with custom config file
  ./rancher-kubeconfig-updater --dry-run -c ~/my-kubeconfig -p
  ```
  > [!TIP]
  > - **Read-only mode**: Authenticates to Rancher and checks token status without making changes
  > - **Preview changes**: Shows which clusters would be updated and why
  > - **No side effects**: Doesn't modify kubeconfig or create backup files
  > - **Safe testing**: Test configuration and credentials before making actual changes
  > - **CI/CD friendly**: Validate token status in pipelines without modifications
  > 
  > **Example output:**
  > ```
  > [DRY-RUN] Mode enabled - no changes will be made to kubeconfig
  > INFO | [DRY-RUN] Would regenerate token | cluster=my-cluster-1 | reason=expires-soon | daysUntilExpiration=15.8
  > INFO | [DRY-RUN] Would skip token regeneration | cluster=my-cluster-2 | reason=still-valid | daysUntilExpiration=45.2
  > INFO | [DRY-RUN] Summary | clustersToUpdate=1 | clustersToSkip=18
  > [DRY-RUN] No changes were made to kubeconfig
  > ```

- **`--insecure-skip-tls-verify`**: Skip TLS certificate verification (see [TLS Certificate Verification](#tls-certificate-verification) section for details)

- **`--threshold-days`**: Set the expiration threshold in days (default: 30)
  ```bash
  # Only regenerate tokens expiring within 7 days
  ./rancher-kubeconfig-updater --threshold-days 7 -p
  
  # Use a longer threshold (60 days)
  ./rancher-kubeconfig-updater --threshold-days 60 -p
  ```
  > [!TIP]
  > - The tool checks token expiration before regenerating
  > - Tokens valid beyond the threshold are skipped (saves API calls)
  > - Can be set via `TOKEN_THRESHOLD_DAYS` environment variable
  > - Default threshold is 30 days

- **`--force-refresh`**: Force regeneration of all tokens, bypassing expiration checks
  ```bash
  # Force regenerate all tokens regardless of expiration
  ./rancher-kubeconfig-updater --force-refresh -p
  
  # Combined with other flags
  ./rancher-kubeconfig-updater --force-refresh --cluster production -p
  ```
  > [!TIP]
  > - Useful when you need to regenerate all tokens immediately
  > - Bypasses all expiration checks
  > - Can be set via `FORCE_REFRESH` environment variable

**Note**: Command line flags take precedence over environment variables.

## Token Expiration Checking

The tool includes smart token expiration checking to avoid unnecessary token regeneration. This feature reduces API load on Rancher and speeds up execution when tokens are still valid.

### How It Works

1. **Checks existing tokens**: Before regenerating, the tool queries the Rancher API to check when each token expires
2. **Compares with threshold**: Tokens are only regenerated if they expire within the threshold period (default: 30 days)
3. **Skips valid tokens**: Tokens that are still valid beyond the threshold are left unchanged
4. **Handles never-expiring tokens**: Tokens with TTL = 0 (never expire) are automatically skipped

### Configuration

**Set expiration threshold:**
```bash
# Via command-line flag (30 days)
./rancher-kubeconfig-updater --threshold-days 30 -p

# Via environment variable
export TOKEN_THRESHOLD_DAYS=30
./rancher-kubeconfig-updater -p

# Via .env file
TOKEN_THRESHOLD_DAYS=30
```

**Force regeneration (bypass checks):**
```bash
# Via command-line flag
./rancher-kubeconfig-updater --force-refresh -p

# Via environment variable
export FORCE_REFRESH=true
./rancher-kubeconfig-updater -p
```

### Example Output

When tokens are checked:

```
INFO | Token is still valid, skipping regeneration | cluster=production | expiresAt=2024-03-15 10:30:00 | daysUntilExpiration=45.2
INFO | Token expires soon, regenerating | cluster=staging | expiresAt=2024-02-10 15:20:00 | daysUntilExpiration=15.8
INFO | Token never expires, skipping regeneration | cluster=development
```

### Benefits

- **Reduced API calls**: Only regenerates tokens when necessary
- **Faster execution**: Skips regeneration for valid tokens
- **Efficient for scheduled runs**: Safe to run frequently without unnecessary load
- **Respects API rate limits**: Fewer API calls = better for Rancher performance
- **Automatic handling**: Works seamlessly with existing workflows

### Error Handling

If the tool cannot check a token's expiration (e.g., due to API errors), it will:
1. Log a warning message
2. **Safely regenerate the token** (fail-safe approach)
3. Continue processing other clusters

This ensures tokens are always kept up-to-date even if expiration checking fails.

## TLS Certificate Verification

By default, the tool verifies TLS certificates when connecting to the Rancher API. However, in certain environments (development, testing, or internal networks), you may need to skip this verification.

### When to Use This Option

✅ **Appropriate Use Cases:**
- Development environments with self-signed certificates
- Testing servers with internal CA certificates
- POC/Demo environments for quick setup
- Internal networks with custom certificate authorities

❌ **NOT Recommended For:**
- Production environments
- Public-facing Rancher instances
- Environments where security is critical

### How to Enable

**Option 1: Command-Line Flag**
```bash
./rancher-kubeconfig-updater --insecure-skip-tls-verify -p
```

**Option 2: Environment Variable**
```bash
export RANCHER_INSECURE_SKIP_TLS_VERIFY=true
./rancher-kubeconfig-updater -p
```

**Option 3: `.env` File**
```bash
# Add to your .env file
RANCHER_INSECURE_SKIP_TLS_VERIFY=true
```

### Security Warning

When TLS verification is disabled, the tool will display prominent warning messages:

```
⚠️  WARNING: TLS certificate verification is disabled!
⚠️  This is insecure and should only be used in development/test environments.
⚠️  Your connection may be vulnerable to man-in-the-middle attacks.
```

> [!CAUTION]
> **Security Risk**: Disabling TLS verification makes your connection vulnerable to man-in-the-middle attacks. Only use this option in trusted, isolated networks where security risks are acceptable.

## License

MIT License - see [LICENSE](LICENSE) file for details.

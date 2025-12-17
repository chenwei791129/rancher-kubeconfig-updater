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
      --auth-type string        Authentication type: 'local' or 'ldap' (default: from RANCHER_AUTH_TYPE env or 'local')
  -a, --auto-create             Automatically create kubeconfig entries for clusters not found in the config
  -h, --help                    help for rancher-kubeconfig-updater
  -p, --password string[="-"]   Rancher Password
  -u, --user string             Rancher Username
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

**Note**: Command line flags take precedence over environment variables.

## License

MIT License - see [LICENSE](LICENSE) file for details.

# install-script Specification

## Purpose

Defines the install script contract for `rancher-kubeconfig-updater`. The repository
ships two scripts at its root that share the same `VERSION` / `INSTALL_DIR`
environment variable model, the same fail-fast platform allowlist, and the same
staging-download pattern:

- `install.sh` — POSIX `sh` for Linux and macOS, invoked via `curl ... | sh`.
- `install.ps1` — PowerShell 5.1+ for Windows, invoked via `irm ... | iex`.

The requirements below cover the one-line invocation, supported platforms,
version selection, install-directory handling (per-platform default + opt-in
non-default paths with platform-specific elevation rules), and the README
content that documents all of the above.

## Requirements

### Requirement: One-line installation entry point

The repository SHALL provide an `install.sh` script at the project root that installs the `rancher-kubeconfig-updater` binary on Linux and macOS via a single `curl ... | sh` invocation. The script SHALL be POSIX `sh` compatible and MUST NOT require `bash`-only features.

#### Scenario: Default install on supported platform

- **WHEN** a user runs `curl -fsSL https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.sh | sh` on a `linux-amd64` or `darwin-arm64` host with `curl` available, `$HOME` set, and no `INSTALL_DIR` override
- **THEN** the script downloads the latest release binary, makes it executable, places it at `$HOME/.local/bin/rancher-kubeconfig-updater` (creating that directory if it does not already exist), and prints a confirmation line containing the installed version and target path

---
### Requirement: Platform allowlist with fail-fast behavior

The script SHALL detect the host operating system and architecture, normalize them to the release artifact naming scheme, and proceed only when the resulting `${OS}-${ARCH}` pair is one of `linux-amd64` or `darwin-arm64`. For any other pair, including `linux-arm64`, `darwin-amd64`, and any unrecognized OS or architecture, the script MUST exit with a non-zero status before attempting any download and MUST emit an error message that names the detected platform and references the Building from Source section of the README.

#### Scenario: Unsupported architecture on Linux

- **WHEN** the script runs on a host where `uname -s` reports `Linux` and `uname -m` reports `aarch64`
- **THEN** the script exits with a non-zero status, prints an error message containing the string `linux-arm64` and a pointer to the Building from Source documentation, and does NOT issue any network request

#### Scenario: Unsupported macOS architecture

- **WHEN** the script runs on a host where `uname -s` reports `Darwin` and `uname -m` reports `x86_64`
- **THEN** the script exits with a non-zero status, prints an error message containing the string `darwin-amd64` and a pointer to the Building from Source documentation, and does NOT issue any network request

#### Scenario: Unrecognized operating system

- **WHEN** the script runs on a host where `uname -s` reports a value that does not normalize to `linux` or `darwin` (for example `FreeBSD` or `MINGW64_NT`)
- **THEN** the script exits with a non-zero status, prints an error message that names the detected OS string, and does NOT issue any network request

##### Example: platform detection matrix

| `uname -s` | `uname -m` | Normalized pair | Behavior |
| ---------- | ---------- | --------------- | -------- |
| `Linux` | `x86_64` | `linux-amd64` | Proceed with download |
| `Darwin` | `arm64` | `darwin-arm64` | Proceed with download |
| `Linux` | `aarch64` | `linux-arm64` | Fail-fast with error |
| `Darwin` | `x86_64` | `darwin-amd64` | Fail-fast with error |
| `FreeBSD` | `amd64` | (unsupported OS) | Fail-fast with error |

---
### Requirement: Version selection via VERSION environment variable

The script SHALL honor a `VERSION` environment variable that selects which release to install. When `VERSION` is unset or equals the literal string `latest`, the script SHALL download from `https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/latest/download/rancher-kubeconfig-updater-${OS}-${ARCH}`. When `VERSION` is set to any other non-empty value, the script SHALL download from `https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/download/${VERSION}/rancher-kubeconfig-updater-${OS}-${ARCH}`. If the HTTP response indicates a 4xx or 5xx status, the script MUST exit non-zero, print an error containing the failed URL, and MUST NOT leave a partial file at the installation target.

#### Scenario: Pinning to a specific release tag

- **WHEN** a user invokes the install pipeline with the environment `VERSION=v1.4.0` set before the piped `sh` invocation
- **THEN** the script constructs the download URL `https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/download/v1.4.0/rancher-kubeconfig-updater-${OS}-${ARCH}` and installs that exact version

#### Scenario: Non-existent version

- **WHEN** the user sets `VERSION` to a tag that does not exist on GitHub
- **THEN** the script exits non-zero, the error message contains the constructed download URL and the HTTP error code, and no file is placed at the installation target

---
### Requirement: Installation directory via INSTALL_DIR environment variable

The script SHALL honor an `INSTALL_DIR` environment variable that selects the target directory for the installed binary, defaulting to `$HOME/.local/bin` when unset. The final binary name installed in that directory SHALL be `rancher-kubeconfig-updater`. When `INSTALL_DIR` resolves to the default value and the directory does not yet exist, the script SHALL create it with `mkdir -p` before downloading; if creation fails the script MUST exit non-zero with an error naming the directory and MUST NOT escalate privileges. When `INSTALL_DIR` resolves to the default value and is not writable by the current user even after creation, the script MUST exit non-zero with an error naming the directory and MUST NOT use `sudo`. When `INSTALL_DIR` is set to any non-default value, the script MUST NOT create the directory automatically and MUST exit non-zero with an error naming the path if the directory does not exist; if the directory exists but is not writable by the current user, the script MAY use `sudo mv` to move the binary into place and MUST print a notice before invoking `sudo`. `$HOME` MUST be resolved from the environment; if `$HOME` is unset or empty the script MUST exit non-zero with an error before downloading.

#### Scenario: Install into default directory that does not exist

- **WHEN** a user runs the install pipeline with `INSTALL_DIR` unset, `$HOME` set to a writable home directory, and `$HOME/.local/bin` does not exist
- **THEN** the script creates `$HOME/.local/bin` with `mkdir -p`, places the binary at `$HOME/.local/bin/rancher-kubeconfig-updater`, does NOT invoke `sudo`, and prints a confirmation line referencing the created path

#### Scenario: Install into pre-existing default directory

- **WHEN** a user runs the install pipeline with `INSTALL_DIR` unset and `$HOME/.local/bin` already exists and is writable
- **THEN** the script places the binary at `$HOME/.local/bin/rancher-kubeconfig-updater`, leaves the directory contents unchanged (any `mkdir -p` call on an existing path is a no-op), does NOT invoke `sudo`, and prints a confirmation line referencing the chosen path

#### Scenario: Install into a user-writable non-default directory

- **WHEN** a user runs the install pipeline with `INSTALL_DIR=$HOME/bin` and `$HOME/bin` exists and is writable
- **THEN** the script places the binary at `$HOME/bin/rancher-kubeconfig-updater`, does NOT invoke `sudo`, and prints a confirmation line referencing the chosen path

#### Scenario: Explicit /usr/local/bin requires elevation

- **WHEN** a user runs the install pipeline with `INSTALL_DIR=/usr/local/bin` and `/usr/local/bin` exists but is not writable by the current user
- **THEN** the script prints a notice that `sudo` will be used, invokes `sudo mv` to place the binary at `/usr/local/bin/rancher-kubeconfig-updater`, and exits zero on success

#### Scenario: Non-default directory does not exist

- **WHEN** a user sets `INSTALL_DIR=/opt/bin` and `/opt/bin` does not exist on the host
- **THEN** the script exits non-zero with an error naming `/opt/bin`, MUST NOT invoke `mkdir`, MUST NOT invoke `sudo`, and MUST NOT leave a partial file at the target path

#### Scenario: HOME is unset

- **WHEN** a user runs the install pipeline with `INSTALL_DIR` unset and `$HOME` unset or empty
- **THEN** the script exits non-zero with an error message naming the missing `HOME` environment variable, MUST NOT issue any network request, and MUST NOT invoke `sudo`

##### Example: INSTALL_DIR decision matrix

| `INSTALL_DIR` setting        | Directory state            | Behavior                                                          |
| ---------------------------- | -------------------------- | ----------------------------------------------------------------- |
| unset (default)              | exists, writable           | `mv` to default, no sudo                                          |
| unset (default)              | does not exist             | `mkdir -p`, then `mv`, no sudo                                    |
| unset (default)              | exists, not writable       | exit non-zero, no sudo                                            |
| unset, `$HOME` unset         | n/a                        | exit non-zero before any network                                  |
| `$HOME/bin`                  | exists, writable           | `mv`, no sudo                                                     |
| `$HOME/bin`                  | does not exist             | exit non-zero, no `mkdir`, no sudo                                |
| `/usr/local/bin`             | exists, not writable       | `sudo mv` with notice                                             |
| `/opt/bin`                   | exists, not writable       | `sudo mv` with notice                                             |
| `/opt/bin`                   | does not exist             | exit non-zero, no `mkdir`, no sudo                                |

---
### Requirement: README installation section uses the script

The `README.md` `Installation` section SHALL present the Linux / macOS install path as a single `curl ... | sh` command pointing at `https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.sh`, document the `VERSION` and `INSTALL_DIR` environment variables, name the supported platforms (`linux-amd64`, `darwin-arm64`), and provide a Building from Source subsection that unsupported-platform users can follow. The Windows installation subsection SHALL remain unchanged.

#### Scenario: Reader installs on supported platform from README

- **WHEN** a reader copies the single-line `curl ... | sh` command from the Linux / macOS subsection on a `linux-amd64` or `darwin-arm64` host
- **THEN** the command completes installation without further README steps, matching the behavior defined in the One-line installation entry point requirement

#### Scenario: Reader on unsupported platform consults README

- **WHEN** a reader on `linux-arm64` or `darwin-amd64` runs the curl command and sees the fail-fast error pointing at Building from Source
- **THEN** the README contains a Building from Source subsection with `git clone` and `go build` instructions sufficient to produce a working binary locally

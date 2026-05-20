## MODIFIED Requirements

### Requirement: One-line installation entry point

The repository SHALL provide install scripts at the project root that install the `rancher-kubeconfig-updater` binary via a single piped invocation, one script per shell family:

- `install.sh` for Linux and macOS — POSIX `sh` compatible (MUST NOT require `bash`-only features), invoked via `curl -fsSL ... | sh`.
- `install.ps1` for Windows — PowerShell 5.1 or later, invoked via `irm ... | iex`.

The two scripts SHALL honor the same environment variable contract (`VERSION`, `INSTALL_DIR`) and the same fail-fast model, differing only where the host platform requires it.

#### Scenario: Default install on supported Unix platform

- **WHEN** a user runs `curl -fsSL https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.sh | sh` on a `linux-amd64` or `darwin-arm64` host with `curl` available, `$HOME` set, and no `INSTALL_DIR` override
- **THEN** the script downloads the latest release binary, makes it executable, places it at `$HOME/.local/bin/rancher-kubeconfig-updater` (creating that directory if it does not already exist), and prints a confirmation line containing the installed version and target path

#### Scenario: Default install on Windows

- **WHEN** a user runs `irm https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.ps1 | iex` on a `windows-amd64` host with `$env:USERPROFILE` set and no `INSTALL_DIR` override
- **THEN** the script downloads the latest release binary, places it at `$env:USERPROFILE\.local\bin\rancher-kubeconfig-updater.exe` (creating that directory if it does not already exist), appends that directory to the User-scope `PATH` environment variable if not already present, and prints a confirmation line containing the installed version, target path, and (when PATH was modified) a notice that the user must restart their shell for the change to take effect

---
### Requirement: Platform allowlist with fail-fast behavior

Each install script SHALL detect the host operating system and architecture, normalize them to the release artefact naming scheme, and proceed only when the resulting `${OS}-${ARCH}` pair is on the allowlist: `linux-amd64`, `darwin-arm64`, or `windows-amd64`. For any other pair — including `linux-arm64`, `darwin-amd64`, `windows-arm64`, and any unrecognized operating system or architecture — the script MUST exit with a non-zero status before attempting any download and MUST emit an error message that names the detected platform and references the Building from Source section of the README.

On Unix, OS detection uses `uname -s` and architecture detection uses `uname -m`. On Windows, OS detection is implicit (the host runs `install.ps1`) and architecture detection uses `$env:PROCESSOR_ARCHITECTURE` (with `$env:PROCESSOR_ARCHITEW6432` consulted when running under WOW64) normalized to `amd64` or `arm64`.

#### Scenario: Unsupported architecture on Linux

- **WHEN** `install.sh` runs on a host where `uname -s` reports `Linux` and `uname -m` reports `aarch64`
- **THEN** the script exits with a non-zero status, prints an error message containing the string `linux-arm64` and a pointer to the Building from Source documentation, and does NOT issue any network request

#### Scenario: Unsupported macOS architecture

- **WHEN** `install.sh` runs on a host where `uname -s` reports `Darwin` and `uname -m` reports `x86_64`
- **THEN** the script exits with a non-zero status, prints an error message containing the string `darwin-amd64` and a pointer to the Building from Source documentation, and does NOT issue any network request

#### Scenario: Unsupported Windows architecture

- **WHEN** `install.ps1` runs on a host where `$env:PROCESSOR_ARCHITECTURE` reports `ARM64` (or `$env:PROCESSOR_ARCHITEW6432` indicates `ARM64`)
- **THEN** the script exits with a non-zero status, prints an error message containing the string `windows-arm64` and a pointer to the Building from Source documentation, and does NOT issue any network request

#### Scenario: Unrecognized operating system

- **WHEN** `install.sh` runs on a host where `uname -s` reports a value that does not normalize to `linux` or `darwin` (for example `FreeBSD` or `MINGW64_NT`)
- **THEN** the script exits with a non-zero status, prints an error message that names the detected OS string, and does NOT issue any network request

##### Example: platform detection matrix

| Script        | OS detector reports        | Arch detector reports        | Normalized pair  | Behavior                |
| ------------- | -------------------------- | ---------------------------- | ---------------- | ----------------------- |
| `install.sh`  | `Linux`                    | `x86_64`                     | `linux-amd64`    | Proceed with download   |
| `install.sh`  | `Darwin`                   | `arm64`                      | `darwin-arm64`   | Proceed with download   |
| `install.ps1` | (Windows, implicit)        | `AMD64`                      | `windows-amd64`  | Proceed with download   |
| `install.sh`  | `Linux`                    | `aarch64`                    | `linux-arm64`    | Fail-fast with error    |
| `install.sh`  | `Darwin`                   | `x86_64`                     | `darwin-amd64`   | Fail-fast with error    |
| `install.ps1` | (Windows, implicit)        | `ARM64`                      | `windows-arm64`  | Fail-fast with error    |
| `install.sh`  | `FreeBSD`                  | `amd64`                      | (unsupported OS) | Fail-fast with error    |

---
### Requirement: Version selection via VERSION environment variable

Each install script SHALL honor a `VERSION` environment variable that selects which release to install. The variable is read as `${VERSION:-latest}` in `install.sh` and as `$env:VERSION` (defaulting to `latest` when unset or empty) in `install.ps1`. When `VERSION` is unset or equals the literal string `latest`, the script SHALL download from `https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/latest/download/<asset>`. When `VERSION` is set to any other non-empty value, the script SHALL download from `https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/download/${VERSION}/<asset>`. The `<asset>` name is `rancher-kubeconfig-updater-${OS}-${ARCH}` for Unix and `rancher-kubeconfig-updater-windows-amd64.exe` for Windows. If the HTTP response indicates a 4xx or 5xx status, the script MUST exit non-zero, print an error containing the failed URL, and MUST NOT leave a partial file at the installation target.

#### Scenario: Pinning to a specific release tag on Unix

- **WHEN** a user invokes the install pipeline with the environment `VERSION=v1.4.0` set before the piped `sh` invocation
- **THEN** the script constructs the download URL `https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/download/v1.4.0/rancher-kubeconfig-updater-${OS}-${ARCH}` and installs that exact version

#### Scenario: Pinning to a specific release tag on Windows

- **WHEN** a user sets `$env:VERSION='v1.4.0'` in PowerShell before running `irm ... | iex`
- **THEN** the script constructs the download URL `https://github.com/chenwei791129/rancher-kubeconfig-updater/releases/download/v1.4.0/rancher-kubeconfig-updater-windows-amd64.exe` and installs that exact version

#### Scenario: Non-existent version

- **WHEN** the user sets `VERSION` to a tag that does not exist on GitHub
- **THEN** the script exits non-zero, the error message contains the constructed download URL and the HTTP error code, and no file is placed at the installation target

---
### Requirement: Installation directory via INSTALL_DIR environment variable

Each install script SHALL honor an `INSTALL_DIR` environment variable that selects the target directory for the installed binary. The platform-specific defaults are:

- `install.sh`: `$HOME/.local/bin`. The installed binary name SHALL be `rancher-kubeconfig-updater`.
- `install.ps1`: `$env:USERPROFILE\.local\bin`. The installed binary name SHALL be `rancher-kubeconfig-updater.exe`.

**Default-value behaviour (`INSTALL_DIR` resolves to the platform default):**

The script SHALL create the directory if it does not yet exist (`mkdir -p` on Unix; `New-Item -ItemType Directory -Force` on Windows). If creation fails the script MUST exit non-zero with an error naming the directory and MUST NOT escalate privileges. If the directory is not writable even after creation, the script MUST exit non-zero with an error naming the directory; on Unix MUST NOT use `sudo`, on Windows MUST NOT attempt UAC elevation. Additionally on Windows, when the resolved default directory is not already present in the User-scope `PATH` (read via `[Environment]::GetEnvironmentVariable('PATH', 'User')`), the script SHALL append it via `[Environment]::SetEnvironmentVariable('PATH', $updated, 'User')` and SHALL print a notice including instructions for the user to restart their shell; the current session's `$env:PATH` SHALL NOT be modified.

**Non-default-value behaviour (`INSTALL_DIR` set to anything other than the platform default):**

The script MUST NOT create the directory automatically. If the directory does not exist the script MUST exit non-zero with an error naming the path; MUST NOT invoke `mkdir` / `New-Item`; MUST NOT escalate privileges. If the directory exists but is not writable by the current user:

- Unix (`install.sh`): the script MAY use `sudo mv` to move the binary into place, and MUST print a notice before invoking `sudo`.
- Windows (`install.ps1`): the script MUST exit non-zero with an error that names the unwritable path and instructs the user to re-run the install pipeline from an elevated PowerShell session; the script MUST NOT attempt UAC elevation via `Start-Process -Verb RunAs` or any equivalent mechanism, and MUST NOT modify the User-scope `PATH` for non-default install directories.

**Environment guards:**

- `install.sh`: `$HOME` MUST be resolved from the environment when `INSTALL_DIR` is unset; if `$HOME` is unset or empty the script MUST exit non-zero with an error before downloading.
- `install.ps1`: `$env:USERPROFILE` MUST be resolved from the environment when `INSTALL_DIR` is unset; if it is unset or empty the script MUST exit non-zero with an error before downloading.

#### Scenario: Install into default directory that does not exist (Unix)

- **WHEN** a user runs the install pipeline with `INSTALL_DIR` unset, `$HOME` set to a writable home directory, and `$HOME/.local/bin` does not exist
- **THEN** the script creates `$HOME/.local/bin` with `mkdir -p`, places the binary at `$HOME/.local/bin/rancher-kubeconfig-updater`, does NOT invoke `sudo`, and prints a confirmation line referencing the created path

#### Scenario: Install into pre-existing default directory (Unix)

- **WHEN** a user runs the install pipeline with `INSTALL_DIR` unset and `$HOME/.local/bin` already exists and is writable
- **THEN** the script places the binary at `$HOME/.local/bin/rancher-kubeconfig-updater`, leaves the directory contents unchanged (any `mkdir -p` call on an existing path is a no-op), does NOT invoke `sudo`, and prints a confirmation line referencing the chosen path

#### Scenario: Install into a user-writable non-default directory (Unix)

- **WHEN** a user runs `install.sh` with `INSTALL_DIR=$HOME/bin` and `$HOME/bin` exists and is writable
- **THEN** the script places the binary at `$HOME/bin/rancher-kubeconfig-updater`, does NOT invoke `sudo`, and prints a confirmation line referencing the chosen path

#### Scenario: Explicit /usr/local/bin requires elevation (Unix)

- **WHEN** a user runs `install.sh` with `INSTALL_DIR=/usr/local/bin` and `/usr/local/bin` exists but is not writable by the current user
- **THEN** the script prints a notice that `sudo` will be used, invokes `sudo mv` to place the binary at `/usr/local/bin/rancher-kubeconfig-updater`, and exits zero on success

#### Scenario: Non-default directory does not exist (Unix)

- **WHEN** a user sets `INSTALL_DIR=/opt/bin` and `/opt/bin` does not exist on the host
- **THEN** the script exits non-zero with an error naming `/opt/bin`, MUST NOT invoke `mkdir`, MUST NOT invoke `sudo`, and MUST NOT leave a partial file at the target path

#### Scenario: HOME is unset (Unix)

- **WHEN** a user runs `install.sh` with `INSTALL_DIR` unset and `$HOME` unset or empty
- **THEN** the script exits non-zero with an error message naming the missing `HOME` environment variable, MUST NOT issue any network request, and MUST NOT invoke `sudo`

#### Scenario: Install into default directory that does not exist (Windows)

- **WHEN** a user runs `irm ... | iex` with `$env:INSTALL_DIR` unset, `$env:USERPROFILE` set to a writable profile path, and `$env:USERPROFILE\.local\bin` does not exist
- **THEN** the script creates `$env:USERPROFILE\.local\bin`, places the binary at `$env:USERPROFILE\.local\bin\rancher-kubeconfig-updater.exe`, does NOT attempt UAC elevation, appends the directory to the User-scope `PATH` if it is not already there, and prints a confirmation line referencing the created path and a notice that the new `PATH` only applies to newly opened shells

#### Scenario: Default directory already on user PATH (Windows)

- **WHEN** a user runs `irm ... | iex` with `$env:INSTALL_DIR` unset, `$env:USERPROFILE\.local\bin` already on the User-scope `PATH`, and the directory writable
- **THEN** the script places the binary at `$env:USERPROFILE\.local\bin\rancher-kubeconfig-updater.exe`, does NOT modify the User-scope `PATH`, does NOT attempt UAC elevation, and the confirmation line MUST NOT contain a PATH-modification notice

#### Scenario: Install into a user-writable non-default directory (Windows)

- **WHEN** a user sets `$env:INSTALL_DIR='C:\tools'` where `C:\tools` exists and is user-writable
- **THEN** the script places the binary at `C:\tools\rancher-kubeconfig-updater.exe`, does NOT modify the User-scope `PATH`, does NOT attempt UAC elevation, and prints a confirmation line referencing the chosen path

#### Scenario: Non-default directory requires admin elevation (Windows)

- **WHEN** a user sets `$env:INSTALL_DIR='C:\Program Files\rancher-kubeconfig-updater'` from a non-elevated PowerShell session and the directory exists but is not writable to the current user
- **THEN** the script exits non-zero with an error message that names the unwritable path and instructs the user to re-run the same command from an elevated PowerShell session, MUST NOT attempt UAC elevation, MUST NOT modify the User-scope `PATH`, and MUST NOT leave a partial file at the target path

#### Scenario: Non-default directory does not exist (Windows)

- **WHEN** a user sets `$env:INSTALL_DIR='C:\does-not-exist'` and the directory does not exist on the host
- **THEN** the script exits non-zero with an error naming the path, MUST NOT invoke `New-Item`, MUST NOT attempt UAC elevation, MUST NOT modify the User-scope `PATH`, and MUST NOT leave a partial file at the target path

#### Scenario: USERPROFILE is unset (Windows)

- **WHEN** a user runs `install.ps1` with `$env:INSTALL_DIR` unset and `$env:USERPROFILE` unset or empty
- **THEN** the script exits non-zero with an error message naming the missing `USERPROFILE` environment variable, MUST NOT issue any network request, and MUST NOT attempt UAC elevation

##### Example: INSTALL_DIR decision matrix

| Platform | `INSTALL_DIR` setting                       | Directory state                  | PATH state                       | Behavior                                                                       |
| -------- | ------------------------------------------- | -------------------------------- | -------------------------------- | ------------------------------------------------------------------------------ |
| Unix     | unset (default)                             | exists, writable                 | n/a                              | `mv` to default, no sudo                                                       |
| Unix     | unset (default)                             | does not exist                   | n/a                              | `mkdir -p`, then `mv`, no sudo                                                 |
| Unix     | unset (default)                             | exists, not writable             | n/a                              | exit non-zero, no sudo                                                         |
| Unix     | unset, `$HOME` unset                        | n/a                              | n/a                              | exit non-zero before any network                                               |
| Unix     | `$HOME/bin`                                 | exists, writable                 | n/a                              | `mv`, no sudo                                                                  |
| Unix     | `$HOME/bin`                                 | does not exist                   | n/a                              | exit non-zero, no `mkdir`, no sudo                                             |
| Unix     | `/usr/local/bin`                            | exists, not writable             | n/a                              | `sudo mv` with notice                                                          |
| Unix     | `/opt/bin`                                  | exists, not writable             | n/a                              | `sudo mv` with notice                                                          |
| Unix     | `/opt/bin`                                  | does not exist                   | n/a                              | exit non-zero, no `mkdir`, no sudo                                             |
| Windows  | unset (default)                             | exists, writable                 | already on User PATH             | move file, no PATH change, no notice                                           |
| Windows  | unset (default)                             | exists, writable                 | NOT on User PATH                 | move file, append to User PATH, print restart-shell notice                     |
| Windows  | unset (default)                             | does not exist                   | n/a                              | create directory, move file, append to User PATH if needed, print notice       |
| Windows  | unset (default)                             | exists, not writable             | n/a                              | exit non-zero, no UAC                                                          |
| Windows  | unset, `$env:USERPROFILE` unset             | n/a                              | n/a                              | exit non-zero before any network                                               |
| Windows  | `C:\tools` (existing, user-writable)        | exists, writable                 | n/a                              | move file, no PATH change                                                      |
| Windows  | `C:\does-not-exist`                         | does not exist                   | n/a                              | exit non-zero, no `New-Item`, no UAC                                           |
| Windows  | `C:\Program Files\...` (non-elevated)       | exists, not writable             | n/a                              | exit non-zero, error directs to elevated PowerShell, no UAC, no PATH change    |

---
### Requirement: README installation section uses the script

The `README.md` `Installation` section SHALL present:

- The Linux / macOS install path as a single `curl ... | sh` command pointing at `https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.sh`, naming the supported Unix platforms (`linux-amd64`, `darwin-arm64`) and documenting `VERSION` and `INSTALL_DIR` with `sh` syntax (`VERSION=...`, `INSTALL_DIR=...`).
- The Windows install path as a single `irm ... | iex` command pointing at `https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.ps1`, naming the supported Windows platform (`windows-amd64`), documenting `VERSION` and `INSTALL_DIR` with PowerShell syntax (`$env:VERSION='...'`, `$env:INSTALL_DIR='...'`), and including a system-wide install example that uses an elevated PowerShell session and sets `$env:INSTALL_DIR` to a path under `C:\Program Files\`.
- A `Building from Source` subsection covering all unsupported platforms (`linux-arm64`, `darwin-amd64`, `windows-arm64`, and any unrecognised host).

#### Scenario: Reader installs on supported Unix platform from README

- **WHEN** a reader copies the single-line `curl ... | sh` command from the Linux / macOS subsection on a `linux-amd64` or `darwin-arm64` host
- **THEN** the command completes installation without further README steps, matching the behavior defined in the One-line installation entry point requirement

#### Scenario: Reader installs on Windows from README

- **WHEN** a reader copies the single-line `irm ... | iex` command from the Windows subsection on a `windows-amd64` host
- **THEN** the command completes installation without further README steps, matching the Default install on Windows scenario defined in the One-line installation entry point requirement

#### Scenario: Reader on unsupported platform consults README

- **WHEN** a reader on `linux-arm64`, `darwin-amd64`, or `windows-arm64` runs the install command for their platform and sees the fail-fast error pointing at Building from Source
- **THEN** the README contains a Building from Source subsection with `git clone` and `go build` instructions sufficient to produce a working binary locally for that platform

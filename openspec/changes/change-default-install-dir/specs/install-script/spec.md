## MODIFIED Requirements

### Requirement: One-line installation entry point

The repository SHALL provide an `install.sh` script at the project root that installs the `rancher-kubeconfig-updater` binary on Linux and macOS via a single `curl ... | sh` invocation. The script SHALL be POSIX `sh` compatible and MUST NOT require `bash`-only features.

#### Scenario: Default install on supported platform

- **WHEN** a user runs `curl -fsSL https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.sh | sh` on a `linux-amd64` or `darwin-arm64` host with `curl` available, `$HOME` set, and no `INSTALL_DIR` override
- **THEN** the script downloads the latest release binary, makes it executable, places it at `$HOME/.local/bin/rancher-kubeconfig-updater` (creating that directory if it does not already exist), and prints a confirmation line containing the installed version and target path

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

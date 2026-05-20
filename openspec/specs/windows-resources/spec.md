# windows-resources Specification

## Purpose

Defines the Windows-specific resources embedded into the `rancher-kubeconfig-updater.exe`
binary — namely the application manifest that declares `requestedExecutionLevel="asInvoker"`
(suppressing the Windows Installer Detection heuristic that would otherwise trigger UAC
on the `updater` filename) and the version info fields surfaced through Windows File
Properties → Details (ProductName, FileDescription, FileVersion, ProductVersion,
CompanyName, LegalCopyright).

The requirements below cover the resource generation source (`versioninfo.json` +
`app.manifest`, both committed), the local and CI build-time generation flow (Makefile
`go generate` + release-please.yml Windows-only step that injects the release tag), and
the cross-platform isolation rules that keep Linux and macOS artefacts byte-identical
through the Go linker's `_windows_amd64.syso` filename-suffix mechanism.

## Requirements

### Requirement: Application manifest suppresses Installer Detection

The `rancher-kubeconfig-updater.exe` Windows binary SHALL embed an application manifest in its PE resource section that declares `<requestedExecutionLevel level="asInvoker" uiAccess="false"/>` so that Windows' Installer Detection heuristic does not trigger UAC elevation for the binary based on its filename. The manifest SHALL be the minimum required to suppress Installer Detection — it MUST NOT include OS compatibility (`<compatibility>`), DPI awareness (`<dpiAware>`), long-path awareness, or any other element that does not directly relate to the privilege requirement.

#### Scenario: Launching the binary from a standard user shell does not prompt for UAC

- **WHEN** a non-administrator user runs `rancher-kubeconfig-updater.exe --help` from an unelevated `cmd.exe` or PowerShell session on Windows 10 or Windows 11
- **THEN** Windows does not display a UAC prompt, the process runs with the caller's standard-user token, stdout and stderr stream back to the calling console, and the process exits with code 0 after printing the usage text

#### Scenario: Filename keyword no longer triggers Installer Detection

- **WHEN** the embedded manifest is inspected via `sigcheck -m rancher-kubeconfig-updater.exe` or `mt.exe -inputresource:rancher-kubeconfig-updater.exe;#1 -out:stdout`
- **THEN** the output contains a `<trustInfo>` block with `<requestedExecutionLevel level="asInvoker" uiAccess="false"/>`, and Windows treats the binary as a standard application despite the `updater` suffix in its filename

---
### Requirement: Windows version info populated from release tag

The Windows binary SHALL embed Windows VS_VERSIONINFO resource fields (`FixedFileInfo.FileVersion`, `FixedFileInfo.ProductVersion`, `StringFileInfo.ProductName`, `StringFileInfo.FileDescription`, `StringFileInfo.CompanyName`, `StringFileInfo.LegalCopyright`) so that Windows File Explorer's `Properties → Details` tab displays accurate metadata. The numeric `FileVersion` and `ProductVersion` fields SHALL be set from the release tag when one exists (a semver tag `vX.Y.Z` maps to the four-part Windows version `X.Y.Z.0`); when no release tag is available (e.g., a local developer build), the version fields SHALL default to `0.0.0.0` from the static configuration file.

#### Scenario: Released binary shows the release tag in File Properties

- **WHEN** the binary built from CI for release tag `v1.4.0` is right-clicked in Windows Explorer and `Properties → Details` is opened
- **THEN** the `Product version` and `File version` rows display `1.4.0.0`, the `Product name` row displays `rancher-kubeconfig-updater`, and the `File description` row displays a non-empty short description sourced from the project's CLI Long description

#### Scenario: Local developer build falls back to placeholder version

- **WHEN** a developer runs `make build` on their workstation with no release tag in scope and then inspects the produced `rancher-kubeconfig-updater.exe` via File Properties on Windows
- **THEN** `Product version` and `File version` display `0.0.0.0`, the string fields (ProductName / FileDescription / CompanyName / LegalCopyright) still display their statically configured values, and the binary still runs without UAC

##### Example: release tag to version field mapping

| Release tag | FileVersion / ProductVersion | Source                          |
| ----------- | ---------------------------- | ------------------------------- |
| `v1.4.0`    | `1.4.0.0`                    | CI parsed tag, overrides JSON   |
| `v2.0.0`    | `2.0.0.0`                    | CI parsed tag, overrides JSON   |
| `v0.9.1`    | `0.9.1.0`                    | CI parsed tag, overrides JSON   |
| (no tag, local build) | `0.0.0.0`           | `versioninfo.json` placeholder  |

---
### Requirement: Resource embedding is Windows-amd64 only and source-controlled

The repository SHALL keep a single source-of-truth resource configuration file `versioninfo.json` at the project root, committed to git, that drives the `goversioninfo` tool. The generated PE resource artefact SHALL be named with a Go platform suffix (`resource_windows_amd64.syso`) so that the Go linker auto-links it only on `GOOS=windows GOARCH=amd64` builds. The generated artefact MUST NOT be committed to git (it is a build product); the repository's `.gitignore` SHALL exclude `*.syso`. Linux and macOS release artefacts MUST be byte-identical to their pre-change counterparts (no PE resource section exists on those platforms, and Go silently ignores the `.syso` due to its filename suffix).

#### Scenario: Linux and macOS artefacts are unaffected

- **WHEN** the release-please workflow runs the matrix build for `goos=linux goarch=amd64` and for `goos=darwin goarch=arm64`
- **THEN** the produced `rancher-kubeconfig-updater-linux-amd64` and `rancher-kubeconfig-updater-darwin-arm64` artefacts are byte-identical to the artefacts produced from the same source tree without this change (verified by hash comparison against a pre-change build of the same commit minus the `versioninfo.json` / `main.go` `//go:generate` directive)

#### Scenario: Generated artefact is not committed

- **WHEN** a contributor runs `go generate ./...` locally to refresh the resource artefact and then runs `git status`
- **THEN** `resource_windows_amd64.syso` (and any other `*.syso` file) does not appear in the status output, because `.gitignore` excludes that suffix

---
### Requirement: Resource generation invoked from build entry points

The local `make build` target and the CI `release-please.yml` Windows matrix build SHALL invoke `goversioninfo` via the project's `go tool` mechanism (added through `go get -tool github.com/josephspurrier/goversioninfo/cmd/goversioninfo`) before invoking `go build`. The local invocation uses the `//go:generate` directive in `main.go` (executed by `go generate ./...`) and produces the artefact with `versioninfo.json`'s static placeholder version. The CI invocation parses the release tag (semver `vX.Y.Z`) into integer major / minor / patch components and passes them as `-ver-major=X -ver-minor=Y -ver-patch=Z` flags to `goversioninfo`, overriding the JSON placeholder so the released binary advertises its real version. On non-Windows matrix rows the resource generation step is skipped (the `.syso` is unused there).

#### Scenario: Local make build embeds placeholder version

- **WHEN** a developer with the `goversioninfo` tool resolved (via `go mod download` of `go.mod` tool deps) runs `make build` on a Windows host
- **THEN** `go generate ./...` runs first, `resource_windows_amd64.syso` is produced from `versioninfo.json` with `0.0.0.0` versions, and `go build .` then links it into the binary; the resulting `rancher-kubeconfig-updater.exe` exhibits the asInvoker manifest behaviour and the placeholder version in File Properties

#### Scenario: CI build injects release tag into version fields

- **WHEN** release-please publishes tag `v1.4.0` and the `build-and-upload` job runs the `goos=windows goarch=amd64` matrix row
- **THEN** the workflow parses `v1.4.0` into `1`, `4`, `0`, invokes `go tool goversioninfo -platform-specific -ver-major=1 -ver-minor=4 -ver-patch=0 versioninfo.json` to produce `resource_windows_amd64.syso`, and the subsequent `go build` links a binary whose File Properties show `1.4.0.0` for both FileVersion and ProductVersion

#### Scenario: Non-Windows CI rows skip generation

- **WHEN** the `goos=linux goarch=amd64` or `goos=darwin goarch=arm64` matrix row runs
- **THEN** the workflow does NOT invoke `goversioninfo`, no `.syso` file is produced on the runner, and the resulting binary is byte-identical to its pre-change counterpart

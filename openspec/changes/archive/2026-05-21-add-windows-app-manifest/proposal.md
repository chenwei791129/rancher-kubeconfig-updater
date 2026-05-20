## Why

Launching the `rancher-kubeconfig-updater.exe` binary on Windows triggers a UAC elevation prompt every time, because Windows' [Installer Detection](https://learn.microsoft.com/en-us/windows/security/identity-protection/user-account-control/how-it-works#installer-detection) heuristic flags executables whose filename contains keywords like `install`, `setup`, `update`, or `updater` as installers — and our binary name ends in **updater**. After the user grants UAC, Windows runs the binary in a new elevated console window that closes when the process exits, manifesting to the user as "the program crashes immediately on launch".

The fix is to embed an application manifest declaring `requestedExecutionLevel="asInvoker"`, which suppresses the Installer Detection heuristic. With the manifest in place the binary inherits the caller's privilege token (no UAC), runs in the existing console, and behaves identically to its Linux / macOS siblings.

We will also embed minimal Windows version info (`ProductName`, `FileDescription`, `FileVersion`, `ProductVersion`, `CompanyName`, `LegalCopyright`) so that `File Properties → Details` in Windows Explorer shows the real release tag instead of being empty. This costs nothing extra because the same `goversioninfo` invocation that emits the manifest also emits version info.

## What Changes

- Add `josephspurrier/goversioninfo` to `go.mod` tool dependencies via `go get -tool`, so it is invoked through `go tool goversioninfo` per the project's existing tool convention.
- New `versioninfo.json` at the repo root: declares the minimal manifest XML (`requestedExecutionLevel="asInvoker" uiAccess="false"`) and the static version info string fields (ProductName, FileDescription, CompanyName, LegalCopyright). FileVersion / ProductVersion placeholders are `0.0.0.0` and are overridden at build time when a release tag is available.
- New `//go:generate go tool goversioninfo -platform-specific versioninfo.json` directive in `main.go`. The directive produces `resource_windows_amd64.syso` at the package root; Go's linker auto-picks files whose name ends in `_windows_amd64.syso` only on `GOOS=windows GOARCH=amd64` builds, so Linux and macOS artefacts stay byte-identical.
- `.gitignore` adds `*.syso` so the generated artefact is not committed.
- `Makefile` `build` target runs `go generate ./...` before `go build .`, ensuring `make build` on any developer machine produces a fresh `.syso` (when on Windows) or a no-op `.syso` that gets ignored at link time (when on other platforms).
- `.github/workflows/release-please.yml`: in the `build-and-upload` job, before invoking `go build`, run `go tool goversioninfo -platform-specific -ver-major=<X> -ver-minor=<Y> -ver-patch=<Z> versioninfo.json` with the major/minor/patch parsed from the release tag name (`v1.4.0` → 1, 4, 0). The build step is unchanged otherwise — the produced `.syso` is picked up automatically. On non-Windows matrix rows the generate step is skipped (it only matters for `goos: windows`).
- `README.md` Building from Source subsection adds a one-liner noting that local `go build` on Windows requires a prior `go generate ./...` (or just `make build`) to embed the manifest. Linux / macOS instructions unchanged.

## Non-Goals

- Embedding an application icon (this is a CLI tool with no GUI window — File Explorer would only show the icon in a thumbnail context that the user does not interact with).
- Code signing the `.exe` (no Authenticode certificate available; out of scope and a separate change with significant CI / secrets cost).
- Resource embedding for Linux or macOS binaries (those platforms do not have a PE resource section; not applicable).
- Supporting `GOOS=windows GOARCH=arm64` — the build matrix only ships `windows-amd64`; if `windows-arm64` is added later, this change must be revisited to produce a second `resource_windows_arm64.syso`.
- Adding compatibility / DPI-awareness / longPathAware elements to the manifest (irrelevant for a CLI tool).
- Pre-commit hook auto-running `go generate` (already covered by `make build` running it; adding a hook is separate hygiene work).

## Alternatives Considered

- **`akavel/rsrc`** instead of `goversioninfo`: rejected. Less actively maintained, supports only manifest + icon (no version info), and project would still need a separate tool or hand-written struct for version info fields.
- **Hand-written `.syso`** (raw PE resource section bytes): rejected. Requires understanding the PE resource format; no maintainability win.
- **Renaming the binary** to avoid the `updater` suffix: rejected. The name is the project name; renaming the artefact diverges binary name from project name and breaks any user / docs that already reference `rancher-kubeconfig-updater`.
- **Committing the generated `.syso`**: rejected (per discuss conclusion). Binary derivative artefacts do not belong in source control; CI / Makefile invoking `go generate` keeps the source-of-truth single (`versioninfo.json`).
- **Static `0.0.0.0` for FileVersion** in CI as well: rejected. Dynamic injection from release tag costs ~5 lines of shell parsing in the workflow and gives Windows users an accurate File Properties view; the marginal complexity is justified.

## Capabilities

### New Capabilities

- `windows-resources`: defines the Windows-specific resources embedded into the `rancher-kubeconfig-updater.exe` binary — namely the application manifest (suppressing UAC Installer Detection by declaring `asInvoker`) and the static / dynamic version info fields surfaced through Windows File Properties. Covers the resource generation source (`versioninfo.json`), the local and CI build-time generation flow, and the cross-platform isolation rules (no impact on Linux / macOS artefacts).

### Modified Capabilities

(none)

## Impact

- Affected specs: new `windows-resources` capability.
- Affected code:
  - New:
    - `versioninfo.json`
  - Modified:
    - `go.mod` (add `tool` entry for `goversioninfo`)
    - `go.sum` (transitive updates)
    - `main.go` (add `//go:generate` directive)
    - `Makefile` (`build` target adds `go generate ./...`)
    - `.github/workflows/release-please.yml` (add tag parsing + goversioninfo invocation before `go build` for the Windows matrix row)
    - `.gitignore` (add `*.syso`)
    - `README.md` (Building from Source note about `go generate` on Windows)
  - Removed:
    - (none)
- Affected build flow: Windows release artefact size grows by a few KB (resource section). Linux / macOS artefacts are unchanged.
- Affected runtime: `rancher-kubeconfig-updater.exe` no longer triggers UAC; Windows File Properties → Details displays ProductName / FileDescription / FileVersion / ProductVersion / CompanyName / LegalCopyright.
- Affected change ordering: depends on `install-script` capability already existing on `main` (PR #84 + #86 already merged). No cross-change conflicts.

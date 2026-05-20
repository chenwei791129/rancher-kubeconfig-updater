<#
.SYNOPSIS
    Installer for rancher-kubeconfig-updater on Windows.

.DESCRIPTION
    Downloads and installs the rancher-kubeconfig-updater binary into a user-writable
    directory. Mirrors the contract of install.sh on Linux and macOS: same VERSION
    and INSTALL_DIR environment variable model, same fail-fast platform allowlist.

    Write-Host is used intentionally for installer console output (colored status
    lines and notices) instead of Write-Information or pipeline output. Unlike
    install.sh, which writes those messages to stderr, this script directs them
    to the PowerShell host stream (the idiomatic Windows convention).

    Usage:
        irm https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.ps1 | iex

    Environment variables:
        VERSION       Release tag (default: latest)
        INSTALL_DIR   Target directory (default: $env:USERPROFILE\.local\bin)

    Supported platforms: windows-amd64. Other architectures must build from source.

.NOTES
    Internal test hooks (intentionally undocumented for end users):
        _ARCH_OVERRIDE       Override architecture detection (used by automated tests).
        _USER_PATH_OVERRIDE  Override the User-scope PATH read for the auto-PATH branch
                             (used by automated tests on non-Windows hosts where the
                             User scope is a silent no-op). Set to the literal string
                             '__EMPTY__' to simulate an empty user PATH.
#>

#Requires -Version 5.1

Set-StrictMode -Version 3.0
$ErrorActionPreference = 'Stop'

try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
} catch {
    # PowerShell Core on non-Windows hosts may not expose ServicePointManager
    # in the same shape. The script's platform allowlist will fail-fast there
    # before any real HTTP call, so this is safe to swallow.
    $null = $_
}

$Repo = 'chenwei791129/rancher-kubeconfig-updater'
$BinaryName = 'rancher-kubeconfig-updater.exe'
$BuildFromSourceUrl = "https://github.com/$Repo#building-from-source"

function Write-Info {
    [Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSAvoidUsingWriteHost', '', Justification='Installer console output')]
    param([string]$Message)
    Write-Host $Message -ForegroundColor Green
}

function Write-Notice {
    [Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSAvoidUsingWriteHost', '', Justification='Installer console output')]
    param([string]$Message)
    Write-Host $Message -ForegroundColor Yellow
}

function Write-Err {
    param([string]$Message)
    [Console]::Error.WriteLine("error: $Message")
}

# Extract the `Location` header from a response Headers object across PS
# versions. PS 5.1 surfaces WebHeaderCollection with a string indexer; PS 7+
# surfaces HttpResponseHeaders with a typed `Location` (System.Uri) property
# and no string indexer (it returns $null silently). Try both shapes.
function Get-LocationHeader {
    param($headers)
    if (-not $headers) { return $null }
    try {
        if ($headers.Location) { return $headers.Location.ToString() }
    } catch {
        $null = $_
    }
    try {
        return $headers['Location']
    } catch {
        $null = $_
        return $null
    }
}

# Architecture detection.
# Keep the allowlist in sync with the build matrix in
# .github/workflows/release-please.yml - release artefacts only exist for
# windows-amd64 (in addition to linux-amd64 and darwin-arm64 covered by install.sh).
$rawArch = if ($env:_ARCH_OVERRIDE) {
    $env:_ARCH_OVERRIDE
} elseif ($env:PROCESSOR_ARCHITEW6432) {
    # Running under WOW64; the real architecture is reported here.
    $env:PROCESSOR_ARCHITEW6432
} else {
    $env:PROCESSOR_ARCHITECTURE
}

$arch = switch -Regex ($rawArch) {
    # Production `$env:PROCESSOR_ARCHITECTURE` reports `AMD64` or `ARM64` on
    # Windows; `x86_64` / `aarch64` are only reached via `_ARCH_OVERRIDE` in
    # cross-platform tests that reuse the install.sh-style arch aliases.
    '^(AMD64|x86_64)$' { 'amd64'; break }
    '^(ARM64|aarch64)$' { 'arm64'; break }
    default { '' }
}

if (-not $arch) {
    Write-Err "unsupported architecture: $rawArch. See $BuildFromSourceUrl to build from source."
    exit 1
}

$platform = "windows-$arch"

if ($platform -ne 'windows-amd64') {
    Write-Err "platform $platform is not in the prebuilt release matrix. See $BuildFromSourceUrl to build from source."
    exit 1
}

# Resolve VERSION.
if ($env:VERSION) {
    $Version = $env:VERSION
} else {
    $Version = 'latest'
}

# Resolve INSTALL_DIR. Only depend on USERPROFILE when defaulting.
if ($env:INSTALL_DIR) {
    $InstallDir = $env:INSTALL_DIR
    if ($env:USERPROFILE) {
        $DefaultInstallDir = (Join-Path $env:USERPROFILE '.local\bin').TrimEnd('\', '/')
    } else {
        $DefaultInstallDir = ''
    }
} else {
    if (-not $env:USERPROFILE) {
        Write-Err "USERPROFILE environment variable is unset or empty; set USERPROFILE or pass an explicit INSTALL_DIR."
        exit 1
    }
    $DefaultInstallDir = (Join-Path $env:USERPROFILE '.local\bin').TrimEnd('\', '/')
    $InstallDir = $DefaultInstallDir
}

$InstallDir = $InstallDir.TrimEnd('\', '/')
if (-not $InstallDir) {
    $InstallDir = '\'
}

$IsDefault = ($InstallDir -ieq $DefaultInstallDir)

# Validate INSTALL_DIR against the requirement contract:
#   - Default value: auto-create with New-Item if missing; never elevate.
#   - Non-default value: caller is responsible for the path. Fail-fast if it
#     does not exist; if it exists but is not writable, exit non-zero with
#     guidance to re-run from an elevated session (no UAC auto-elevation).
if ($IsDefault) {
    if (-not (Test-Path -LiteralPath $InstallDir -PathType Container)) {
        try {
            New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
        } catch {
            Write-Err "failed to create default install directory ${InstallDir}: $($_.Exception.Message)"
            exit 1
        }
    }
    $probe = Join-Path $InstallDir ('.write-probe-' + [Guid]::NewGuid().ToString('N'))
    try {
        Set-Content -LiteralPath $probe -Value '' -ErrorAction Stop
        Remove-Item -LiteralPath $probe -Force -ErrorAction SilentlyContinue
    } catch {
        Write-Err "default install directory $InstallDir is not writable"
        exit 1
    }
} else {
    if (-not (Test-Path -LiteralPath $InstallDir -PathType Container)) {
        Write-Err "INSTALL_DIR=$InstallDir does not exist or is not a directory"
        exit 1
    }
    # Probe writability before downloading so unwritable non-default paths
    # (typically system locations like `C:\Program Files\`) fail fast with
    # an actionable message instead of wasting a multi-MB download.
    $probe = Join-Path $InstallDir ('.write-probe-' + [Guid]::NewGuid().ToString('N'))
    try {
        Set-Content -LiteralPath $probe -Value '' -ErrorAction Stop
        Remove-Item -LiteralPath $probe -Force -ErrorAction SilentlyContinue
    } catch {
        Write-Err "INSTALL_DIR=$InstallDir is not writable ($($_.Exception.Message)); re-run the install pipeline from an elevated PowerShell session, or move the binary manually."
        exit 1
    }
}

$Asset = "rancher-kubeconfig-updater-$platform.exe"
if ($Version -eq 'latest') {
    $Url = "https://github.com/$Repo/releases/latest/download/$Asset"
} else {
    $Url = "https://github.com/$Repo/releases/download/$Version/$Asset"
}

# Resolve "latest" to a concrete release tag for the confirmation message.
# Fall back to the literal "latest" on any resolution failure - the actual
# download still uses the original $Url.
$DisplayVersion = $Version
if ($Version -eq 'latest') {
    $location = $null
    try {
        $headResp = Invoke-WebRequest -Uri $Url -Method Head -MaximumRedirection 0 -UseBasicParsing -ErrorAction Stop
        $location = Get-LocationHeader $headResp.Headers
    } catch {
        $location = Get-LocationHeader $_.Exception.Response.Headers
    }
    if ($location -is [array]) {
        $location = $location[0]
    }
    if ($location -and ($location -match '/releases/download/([^/]+)/')) {
        $DisplayVersion = $Matches[1]
    }
}

$tmpFile = [System.IO.Path]::Combine(
    [System.IO.Path]::GetTempPath(),
    [System.IO.Path]::GetRandomFileName() + '.exe'
)

try {
    Write-Info "Downloading $Asset ($DisplayVersion) from $Url"
    try {
        Invoke-WebRequest -Uri $Url -OutFile $tmpFile -UseBasicParsing -ErrorAction Stop | Out-Null
    } catch {
        $httpStatus = $null
        try {
            $httpStatus = [int]$_.Exception.Response.StatusCode
        } catch {
            $null = $_
        }
        if ($httpStatus) {
            Write-Err "failed to download $Url (HTTP $httpStatus)"
        } else {
            Write-Err "failed to download ${Url}: $($_.Exception.Message)"
        }
        exit 1
    }

    $Target = Join-Path $InstallDir $BinaryName

    try {
        Move-Item -LiteralPath $tmpFile -Destination $Target -Force -ErrorAction Stop
    } catch {
        if ($IsDefault) {
            Write-Err "failed to move binary to ${Target}: $($_.Exception.Message)"
        } else {
            Write-Err "INSTALL_DIR=$InstallDir is not writable ($($_.Exception.Message)); re-run the install pipeline from an elevated PowerShell session, or move the binary manually."
        }
        exit 1
    }

    Write-Info "Installed $BinaryName ($DisplayVersion) to $Target"

    # User PATH auto-write - only for the default install directory.
    # The current session's $env:PATH is intentionally NOT modified; the
    # registry write only affects newly spawned shells.
    if ($IsDefault) {
        $userPath = $null
        if ($env:_USER_PATH_OVERRIDE) {
            $userPath = if ($env:_USER_PATH_OVERRIDE -eq '__EMPTY__') { '' } else { $env:_USER_PATH_OVERRIDE }
        } else {
            try {
                $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
            } catch {
                # 'User' scope is Windows-only - silently skip on other platforms
                # (this branch is unreachable in production since the allowlist
                # gates non-Windows hosts).
                $null = $_
            }
        }
        $alreadyOnPath = $false
        if ($userPath) {
            foreach ($seg in $userPath.Split(';')) {
                if ($seg -and ($seg.TrimEnd('\', '/') -ieq $InstallDir)) {
                    $alreadyOnPath = $true
                    break
                }
            }
        }
        if (-not $alreadyOnPath) {
            $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
            try {
                [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
                Write-Notice "Added $InstallDir to your user PATH. Restart your shell (or sign out and back in) for the change to take effect."
            } catch {
                Write-Notice "Could not update user PATH automatically: $($_.Exception.Message). Add $InstallDir to your PATH manually."
            }
        }
    }
} finally {
    Remove-Item -LiteralPath $tmpFile -Force -ErrorAction SilentlyContinue
}

exit 0

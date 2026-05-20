#!/bin/sh
# install.sh — installer for rancher-kubeconfig-updater on Linux and macOS.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.sh | sh
#
# Environment variables:
#   VERSION       Release tag (default: latest)
#   INSTALL_DIR   Target directory (default: /usr/local/bin)
#
# Supported platforms: linux-amd64, darwin-arm64.
# Other platforms must build from source — see the README.
#
# Internal test hooks (intentionally undocumented for end users):
#   _OS_OVERRIDE   Override `uname -s` output (used by automated tests).
#   _ARCH_OVERRIDE Override `uname -m` output (used by automated tests).

set -eu

REPO="chenwei791129/rancher-kubeconfig-updater"
BINARY_NAME="rancher-kubeconfig-updater"
BUILD_FROM_SOURCE_URL="https://github.com/${REPO}#building-from-source"
DEFAULT_INSTALL_DIR="/usr/local/bin"

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-${DEFAULT_INSTALL_DIR}}"
INSTALL_DIR="${INSTALL_DIR%/}"
if [ -z "${INSTALL_DIR}" ]; then
    INSTALL_DIR="/"
fi

if [ -t 2 ] && [ -z "${NO_COLOR:-}" ]; then
    C_RED=$(printf '\033[31m')
    C_GREEN=$(printf '\033[32m')
    C_YELLOW=$(printf '\033[33m')
    C_RESET=$(printf '\033[0m')
else
    C_RED=""
    C_GREEN=""
    C_YELLOW=""
    C_RESET=""
fi

info() {
    printf '%s%s%s\n' "${C_GREEN}" "$*" "${C_RESET}" >&2
}

warn() {
    printf '%s%s%s\n' "${C_YELLOW}" "$*" "${C_RESET}" >&2
}

err() {
    printf '%serror: %s%s\n' "${C_RED}" "$*" "${C_RESET}" >&2
}

raw_os="${_OS_OVERRIDE:-$(uname -s)}"
raw_arch="${_ARCH_OVERRIDE:-$(uname -m)}"

case "${raw_os}" in
    Linux) os="linux" ;;
    Darwin) os="darwin" ;;
    *)
        err "unsupported operating system: ${raw_os}. See ${BUILD_FROM_SOURCE_URL} to build from source."
        exit 1
        ;;
esac

case "${raw_arch}" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
        err "unsupported architecture: ${raw_arch} on ${os}. See ${BUILD_FROM_SOURCE_URL} to build from source."
        exit 1
        ;;
esac

platform="${os}-${arch}"

# Keep this allowlist in sync with the build matrix in
# .github/workflows/release-please.yml — release artefacts only exist for
# these platforms.
case "${platform}" in
    linux-amd64|darwin-arm64) ;;
    *)
        err "platform ${platform} is not in the prebuilt release matrix. See ${BUILD_FROM_SOURCE_URL} to build from source."
        exit 1
        ;;
esac

if ! command -v curl > /dev/null 2>&1; then
    err "curl is required but not found. Install curl and retry."
    exit 1
fi

if [ "${INSTALL_DIR}" != "${DEFAULT_INSTALL_DIR}" ]; then
    if [ ! -d "${INSTALL_DIR}" ]; then
        err "INSTALL_DIR=${INSTALL_DIR} does not exist or is not a directory"
        exit 1
    fi
    if [ ! -w "${INSTALL_DIR}" ]; then
        err "INSTALL_DIR=${INSTALL_DIR} is not writable; rerun with a writable directory or move the binary manually."
        exit 1
    fi
fi

asset="${BINARY_NAME}-${platform}"
if [ "${VERSION}" = "latest" ]; then
    url="https://github.com/${REPO}/releases/latest/download/${asset}"
else
    url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
fi

tmpdir=""
trap '[ -n "${tmpdir}" ] && rm -rf "${tmpdir}"' EXIT INT TERM
tmpdir="$(mktemp -d)"
tmpfile="${tmpdir}/${BINARY_NAME}"

# Resolve "latest" to a concrete release tag so the confirmation message
# reports the actual installed version. Fall back to "latest" on any
# resolution failure — this is purely cosmetic, the download still uses the
# original "latest" URL.
display_version="${VERSION}"
if [ "${VERSION}" = "latest" ]; then
    redirect="$(curl -fsSI -o /dev/null -w '%{redirect_url}' "${url}" 2>/dev/null || true)"
    case "${redirect}" in
        */releases/download/*)
            display_version="$(printf '%s' "${redirect}" | sed -E 's|.*/releases/download/([^/]+)/.*|\1|')"
            ;;
    esac
fi

info "Downloading ${asset} (${display_version}) from ${url}"
if ! curl -fsSL -o "${tmpfile}" "${url}"; then
    err "failed to download ${url}"
    exit 1
fi

chmod +x "${tmpfile}"

target="${INSTALL_DIR}/${BINARY_NAME}"

# At this point any non-default INSTALL_DIR has already been validated as
# writable above, so this branch only ever fires for the default path.
if [ ! -w "${INSTALL_DIR}" ]; then
    warn "Elevation required to write to ${INSTALL_DIR}; using sudo."
    sudo mv "${tmpfile}" "${target}"
else
    mv "${tmpfile}" "${target}"
fi

info "Installed ${BINARY_NAME} (${display_version}) to ${target}"

case ":${PATH:-}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        warn "${INSTALL_DIR} is not in PATH. Add it to your shell profile to run ${BINARY_NAME} directly."
        ;;
esac

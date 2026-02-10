#!/bin/bash
set -euo pipefail

# AgentX installer — downloads the latest (or pinned) release binary.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/jefelabs/agentx/main/scripts/install.sh | bash
#
# Environment variables:
#   AGENTX_VERSION      Pin a specific version (e.g., "1.2.0"). Default: latest.
#   AGENTX_MIRROR       Override download base URL. Default: GitHub Releases.
#   AGENTX_INSTALL_DIR  Override install directory. Default: ~/.local/bin.

REPO="jefelabs/agentx"
DEFAULT_BASE_URL="https://github.com/${REPO}/releases/download"
BASE_URL="${AGENTX_MIRROR:-$DEFAULT_BASE_URL}"
INSTALL_DIR="${AGENTX_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${AGENTX_VERSION:-}"

# --- Detect platform ---

detect_os() {
  local os
  os="$(uname -s)"
  case "$os" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)       echo "Unsupported OS: $os" >&2; exit 1 ;;
  esac
}

detect_arch() {
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64)  echo "amd64" ;;
    amd64)   echo "amd64" ;;
    aarch64) echo "arm64" ;;
    arm64)   echo "arm64" ;;
    *)       echo "Unsupported architecture: $arch" >&2; exit 1 ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

# --- Resolve version ---

if [ -z "$VERSION" ]; then
  echo "Fetching latest release version..."
  if [ -n "${AGENTX_MIRROR:-}" ]; then
    # Enterprise mirror: resolve latest version from version.txt
    VERSION="$(curl -sSL --fail "${BASE_URL}/latest/version.txt" 2>/dev/null)" || {
      echo "Error: could not fetch latest version from ${BASE_URL}/latest/version.txt" >&2
      echo "Set AGENTX_VERSION to pin a specific version." >&2
      exit 1
    }
  else
    # GitHub: resolve latest version from the API
    VERSION="$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' \
      | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')"
  fi
  if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version." >&2
    exit 1
  fi
fi

# Strip leading "v" if present
VERSION="${VERSION#v}"

ARCHIVE="agentx_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="${BASE_URL}/v${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="${BASE_URL}/v${VERSION}/checksums.txt"

echo "Installing agentx v${VERSION} (${OS}/${ARCH})..."
echo "  From: ${DOWNLOAD_URL}"
echo "  To:   ${INSTALL_DIR}/agentx"

# --- Download ---

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

curl -sSL --fail -o "${TMP_DIR}/${ARCHIVE}" "$DOWNLOAD_URL" || {
  echo "Error: failed to download ${DOWNLOAD_URL}" >&2
  echo "Check that version v${VERSION} exists at https://github.com/${REPO}/releases" >&2
  exit 1
}

# --- Verify checksum (optional, best-effort) ---

verify_checksum() {
  local checksums_file="${TMP_DIR}/checksums.txt"
  if ! curl -sSL --fail -o "$checksums_file" "$CHECKSUMS_URL" 2>/dev/null; then
    echo "  Checksum file not available, skipping verification."
    return 0
  fi

  local expected
  expected="$(grep "${ARCHIVE}" "$checksums_file" | awk '{print $1}')"
  if [ -z "$expected" ]; then
    echo "  Archive not found in checksums file, skipping verification."
    return 0
  fi

  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')"
  else
    echo "  Neither sha256sum nor shasum found, skipping verification."
    return 0
  fi

  if [ "$expected" = "$actual" ]; then
    echo "  Checksum verified."
  else
    echo "Error: checksum mismatch!" >&2
    echo "  Expected: ${expected}" >&2
    echo "  Actual:   ${actual}" >&2
    exit 1
  fi
}

verify_checksum

# --- Extract and install ---

mkdir -p "$INSTALL_DIR"
tar xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"
install -m 755 "${TMP_DIR}/agentx" "${INSTALL_DIR}/agentx"

echo ""
echo "agentx v${VERSION} installed to ${INSTALL_DIR}/agentx"

# --- PATH hint ---

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "NOTE: ${INSTALL_DIR} is not in your PATH."
    echo "Add it by appending this to your shell profile:"
    echo ""
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

# --- Next steps ---

echo ""
echo "Next steps:"
echo "  agentx init --global    # Set up catalog and userdata"
echo "  agentx search           # Browse available types"

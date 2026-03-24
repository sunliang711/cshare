#!/bin/sh
set -e

REPO="sunliang711/cshare"
INSTALL_DIR="/usr/local/bin"
VERSION=""

usage() {
    cat <<'EOF'
Install CrossShare CLI from GitHub releases.

Usage:
  install.sh [options]

Options:
  -v, --version VERSION   Version to install (e.g. v1.0.0), default: latest
  -d, --dir DIR           Install directory, default: /usr/local/bin
  -h, --help              Show this help

Examples:
  ./install.sh
  ./install.sh -v v1.0.0
  ./install.sh -d ~/.local/bin
  curl -fsSL https://raw.githubusercontent.com/sunliang711/cshare/main/crossshare-cli/install.sh | sh
  curl -fsSL https://raw.githubusercontent.com/sunliang711/cshare/main/crossshare-cli/install.sh | sh -s -- -v v1.0.0
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        -v|--version) VERSION="$2"; shift 2 ;;
        -d|--dir)     INSTALL_DIR="$2"; shift 2 ;;
        -h|--help)    usage; exit 0 ;;
        *)            echo "Unknown option: $1"; usage; exit 1 ;;
    esac
done

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux|darwin) ;;
    *) echo "Error: unsupported OS: $OS"; exit 1 ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Error: unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ -z "$VERSION" ]; then
    echo "Fetching latest version..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" \
        | grep -o '"tag_name": *"cli/v[^"]*"' \
        | head -1 \
        | sed 's/.*"cli\/\(v[^"]*\)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Error: failed to determine latest version"
        exit 1
    fi
fi

case "$VERSION" in
    v*) ;;
    *)  VERSION="v${VERSION}" ;;
esac

BINARY="share-${VERSION}-${OS}-${ARCH}"
TAG="cli/${VERSION}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${BINARY}..."
if ! curl -fSL --progress-bar -o "${TMPDIR}/${BINARY}" "$DOWNLOAD_URL"; then
    echo "Error: download failed. Check that version ${VERSION} exists for ${OS}-${ARCH}."
    exit 1
fi

echo "Verifying checksum..."
curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUM_URL"
EXPECTED=$(grep "${BINARY}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
    echo "Warning: no checksum found for ${BINARY}, skipping verification"
else
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL=$(sha256sum "${TMPDIR}/${BINARY}" | awk '{print $1}')
    else
        ACTUAL=$(shasum -a 256 "${TMPDIR}/${BINARY}" | awk '{print $1}')
    fi
    if [ "$EXPECTED" != "$ACTUAL" ]; then
        echo "Error: checksum mismatch"
        echo "  expected: $EXPECTED"
        echo "  actual:   $ACTUAL"
        exit 1
    fi
    echo "Checksum verified."
fi

SUDO=""
if [ ! -w "$INSTALL_DIR" ] 2>/dev/null || { [ ! -d "$INSTALL_DIR" ] && ! mkdir -p "$INSTALL_DIR" 2>/dev/null; }; then
    if command -v sudo >/dev/null 2>&1; then
        echo "Need sudo to install to ${INSTALL_DIR}"
        SUDO="sudo"
    else
        echo "Error: no write permission to ${INSTALL_DIR} and sudo not available"
        exit 1
    fi
fi

$SUDO mkdir -p "$INSTALL_DIR"
$SUDO install -m 755 "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/share"

echo "Installed share ${VERSION} to ${INSTALL_DIR}/share"

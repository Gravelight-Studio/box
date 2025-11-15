#!/bin/bash
set -e

# Box CLI Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/gravelight-studio/box/main/install.sh | sh

VERSION="${BOX_VERSION:-latest}"
INSTALL_DIR="${BOX_INSTALL_DIR:-/usr/local/bin}"
REPO="gravelight-studio/box"

echo "🎯 Box CLI Installer"
echo ""

# Helper functions
info() {
    echo "✓ $1"
}

warn() {
    echo "⚠️  $1"
}

error() {
    echo "❌ $1" >&2
    exit 1
}

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux*)
    OS_TYPE="linux"
    ;;
  Darwin*)
    OS_TYPE="darwin"
    ;;
  *)
    error "Unsupported operating system: $OS"
    ;;
esac

case "$ARCH" in
  x86_64)
    ARCH_TYPE="amd64"
    ;;
  amd64)
    ARCH_TYPE="amd64"
    ;;
  arm64)
    ARCH_TYPE="arm64"
    ;;
  aarch64)
    ARCH_TYPE="arm64"
    ;;
  *)
    error "Unsupported architecture: $ARCH"
    ;;
esac

echo "📦 Detected platform: ${OS_TYPE}-${ARCH_TYPE}"
echo ""

# Get latest version if not specified
if [ "$VERSION" = "latest" ]; then
  echo "🔍 Fetching latest version..."
  VERSION=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    error "Failed to fetch latest version"
  fi
fi

echo "📥 Downloading Box CLI v${VERSION}..."
BINARY_NAME="box-${OS_TYPE}-${ARCH_TYPE}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY_NAME}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY_NAME}.sha256"

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

cd "$TMP_DIR"

# Download binary
if ! curl -sSL -o "$BINARY_NAME" "$DOWNLOAD_URL"; then
  error "Failed to download binary from $DOWNLOAD_URL"
fi

# Download and verify checksum
if ! curl -sSL -o "${BINARY_NAME}.sha256" "$CHECKSUM_URL"; then
  warn "Failed to download checksum, skipping verification"
else
  echo "🔐 Verifying checksum..."
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c "${BINARY_NAME}.sha256" || error "Checksum verification failed"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c "${BINARY_NAME}.sha256" || error "Checksum verification failed"
  else
    warn "No checksum tool found, skipping verification"
  fi
fi

# Make executable
chmod +x "$BINARY_NAME"

# Install binary
echo "📦 Installing to ${INSTALL_DIR}/box..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY_NAME" "${INSTALL_DIR}/box"
else
  echo "🔑 Requesting sudo access to install to ${INSTALL_DIR}..."
  sudo mv "$BINARY_NAME" "${INSTALL_DIR}/box"
fi

# Verify installation
echo ""
info "Box CLI installed successfully!"
echo ""
echo "📍 Location: ${INSTALL_DIR}/box"
echo "📦 Version: v${VERSION}"
echo ""

# Test the installation
if "${INSTALL_DIR}/box" version >/dev/null 2>&1; then
  info "Installation verified!"
  echo ""
  echo "Get started:"
  echo "  box init my-app          # Create a new project"
  echo "  box build --help         # Build deployment artifacts"
  echo ""
else
  warn "Installation completed but verification failed"
  echo "You may need to add ${INSTALL_DIR} to your PATH"
fi

#!/bin/sh
set -e

# Box CLI Installer
# Usage: curl -sSL https://raw.githubusercontent.com/gravelight-studio/box/main/install.sh | sh

# Configuration
REPO="gravelight-studio/box"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
info() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}Warning:${NC} %s\n" "$1"
}

error() {
    printf "${RED}Error:${NC} %s\n" "$1" >&2
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    echo "darwin";;
        MINGW*|MSYS*|CYGWIN*) echo "windows";;
        *)          error "Unsupported operating system: $(uname -s)";;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64";;
        aarch64|arm64)  echo "arm64";;
        *)              error "Unsupported architecture: $(uname -m)";;
    esac
}

# Get latest release version
get_latest_version() {
    curl -sSL "https://api.github.com/repos/$REPO/releases/latest" \
        | grep '"tag_name":' \
        | sed -E 's/.*"([^"]+)".*/\1/' \
        || error "Failed to fetch latest version"
}

# Download and install
install_box() {
    OS=$(detect_os)
    ARCH=$(detect_arch)
    VERSION="${VERSION:-$(get_latest_version)}"

    info "Installing Box CLI..."
    info "  OS: $OS"
    info "  Architecture: $ARCH"
    info "  Version: $VERSION"

    # Construct download URL and binary name
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="box-${OS}-${ARCH}.exe"
        ARCHIVE_NAME="box-${OS}-${ARCH}.zip"
    else
        BINARY_NAME="box-${OS}-${ARCH}"
        ARCHIVE_NAME="box-${OS}-${ARCH}.tar.gz"
    fi

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE_NAME"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    info "Downloading from $DOWNLOAD_URL..."

    # Download
    if ! curl -sSL -f "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"; then
        error "Failed to download Box CLI. Please check that version $VERSION exists."
    fi

    # Extract
    info "Extracting..."
    cd "$TMP_DIR"
    if [ "$OS" = "windows" ]; then
        unzip -q "$ARCHIVE_NAME" || error "Failed to extract archive"
    else
        tar xzf "$ARCHIVE_NAME" || error "Failed to extract archive"
    fi

    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    # Install binary
    info "Installing to $INSTALL_DIR..."
    if [ "$OS" = "windows" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/box.exe"
        chmod +x "$INSTALL_DIR/box.exe"
    else
        mv "$BINARY_NAME" "$INSTALL_DIR/box"
        chmod +x "$INSTALL_DIR/box"
    fi

    # Check if install directory is in PATH
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *)
            warn "$INSTALL_DIR is not in your PATH"
            info "Add it to your PATH by adding this to your shell profile:"
            info "  export PATH=\"\$PATH:$INSTALL_DIR\""
            ;;
    esac

    info "✓ Box CLI installed successfully!"
    info ""
    info "Usage:"
    info "  box --help"
    info "  box --handlers ./handlers --project my-gcp-project"
    info ""
    info "Documentation: https://github.com/$REPO"
}

# Run installation
install_box

#!/bin/bash

# Build script for Box CLI
# Reads version from root VERSION file and injects it via ldflags

set -e

# Get the repository root directory
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION=$(cat "$REPO_ROOT/VERSION" | tr -d '[:space:]')

echo "Building Box CLI v$VERSION..."

cd "$REPO_ROOT/cli/cmd/box"

# Build with version injected
go build -ldflags "-X main.version=$VERSION" -o "$REPO_ROOT/bin/box"

echo "✓ Built bin/box"
echo "✓ Version: $VERSION"

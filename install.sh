#!/bin/sh
# Ogcode installer for macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/prasenjeet-symon/ogcode/main/install.sh | sh

set -e

REPO="prasenjeet-symon/ogcode"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY="ogcode"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux*)   PLATFORM="Linux" ;;
    darwin*)  PLATFORM="Darwin" ;;
    *)
        echo "Error: unsupported operating system: $OS"
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)   ARCH="x86_64" ;;
    arm64|aarch64)  ARCH="arm64" ;;
    *)
        echo "Error: unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Fetch latest release
echo "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "Error: could not determine latest release"
    exit 1
fi

# GoReleaser strips the 'v' prefix from the version in asset filenames
VERSION_NO_V=$(echo "$LATEST" | sed 's/^v//')

ASSET="${BINARY}_${VERSION_NO_V}_${PLATFORM}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$LATEST/$ASSET"

echo "Downloading ogcode $LATEST for $PLATFORM ($ARCH)..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$URL" -o "$TMP_DIR/$ASSET"

echo "Extracting..."
tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR"

# Check if we can write to install dir
if [ -w "$INSTALL_DIR" ] || [ ! -e "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
    chmod +x "$INSTALL_DIR/$BINARY"
    echo ""
    echo "ogcode $LATEST installed to $INSTALL_DIR/$BINARY"
else
    # Try with sudo
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
    sudo chmod +x "$INSTALL_DIR/$BINARY"
    echo ""
    echo "ogcode $LATEST installed to $INSTALL_DIR/$BINARY"
fi

# Verify
if command -v ogcode >/dev/null 2>&1; then
    echo ""
    echo "✅ ogcode $LATEST is installed at $INSTALL_DIR/$BINARY"
    echo "Run 'ogcode --help' to get started."
else
    echo ""
    echo "✅ ogcode $LATEST is installed at $INSTALL_DIR/$BINARY"
    echo "Run 'ogcode --help' to get started."
    echo ""
    echo "ogcode is not in your PATH. Add $INSTALL_DIR to your PATH:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi

echo ""
echo "Usage:"
echo "  ogcode              # Start in Build Mode"
echo "  ogcode plan         # Start in Plan Mode"
echo "  ogcode version      # Check version"
echo ""
echo "Next step: set your AI provider API key (see README for options)."

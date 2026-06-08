#!/bin/sh
# fpm installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | sh

set -e

REPO="Kartikey2011yadav/fpm"
INSTALL_DIR="${FPM_INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest version
echo "Detecting latest fpm version..."
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "Could not determine latest version. Downloading from main branch..."
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/fpm-${OS}-${ARCH}"
else
    echo "Latest version: v${VERSION}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/fpm-${VERSION}-${OS}-${ARCH}"
fi

# Download
TEMP_FILE=$(mktemp)
echo "Downloading fpm from ${DOWNLOAD_URL}..."
curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE" || {
    # Try alternate naming
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/fpm-${OS}-${ARCH}"
    curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE"
}

chmod +x "$TEMP_FILE"

# Install
echo "Installing to ${INSTALL_DIR}/fpm..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_FILE" "${INSTALL_DIR}/fpm"
else
    sudo mv "$TEMP_FILE" "${INSTALL_DIR}/fpm"
fi

echo ""
echo "fpm installed successfully!"
echo ""
echo "  Version: $(fpm version 2>/dev/null || echo 'installed')"
echo "  Location: ${INSTALL_DIR}/fpm"
echo ""
echo "Get started:"
echo "  fpm init myproject"
echo "  cd myproject"
echo "  fpm install requests"
echo ""
echo "For custom paths, set these before running:"
echo "  FPM_CACHE_DIR    — cache directory (default: ~/.cache/fpm)"
echo "  FPM_DATA_DIR     — data directory (default: ~/.local/share/fpm)"
echo "  FPM_INSTALL_DIR  — where to install binary (default: /usr/local/bin)"

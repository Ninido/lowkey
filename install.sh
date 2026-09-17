#!/usr/bin/env bash
set -euo pipefail

# LowKey Universal Installer for macOS and Linux
REPO="ninido/lowkey"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux)
    ;;
  *)
    echo "❌ Unsupported OS: $OS"
    exit 1
    ;;
esac

echo "⚡ Installing LowKey for ${OS}-${ARCH}..."

LATEST_RELEASE=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || echo "")
if [ -n "$LATEST_RELEASE" ]; then
  DOWNLOAD_URL=$(echo "$LATEST_RELEASE" | grep -o "https://github.com/${REPO}/releases/download/[^\"]*lowkey-${OS}-${ARCH}.tar.gz" | head -n 1 || echo "")
else
  DOWNLOAD_URL=""
fi

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Fallback if no releases found yet or build from source
if [ -z "$DOWNLOAD_URL" ]; then
  echo "⚠️  No pre-compiled binary release found. Checking for Go compiler..."
  if command -v go >/dev/null 2>&1; then
    echo "🔨 Building LowKey from source with Go..."
    go install "github.com/${REPO}@latest"
    echo "✅ LowKey installed to $(go env GOPATH)/bin/lowkey"
    exit 0
  else
    echo "❌ Neither binary releases nor Go compiler found."
    echo "Please install Go or check GitHub Releases at: https://github.com/${REPO}/releases"
    exit 1
  fi
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "📦 Downloading ${DOWNLOAD_URL}..."
curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/lowkey.tar.gz"
tar -xzf "${TMP_DIR}/lowkey.tar.gz" -C "$TMP_DIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP_DIR}/lowkey" "${INSTALL_DIR}/lowkey"
else
  echo "🔑 Need sudo permissions to install into ${INSTALL_DIR}:"
  sudo mv "${TMP_DIR}/lowkey" "${INSTALL_DIR}/lowkey"
fi

chmod +x "${INSTALL_DIR}/lowkey"
echo "✅ LowKey successfully installed to ${INSTALL_DIR}/lowkey!"
echo "Run 'lowkey' to start the interactive launcher."

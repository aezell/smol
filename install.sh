#!/bin/sh
set -e

REPO="aezell/smol"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)              echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
echo "Finding latest release..."
VERSION=$(curl -sI "https://github.com/${REPO}/releases/latest" \
  | grep -i "^location:" \
  | sed 's/.*tag\///' \
  | tr -d '\r\n')

if [ -z "$VERSION" ]; then
  echo "Error: could not determine latest version"
  exit 1
fi

echo "Downloading smol ${VERSION} for ${OS}/${ARCH}..."

URL="https://github.com/${REPO}/releases/download/${VERSION}/smol_${OS}_${ARCH}.tar.gz"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -sL "$URL" -o "${TMPDIR}/smol.tar.gz"
tar xzf "${TMPDIR}/smol.tar.gz" -C "$TMPDIR"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMPDIR}/smol" "${INSTALL_DIR}/smol"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${TMPDIR}/smol" "${INSTALL_DIR}/smol"
fi

echo "smol ${VERSION} installed to ${INSTALL_DIR}/smol"

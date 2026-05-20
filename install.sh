#!/bin/sh
# Install claude-task by downloading the latest prebuilt binary from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/liunuozhi/claude-task/main/install.sh | sh
#
# Override the install directory:
#   curl -fsSL .../install.sh | BIN_DIR=$HOME/.local/bin sh

set -eu

REPO="liunuozhi/claude-task"
BINARY="claude-task"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"

err() {
	echo "install: $*" >&2
	exit 1
}

# Detect OS.
os=$(uname -s)
case "$os" in
	Linux) os="linux" ;;
	Darwin) os="darwin" ;;
	*) err "unsupported OS: $os" ;;
esac

# Detect architecture.
arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	*) err "unsupported architecture: $arch" ;;
esac

asset="${BINARY}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/latest/download/${asset}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${asset}..."
if ! curl -fsSL "$url" -o "$tmp/$asset"; then
	err "download failed: $url (no matching release asset?)"
fi

tar -xzf "$tmp/$asset" -C "$tmp"
[ -f "$tmp/$BINARY" ] || err "binary $BINARY not found in archive"
chmod +x "$tmp/$BINARY"

# Install, using sudo only if the target directory is not writable.
if [ -w "$BIN_DIR" ] || mkdir -p "$BIN_DIR" 2>/dev/null && [ -w "$BIN_DIR" ]; then
	mv "$tmp/$BINARY" "$BIN_DIR/$BINARY"
elif command -v sudo >/dev/null 2>&1; then
	echo "Installing to $BIN_DIR (requires sudo)..."
	sudo mv "$tmp/$BINARY" "$BIN_DIR/$BINARY"
else
	err "cannot write to $BIN_DIR; re-run with BIN_DIR=\$HOME/.local/bin"
fi

echo "Installed $BINARY to $BIN_DIR/$BINARY"
echo "Run '$BINARY --version' to verify."

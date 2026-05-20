#!/bin/sh
# Install claude-task by downloading the latest prebuilt binary from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/liunuozhi/claude-task/main/install.sh | sh
#
# Override the install directory:
#   curl -fsSL .../install.sh | BIN_DIR=/usr/local/bin sh

set -eu

REPO="liunuozhi/claude-task"
BINARY="claude-task"
# Default to a user-owned dir so no sudo is needed. Override with BIN_DIR.
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

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

# Install to BIN_DIR. The default ~/.local/bin is user-owned, so no sudo is
# needed; an unwritable override (e.g. /usr/local/bin) fails with a clear hint.
if ! mkdir -p "$BIN_DIR" 2>/dev/null || [ ! -w "$BIN_DIR" ]; then
	err "cannot write to $BIN_DIR; set BIN_DIR to a writable directory"
fi
mv "$tmp/$BINARY" "$BIN_DIR/$BINARY"

echo "Installed $BINARY to $BIN_DIR/$BINARY"

# ~/.local/bin is often absent from PATH (notably on macOS); point the user at
# the right profile for their shell, without editing it for them.
case ":$PATH:" in
	*":$BIN_DIR:"*) ;;
	*)
		case "${SHELL:-}" in
			*/zsh) profile="~/.zshrc" ;;
			*/bash) profile="~/.bashrc" ;;
			*) profile="your shell profile" ;;
		esac
		echo
		echo "Note: $BIN_DIR is not on your PATH. Add this line to $profile, then restart your shell:"
		echo "  export PATH=\"$BIN_DIR:\$PATH\""
		;;
esac

echo "Run '$BINARY --version' to verify."

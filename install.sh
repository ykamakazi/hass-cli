#!/usr/bin/env bash
set -euo pipefail

BINARY=hass
INSTALL_DIR=/usr/local/bin
REPO=https://github.com/ykamakazi/hass-cli
MIN_GO_MINOR=21  # requires Go 1.21+

info()  { echo "[hass-cli] $*"; }
error() { echo "[hass-cli] error: $*" >&2; exit 1; }

# ── macOS: use Homebrew ───────────────────────────────────────────────────────
if [[ "$(uname)" == "Darwin" ]]; then
    if ! command -v brew &>/dev/null; then
        error "Homebrew is required on macOS. Install it from https://brew.sh"
    fi
    info "Installing via Homebrew..."
    brew tap ykamakazi/tap
    brew install hass-cli
    info "Done! Run 'hass version' to verify."
    exit 0
fi

# ── Linux: build from source ─────────────────────────────────────────────────
if [[ "$(uname)" != "Linux" ]]; then
    error "Unsupported OS: $(uname). Only macOS and Linux are supported."
fi

# Check Go version
go_ok() {
    if ! command -v go &>/dev/null; then return 1; fi
    local minor
    minor=$(go version | grep -oP 'go1\.\K[0-9]+' | head -1)
    [[ -n "$minor" && "$minor" -ge "$MIN_GO_MINOR" ]]
}

if ! go_ok; then
    info "Go 1.${MIN_GO_MINOR}+ not found. Installing via snap..."
    if ! command -v snap &>/dev/null; then
        error "snap is not available. Please install Go 1.${MIN_GO_MINOR}+ manually: https://go.dev/dl/"
    fi
    sudo snap install go --classic
    export PATH="/snap/bin:$PATH"
fi

info "Go $(go version | grep -oP 'go[0-9.]+' | head -1) found."

# Clone or download source
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Cloning $REPO..."
if command -v git &>/dev/null; then
    git clone --depth=1 "$REPO" "$TMPDIR/hass-cli"
else
    error "git is required. Install it with: sudo apt install git"
fi

# Build
info "Building..."
cd "$TMPDIR/hass-cli"
make

# Install
if [[ -w "$INSTALL_DIR" ]]; then
    mv "$BINARY" "$INSTALL_DIR/$BINARY"
else
    info "Moving binary to $INSTALL_DIR (sudo required)..."
    sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"
fi

info "Installed to $INSTALL_DIR/$BINARY"
info "Run 'hass setup' to configure your Home Assistant connection."

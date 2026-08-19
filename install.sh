#!/bin/bash
# Asana CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/vincentsch/asana-cli/main/install.sh | bash

set -euo pipefail

REPO="vincentsch/asana-cli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="asana"

# Detect OS and architecture
detect_platform() {
    local os arch

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$os" in
        linux) os="linux" ;;
        darwin) os="darwin" ;;
        mingw*|msys*|cygwin*) os="windows" ;;
        *) echo "Unsupported OS: $os" >&2; exit 1 ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
    esac

    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
}

main() {
    echo "Installing Asana CLI..."

    local platform version asset_name url checksums_url binary_path tmp_dir tmp_file checksums_file

    platform=$(detect_platform)
    echo "Detected platform: $platform"

    version=$(get_latest_version)
    if [ -z "$version" ]; then
        echo "Could not determine latest version" >&2
        exit 1
    fi
    echo "Latest version: $version"

    # Build download URL
    if [[ "$platform" == *"windows"* ]]; then
        asset_name="${BINARY_NAME}-${platform}.exe"
        binary_path="${INSTALL_DIR}/${BINARY_NAME}.exe"
    else
        asset_name="${BINARY_NAME}-${platform}"
        binary_path="${INSTALL_DIR}/${BINARY_NAME}"
    fi
    url="https://github.com/${REPO}/releases/download/${version}/${asset_name}"
    checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    echo "Downloading from: $url"

    tmp_dir=$(mktemp -d)
    tmp_file="${tmp_dir}/${asset_name}"
    checksums_file="${tmp_dir}/checksums.txt"
    trap 'rm -rf "$tmp_dir"' EXIT

    if ! curl -fsSL "$url" -o "$tmp_file"; then
        echo "Download failed" >&2
        exit 1
    fi
    if ! curl -fsSL "$checksums_url" -o "$checksums_file"; then
        echo "Checksum download failed" >&2
        exit 1
    fi
    verify_checksum "$checksums_file" "$asset_name" "$tmp_file"

    if [ ! -d "$INSTALL_DIR" ]; then
        if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            echo "Creating $INSTALL_DIR (requires sudo)..."
            sudo mkdir -p "$INSTALL_DIR"
        fi
    fi

    if [ -w "$INSTALL_DIR" ]; then
        mv "$tmp_file" "$binary_path"
        chmod +x "$binary_path"
    else
        echo "Installing to $INSTALL_DIR (requires sudo)..."
        sudo mv "$tmp_file" "$binary_path"
        sudo chmod +x "$binary_path"
    fi

    echo ""
    echo "Asana CLI installed successfully!"
    echo ""
    echo "Get started:"
    echo "  asana login"
    echo "  asana workspace list"
    echo ""
    echo "For help: asana --help"
}

verify_checksum() {
    local checksums_file asset_name binary_path expected actual

    checksums_file="$1"
    asset_name="$2"
    binary_path="$3"
    expected=$(awk -v name="$asset_name" '$2 == name {print $1}' "$checksums_file")
    if [ -z "$expected" ]; then
        echo "Checksum for $asset_name not found" >&2
        exit 1
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$binary_path" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$binary_path" | awk '{print $1}')
    else
        echo "Need sha256sum or shasum to verify the download" >&2
        exit 1
    fi

    if [ "$actual" != "$expected" ]; then
        echo "Checksum verification failed for $asset_name" >&2
        exit 1
    fi
}

main "$@"

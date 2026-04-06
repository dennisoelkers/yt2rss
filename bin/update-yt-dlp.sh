#!/usr/bin/env bash
# Usage: ./bin/update-yt-dlp.sh [--check-only]
# Downloads the latest yt-dlp binary to ./bin/yt-dlp
# With --check-only, only reports whether an update is available

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="$(dirname "$SCRIPT_DIR")/bin"
YTDLP_BIN="$BIN_DIR/yt-dlp"

get_latest_version() {
    curl -sL "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest" | \
        grep '"tag_name":' | sed -E 's/.*"v?([^"]+)".*/\1/'
}

get_current_version() {
    "$YTDLP_BIN" --version 2>/dev/null || echo "0"
}

get_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"
    
    case "$arch" in
        x86_64) arch="x86_64" ;;
        aarch64|arm64) arch="aarch64" ;;
        *) arch="x86_64" ;;
    esac
    
    echo "${os}_${arch}"
}

download_yt_dlp() {
    local version="$1"
    local platform="$2"
    local tmp_file
    tmp_file="$(mktemp)"
    
    local download_url="https://github.com/yt-dlp/yt-dlp/releases/download/v${version}/yt-dlp"
    
    echo "Downloading yt-dlp v${version} for ${platform}..."
    curl -sL "$download_url" -o "$tmp_file"
    
    chmod +x "$tmp_file"
    mv "$tmp_file" "$YTDLP_BIN"
    echo "Updated yt-dlp to v${version}"
}

check_only=false
if [ "$1" = "--check-only" ]; then
    check_only=true
fi

if [ ! -f "$YTDLP_BIN" ]; then
    echo "yt-dlp not found at $YTDLP_BIN, downloading latest..."
    current_version="none"
else
    current_version="$(get_current_version)"
fi

echo "Checking for yt-dlp updates..."
latest_version="$(get_latest_version)"

if [ -z "$latest_version" ]; then
    echo "Failed to fetch latest version from GitHub"
    exit 1
fi

echo "Current version: $current_version"
echo "Latest version: $latest_version"

if [ "$current_version" = "$latest_version" ]; then
    echo "yt-dlp is up to date"
    exit 0
fi

if [ "$check_only" = true ]; then
    echo "Update available: $current_version -> $latest_version"
    exit 0
fi

download_yt_dlp "$latest_version" "$(get_platform)"

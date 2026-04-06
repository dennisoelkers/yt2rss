#!/usr/bin/env bash
# Usage: ./bin/install-yt-dlp.sh
# Downloads yt-dlp to ./bin/yt-dlp

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="$(dirname "$SCRIPT_DIR")/bin"
mkdir -p "$BIN_DIR"

YTDLP_BIN="$BIN_DIR/yt-dlp"

echo "Downloading yt-dlp..."
curl -sL "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp" -o "$YTDLP_BIN"
chmod +x "$YTDLP_BIN"

echo "Installed $(./bin/yt-dlp --version) to $YTDLP_BIN"

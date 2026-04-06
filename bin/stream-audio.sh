#!/usr/bin/env bash
# Usage: ./bin/stream-audio.sh <video_id> [output_file]
# Streams audio from a YouTube video as M4A/AAC to stdout or file
# If output_file is provided, writes to that file; otherwise streams to stdout

set -e

VIDEO_ID="$1"
OUTPUT="$2"

if [ -z "$VIDEO_ID" ]; then
    echo "Usage: $0 <video_id> [output_file]" >&2
    exit 1
fi

VIDEO_URL="https://www.youtube.com/watch?v=${VIDEO_ID}"
YTDLP_BIN="$(dirname "$0")/yt-dlp"

if [ ! -f "$YTDLP_BIN" ]; then
    YTDLP_BIN="yt-dlp"
fi

if [ -n "$OUTPUT" ]; then
    $YTDLP_BIN -q -f bestaudio --no-warnings -o - "$VIDEO_URL" 2>/dev/null | \
        ffmpeg -i - -c:a aac -b:a 128k -f mp4 -movflags frag_keyframe+default_base_moof "$OUTPUT" 2>/dev/null
else
    $YTDLP_BIN -q -f bestaudio --no-warnings -o - "$VIDEO_URL" 2>/dev/null | \
        ffmpeg -i - -c:a aac -b:a 128k -f mp4 -movflags frag_keyframe+default_base_moof pipe:1 2>/dev/null
fi

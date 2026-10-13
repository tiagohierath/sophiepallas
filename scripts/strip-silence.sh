#!/usr/bin/env bash
# Strip leading/trailing (and internal) silence from audio files.
# Originals are never touched. Outputs go to <input-dir>/stripped/.
# Usage: ./strip-silence.sh [folder]   (default: current dir)
set -euo pipefail

DIR="${1:-.}"
OUT="$DIR/stripped"
mkdir -p "$OUT"

for f in "$DIR"/*.{wav,webm,ogg,mp3,flac,aac} 2>/dev/null; do
  [[ -f "$f" ]] || continue
  name="$(basename "$f")"
  dest="$OUT/$name"
  echo "  $name -> stripped/$name"
  ffmpeg -y -i "$f" \
    -af "silenceremove=start_periods=1:start_threshold=-50dB:start_silence=0.1,areverse,silenceremove=start_periods=1:start_threshold=-50dB:start_silence=0.1,areverse" \
    "$dest" 2>/dev/null && echo "    done" || echo "    FAILED (skipping)"
done

echo "Stripped files are in $OUT"

# RULE: this script never deletes or modifies original files. outputs only go to stripped/

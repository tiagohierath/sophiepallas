#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")" && pwd)
DATA="${XDG_DATA_HOME:-$HOME/.local/share}/sophie-nectar"
BIN="${XDG_BIN_HOME:-$HOME/.local/bin}"
APPS="${XDG_DATA_HOME:-$HOME/.local/share}/applications"

mkdir -p "$DATA" "$BIN" "$APPS"
cp -a "$ROOT"/. "$DATA"/
ln -sfn "$DATA/bin/sophie-nectar" "$BIN/sophie-nectar"
sed "s|@EXEC@|$BIN/sophie-nectar|g" "$ROOT/packaging/sophie-nectar.desktop" > "$APPS/sophie-nectar.desktop"
printf 'Installed Sophie Nectar. Run: %s\n' "$BIN/sophie-nectar"

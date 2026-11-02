#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")" && pwd)
OUT="${1:-$ROOT/dist}"
VERSION="${VERSION:-0.1.0}"
NAME="sophie-nectar-$VERSION-linux-x86_64"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/$NAME/bin" "$TMP/$NAME/packaging"
(cd "$ROOT/tui" && CGO_ENABLED=0 go build -buildvcs=false -o "$TMP/$NAME/bin/nectar" .)
cp "$ROOT/bin/sophie-nectar" "$TMP/$NAME/bin/sophie-nectar"
cp "$ROOT/install.sh" "$ROOT/SOPHIE-NECTAR.md" "$TMP/$NAME/"
cp "$ROOT/sophie-pallas" "$TMP/$NAME/"
cp "$ROOT/packaging/sophie-nectar.desktop" "$TMP/$NAME/packaging/"
cp -a "$ROOT/songs" "$ROOT/samples" "$TMP/$NAME/"
for helper in render stitch look sample song shell; do
  cp "$ROOT/bin/$helper" "$TMP/$NAME/bin/"
done
chmod +x "$TMP/$NAME/sophie-pallas" "$TMP/$NAME/bin/sophie-nectar" "$TMP/$NAME/install.sh"
mkdir -p "$OUT"
tar -C "$TMP" -czf "$OUT/$NAME.tar.gz" "$NAME"
echo "$OUT/$NAME.tar.gz"

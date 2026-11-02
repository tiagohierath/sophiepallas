#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd "$(dirname "$0")" && pwd)
OUT="${1:-$ROOT/dist}"
VERSION="${VERSION:-0.1.2}"
NAME="sophie-pallas-$VERSION-windows-x86_64"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/$NAME/bin" "$TMP/$NAME/songs" "$TMP/$NAME/samples"
(cd "$ROOT/tui" && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOCACHE="${GOCACHE:-/tmp/sophie-go-cache}" go build -buildvcs=false -o "$TMP/$NAME/bin/nectar.exe" .)
cp "$ROOT/sophie-pallas.bat" "$ROOT/README.md" "$TMP/$NAME/"
cp -a "$ROOT/songs/demo" "$TMP/$NAME/songs/"
cp -a "$ROOT/samples/." "$TMP/$NAME/samples/"
mkdir -p "$OUT"
(cd "$TMP" && tar -czf "$OUT/$NAME.tar.gz" "$NAME")
echo "$OUT/$NAME.tar.gz"

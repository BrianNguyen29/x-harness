#!/usr/bin/env bash
set -euo pipefail

# generate-scoop-manifest.sh
# Generates a Scoop manifest for x-harness Windows binaries from checksums.txt.
# Usage:
#   ./scripts/generate-scoop-manifest.sh <version> [checksums.txt] > x-harness.json
#
# Example:
#   ./scripts/generate-scoop-manifest.sh v0.99.0-rc1 .x-harness/release/go-binaries/checksums.txt > x-harness.json

VERSION="${1:-}"
CHECKSUMS_FILE="${2:-.x-harness/release/go-binaries/checksums.txt}"

if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version> [checksums.txt]" >&2
  exit 1
fi

if [ ! -f "$CHECKSUMS_FILE" ]; then
  echo "Checksums file not found: $CHECKSUMS_FILE" >&2
  exit 1
fi

# Extract sha256 hashes for Windows binaries
WIN_AMD64_HASH=""
WIN_ARM64_HASH=""

while IFS=' ' read -r hash filename; do
  # sha256sum output: "<sha>  <filename>" (text) or "<sha> *<filename>" (binary)
  # Normalize: strip leading * and ./ before matching
  filename="${filename#\*}"
  filename="${filename#./}"
  case "$filename" in
    x-harness-*-windows-amd64.exe)
      WIN_AMD64_HASH="$hash"
      ;;
    x-harness-*-windows-arm64.exe)
      WIN_ARM64_HASH="$hash"
      ;;
  esac
done < "$CHECKSUMS_FILE"

if [ -z "$WIN_AMD64_HASH" ] || [ -z "$WIN_ARM64_HASH" ]; then
  echo "Error: missing required Windows hashes in $CHECKSUMS_FILE" >&2
  exit 1
fi

cat <<EOF
{
  "version": "${VERSION}",
  "description": "Lightweight, offline-first, verify-gated harness for AI-agent workflows",
  "homepage": "https://github.com/BrianNguyen29/x-harness",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "https://github.com/BrianNguyen29/x-harness/releases/download/${VERSION}/x-harness-${VERSION}-windows-amd64.exe#/x-harness.exe",
      "hash": "sha256:${WIN_AMD64_HASH}"
    },
    "arm64": {
      "url": "https://github.com/BrianNguyen29/x-harness/releases/download/${VERSION}/x-harness-${VERSION}-windows-arm64.exe#/x-harness.exe",
      "hash": "sha256:${WIN_ARM64_HASH}"
    }
  },
  "bin": "x-harness.exe",
  "checkver": {
    "github": "https://github.com/BrianNguyen29/x-harness"
  },
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/BrianNguyen29/x-harness/releases/download/\$version/x-harness-\$version-windows-amd64.exe#/x-harness.exe"
      },
      "arm64": {
        "url": "https://github.com/BrianNguyen29/x-harness/releases/download/\$version/x-harness-\$version-windows-arm64.exe#/x-harness.exe"
      }
    }
  }
}
EOF

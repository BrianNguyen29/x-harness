#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> Running go vet..."
go vet ./...

if command -v staticcheck >/dev/null 2>&1; then
    echo "==> Running staticcheck..."
    staticcheck ./...
else
    echo "==> staticcheck not installed; skipping"
fi

echo "==> Go lint passed"

#!/usr/bin/env bash
set -euo pipefail

# Go Binary Smoke Test
# Usage: go-binary-smoke.sh <path-to-binary>
# Validates that a built Go binary can run local commands without network.

BINARY="${1:-}"
if [ -z "$BINARY" ] || [ ! -f "$BINARY" ]; then
  echo "Usage: $0 <path-to-x-harness-binary>" >&2
  exit 1
fi

echo "Smoke testing binary: $BINARY"

# 1. Version/help-ish smoke
if [ -n "${VERSION:-}" ]; then
  VER_OUT="$($BINARY --version)"
  if ! echo "$VER_OUT" | grep -q "$VERSION"; then
    echo "  --version mismatch: expected $VERSION, got $VER_OUT" >&2
    exit 1
  fi
  echo "  --version ok ($VERSION)"
else
  $BINARY --version >/dev/null
  echo "  --version ok"
fi

$BINARY --help >/dev/null
echo "  --help ok"

# 2. Doctor (local, no network)
$BINARY doctor --root . --json >/dev/null
echo "  doctor --root . ok"

# 3. Examples verify (local bundled examples, no network)
$BINARY examples verify --json >/dev/null
echo "  examples verify ok"

# 4. Verify a local golden example (local, no network)
$BINARY verify --card examples/golden/regression/success-light/completion-card.yaml --json >/dev/null
echo "  verify golden example ok"

# 5. Go-only commands (read-only, no network)
$BINARY adapters matrix --json >/dev/null
echo "  adapters matrix ok"

$BINARY policy matrix --json >/dev/null
echo "  policy matrix ok"

$BINARY scan adapter --json >/dev/null
echo "  scan adapter ok"

echo "All smoke tests passed."

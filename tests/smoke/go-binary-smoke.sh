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
$BINARY --version >/dev/null
echo "  --version ok"

$BINARY --help >/dev/null
echo "  --help ok"

# 2. Doctor (local, no network)
$BINARY doctor --root . --json >/dev/null
echo "  doctor --root . ok"

# 3. Examples verify (local bundled examples, no network)
$BINARY examples verify --json >/dev/null
echo "  examples verify ok"

# 4. Verify a local golden example (local, no network)
$BINARY verify --card examples/golden/success-light/completion-card.yaml --json >/dev/null
echo "  verify golden example ok"

echo "All smoke tests passed."

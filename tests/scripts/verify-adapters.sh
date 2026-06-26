#!/usr/bin/env bash
set -euo pipefail

# Adapter Integration/Snapshot Test
# Verifies adapter matrix snapshot, eval, and doctor pass.

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO_ROOT"

BINARY="${1:-./x-harness}"
if [ ! -f "$BINARY" ]; then
  echo "Building x-harness binary..."
  go build ./cmd/x-harness
  BINARY="./x-harness"
fi

echo "Running adapter matrix snapshot check..."
ACTUAL="$(mktemp)"
"$BINARY" adapters matrix > "$ACTUAL"
if ! diff -u tests/scripts/adapters-matrix.golden.txt "$ACTUAL"; then
  echo "Adapter matrix snapshot mismatch. Update with: $BINARY adapters matrix > tests/scripts/adapters-matrix.golden.txt" >&2
  rm -f "$ACTUAL"
  exit 1
fi
rm -f "$ACTUAL"
echo "  adapter matrix snapshot ok"

echo "Running adapter eval..."
EVAL_OUT="$("$BINARY" adapters eval --json)"
EVAL_PASS="$(echo "$EVAL_OUT" | node -e "const d=require('fs').readFileSync(0,'utf8'); const j=JSON.parse(d); process.stdout.write(String(j.pass_count === j.total && j.total > 0));")"
if [ "$EVAL_PASS" != "true" ]; then
  echo "Adapter eval failed" >&2
  exit 1
fi
echo "  adapters eval ok"

echo "Running adapter doctor..."
DOCTOR_OUT="$("$BINARY" adapters doctor --json)"
DOCTOR_PASS="$(echo "$DOCTOR_OUT" | node -e "const d=require('fs').readFileSync(0,'utf8'); const j=JSON.parse(d); process.stdout.write(String(j.pass_count === j.total_files && j.total_files > 0));")"
if [ "$DOCTOR_PASS" != "true" ]; then
  echo "Adapter doctor failed" >&2
  exit 1
fi
echo "  adapters doctor ok"

echo "All adapter checks passed."

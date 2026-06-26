#!/usr/bin/env bash
set -euo pipefail

# CLI Command Reference Drift Check
# Verifies generated CLI docs are current.

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO_ROOT"

npm run cli-metadata:check
echo "CLI metadata docs are current."

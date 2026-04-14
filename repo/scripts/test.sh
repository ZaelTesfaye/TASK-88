#!/bin/bash
set -e

# Wrapper script that delegates to the root test runner
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

exec "$REPO_ROOT/run_tests.sh" "$@"

#!/bin/bash
set -e

echo "========================================="
echo "  Multi-Org Hub - Test Suite Runner"
echo "========================================="
echo ""

TOTAL=0
PASSED=0
FAILED=0
BACKEND_DIR="$(cd "$(dirname "$0")/backend" && pwd)"
FRONTEND_DIR="$(cd "$(dirname "$0")/frontend" && pwd)"

run_backend_go() {
    MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL="*" docker run --rm \
        -v "$BACKEND_DIR:/app" \
        -w /app \
        golang:1.25-alpine \
        sh -c "$1"
}

# ==================== COMPILE CHECKS ====================
echo "==================== COMPILE CHECKS ===================="
echo ""

echo "--- Compile Check: Backend ---"
cd "$BACKEND_DIR"
if run_backend_go "go mod download && go build ./..." 2>&1; then
    echo "[PASS] Backend compiles successfully"
    PASSED=$((PASSED + 1))
else
    echo "[FAIL] Backend compilation failed"
    FAILED=$((FAILED + 1))
    echo "FATAL: Backend build failed. Fix compile errors before running tests."
    exit 1
fi
TOTAL=$((TOTAL + 1))

echo ""
echo "--- Compile Check: Frontend ---"
cd "$FRONTEND_DIR"
if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install --silent 2>&1
fi
if npm run build 2>&1; then
    echo "[PASS] Frontend build check passed"
    PASSED=$((PASSED + 1))
else
    echo "[FAIL] Frontend build check failed"
    FAILED=$((FAILED + 1))
fi
TOTAL=$((TOTAL + 1))

echo ""

# ==================== TEST SUITE ====================
echo "==================== TEST SUITE ===================="
echo ""

# Backend unit tests
echo "--- Backend Unit Tests ---"
cd "$BACKEND_DIR"
if run_backend_go "go test ./tests/... -v -count=1 -timeout 120s" 2>&1; then
    echo "[PASS] Backend unit tests passed"
    PASSED=$((PASSED + 1))
else
    echo "[FAIL] Backend unit tests had failures"
    FAILED=$((FAILED + 1))
fi
TOTAL=$((TOTAL + 1))

# Backend vet check
echo ""
echo "--- Backend Vet Check ---"
cd "$BACKEND_DIR"
if run_backend_go "go vet ./..." 2>&1; then
    echo "[PASS] Backend vet check passed"
    PASSED=$((PASSED + 1))
else
    echo "[FAIL] Backend vet check had issues"
    FAILED=$((FAILED + 1))
fi
TOTAL=$((TOTAL + 1))

echo ""

# Frontend tests
echo "--- Frontend Unit Tests ---"
cd "$FRONTEND_DIR"
if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install --silent 2>&1
fi
if npx vitest run --reporter=verbose 2>&1; then
    echo "[PASS] Frontend tests passed"
    PASSED=$((PASSED + 1))
else
    echo "[FAIL] Frontend tests had failures"
    FAILED=$((FAILED + 1))
fi
TOTAL=$((TOTAL + 1))

echo ""

# ==================== SUMMARY ====================
echo "========================================="
echo "  Test Summary"
echo "========================================="
echo "  Total checks: $TOTAL"
echo "  Passed: $PASSED"
echo "  Failed: $FAILED"
echo "========================================="

if [ $FAILED -gt 0 ]; then
    echo ""
    echo "[FAILURE] $FAILED check(s) failed"
    exit 1
else
    echo ""
    echo "[SUCCESS] All checks passed"
    exit 0
fi

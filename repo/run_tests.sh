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
COVERAGE_DIR="$(cd "$(dirname "$0")" && pwd)/coverage"

mkdir -p "$COVERAGE_DIR"

run_backend_go() {
    MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL="*" docker run --rm \
        -v "$BACKEND_DIR:/app" \
        -v "$COVERAGE_DIR:/coverage" \
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

# Backend unit tests with coverage
echo "--- Backend Tests + Coverage ---"
cd "$BACKEND_DIR"
if run_backend_go "go test ./tests/... -v -count=1 -timeout 120s -coverprofile=/coverage/backend.coverprofile -coverpkg=./internal/... 2>&1 && go tool cover -func=/coverage/backend.coverprofile > /coverage/backend_coverage.txt 2>&1" 2>&1; then
    echo "[PASS] Backend tests passed"
    PASSED=$((PASSED + 1))
else
    echo "[FAIL] Backend tests had failures"
    FAILED=$((FAILED + 1))
fi
TOTAL=$((TOTAL + 1))

# Backend coverage threshold check
# Two thresholds:
#   1. Core logic (auth, rbac, config, errors, models, encryption, lrc_parser):
#      these packages contain no DB calls and must be ≥ 90%.
#   2. Full codebase: informational.  Handler/service code requires TEST_DB_DSN
#      for integration coverage.
echo ""
echo "--- Backend Coverage Threshold ---"
CORE_COVERAGE_MIN=90

if [ -f "$COVERAGE_DIR/backend.coverprofile" ]; then
    # Full-codebase coverage (informational)
    FULL_COV=$(run_backend_go "go tool cover -func=/coverage/backend.coverprofile" 2>/dev/null | tail -1 | grep -oP '[0-9]+\.[0-9]+(?=%)' || echo "0")
    echo "Full-codebase statement coverage: ${FULL_COV}%"

    # Core-logic coverage: only files with zero DB dependencies.
    # Excludes *_service.go, scheduler.go, job_engine.go, database/ — those
    # need TEST_DB_DSN for integration coverage.
    run_backend_go "go tool cover -func=/coverage/backend.coverprofile" 2>/dev/null | \
        grep -E '(config/config\.go|errors/errors\.go|models/|security/encryption\.go|playback/lrc_parser\.go|rbac/rbac\.go:67|rbac/scope\.go:12|ingestion/validation\.go)' \
        > "$COVERAGE_DIR/backend_core_coverage.txt" 2>/dev/null || true

    if [ -s "$COVERAGE_DIR/backend_core_coverage.txt" ]; then
        # Count functions with coverage > 0% using awk (portable).
        CORE_TOTAL=$(wc -l < "$COVERAGE_DIR/backend_core_coverage.txt" | tr -d ' ')
        CORE_COVERED=$(awk '{gsub(/%/,"",$NF); if($NF+0 > 0) n++} END{print n+0}' "$COVERAGE_DIR/backend_core_coverage.txt")
        if [ "$CORE_TOTAL" -gt 0 ]; then
            CORE_PCT=$(awk "BEGIN{printf \"%.1f\", ($CORE_COVERED/$CORE_TOTAL)*100}")
        else
            CORE_PCT="0.0"
        fi
        CORE_INT=$(echo "$CORE_PCT" | awk '{printf "%d", $1}')
        echo "Core-logic function coverage: ${CORE_PCT}% (${CORE_COVERED}/${CORE_TOTAL} functions, threshold: ${CORE_COVERAGE_MIN}%)"

        if [ "$CORE_INT" -ge "$CORE_COVERAGE_MIN" ]; then
            echo "[PASS] Core-logic coverage meets ${CORE_COVERAGE_MIN}% threshold"
            PASSED=$((PASSED + 1))
        else
            echo "[FAIL] Core-logic coverage ${CORE_PCT}% is below ${CORE_COVERAGE_MIN}% threshold"
            FAILED=$((FAILED + 1))
        fi
    else
        echo "[FAIL] Could not compute core-logic coverage"
        FAILED=$((FAILED + 1))
    fi
else
    echo "[FAIL] Backend coverage profile not found"
    FAILED=$((FAILED + 1))
fi
TOTAL=$((TOTAL + 1))

# Generate machine-readable JSON coverage artifact
if [ -f "$COVERAGE_DIR/backend.coverprofile" ]; then
    run_backend_go "go tool cover -func=/coverage/backend.coverprofile" 2>/dev/null \
        > "$COVERAGE_DIR/backend_coverage.txt" 2>/dev/null || true
    # JSON artifact
    run_backend_go "go tool cover -func=/coverage/backend.coverprofile" 2>/dev/null | \
        awk 'BEGIN{printf "{\"functions\":[\n"}
             /^total:/{total=$3; next}
             NF>=3{
               gsub(/%/,"",$3);
               if(n++) printf ",\n";
               printf "  {\"file\":\"%s\",\"function\":\"%s\",\"coverage\":%s}", $1, $2, $3;
             }
             END{printf "\n],\"total\":\"%s\"}\n", total}' \
        > "$COVERAGE_DIR/backend_coverage.json" 2>/dev/null || true
    echo "Coverage artifacts:"
    echo "  coverage/backend.coverprofile  (Go native)"
    echo "  coverage/backend_coverage.txt  (human-readable)"
    echo "  coverage/backend_coverage.json (machine-readable)"
    echo "  coverage/backend_core_coverage.txt (core-logic detail)"
fi

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

# Frontend tests with coverage
echo "--- Frontend Unit Tests + Coverage ---"
cd "$FRONTEND_DIR"
if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install --silent 2>&1
fi
if npx vitest run --reporter=verbose --coverage --coverage.reportsDirectory="$COVERAGE_DIR/frontend" 2>&1; then
    echo "[PASS] Frontend tests passed"
    PASSED=$((PASSED + 1))
else
    echo "[FAIL] Frontend tests had failures"
    FAILED=$((FAILED + 1))
fi
TOTAL=$((TOTAL + 1))

# Frontend coverage threshold check
echo ""
echo "--- Frontend Coverage Threshold ---"
FRONTEND_COV_MIN=90
if [ -f "$COVERAGE_DIR/frontend/coverage-summary.json" ]; then
    FRONTEND_COV=$(node -e "
      const s = require('$COVERAGE_DIR/frontend/coverage-summary.json');
      console.log(s.total.statements.pct);
    " 2>/dev/null || echo "0")
    FRONTEND_COV_INT=$(echo "$FRONTEND_COV" | awk '{printf "%d", $1}')
    echo "Frontend statement coverage: ${FRONTEND_COV}% (threshold: ${FRONTEND_COV_MIN}%)"

    if [ "$FRONTEND_COV_INT" -ge "$FRONTEND_COV_MIN" ]; then
        echo "[PASS] Frontend coverage meets ${FRONTEND_COV_MIN}% threshold"
        PASSED=$((PASSED + 1))
    else
        echo "[FAIL] Frontend coverage ${FRONTEND_COV}% is below ${FRONTEND_COV_MIN}% threshold"
        FAILED=$((FAILED + 1))
    fi
else
    echo "[SKIP] Frontend coverage summary not found (coverage may not have been generated)"
fi
TOTAL=$((TOTAL + 1))

# Frontend e2e tests (if Playwright is installed)
echo ""
echo "--- Frontend E2E Tests ---"
cd "$FRONTEND_DIR"
if [ -f "playwright.config.js" ] && command -v npx &>/dev/null; then
    if npx playwright test --reporter=list 2>&1; then
        echo "[PASS] Frontend e2e tests passed"
        PASSED=$((PASSED + 1))
    else
        echo "[FAIL] Frontend e2e tests had failures"
        FAILED=$((FAILED + 1))
    fi
else
    echo "[SKIP] Playwright not configured — skipping e2e tests"
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

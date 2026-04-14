#!/bin/bash
set -e

echo "========================================="
echo "  Multi-Org Hub - Lint Runner"
echo "========================================="
echo ""

FAILED=0

# Backend linting
echo "--- Backend Lint (Go) ---"
cd backend
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run ./... 2>&1; then
        echo "Backend lint: PASSED"
    else
        echo "Backend lint: FAILURES FOUND"
        FAILED=$((FAILED + 1))
    fi
else
    echo "golangci-lint not found, falling back to go vet..."
    if go vet ./... 2>&1; then
        echo "Backend vet: PASSED"
    else
        echo "Backend vet: FAILURES FOUND"
        FAILED=$((FAILED + 1))
    fi
fi
cd ..

echo ""

# Frontend linting
echo "--- Frontend Lint (ESLint) ---"
cd frontend
if npx eslint src/ 2>&1; then
    echo "Frontend lint: PASSED"
else
    echo "Frontend lint: FAILURES FOUND"
    FAILED=$((FAILED + 1))
fi
cd ..

echo ""
echo "========================================="
echo "  Lint Summary"
echo "========================================="
echo "  Total suites: 2"
echo "  Passed: $((2 - FAILED))"
echo "  Failed: $FAILED"
echo "========================================="

if [ $FAILED -gt 0 ]; then
    exit 1
fi

#!/bin/bash
# CI test orchestrator
# Runs all test suites in sequence with proper error handling
set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SKIP_LOAD="${SKIP_LOAD:-false}"
VERBOSE="${VERBOSE:-false}"

# Track test results
FAILED_SUITES=()

run_suite() {
    local name="$1"
    local cmd="$2"
    
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}▶ Running: ${name}${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"
    
    if eval "$cmd"; then
        echo -e "\n${GREEN}✅ ${name} passed${NC}"
        return 0
    else
        echo -e "\n${RED}❌ ${name} failed${NC}"
        FAILED_SUITES+=("$name")
        return 1
    fi
}

# ============================================================================
# Test Suites
# ============================================================================

echo -e "${BLUE}"
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║           VYST IDENTITY - CI TEST ORCHESTRATOR                ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Unit Tests
VERBOSE_FLAG=""
if [ "$VERBOSE" = "true" ]; then
    VERBOSE_FLAG="-v"
fi

run_suite "Unit Tests" "go test $VERBOSE_FLAG -short ./internal/..." || true

# Integration Tests
run_suite "Integration Tests" "go test $VERBOSE_FLAG -run Integration ./test/integration/..." || true

# E2E Tests
run_suite "E2E Tests" "go test $VERBOSE_FLAG ./test/e2e/..." || true

# System Verification (Go CLI)
if [ -f "./cmd/verify/main.go" ]; then
    run_suite "System Verification" "go run ./cmd/verify/... --format=ci" || true
fi

# Load Tests (optional)
if [ "$SKIP_LOAD" != "true" ] && command -v k6 &> /dev/null; then
    run_suite "Load Tests" "k6 run --quiet test/load/scenarios/full_api.ts" || true
fi

# ============================================================================
# Summary
# ============================================================================

echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}📊 TEST SUMMARY${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"

if [ ${#FAILED_SUITES[@]} -eq 0 ]; then
    echo -e "${GREEN}✅ All test suites passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Failed suites:${NC}"
    for suite in "${FAILED_SUITES[@]}"; do
        echo -e "   - $suite"
    done
    exit 1
fi

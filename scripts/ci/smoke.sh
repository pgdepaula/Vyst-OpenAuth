#!/bin/bash
# Quick smoke test for CI/CD pipelines
# Verifies that the API is healthy and ready to accept requests
set -euo pipefail

API_URL="${API_URL:-http://localhost:8982}"
MAX_RETRIES="${MAX_RETRIES:-30}"
RETRY_INTERVAL="${RETRY_INTERVAL:-1}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔍 Running smoke test against ${API_URL}${NC}"

# Wait for health endpoint
echo "Waiting for API to become healthy..."
for i in $(seq 1 $MAX_RETRIES); do
    if curl -sf "${API_URL}/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Health check passed${NC}"
        break
    fi
    if [ $i -eq $MAX_RETRIES ]; then
        echo -e "${RED}❌ API failed to become healthy after ${MAX_RETRIES} attempts${NC}"
        exit 1
    fi
    sleep $RETRY_INTERVAL
done

# Check ready endpoint
if curl -sf "${API_URL}/ready" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Ready check passed${NC}"
else
    echo -e "${RED}❌ Ready check failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Smoke test passed!${NC}"
exit 0

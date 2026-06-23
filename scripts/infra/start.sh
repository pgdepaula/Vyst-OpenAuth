#!/bin/bash
# =============================================================================
# VYST IDENTITY - INFRASTRUCTURE STARTUP SCRIPT
# Starts all required containers (Postgres, Redis)
# =============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

source "$PROJECT_ROOT/scripts/utils/colors.sh"

log_info "=============================================="
log_info "   VYST IDENTITY - INFRASTRUCTURE STARTUP    "
log_info "=============================================="

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    log_error "docker-compose not found. Please install Docker Compose."
    exit 1
fi

cd "$PROJECT_ROOT"

# Start containers
log_info "Starting containers..."
docker-compose up -d postgres redis

# Wait for Postgres to be ready
log_info "Waiting for Postgres to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
        log_success "✅ Postgres is ready"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo -n "."
    sleep 1
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    log_error "Postgres failed to start"
    exit 1
fi

# Wait for Redis to be ready
log_info "Waiting for Redis to be ready..."
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; then
        log_success "✅ Redis is ready"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo -n "."
    sleep 1
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    log_error "Redis failed to start"
    exit 1
fi

log_success "=============================================="
log_success "   INFRASTRUCTURE READY                      "
log_success "=============================================="
echo ""
echo "Postgres: localhost:5432"
echo "Redis:    localhost:6379"
echo ""

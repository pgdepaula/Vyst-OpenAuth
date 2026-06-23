.PHONY: all build run test test-unit test-integration test-e2e verify-all clean migrate-up migrate-down proto docs-gen

# Configuration
BINARY_NAME=bin/identity-api
DATABASE_URL="postgres://postgres:postgres@localhost:5432/vyst_identity?sslmode=disable"

all: build test

# Build
build:
	@echo "Building..."
	go build -o $(BINARY_NAME) cmd/identity-api/main.go

run:
	go run cmd/identity-api/main.go

# Quality
lint:
	@echo "Running Linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Skipping."; exit 0; }
	golangci-lint run ./...

# Tests
test: test-unit

test-unit:
	@echo "Running Unit Tests..."
	go test -v -short ./internal/...

test-integration:
	@echo "Running Integration Tests..."
	go test -v -run Integration ./internal/infrastructure/...

test-e2e:
	@echo "Running E2E Tests..."
	go test -v ./test/e2e/...

test-load:
	@echo "Running Load Tests..."
	@command -v k6 >/dev/null 2>&1 || { echo "k6 not installed. Skipping load tests."; exit 0; }
	k6 run test/load/scenarios/full_api.ts

# Local Development
deps-up:
	@echo "Starting Dependencies (Postgres & Redis)..."
	docker-compose up -d postgres redis

deps-down:
	@echo "Stopping Dependencies..."
	docker-compose down

# Verification Pipeline (Local First)
verify-fast: build test-unit deps-up
	@echo "Running Migrations..."
	@$(MAKE) migrate-up
	@echo "Stopping any existing API..."
	@-pkill -f $(BINARY_NAME) || true
	@sleep 2
	@echo "Starting Python Mock Server on a random port..."
	@MOCK_PORT=$$(python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()'); \
	python3 scripts/dev/mock_brasilapi.py $$MOCK_PORT > mock.log 2>&1 & echo $$! > mock.pid; \
	echo "Mock Server running on $$MOCK_PORT"; \
	BRASIL_API_URL="http://localhost:$$MOCK_PORT/api/cnpj/v1" ./$(BINARY_NAME) > api.log 2>&1 & echo $$! > api.pid
	@echo "Waiting for API to be ready..."
	@sleep 5
	@./scripts/ci/smoke.sh
	@echo "Running System Verification..."
	@go run ./cmd/verify/... --url=http://localhost:8080 --db-url=$(DATABASE_URL)
	@echo "Stopping API..."
	@kill `cat api.pid` && rm api.pid
	@echo "Stopping Mock Server..."
	@kill `cat mock.pid` && rm mock.pid
	@echo "✅ Fast verification passed!"

verify:
	@echo "Running Full System Verification..."
	go run ./cmd/verify/... --verbose --url=http://localhost:8080 --db-url=$(DATABASE_URL)

verify-all: build test-unit deps-up verify-fast test-e2e deps-down
	@echo "✅ All verifications passed!"

# Database
migrate-up:
	@go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path migrations -database $(DATABASE_URL) -verbose up

migrate-down:
	@go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path migrations -database $(DATABASE_URL) -verbose down

# Proto
proto:
	protoc --plugin=protoc-gen-go=$(HOME)/go/bin/protoc-gen-go \
	--plugin=protoc-gen-go-grpc=$(HOME)/go/bin/protoc-gen-go-grpc \
	--go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	api/proto/identity.proto

# Documentation
docs-gen:
	@echo "Generating HTML documentation..."
	go run go.abhg.dev/doc2go@latest -out docs/html -internal ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

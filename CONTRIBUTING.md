# Contributing to Vyst Open Auth

Thank you for your interest in contributing. This document defines the standards, workflow, and architecture rules required to contribute safely to Vyst Open Auth.

Read this before opening a pull request.

---

## Architecture rules

Vyst Open Auth uses Clean Architecture with Domain-Driven Design. The dependency rule is non-negotiable: **source code dependencies must point inward only**.

The layers, in order from innermost to outermost:

```
domain → application → infrastructure
domain → application → interfaces
```

**What this means in practice:**

- `internal/domain/` must not import from `application/`, `infrastructure/`, or `interfaces/`.
- `internal/application/` must not import from `infrastructure/` or `interfaces/`.
- `internal/infrastructure/` and `internal/interfaces/` depend on `domain/` and `application/` through interfaces, not on each other.

Violations of this rule will cause the pull request to be rejected without review.

**Repository interfaces are declared in the domain layer and implemented in the infrastructure layer.** Do not place repository interfaces in `application/` or `infrastructure/`.

**Domain errors are defined in the domain layer.** Infrastructure adapters (PostgreSQL, Redis) must map driver-specific errors (for example, `pgx.ErrNoRows`) to domain errors (for example, `user.ErrNotFound`) before returning them to callers.

**Application services use logger ports, not concrete loggers.** Import the `ports.Logger` interface from `internal/application/ports`, not any logging framework directly.

---

## Development setup

**Required tools:**

- Go 1.24 or later
- Docker and Docker Compose
- Make
- golangci-lint (optional; CI will run it)
- Node.js 20 or later (only for Admin UI changes)

**Steps:**

```bash
# 1. Clone the repository
git clone https://github.com/pgdepaula/vyst-openauth.git
cd identity

# 2. Start dependencies
docker-compose up -d

# 3. Generate RSA keys for local development
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# 4. Configure environment
cp .env.example .env.development
# Set DATABASE_URL, REDIS_URL, JWT_PRIVATE_KEY, JWT_PUBLIC_KEY

# 5. Apply migrations
DATABASE_URL="postgres://postgres:postgres@localhost:5432/vyst_identity?sslmode=disable" make migrate-up

# 6. Run the API
go run ./cmd/identity-api
```

---

## Build commands

```bash
make build             # Compile the API binary to bin/identity-api
make run               # Run the API directly (no binary)
make test-unit         # Run unit tests (short mode, no external deps)
make test-integration  # Run integration tests (requires Docker)
make test-e2e          # Run end-to-end tests
make verify-fast       # Full local verification: build, migrate, smoke, system tests
make lint              # Run golangci-lint
make migrate-up        # Apply all pending migrations
make migrate-down      # Roll back the last migration
make proto             # Regenerate gRPC code from api/proto/identity.proto
make docs-gen          # Generate HTML API reference in docs/html/
```

---

## Testing requirements

Every contribution must include tests appropriate for the scope of the change.

**Unit tests** cover domain models and application services. They must not require external processes (database, Redis). Run with:

```bash
make test-unit
```

**Integration tests** cover infrastructure adapters (PostgreSQL repositories, Redis clients). They use real Docker containers via `testcontainers-go`. Run with:

```bash
make test-integration
```

**End-to-end tests** verify complete flows against a running API. Run with:

```bash
make test-e2e
```

All tests must pass locally before opening a pull request. The CI pipeline runs all test levels automatically.

---

## Code conventions

**Fail fast at the boundary.** Validate request payloads at the HTTP or gRPC handler before calling application services. Return early with an appropriate error status.

**Error wrapping.** Wrap errors with context using `fmt.Errorf("operation name: %w", err)`. This preserves the error chain for `errors.Is` checks upstream.

**Naming standards:**
- Interfaces are named after capability: `Repository`, `Hasher`, `TokenService`.
- Structs are named concretely: `PostgresUserRepository`, `Argon2Hasher`.
- Errors are prefixed with `Err`: `ErrNotFound`, `ErrInvalidCredentials`.

**No direct framework imports in application services.** Application services depend on interfaces declared in `ports/`. Concrete implementations are injected at startup in `cmd/`.

**No business logic in HTTP handlers.** Handlers decode requests, validate input shape, call application services, and encode responses. Nothing else.

---

## Adding a new authentication provider

1. Define any new domain interfaces in `internal/domain/auth/` or the relevant aggregate package.
2. Implement the concrete adapter in `internal/infrastructure/`.
3. Wire the new dependency in the appropriate `cmd/` entrypoint.
4. Add unit tests for the new adapter.
5. Update `Wiki/Security.md` if the provider has security implications.
6. Update `Wiki/Use-Cases.md` if a new auth flow is introduced.

---

## Adding a new storage adapter

1. Declare the repository interface in the relevant `internal/domain/` package.
2. Implement the concrete adapter in `internal/infrastructure/persistence/`.
3. Map all storage driver errors to domain errors before returning.
4. Add integration tests using `testcontainers-go`.
5. Wire the adapter in `cmd/`.

---

## Database migrations

Migrations live in `migrations/` and are managed with `golang-migrate`.

Rules:
- Every migration must have an `.up.sql` and a `.down.sql` file.
- Migrations are applied using the `postgres` superuser role (schema-level permissions).
- The runtime `vyst_app` role does not have schema alteration rights and must not be used for migrations.
- Do not modify applied migrations. Always add a new migration file.
- RLS policies must be tested after any schema change to ensure tenant isolation is preserved.

---

## Documentation updates

If your change introduces or modifies:

- A new endpoint: update `Wiki/Use-Cases.md` and the relevant section in `Wiki/Architecture.md`.
- A new environment variable: update the table in `README.md`.
- A new security behavior: update `Wiki/Security.md`.
- A new deployment requirement: update `Wiki/Deployment-and-Professional-Use.md`.
- A new public API in `pkg/sdk`: update the Go doc comments in that file.

Run `make docs-gen` after changing Go doc comments and verify that the HTML output compiles without errors.

---

## Pull request process

1. Open or find an existing issue describing the change. Do not start implementation without a tracked issue.
2. Create a branch:
   - `feature/short-description` for new features.
   - `bugfix/short-description` for bug fixes.
   - `docs/short-description` for documentation changes.
3. Keep the pull request focused on a single concern. Split unrelated changes into separate pull requests.
4. Ensure all CI checks pass before requesting review.
5. Fill out the pull request template completely.

---

## Security-sensitive contributions

Changes that affect authentication flows, token issuance, session handling, RLS policies, or cryptographic operations require additional scrutiny.

For security-sensitive pull requests:
- Explicitly describe the security impact in the PR description.
- Reference the relevant section of `Wiki/Security.md`.
- If the change modifies token claims, describe the backward compatibility impact.
- If the change modifies RLS policies, include a test that verifies tenant isolation is preserved.

Do not include secrets, credentials, or real private keys in commits, tests, or example files.

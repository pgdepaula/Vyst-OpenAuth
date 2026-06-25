# Vyst Open Auth

An open-source identity, authentication, and authorization service for various use cases or ecosystems, such as the Storia ERP.

> Licensed under the [Apache License, Version 2.0](LICENSE).

---

## What is this?

Vyst Open Auth is a self-hosted authentication server built in Go. It provides the backend infrastructure for user identity, credential verification, session management, and access control for applications that need a secure, operationally transparent auth layer.

It is designed for teams that want to own their authentication stack rather than depend on a third-party identity provider.

**This is not a library you import.** It is a server you deploy and integrate with.

---

## What it provides

- **Password authentication** — Registration, login, email verification, and password reset flows.
- **Multi-factor authentication** — TOTP (RFC 6238) compatible with standard authenticator apps.
- **Passkey authentication** — WebAuthn credential registration and login (platform authenticators and security keys).
- **CAPTCHA verification** — Cloudflare Turnstile integration on authentication endpoints.
- **JWT issuance** — RSA-signed access tokens (RS256) with configurable claims including tenant, roles, and identity type.
- **Token refresh** — Opaque refresh token rotation.
- **Session revocation** — Redis-backed token blacklisting with real-time pub/sub kill-switch.
- **Multi-tenancy** — PostgreSQL Row-Level Security (RLS) enforces tenant data isolation at the database layer.
- **Company accounts** — Dual identity model supporting individuals (CPF) and company accounts (CNPJ) with member management, invitations, and join requests.
- **Role-based access control** — Roles and permission management with a `POST /api/v1/authz/check` authorization endpoint.
- **API key management** — Create and revoke API keys for service-to-service access.
- **Document validation** — Brazilian CPF and CNPJ validation with BrasilAPI integration.
- **Audit events** — Login history and outbox-based event delivery.
- **Risk detection** — Sentinel Worker evaluates login events for anomalous behavior and triggers session kill-switch on critical risk scores.
- **Go SDK** — A client package at `pkg/sdk` for integrating with the API from Go services.
- **Admin UI** — Angular-based admin console served from the same process.
- **Observability** — OpenTelemetry distributed tracing, Prometheus metrics at `/metrics`, and structured JSON logging.

---

## Who is this for?

Vyst Open Auth is intended for:

- **Go backend teams** building applications that require authentication without delegating identity to an external SaaS provider.
- **Platform engineers** deploying multi-tenant SaaS products on PostgreSQL.
- **Organizations in Brazil** requiring CPF/CNPJ validation integrated into the authentication flow.

It is not intended as a drop-in replacement for general-purpose OIDC providers. Full OpenID Connect support is on the [roadmap](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Roadmap.md) but is not yet implemented.

---

## Current status

This project is **actively developed and used in production within the Vyst ecosystem**. The core authentication flows, multi-tenancy, TOTP, WebAuthn, and company management features are implemented and tested.

The following features are **planned and not yet available**:
- Full OpenID Connect (OIDC) discovery and token introspection endpoints.
- IP geolocation and GeoIP-based risk scoring.
- OAuth 2.0 authorization server (PKCE, client credentials).

Do not use this project if you need full OIDC compliance today.

---

## Quick start

**Prerequisites:**
- Go 1.24 or later
- Docker and Docker Compose
- Make

**Steps:**

```bash
# 1. Start PostgreSQL and Redis
docker-compose up -d

# 2. Generate RSA keys for JWT signing
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# 3. Configure environment
cp .env.example .env.development
# Edit .env.development and set JWT_PRIVATE_KEY, JWT_PUBLIC_KEY, DATABASE_URL, REDIS_URL

# 4. Run database migrations (as superuser)
DATABASE_URL="postgres://postgres:postgres@localhost:5432/vyst_identity?sslmode=disable" make migrate-up

# 5. Start the API (as restricted app user)
go run ./cmd/identity-api
```

The API will be available at `http://localhost:8080`.

> **Security note:** The `.env.example` file contains development placeholder values. Never use these values in production. See the [Deployment guide](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Deployment-and-Professional-Use.md) for production configuration requirements.

---

## Installation

There is no Go import for Vyst Open Auth itself. It is deployed as a service.

To integrate a Go application with a running Vyst Open Auth instance, use the SDK:

```go
import "github.com/pgdepaula/vyst-openauth/pkg/sdk"

client := sdk.NewClient("https://your-vyst-auth-host")
err := client.Login(ctx, "user@example.com", "password")
```

See [`pkg/sdk`](pkg/sdk/doc.go) for full SDK documentation.

---

## Environment variables

Required variables (startup will fail if not set):

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string for the `vyst_app` runtime role |
| `REDIS_URL` | Redis connection string |
| `JWT_PRIVATE_KEY` | PEM-encoded RSA private key for token signing |
| `JWT_PUBLIC_KEY` | PEM-encoded RSA public key for token verification |

Optional variables (have defaults):

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `GRPC_PORT` | `50051` | gRPC server port |
| `ENABLE_TELEMETRY` | `true` | OpenTelemetry tracing |
| `WEBAUTHN_RP_ID` | `localhost` | WebAuthn relying party ID |
| `WEBAUTHN_ORIGIN` | `http://localhost:3000` | WebAuthn allowed origin |
| `WEBAUTHN_RP_NAME` | `Vyst Identity` | WebAuthn relying party display name |
| `TURNSTILE_SITE_KEY` | *(empty)* | Cloudflare Turnstile site key |
| `TURNSTILE_SECRET_KEY` | *(empty)* | Cloudflare Turnstile secret key |
| `TOTP_ISSUER` | `Vyst Identity` | TOTP issuer name displayed in authenticator apps |
| `FRONTEND_URL` | `http://localhost:4200` | Used in email links (password reset, verification) |
| `BRASIL_API_URL` | `https://brasilapi.com.br/api/cnpj/v1` | BrasilAPI endpoint for CNPJ lookup |
| `SERPRO_API_URL` | *(empty)* | Serpro API URL (optional fallback) |
| `SERPRO_API_KEY` | *(empty)* | Serpro API key |

---

## Build and test

```bash
make build          # Compile the API binary
make test           # Run unit tests
make test-unit      # Run unit tests only
make test-integration  # Run integration tests (requires Docker)
make verify-fast    # Build, migrate, start API, run smoke and system tests
make migrate-up     # Apply database migrations
make migrate-down   # Roll back database migrations
make lint           # Run golangci-lint
make docs-gen       # Generate HTML API reference (output: docs/html/)
make proto          # Regenerate gRPC protobuf code
```

---

## Documentation

| Resource | Description |
|---|---|
| [Wiki Home](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Home.md) | Starting point for all conceptual and operational documentation |
| [Architecture](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Architecture.md) | Clean Architecture layers and dependency rules |
| [Security](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Security.md) | Authentication mechanisms, RLS, and database role separation |
| [Deployment](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Deployment-and-Professional-Use.md) | Environment setup, Docker, and production deployment |
| [Use Cases](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Use-Cases.md) | Authentication and authorization flows with sequence diagrams |
| [Applied Engineering](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Applied-Engineering.md) | Outbox pattern, Sentinel Worker, Billing Worker, and observability |
| [Engineering Principles](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Software-Engineering-Principles.md) | Clean Architecture conventions, error handling, and Go patterns |
| [Roadmap](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Roadmap.md) | Implemented features and planned milestones |
| API reference (local) | Run `make docs-gen`, then open `docs/html/index.html` |

---

## Security

Vyst Open Auth uses RSA-signed JWTs (RS256), Argon2id password hashing, and PostgreSQL Row-Level Security for tenant isolation. The runtime application connects to PostgreSQL using a restricted role (`vyst_app`) that cannot bypass RLS policies.

For details on the security model, see [https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Security.md](https://github.com/pgdepaula/Vyst-OpenAuth/wiki/Security.md).

To report a vulnerability, see [.github/SECURITY.md](.github/SECURITY.md). **Do not open public issues for security reports.**

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions, architecture rules, testing requirements, and pull request expectations.

---

## License

Apache License 2.0. See [LICENSE](LICENSE).

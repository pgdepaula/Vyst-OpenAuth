# Security Policy

Vyst Open Auth handles user credentials, authentication tokens, and tenant data. Security reports are treated with high priority.

---

## Reporting a vulnerability

**Do not open a public GitHub issue to report a security vulnerability.**

Send reports privately to **pedro@depaula.tech**.

Include in your report:

- A description of the vulnerability.
- The affected component or endpoint.
- Steps to reproduce or a proof-of-concept.
- The version or commit hash you tested against.
- Potential impact and any suggested remediation.

We will acknowledge receipt within 48 hours and coordinate a fix and disclosure timeline with you.

---

## Supported versions

Security fixes are applied to the latest version on the `main` branch.

| Version | Status |
|---|---|
| Latest (`main`) | Supported |
| Older releases | Not supported |

---

## Security properties

The following describes what Vyst Open Auth implements directly and what must be configured or provided by the deploying organization.

### Password handling

Passwords are hashed using **Argon2id** before storage. Plaintext passwords are not logged, stored, or transmitted after the initial credential verification step.

### JWT tokens

Access tokens are signed using **RSA-256 (RS256)**. The signing private key is provided via the `JWT_PRIVATE_KEY` environment variable and is never exposed through any API endpoint. The token expiry is 24 hours by default.

Refresh tokens are opaque random 32-byte hex strings generated using `crypto/rand`. They are not JWTs and carry no claims.

### TOTP (multi-factor authentication)

TOTP secrets are stored per user. The implementation follows RFC 6238. Compatible with Google Authenticator, Bitwarden, and similar TOTP clients.

### WebAuthn (passkeys)

Vyst Open Auth implements the WebAuthn standard for passwordless authentication using platform authenticators (Apple FaceID/TouchID, Windows Hello) and hardware security keys (YubiKey). The relying party ID and origin are configurable via `WEBAUTHN_RP_ID` and `WEBAUTHN_ORIGIN`.

> **Production requirement:** `WEBAUTHN_RP_ID` must be set to your actual domain. `WEBAUTHN_ORIGIN` must use HTTPS. Using `localhost` or HTTP origins in production is insecure.

### Database isolation

Multi-tenant data isolation is enforced at the PostgreSQL layer using **Row-Level Security (RLS)** policies. Every tenant-scoped table requires a matching `tenant_id` in the active transaction context.

The runtime application connects using the **`vyst_app`** database role. This role has only `SELECT`, `INSERT`, `UPDATE`, and `DELETE` privileges. It cannot alter schemas, create roles, or bypass RLS policies.

Database migrations run using the **`postgres`** superuser role. This role is never used at runtime.

### Session revocation

Active tokens can be revoked in real time through the Redis-backed blacklist. The Sentinel Worker publishes kill-switch signals over Redis Pub/Sub, and all API server instances immediately blacklist the affected user's tokens.

### CAPTCHA

Login, registration, and password reset endpoints verify a **Cloudflare Turnstile** token before processing credentials. CAPTCHA is disabled when `TURNSTILE_SECRET_KEY` is not configured, which is acceptable for local development but not for production.

### Security headers

The HTTP server sets security-relevant headers on all responses. These include:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`

HTTPS and HSTS enforcement must be configured at the infrastructure level (reverse proxy or load balancer). The application itself does not terminate TLS.

---

## Deployment security requirements

The following conditions must be met in production. Failure to meet these conditions weakens the security guarantees described above.

- `DATABASE_URL` must use the `vyst_app` role, not `postgres`.
- `DATABASE_URL` must include `sslmode=require` or `sslmode=verify-full`.
- `JWT_PRIVATE_KEY` must be a unique RSA key generated for this deployment. Do not reuse development keys.
- `TURNSTILE_SECRET_KEY` must be configured with a real Cloudflare Turnstile secret.
- `WEBAUTHN_ORIGIN` must use HTTPS.
- The API must be deployed behind a TLS-terminating reverse proxy or load balancer.
- Redis must be accessible only from the application network and must not be exposed publicly.

---

## What is not covered

The following threats are not mitigated by Vyst Open Auth directly:

- **Authorization logic beyond role and permission checks.** The `POST /api/v1/authz/check` endpoint evaluates configured permissions, but business-level authorization (for example, whether a user may access a specific resource in a downstream service) is the responsibility of the consuming application.
- **Brute force beyond rate limiting.** The API implements per-IP rate limiting via Redis. Distributed brute force attacks across many source IPs are not mitigated by the application layer.
- **TLS termination.** Handled by infrastructure.
- **Key rotation automation.** RSA key rotation requires redeployment with a new key pair. Tokens signed with old keys will become invalid after the key changes.

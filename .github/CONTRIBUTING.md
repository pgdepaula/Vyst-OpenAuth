# Contributing to Vyst Open Auth

This file mirrors the contributing guidelines in the root [CONTRIBUTING.md](../CONTRIBUTING.md). Please read that file for the complete guide, including architecture rules, testing requirements, code conventions, and pull request process.

---

## Quick reference

**Architecture rule:** Dependencies must point inward only. `domain` → `application` → `infrastructure`/`interfaces`. No exceptions.

**Branch naming:**
- `feature/description` for new features
- `bugfix/description` for bug fixes
- `docs/description` for documentation changes

**Required before opening a PR:**
- All tests pass: `make verify-fast`
- Lint passes: `make lint`
- Documentation updated if the change affects public behavior, environment variables, or security

For security-sensitive changes, see the Security-sensitive contributions section in [CONTRIBUTING.md](../CONTRIBUTING.md).

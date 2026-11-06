# School Management System

A multi-tenant School Management System being developed as a production-oriented modular monolith.

## Product scope

- **Phase 1:** Web application for Platform Admin, School Admin and school staff.
- **Phase 2:** Parent and Driver mobile experiences, planned after the Phase 1 web product. Parent OTP/SMS delivery and Parent/Driver interfaces are intentionally paused.
- **Planning scale:** 25–30 schools, approximately 400–500 students per school. These are targets, not tested capacity limits.

## Technology

| Area | Technology | Status |
|---|---|---|
| Backend API | Go, Gin | Implemented and actively developed |
| Database | PostgreSQL, pgx/pgxpool | Implemented |
| Migrations | golang-migrate | Versioned migration chain |
| Web frontend | React, TypeScript, Vite | Planned; frontend source is not yet present |
| Mobile apps | To be decided | Phase 2 |

## Current repository status

**As of 2026-11-06:** the backend includes principal-aware authentication, refresh-session family validation and rotation, mandatory administrator TOTP MFA, MFA recovery and encryption-key rotation, live database-backed RBAC, delegation and permission overrides, staff and Platform Admin account lifecycle, school onboarding, and selected academic, attendance and guardian APIs.

The latest merged backend milestone also exposes the caller's current effective permission keys through `GET /auth/me` for frontend navigation bootstrap. Those keys are informational only; every protected operation is still authorized by the backend.

The backend is **not yet the complete school product and is not production-ready**. Full resource/tenant/assignment/field-level authorization across all modules, remaining academic/business modules, deployment hardening, and the web frontend remain to be completed.

## Documentation map

- [Project status](docs/project/STATUS.md) — authoritative implementation status, verification evidence and next work.
- [Product requirements](docs/product/REQUIREMENTS.md) — approved scope and delivery boundaries.
- [Phase 1 roles and permissions](docs/product/PHASE1-RBAC.md) — role, account and authorization requirements.
- [Architecture](docs/architecture/ARCHITECTURE.md) — system structure and tenant boundaries.
- [Backend documentation](docs/backend/README.md) — API, database, security and testing references.
- [Frontend documentation](docs/frontend/README.md) — frontend status and planned integration contract.

## Development principles

1. Continue from this repository; do not replace it with a parallel implementation.
2. Keep implemented behavior separate from approved requirements and future plans.
3. Enforce authorization and school/resource scope on the server; frontend visibility is never a security boundary.
4. Use additive, reviewed database migrations. Never run migrations against a persistent database without explicit approval.
5. Keep credentials, encryption keys, `.env` files and real student data out of Git.
6. Verify changes with automated tests and `go vet`; do not claim production readiness from compilation or tests alone.

## Local backend checks

From `backend\` on Windows:

```cmd
go test -count=1 -p 1 ./...
go vet ./...
```

PostgreSQL integration tests are opt-in and require a dedicated test database. Follow [backend testing documentation](docs/backend/TESTING.md). Never point tests at a production database.

## Getting started

See [backend/README.md](backend/README.md) for configuration and API startup. The frontend is not yet implemented, so there is currently no frontend development server to run.

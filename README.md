# School Management System

A commercial, multi-tenant School Management System built incrementally with production-oriented architecture, security practices, and automated testing.

## Technology Stack

- **Frontend:** React + TypeScript + Vite (planned; not yet implemented)
- **Backend:** Go + Gin
- **Database:** PostgreSQL
- **Database access:** pgx / pgxpool
- **Migrations:** golang-migrate
- **Password hashing:** bcrypt
- **Authentication:** JWT access tokens and opaque refresh tokens with rotation and refresh-session revocation

## Initial Target

- 25–30 schools
- Approximately 400–500 students per school
- Approximately 10,000–15,000 students in total

These are planning targets, not verified capacity limits. Production capacity will require load testing and monitoring.

## Project Principles

- Enforce school-level data isolation on the backend.
- Use version-controlled database migrations.
- Never commit secrets, credentials, or real student data.
- Test implemented behavior before moving to the next feature.
- Document architecture, API behavior, database changes, and security decisions.
- Implement features incrementally and verify each milestone.
- Keep implemented functionality clearly separated from planned functionality.

## Approved Product Delivery Direction (2026-11-03)

**Phase 1 is the complete web-based school management product.** It includes:
- Platform Admin dashboard for platform operations and school onboarding.
- School Admin dashboard for complete school administration.
- Role-based dashboards and workflows for school staff, including teachers and future school-scoped operational roles as their modules are delivered.

**Phase 2 follows after the web product is complete:** mobile applications/interfaces for Parents, Drivers and other approved user groups. Parent access is intended to be read-only initially; its authentication and OTP provider will be decided when Phase 2 is planned. Parent-specific implementation is paused in the meantime. Existing Parent/Guardian schema and OTP foundation must be preserved, not deleted or expanded during Phase 1 without a separately approved requirement.

The database must be designed incrementally but with future compatibility in mind: stable student identity separate from academic enrollment, school-scoped ownership, extensible roles/permissions, guardian relationships, transport-ready concepts, audit history and API contracts reusable by web and mobile clients. Future-proofing does not authorize speculative tables or premature migrations. Every schema change requires source review, additive migration design, tests on the dedicated test database and explicit approval before applying migrations.

## Current Status

**Stage (2026-11-05):** Principal-aware authentication, administrative MFA, refresh-family security, live RBAC/delegation/override workflows, staff account lifecycle, academic foundations, assigned-teacher enrollment roster, attendance foundations, and school-scoped Guardian Management APIs are implemented in the backend. The React frontend is not yet implemented and several school business modules remain pending. Parent OTP/SMS and Phase 2 Parent/Driver interfaces remain paused. Active branch: `fix/v4-auth-security-hardening`. This is not production-ready.

### Implemented

- Go + Gin API server.
- Environment-based application configuration.
- PostgreSQL connection pool using pgxpool.
- Health and database-readiness endpoints.
- HTTP server timeouts and graceful shutdown handling.
- Versioned PostgreSQL migrations for shared principals, RBAC, account subtypes, MFA/OTP foundations, audit history, student/guardian records and new-school system-role provisioning.
- User and school repository operations.
- bcrypt password hashing and verification, including password-length validation.
- Authentication service operations for user creation and credential verification.
- Principal-aware JWT access-token generation and validation; tokens carry principal identity/type and token version, while permissions are loaded live from PostgreSQL.
- Staff login by globally unique username and a separate Platform Admin email login.
- Principal-aware refresh sessions with rotation, replay/family revocation, and preserved transaction-lock concurrency protections.
- Database-backed principal status/token-version validation and effective system-role permission loading on protected requests.
- RequirePermission middleware and authenticated /auth/me route.
- Mandatory TOTP MFA enrollment/verification and recovery-code flows for Platform Admin and School Admin.
- Five-failure/15-minute username lockout, atomic IP login rate limiting, and fail-closed behavior if either security control is unavailable.
- Principal-wide five-failure/15-minute MFA lockout across replacement challenges and MFA challenge purposes, with append-only failure auditing and reset after successful verification.
- Mandatory TOTP MFA for Platform Admin and School Admin; admin MFA enrollment is revalidated on protected requests.
- Per-device 30-minute idle timeout for Platform Admin and School Admin sessions, bound to the refresh-session family.
- Live RBAC administration: custom roles, exact-delegation-bound permission grants, user ALLOW/DENY overrides, staff role assignment/revocation, and route-level permission checks.
- Database and repository guards against self-assignment, self-granted permission overrides, and self-removal of DENY overrides (V22).
- Staff account creation, listing, update, disable/re-enable, password change/reset, forced temporary-password change, token-version invalidation, and audit events.
- Teacher enrollment roster limited to assigned class/subject sections and the approved minimal fields: student name, roll number, class and section.
- Parent authentication and guardian matching require an active linked school. Parent OTP delivery and Parent/Driver Phase 2 flows remain intentionally paused.
- Transactional Platform Admin school onboarding, with automatic School Admin/Teacher system-role provisioning for new schools.
- One-time first Platform Admin bootstrap CLI; it does not create a permanent Platform Admin account automatically.
- `POST /auth/login` endpoint that issues access and refresh tokens.
- `POST /auth/refresh` endpoint for refresh-token rotation and replay detection.
- `POST /auth/logout` endpoint to revoke every active refresh session in the supplied token family.
- Generic unauthorized responses for invalid login credentials.
- Login response DTO that excludes password hashes.
- Authentication middleware that validates Bearer access tokens, reloads principal status/token version and effective permissions, and places canonical principal context in the request.
- Protected `GET /auth/me` endpoint returning principal ID/type and staff-only school/username context.
- Unit and handler tests for implemented authentication behavior.
- Authentication middleware and API route tests.
- Development-only seed command with environment safeguards.
- School-scoped Guardian Management APIs for guardian contacts and student relationships, including audited creation/update, primary/emergency contacts, history-preserving revocation, dedicated RBAC permissions, and V19 controlled delegation.
- Documentation for requirements, architecture, database design, API behavior, security, testing, progress, and handoff.

### Authentication Endpoints

#### `POST /auth/login`

Accepts a globally unique staff username and password (no school ID). Privileged School Admin login requires TOTP MFA; successful authentication after MFA returns principal-based access and refresh tokens. Invalid credentials return a generic unauthorized response.

#### `POST /auth/refresh`

Exchanges a refresh token for a new access token and replacement refresh token. Refresh tokens are opaque, cryptographically random values; only their SHA-256 hashes are persisted.

A reused revoked refresh token triggers revocation of active sessions in its token family. Refresh and logout operations serialize on a PostgreSQL transaction advisory lock scoped to the token family. Rotation additionally acquires a per-user transaction advisory lock to serialize rotations across that user's token families. Rotation performs a fresh row-locked lookup after acquiring the locks. Opt-in PostgreSQL repository integration tests cover refresh-session rotation, replay/family revocation, family-wide logout, idempotency, expiry, and concurrent refresh behavior. These tests do not cover every possible database failure or concurrency interleaving.

#### `POST /auth/logout`

Revokes all active refresh sessions in the token family associated with the supplied refresh token. Repeating logout is safe; an unknown token is a no-op. Already-issued access JWTs remain valid until expiry because access-token revocation enforcement has not been implemented.

#### `GET /auth/me`

Requires an `Authorization: Bearer <access-token>` header. The middleware validates the token, reloads current principal and authorization state, and returns principal ID/type plus staff-only school/username context.

This endpoint returns identity context only; complete user-profile retrieval and business-resource authorization remain pending.

Platform Admin uses POST /auth/platform-admin/login with email/password and must complete TOTP enrollment or verification through /auth/mfa/enroll/verify or /auth/mfa/verify. MFA recovery-code regeneration routes are also present. See docs/API.md for request and response contracts.

## Database Migrations

The repository includes additive migrations through V22. GitHub Actions run `41960953660` applied migrations V1–V22 to a fresh ephemeral PostgreSQL 18 database. This verifies the migration chain in CI only; it does not establish or change the version of either persistent local database. The one-time first Platform Admin bootstrap command is separate from migrations and must only be run deliberately by an authorized operator; no permanent Platform Admin account is created automatically. Never apply migrations to `school_management` without separate explicit approval.

## Verification

Latest verified remote checkpoint: commit `1b942cab4ebd76f849c579d1aa3324e839fc2741`, GitHub Actions run [41960953660](https://github.com/TheSumitNain/school-management-system/actions/runs/41960953660). CI applied V1–V22 to ephemeral PostgreSQL 18, ran the complete backend suite including opt-in PostgreSQL integration tests, and passed `go vet ./...`. This does not verify the user's local working tree or migrate either persistent local database. Pull the latest branch and repeat local tests before relying on local verification.

## Not Yet Implemented

- Complete user-profile retrieval beyond the identity context currently returned by `GET /auth/me`.
- Immediate revocation of already-issued access JWTs on ordinary staff logout; refresh-family logout revokes refresh sessions, while already-issued access tokens otherwise expire naturally. Principal token-version changes do invalidate access tokens on subsequent requests.
- Full resource-level tenant, assignment and field-level authorization across every current and future business endpoint; implemented student/enrollment/attendance/guardian scopes still need broader coverage.
- Remaining school business modules such as fees, exams/results, timetable, homework, notices, transport and reporting.
- Exhaustive HTTP end-to-end tests with real JWT signature validation across every protected route, plus additional race/failure-mode testing.
- Production database least-privilege grants, deployment hardening, TLS/proxy configuration, monitoring, alerting, backup and recovery procedures.
- React frontend and Platform Admin, School Admin, teacher and school-staff dashboards (Phase 1).
- Parent/Driver mobile interfaces and production Parent OTP/SMS flow (Phase 2; intentionally deferred).
- Comprehensive security review, penetration testing, capacity/load testing and operational readiness verification.

**Important:** This project is under active development and is not production-ready. Authentication components have been implemented incrementally, but the complete authentication and authorization system has not yet been validated as production-ready.

## Documentation

- `PROGRESS.md` — implementation progress and next steps.
- `docs/REQUIREMENTS.md` — product requirements.
- `docs/ARCHITECTURE.md` — architecture and component boundaries.
- `docs/DATABASE.md` — schema and migration decisions.
- `docs/API.md` — API conventions and endpoint contracts.
- `docs/SECURITY.md` — security requirements and implementation status.
- `docs/TESTING.md` — testing strategy and verification.
- `docs/HANDOFF.md` — handoff instructions for continuing in a new chat.
- `docs/PHASE1-RBAC.md` — approved Phase 1 web roles, permission catalogue, teacher scopes, staff account rules, and future Parent mobile access model.
- `CHANGELOG.md` — milestone history.

## Development

Run backend commands from the backend directory:

```cmd
cd /d D:\school-management-system\backend
go test -count=1 ./...
go vet ./...
```

Run database migrations from the same directory:

```cmd
go run ./cmd/migrate
```

Use the project's configured development workflow to start the API server.

Keep local environment files and credentials out of Git. Never share secrets in chat or commit them to the repository.

Review migration changes carefully before applying them to any production database. Test production migration procedures against a separate, backed-up database.

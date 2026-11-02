> **2026-11-02 current checkpoint:** The user pulled branch `fix/v4-auth-security-hardening` through commit `79c2717` (MFA recovery-code integration fixture repeatability fix). Migration V11 was successfully applied to the dedicated `school_management_test` database; `schema_migrations` reported version `11`, `dirty = false`. With `DB_NAME=school_management_test` and `RUN_POSTGRES_INTEGRATION_TESTS=1`, the user ran `go test -count=1 -p 1 ./...`; all packages with tests passed, including repository integration tests. `go vet ./...` completed without diagnostics. The focused recovery-code integration test also passed twice consecutively. These results are user-reported on Windows. The local working tree still has user-owned helper/untracked files; inspect status before any commit. Complete business APIs, route-level RBAC, tenant isolation, and parent relationship authorization remain unfinished.

# Project Progress

## Current Stage

**Current milestone:** V11 student/guardian foundation is applied and PostgreSQL-tested on the dedicated `school_management_test` database at migration version 11 (`dirty = false`). The full Go suite with PostgreSQL integration enabled and `go vet ./...` passed at local commit `79c2717` plus local `gofmt` of the recovery-code integration test. The focused recovery-code integration test passed twice consecutively. Complete business APIs, route-level RBAC, tenant isolation, and parent relationship authorization remain pending.

The project has a working authentication foundation, including login, JWT authentication middleware, a protected identity endpoint, refresh-token rotation, replay detection, family-wide logout revocation, family-scoped transaction advisory locking, and per-user serialization of concurrent rotations across token families.

The last recorded remote documentation checkpoint before the current local verification was:

- **Commit:** `cb6af3b` — `docs: finalize phase 1 RBAC and account decisions`
- **Full commit hash in ZIP root:** `cb6af3b5abc5249fba420e2699d16e87b1d14664`
- **Branch:** `main`
- **Repository:** https://github.com/TheSumitNain/school-management-system
- **Source snapshot:** supplied ZIP `school-management-system-audit(1).zip`; it contains source and docs but no `.git` directory.
- **Local working-tree state:** not established by the ZIP. The user previously reported an untracked `APPLY-DOCS.txt`; inspect `git status -sb` in the actual local repository before committing. Do not assume the working tree is clean.

The refresh-session work includes opt-in PostgreSQL integration tests for rotation, replay/family revocation, family-wide logout, idempotency, expiry, and concurrent refresh requests.

The user reported the full backend test suite and `go vet` passing after the concurrency changes. The dedicated PostgreSQL integration run included six refresh-session lifecycle subtests plus the cross-family inactive-user race regression test. `git diff --cached --check` passed and the `cb6af3b` commit was pushed. These are historical user-reported results; rerun them in the actual local repository before relying on them.

**Important:** Passing tests does not establish production readiness. RBAC, tenant authorization, account-security controls, deployment hardening, and other production requirements remain outstanding.

## Project Overview

- **Product:** Commercial multi-tenant School Management System
- **Initial planning target:** 25–30 schools, approximately 400–500 students per school
- **Frontend:** React + TypeScript + Vite
- **Backend:** Go + Gin
- **Database:** PostgreSQL
- **Database access:** pgx / pgxpool
- **Architecture:** Modular monolith using a shared database and shared schema initially
- **Tenant isolation:** Must be enforced by the backend
- **Repository:** https://github.com/TheSumitNain/school-management-system
- **Main branch:** `main`
- **Default local API port:** `8081`
- **Local repository:** `D:\school-management-system`
- **Backend directory:** `D:\school-management-system\backend`
- **Go module:** `school-management-system`

The school and student counts are planning targets, not verified capacity limits.

## Completed

### Planning and Documentation

- [x] Initial product requirements documented
- [x] Architecture principles documented
- [x] Database design principles documented
- [x] API conventions and endpoint documentation created
- [x] Security requirements documented
- [x] Testing strategy documented
- [x] Project handoff documentation created
- [x] Progress and changelog files maintained
- [x] Documentation consistency audit against supplied `cb6af3b` source snapshot; RBAC decisions documented as approved requirements

### Backend Foundation

- [x] Go module and Gin API scaffold
- [x] Environment-based configuration
- [x] Configuration validation
- [x] PostgreSQL connection pool using pgxpool
- [x] Database connectivity and readiness check
- [x] `GET /health`
- [x] `GET /ready`
- [x] HTTP server timeouts
- [x] Graceful shutdown handling
- [x] Gin trusted proxies explicitly configured
- [x] Configurable application port, default `8081`
- [x] Development and production configuration distinctions

### Database and Migrations

- [x] Local `school_management` database configured
- [x] Version-controlled migration tooling
- [x] Migration v1 for the `schools` table
- [x] Migration v2 for `users` and `refresh_sessions`
- [x] Migration v3 for refresh-token family identifiers
- [x] Migration v3 applied to the development database and previously reported clean (`dirty = false`)
- [x] Migration v3 applied to the dedicated `school_management_test` database and reported clean (`dirty = false`)
- [x] User and school repository operations implemented
- [x] Refresh-session repository integration tests against PostgreSQL
- [ ] Automated migration tests against a dedicated clean test database
- [ ] Production migration and recovery procedure fully validated

**Migration safety:** Always verify the target database and migration state before deployment. Never run destructive down migrations against production without an approved backup and recovery plan.

### Authentication

- [x] User model and repository operations
- [x] School repository lookup
- [x] bcrypt password hashing and verification
- [x] Password validation and password-service tests
- [x] Authentication service for creating users in an active school
- [x] Authentication service for verifying credentials
- [x] Authentication service unit tests
- [x] JWT access-token generation and validation
- [x] JWT issuer, expiry and required-claim validation
- [x] HTTP login handler
- [x] `POST /auth/login`
- [x] Login response excludes password hashes
- [x] Invalid credentials return an unauthorized response
- [x] Login handler tests
- [x] Development-only seed command with environment safeguards
- [x] JWT authentication middleware for protected routes
- [x] Bearer access-token validation
- [x] Validated user ID and school ID stored in Gin request context
- [x] Missing or invalid authentication rejected by middleware
- [x] Invalid token identity claims rejected
- [x] Protected `GET /auth/me`
- [x] `/auth/me` returns authenticated user ID and school ID
- [x] Opaque refresh-token generation
- [x] SHA-256 hashes stored for refresh tokens rather than raw tokens
- [x] Configurable refresh-token expiry through `REFRESH_TOKEN_EXPIRY_DAYS`
- [x] Default refresh-token expiry of 7 days, with configuration validation
- [x] `POST /auth/refresh`
- [x] Refresh-token rotation
- [x] Token-family tracking for replay detection
- [x] Refresh-token family revocation on detected replay
- [x] `POST /auth/logout`
- [x] Session-service unit tests and authentication-handler tests
- [x] PostgreSQL integration coverage for refresh-session lifecycle
- [x] Concurrent refresh-token reuse detection tested against PostgreSQL
- [x] Cross-family inactive-user race regression test included: concurrent rotations across separate token families are serialized per user so an inactive-user revocation cannot be bypassed by a competing family rotation

**Known limitations:**

- `/auth/me` returns the authenticated identity; it does not yet provide a complete user profile.
- Authentication middleware establishes identity but does not itself enforce roles, permissions or complete tenant authorization.
- PostgreSQL integration tests cover the listed refresh-session scenarios, but they do not prove every possible database failure or concurrency scenario is safe.
- Immediate revocation of already-issued access tokens requires a separately designed and implemented strategy.
- Authentication endpoints do not mean that public registration or the full school-onboarding workflow exists.

## Latest Verification Results

The following checks were reported after the PostgreSQL integration test work was completed. These are historical results; rerun them after any code changes.

### Repository integration tests

Command, run from `D:\school-management-system\backend`:

```bat
go test -count=1 -v ./internal/repository
```

**Result:** PASS. A repeated run with `go test -count=20 ./internal/repository` was also reported successful.

Verified scenarios:

- [x] Refresh-token rotation revokes the old session
- [x] Reuse of a rotated token is detected
- [x] Replay detection revokes the replacement token in the same family
- [x] Logout revokes the complete refresh-token family
- [x] Logout revocation is idempotent
- [x] Expired refresh tokens are rejected
- [x] Concurrent refresh requests produce one successful rotation and one reuse-detection result
- [x] Cross-family inactive-user race regression is covered by a dedicated PostgreSQL-backed repository test

### Full backend tests

Command:

```bat
go test -count=1 ./...
```

**Result:** PASS.

Verified packages:

- [x] `cmd/api`
- [x] `internal/config`
- [x] `internal/handlers`
- [x] `internal/middleware`
- [x] `internal/repository`
- [x] `internal/services`

The `cmd/devseed`, `cmd/migrate`, `internal/database`, and `internal/models` packages reported no test files.

### Static analysis

Command:

```bat
go vet ./...
```

**Result:** PASS, with no reported diagnostics.

### Git verification

- [x] Integration-test file staged and reviewed
- [x] `git diff --cached --check` completed without whitespace errors
- [x] Latest user-reported pushed code checkpoint represented by supplied ZIP: `cb6af3b` (`docs: finalize phase 1 RBAC and account decisions`)
- [ ] Verify actual local Git status and remote alignment before committing documentation; the ZIP has no `.git` metadata

These results describe the last verified state. Repeat relevant checks after future code changes.

## Approved Phase 1 Product and Authorization Decisions (Implementation Pending)

The user has finalized the Phase 1 account and RBAC direction. The detailed authoritative matrix is `docs/PHASE1-RBAC.md`. Key decisions:

- Core Phase 1 roles: Platform Admin (global), School Admin (one school), Teacher (one school), Parent (verified linked children).
- Every staff account is school-specific. A person employed by multiple schools receives separate accounts, passwords, and sessions for each school. Staff usernames are globally unique and case-insensitive.
- School Admin provisions staff with a temporary initial password; staff must change it at first login and must enter the current password for self-service changes. School Admin can reset same-school staff passwords without the old password; reset revokes sessions/access, requires a change at next login, and is audited.
- Platform Admin provisions School Admin accounts. School Admin cannot create another School Admin or Platform Admin.
- School Admin may create school-scoped custom roles from the platform permission catalogue, only within an explicit delegation ceiling. Operational roles (Clerk, Manager, Librarian, Gate Keeper, Helper, Bus Conductor, etc.) are future scope, not Phase 1 implementations.
- Multiple roles per staff account are supported within that account's single school. Role permissions plus per-user allow/deny overrides are supported; explicit deny overrides role allow, explicit authorized allow can add a permission otherwise absent, and default is deny.
- Teaching assignment and class-in-charge assignment are separate. Subject teachers see only basic student fields (name, roll number, class, section) for assigned class/subject and relevant permitted teaching data. A teaching assignment alone never grants attendance access.
- A class-in-charge teacher can access attendance, same-day attendance correction within the configured cutoff, basic student details, timetable, homework, and applicable notices only for the specific class/section in charge. Past attendance correction is School Admin-only.
- One verified parent mobile/OTP account can access all officially linked children, including across schools. Parent selects a child; every API request verifies the relationship. Parent access is read-only; no student login accounts in Phase 1.
- Permissions are checked server-side on each protected request; permission changes apply to the next request. Tenant, assignment, field-level, account-status, and school-status checks cannot be bypassed by overrides.

These are approved requirements, not proof of complete RBAC enforcement. The V4 SQL draft now contains source-level corrective audit items C1–C14, and the 2026-11-01 Go principal-auth cutover candidate is documented in `docs/CURRENT-BUILD-FIX-HANDOFF-20261101.md`. The migration remains unapplied and PostgreSQL-unverified; full Go tests/vet for the cutover candidate must be verified on the user's Windows Go 1.27.1 environment. Custom-role grants and ALLOW overrides remain intentionally fail-closed, and business-resource/tenant/teacher/parent authorization routes remain pending.

## In Progress

- [x] Audit project documentation against the latest reported implementation and test coverage
- [x] Cross-check README, progress, changelog, requirements, architecture, database, API, security, testing and handoff documentation for consistency
- [ ] Review remaining authentication security edge cases against the current local working tree
- [ ] Re-review refresh-token integration-test isolation and cleanup after any test changes
- [ ] Add migration tests against a dedicated clean database
- [x] Verify that `backend/.env` is ignored by Git and is not tracked
- [ ] Run the full verification workflow after each implementation milestone

## Not Yet Implemented or Verified

### Authentication and Account Security

- [ ] Complete user-profile retrieval
- [ ] Production school onboarding and secure invitation workflow
- [ ] Password-reset workflow and one-time reset tokens
- [ ] Mandatory MFA for Platform Admin and School Admin
- [ ] MFA recovery codes, recovery workflows and audit logging
- [ ] Failed-login lockout: 5 failed attempts followed by a 15-minute lock
- [ ] Login and account/IP rate limiting
- [ ] Security-event logging and audit trail
- [ ] Account deactivation and complete session invalidation
- [ ] Admin idle-session timeout of 30 minutes
- [ ] Verify all agreed password policy requirements
- [ ] Complete access-token revocation strategy

### Authorization and Multi-Tenancy

- [ ] Implement application-level role-based access control (RBAC)
- [x] Prepare V4 identity/RBAC SQL migration draft (not applied or PostgreSQL-tested)
- [ ] Implement permission checks on protected endpoints
- [ ] Support authorized permission changes taking effect on subsequent protected requests
- [ ] Implement school-level tenant authorization
- [ ] Ensure client-provided school IDs never independently grant access
- [ ] Add cross-school authorization and data-isolation tests
- [ ] Restrict teachers to assigned classes and subjects
- [ ] Restrict parents to verified linked children
- [ ] Implement authorized role and school-membership management

### Database and School Features

- [ ] Complete production database schema
- [ ] Add automated migration tests against a clean database
- [ ] Student APIs
- [ ] Teacher and staff APIs
- [ ] Class and subject APIs
- [ ] Daily attendance APIs
- [ ] Parent-child relationship management
- [ ] Timetable, homework, exam results and notices
- [ ] Additional school workflows and reporting

### Frontend and Operations

- [ ] React frontend implementation
- [ ] Login and authentication integration
- [ ] Role-specific dashboards
- [ ] Parent-facing interfaces
- [ ] CI/CD and deployment configuration
- [ ] Production TLS and certificate verification
- [ ] Production database connection-pool tuning
- [ ] Centralized logging, monitoring and alerting
- [ ] Backup and recovery procedures
- [ ] Security testing and load testing
- [ ] Production readiness review

## Next Steps

1. Review `docs/V4-IMPLEMENTATION-NOTES.md` and the V4 migration draft. Do not apply it to the development database yet.
2. Complete principal-aware Go authentication/session refactoring and automated migration tests before applying V4.
3. Verify `backend/.env` is ignored and not tracked. Never paste secret contents into chat.
4. Run `git diff --check` and inspect `git status -sb`, including the previously reported untracked `APPLY-DOCS.txt`.
5. Re-run `go test -count=1 ./...` and `go vet ./...` from the backend directory after any code changes.
6. Run the opt-in PostgreSQL integration tests against the dedicated `school_management_test` database, including the cross-family inactive-user race test.
7. Review and commit the documentation diff only after checking the exact files; then push if the user approves the milestone.
8. Implement principal-aware login, JWT claims, refresh-session rotation/revocation, token-version checks, and DB-backed permission evaluation against the reviewed V4 schema.
9. Implement roles, permissions, custom roles, bounded overrides, staff password lifecycle, teacher assignment scopes, and parent-child authorization with automated tests.
10. Implement login rate limiting and failed-login lockout with automated tests.
11. Continue school-management APIs and frontend integration in small, verified milestones.
12. Complete production security, deployment, monitoring and recovery checks before release.

## Working Rules

- Work on one small task at a time.
- Explain code in Hindi/Hinglish, step by step.
- Wait for command output before proceeding.
- Do not mark features complete without verifying implementation and relevant tests.
- Keep `PROGRESS.md` and `docs/HANDOFF.md` synchronized at meaningful milestones.
- Never commit `.env` files, passwords, signing secrets or real student data.
- Never share secrets in chat.
- Never trust client-provided identifiers for authorization.
- Do not run destructive production migrations without an approved backup and recovery plan.
- Do not claim production readiness until security, testing, deployment, monitoring and recovery requirements have been addressed.

## V4 Corrective Source Audit — 2026-11-01/02

- [x] C1–C17 source-level migration review and corrective changes are recorded in `docs/V4-IMPLEMENTATION-NOTES.md`
- [x] Principal role assignment, permission delegation, permission catalogue, MFA recovery-code, and parent OTP history guard source changes are present
- [x] Corrected PL/pgSQL dollar-quote delimiters in the five identified V4 trigger functions
- [ ] Re-run the corrected V4 source through a clean PostgreSQL 18 v3-to-v4 upgrade and trigger failure-case suite after syncing the latest branch

The user subsequently pulled the V11 and recovery-test fix commits. On the dedicated `school_management_test` database, V11 is applied and clean (`schema_migrations.version = 11`, `dirty = false`). The full Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1` and `go vet ./...` both passed on the user's Windows environment. Do not apply migrations to `school_management` without explicit approval.

## V11 Student and Guardian Foundation — 2026-11-02

- [x] Added `000011_add_student_guardian_foundation.up.sql`
- [x] Added a guarded down migration that refuses rollback when V11 data exists
- [x] Added school-scoped `students` and `guardians` records
- [x] Added many-to-many `student_guardians` mapping with composite same-school foreign keys
- [x] Added immutable school ownership for student and guardian records
- [x] Added PostgreSQL integration test source for many-to-many links, duplicate prevention, cross-school isolation, normalized admission-number uniqueness, and school-scope immutability
- [x] Updated `docs/DATABASE.md` and `CHANGELOG.md`
- [x] Pull latest branch through recovery-code fixture fix `79c2717`
- [x] Apply V11 only to `school_management_test`; verified migration version `11`, `dirty = false`
- [x] Run full Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1` and `go vet ./...`; both passed
- [x] Run focused recovery-code integration test twice consecutively; both runs passed

V11 is **applied and PostgreSQL-verified on the dedicated test database**. This verification does not imply production readiness or complete parent authorization. It does not create academic-year enrollment, class/section placement, parent login endpoints, SMS/OTP delivery, or authenticated parent-to-child authorization. Guardian contact records remain separate from authenticated `parent_accounts`; V12 must add the reviewed verification/revocation relationship before parent access is enabled.

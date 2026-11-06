# Project Status

**Last reconciled:** 2026-11-06  
**Repository:** `TheSumitNain/school-management-system`  
**Default branch:** `main`  
**Latest known merged commit:** `f506dfeb5d2992b6ca9793c3651776e2b0f54b46`

This file is the single current implementation-status reference. Older checkpoint notes must not override it.

## Delivery scope

- **Phase 1:** Complete web product for Platform Admin, School Admin and school staff.
- **Phase 2:** Parent and Driver mobile experiences after Phase 1 web completion. Parent OTP/SMS delivery and Parent/Driver workflows remain paused.
- **Scale target:** 25–30 schools and 400–500 students per school; capacity is not yet load-tested.

## Repository reality

- Backend: Go + Gin, PostgreSQL + pgx/pgxpool.
- Frontend: React + TypeScript + Vite is the intended stack, but no frontend source tree exists yet.
- Deployment: production deployment architecture and operational controls are not yet complete.

## Implemented backend capabilities

### Identity and session security
- Principal-aware staff and Platform Admin authentication.
- JWT access tokens with database-backed principal/token-version checks.
- Opaque refresh tokens, rotation, replay/family revocation, transaction-lock concurrency protections, and protected-request refresh-family validation.
- Mandatory TOTP MFA for Platform Admin and School Admin, recovery-code workflows, attempt lockout/auditing, and versioned TOTP encryption-key rotation.
- Staff and Platform Admin password lifecycle, temporary-password enforcement, token-version invalidation and session revocation.
- Login lockout, IP rate limiting, breached-password checks and administrative idle timeout.

### RBAC and platform administration
- Live database-backed permission resolution on protected requests.
- System/custom roles, scoped delegation, exact-delegation-bound grants, user ALLOW/DENY overrides and audit events.
- Route-level permission middleware and repository/database guards against self-escalation.
- Platform Admin lifecycle and transactional school onboarding with school system-role provisioning.
- `GET /auth/me` returns sorted current permission keys for Staff and Platform Admin principals as a frontend navigation hint; server authorization remains authoritative.

### Selected school workflows
- School-scoped student, enrollment, promotion/history, class, section and subject foundations.
- Teacher-subject and class-in-charge assignments.
- Teacher enrollment roster restricted to assigned sections and minimal approved fields.
- Attendance APIs with targeted permission and scope checks.
- School-scoped guardian management APIs and parent/guardian eligibility foundations.

## Not yet complete

- React web frontend, dashboards, forms and end-to-end browser workflows.
- Complete tenant/resource/assignment/field-level authorization coverage across every business module and route.
- Remaining academic/business modules, including timetable, examinations and results.
- Parent OTP/SMS delivery and Parent/Driver Phase 2 interfaces.
- Exhaustive end-to-end HTTP coverage with real JWT signature validation for every protected route.
- Production deployment hardening, least-privilege database grants, monitoring/alerting, backup/restore drills, penetration testing and load/capacity testing.

## Verification evidence

- User-reported local Windows checks on the merged backend: `go test -count=1 -p 1 ./...` passed and `go vet ./...` passed.
- Latest known GitHub Actions verification for the merged backend applied the migration chain to an ephemeral PostgreSQL database, ran backend tests including PostgreSQL integration tests, and ran `go vet ./...` successfully.
- These checks do not prove production readiness or load capacity.
- CI uses disposable infrastructure; do not infer the migration version of the user's persistent `school_management` or `school_management_test` databases from CI.

## Safe next sequence

1. Finish documentation reconciliation so each document has one clear purpose and no stale checkpoint claims.
2. Reconcile remaining backend authorization gaps against the actual route/repository surface; add tests for each fix.
3. Complete remaining backend business modules and their tenant/assignment/field-level boundaries.
4. Design and implement the React web foundation against the actual API contracts.
5. Complete browser-level workflows, operational hardening and production readiness checks.

## Git and database safety

- Continue on `main` unless a separately agreed workflow requires otherwise.
- Do not stage unrelated helper files or commit secrets.
- Do not run migrations against persistent databases without explicit approval.
- Keep `.env`, passwords, JWT secrets, TOTP encryption keys and real student data out of Git.

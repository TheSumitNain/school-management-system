# Project Status

**Last reconciled:** 2026-11-08  
**Repository:** `TheSumitNain/school-management-system`  
**Reviewed branch:** `feature/school-attendance-management`  
**Reviewed branch HEAD before this status reconciliation:** `4112ac549fc46e19bd9cbb1ef2865a48b9119675`

This file is the current implementation-status reference. Older checkpoint notes must not override it. Verification statements below distinguish repository inspection, user-reported local checks and real browser/database operations.

## Delivery scope and current priority

- **Phase 1:** Complete web product for Platform Admin, School Admin and school staff.
- **Immediate priority:** Complete the School Admin workspace and verify its end-to-end operational workflows.
- **Phase 2:** Parent and Driver experiences only after Phase 1 school web delivery.
- **Parent OTP/SMS:** Explicitly deferred. Do not implement parent OTP login, SMS delivery or parent-facing authentication now. Preserve existing guardian/parent relationship foundations without expanding this workstream until Phase 2 is approved.
- **Scale target:** 25–30 schools and 400–500 students per school; capacity is not yet load-tested.

## Next scheduled task — persistent authenticated sessions (planned, not implemented)

The next security task is to revise session persistence while preserving the existing authentication and authorization safeguards. This work has **not started** and must not be reported as complete until code, tests and documentation are updated and verified.

- Keep users signed in across browser close/reopen and laptop shutdown/restart, subject to the browser retaining its persistent cookie. Renew the persistent refresh cookie during successful token refresh; do not promise persistence beyond browser-enforced cookie policies.
- Platform Admin and School Admin must complete mandatory MFA on every fresh login. MFA must not be repeated during background access-token refresh or restoration of an existing valid session.
- Remove the current 30-minute administrative idle timeout and the current server-side absolute refresh-session expiry, without weakening session-family validation.
- A session remains valid until the user manually logs out or a security/account event requires revocation (for example, password/security reset, account disablement or confirmed compromise).
- Manual logout must revoke the session family. Preserve opaque refresh-token rotation, replay detection/family revocation, transaction/concurrency protections, HttpOnly + Secure cookie policy, cross-tab logout synchronization, principal token-version checks and live database-backed RBAC authorization.
- Update backend and frontend behavior, regression/integration tests, and the relevant security/session documentation. Verify fresh-login MFA separately from background refresh and browser session restoration.
- Work on the existing `feature/school-attendance-management` branch. Do not create a new branch or merge PR #6; neither is part of this task.

## Future product direction (documented, not current implementation scope)

- **Self-service school registration is deferred until the School Admin dashboard and the Phase 1 school web experience are complete and stable.** Do not build public registration, email-verification onboarding, or self-service tenant creation during the current workstream.
- The current supported school-creation path remains Platform Admin-assisted onboarding. Complete and verify that path first, including its real PostgreSQL transaction and failure handling.
- When the deferred self-service phase is approved, schools should be able to register and create their own initial School Admin account without manual Platform Admin intervention. Email ownership verification will be required, but email verification alone must not be treated as proof that an institution is genuine. Abuse controls and any school-authenticity review policy must be designed before release.
- Platform Admin-assisted onboarding must remain available for assisted/special-case onboarding.
- Both onboarding entry points must call one shared, server-authoritative school-onboarding service so tenant creation, initial role provisioning, transaction boundaries, validation and audit behavior remain consistent.
- The public school website is also a future product capability and should use the same product domain (school-specific paths such as `/s/{school-slug}`); do not introduce a separate domain per school as a requirement.
- Record this direction in the roadmap only. Do not create placeholder registration routes, forms, mock flows or database changes before the deferred phase is explicitly started.

## Repository reality

- Backend: Go + Gin, PostgreSQL + pgx/pgxpool.
- Frontend: React + TypeScript + Vite. A working frontend source tree exists, including the shared application shell, session-aware login/MFA flow, theme support, Platform Admin screens and School Admin feature pages.
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
- `GET /auth/me` returns current effective permission keys for Staff and Platform Admin principals as a frontend navigation hint; server authorization remains authoritative.

### Selected school workflows
- School-scoped staff account lifecycle and password reset.
- School-scoped system/custom role management, permission delegation and staff permission overrides.
- Academic years, classes, sections, subjects, teacher-subject assignments and class-in-charge assignments.
- Student records, enrollment, promotion/history and teacher roster restrictions.
- Attendance APIs with targeted permission and scope checks.
- School-scoped guardian management APIs and parent/guardian eligibility foundations.

## Frontend implementation present on the reviewed branch

- Shared login and authenticated application shell with role/permission-aware navigation.
- Platform Admin account list and school onboarding form.
- School Admin pages for staff accounts, roles/permissions, academic structure (including teacher-subject and class-in-charge assignment management), students/enrollments and attendance.
- Separate page CSS, shared design tokens and Light/Dark theme foundations.
- Typed API modules and TanStack Query-based server state for school administration.

The presence of a page or API client does not mean its full browser workflow has been accepted. Continue testing against the real local API and verify every permission boundary. Do not present mock data or frontend-only checks as completed backend workflows.

## Recent verification evidence

### User-reported local checks
- Full backend: `go test -count=1 -p 1 ./...` passed.
- Backend static analysis: `go vet ./...` passed.
- School onboarding handler tests: `go test -count=1 ./internal/handlers` passed after the latest pull. Windows printed a temporary test executable cleanup warning (`go: unlinkat ... handlers.test.exe`); the test package itself passed.
- Frontend production build passed after the latest pull, including the student-search enrollment updates (Vite 7.3.7; 1,657 modules transformed).
- School onboarding UI required-field checks were manually exercised for school name, school code minimum length, username minimum length and password minimum length.
- Student enrollment selection now uses server-side search (name/admission number) instead of loading only the first 100 students; a student must be selected before submission. The user pulled these changes and confirmed the production build passes.
- Platform Admin login and MFA verification succeeded against the user's `school_management` database.

### Scope and limitations of that verification
- No school was created during onboarding UI validation. A successful real onboarding transaction and live duplicate-code/username checks have not been demonstrated in this workflow.
- The onboarding handler/service tests cover success/error mapping and validation paths, but do not by themselves prove a real database transaction in the user's persistent database.
- The initial Platform Admin list request returned one HTTP 500 followed by HTTP 200; the page subsequently loaded, but the first response's cause has not been established.
- These checks do not prove production readiness, exhaustive authorization coverage or load capacity.
- Never infer the migration version or contents of the user's persistent databases from disposable CI environments.

## Not yet complete

- End-to-end browser verification of all School Admin workflows, including staff creation/role assignment, academic setup, student enrollment and attendance.
- Full browser verification of Platform Admin onboarding, lifecycle and delegation workflows.
- Complete tenant/resource/assignment/field-level authorization coverage across every business module and route.
- Remaining academic/business modules, including timetable, examinations and results.
- Teacher/staff-specific operational screens and assignment-limited browser workflows beyond the new School Admin assignment-management screen.
- Exhaustive HTTP coverage with real JWT signature validation for every protected route.
- Production deployment hardening, least-privilege database grants, monitoring/alerting, backup/restore drills, penetration testing and load/capacity testing.
- Parent OTP/SMS delivery and Parent/Driver Phase 2 interfaces remain deferred and are not current work.

## Safe next sequence

1. Continue reviewing and completing the School Admin browser workflows against the existing API contracts.
2. Fix verified frontend/backend integration defects and add regression tests for each fix.
3. Reconcile remaining backend authorization gaps against the actual route/repository surface.
4. Complete the remaining school business modules and teacher/staff workflows needed for Phase 1.
5. Complete browser-level, responsive, accessibility, theme, session-recovery and operational verification.
6. Start Parent OTP/SMS and Parent/Driver Phase 2 work only after Phase 1 school web product completion and explicit reprioritization.

## Git and database safety

- Prefer the agreed primary development branch workflow; do not create unnecessary branches or merge a pull request without explicit approval.
- Do not stage unrelated helper files or commit secrets.
- Do not run migrations against persistent databases without explicit approval.
- Keep `.env`, passwords, JWT secrets, TOTP encryption keys and real student data out of Git.

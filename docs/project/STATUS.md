# Project Status

**Last reconciled:** 2026-11-14  
**Repository:** `TheSumitNain/school-management-system`  
**Reviewed branch:** `work`  
**Reviewed through:** `d4fcf7e0` on `work`; V32–V38 were locally verified in the user's disposable `school_management_test` database; the latest completed backend CI (including PostgreSQL integration and `go vet`) passed on `2a0ced44`; the frontend production build passed on `d4fcf7e0`; backend CI for the latest frontend styling commit is pending; local V39–V41 database application and browser acceptance remain pending

## Administrator access correction — School Admin guardian permissions (V39; local verification pending)

- Root cause of the missing Parent/Sibling area: the guardian UI and APIs existed, but the School Admin system-role bundle and its runtime allowlist omitted `guardians.read` and `guardians.manage`.
- V39 adds both school-scoped permissions to every existing School Admin system role and updates the school-creation provisioning function so newly onboarded schools receive the same bundle. The immutable system-role protection trigger is removed only around the reviewed migration insert and is recreated immediately.
- Teacher permissions remain unchanged; Teacher system roles receive neither guardian permission. The School Admin runtime permission allowlist and provisioning integration expectations now include both keys.
- Added PostgreSQL integration coverage for existing-school grants, Teacher exclusion and restoration of the system-role immutability trigger. Newly created-school provisioning coverage also expects the updated 53-permission School Admin bundle.
- No guardian/student records are modified. No frontend permission check or backend authorization middleware is bypassed.
- **V39 has been committed to `work` but has not been run against any local database.** Verify it only on the disposable `school_management_test` database first. Do not apply it to the persistent `school_management` database without separate explicit approval and a verified backup.
- **Platform Admin cross-school business operations remain an explicit architecture gap:** Platform Admin is a platform-scoped principal, while current school business APIs require a school-scoped staff context. Do not impersonate a School Admin or bypass tenant checks. Implement a reviewed selected-school administrative context with explicit target-school authorization and audit before exposing Platform Admin to school business records.

## Student profile UI and guardian relationship verification (2026-11-14)

- The guardian relationship edit regression test was corrected to assert that the previously selected primary contact is cleared, while the newly selected primary contact remains selected. The backend implementation clears the previous primary only within the same student's active guardian links.
- The full backend CI run on commit `3fd23690` passed: complete migrations on ephemeral PostgreSQL, backend tests including PostgreSQL integration, and `go vet`.
- The database engineering standards in [DATABASE.md](../backend/DATABASE.md) now cover immutable versioned migrations, integrity and tenant constraints, transaction/concurrency rules, query/index performance, fresh-install and upgrade verification, and backup/restore safety.
- Address & Communication and Medical & Physical reuse the `.student-address-tabs` class. Its missing styles have now been added with active, hover, keyboard-focus and narrow-screen states. The frontend production build passed on commit `d4fcf7e0`; backend CI for that exact commit is still pending.
- Source review confirms the Basic Details and Address/Contact save actions intentionally remain unavailable while their initial read request is loading or has failed. This prevents saving blank/default form state over data that could not be loaded. The UI exposes the read error and retry action. The precise cause of the previously reported disabled Save buttons has not been established without a live authenticated browser session; do not remove these safeguards or claim the issue resolved until browser acceptance verifies the read endpoints and permissions.
- Browser acceptance remains pending: verify view/edit permissions, load and retry behavior, successful save and reload persistence for Basic Details, current/permanent address, contact details, and Parent Details. Parent OTP/authentication remains deferred.

## Student Management — approved yearly admission model (V31–V38 schema; application cutover in progress)

- Product decisions are recorded in [Student Management — Domain and Database Design](../product/STUDENT-MANAGEMENT-DESIGN.md): one permanent school-scoped student identity, one enrollment/admission number per academic year, combined initial admission, and independently editable profile sections.
- V31 adds nullable `admission_number` and `admission_date` to `student_enrollments`, a paired-nullability check, a non-empty/length check, and a normalized unique index scoped to school + academic year. Historical enrollments may remain NULL/NULL when their original identity cannot be proven.
- V32 decouples permanent student identity from the legacy school-wide admission number: `students.admission_number` becomes nullable, its school-wide unique index is replaced with a non-unique legacy lookup index, and existing values remain untouched. Its down migration refuses rollback if null or duplicate legacy identities exist.
- The enrollment repository read model now exposes year-specific admission number/date. Attendance projections now read admission number from the enrollment and return an empty value for historical NULL rows. Student-profile reads safely tolerate a NULL legacy number.
- New enrollment and promotion repository writes now require an admission number and admission date, validate both dates against the selected academic year, and write those values with audit metadata. Promotion preserves the prior enrollment and creates the destination-year identity.
- V33 extends the existing enrollment-history trigger so admission number/date are immutable from the moment an enrollment is created, including while active. The existing active-to-terminal transition remains allowed without rewriting identity; a dedicated PostgreSQL integration test covers both rules.
- New student creation now has one transactional repository operation that creates the permanent student identity (legacy admission number NULL), its first year-specific enrollment, and both audit events. The `POST /school/students` route now uses this combined operation; an enrollment uniqueness conflict rolls back the student insert.
- Student directory/profile reads now display and search the current academic year's enrollment admission number. If a student has a current-year enrollment with unknown legacy admission identity, the directory deliberately shows no number rather than incorrectly carrying forward an unverified old-year value. Attendance and enrollment lists also expose the enrollment-level number.
- The Student Management page now uses a guided first-admission form (student name, admission number, academic year, section, optional roll number and admission date) and requires year-specific admission number/date when enrolling an existing student.
- The student directory now opens a profile detail view showing the permanent student identity and year-wise enrollment history, including each year's admission number/date, class/section, roll number and enrollment lifecycle dates. Address & Communication and Parents & Siblings are launched as separate profile-section actions; the directory row no longer carries redundant section shortcuts.
- V34 adds tenant-aware hierarchical locations and separate current/permanent student address records. Backend APIs support location search/add/deactivate and address read/update; the frontend now has an independently editable Address & Communication section with dependent State → District → Block/Tehsil → Village/City selectors and school-owned additions. At V34, no guessed national geography catalogue was seeded; V40 later adds only the verified national State/UT reference list.
- V35 adds optional primary phone, alternate phone and email contact details with live read/update permissions, audit metadata that excludes PII values, and a Contact Details tab in the same independently editable profile section. Phone/email values are intentionally non-unique.
- The Parents & Siblings section now reuses the existing V11 guardian model: staff can create guardian contacts, link multiple guardians with relationship/primary/emergency flags, and revoke links with reasons while retaining history. Existing V12 partial uniqueness allows a revoked relationship to be re-established as a new history row. A related-student list is derived from active shared guardian links and explicitly does not claim biological sibling status. Parent OTP/login remains deferred.
- V36 adds optional Medical and Physical profile sections. Medical data includes blood group, allergies, known conditions, ongoing medication notes and emergency medical notes; every edit preserves the previous snapshot in append-only version history. Physical data is a dated append-only measurement history for height/weight and optional vision/physical notes; new measurements never overwrite old ones.
- V37 adds an optional one-to-one Basic Details profile section for date/place of birth, gender, nationality, religion, social category, EWS status and school registration number. It is not required during admission; reads require `students.read_sensitive`, writes require `students.update`, and audit metadata records field-presence flags rather than demographic values. Aadhaar is intentionally excluded. The section is linked from Student Profile and supports view-only access.
- V38 extends school-owned guardian contacts with optional alternate mobile, occupation, education and workplace/organization fields. These details belong to the guardian identity and are shared across linked students; relationship type and primary/emergency flags remain per-student relationship data. Existing guardian rows remain valid with NULL optional fields. Student address remains managed in the separate Address & Communication section; no duplicate guardian address is stored. The user locally verified the V38 migration, full backend test suite, `go vet`, frontend typecheck and production build on 2026-11-14. The V38 down migration refuses rollback once optional guardian profile data exists, preventing silent data loss.
- Medical/physical reads use `students.read_sensitive` and are audited; updates use `students.update` and are audited with version/presence metadata only. Health values and notes are excluded from audit metadata, Teacher access is denied, and sensitive query caching is disabled on the frontend. Documents remain deferred until private storage and document-specific access controls are designed.
- **User-reported local verification (2026-11-14, after pull through `2a5da95`):** with `DB_NAME=school_management_test` and `RUN_POSTGRES_INTEGRATION_TESTS=1`, `go run ./cmd/migrate` applied the full migration chain successfully through V37; `go test -count=1 -p 1 ./...` passed across all backend packages; and the chained `go vet ./...` completed without an error. `GOTMPDIR=D:\\go-test-tmp` was used to avoid Windows temporary test-executable cleanup conflicts. This verifies the disposable test database and backend code only; it does not verify the frontend in a browser.
- **GitHub Actions verification (2026-11-14):** backend CI passed on commit `429985b`, including all migrations on an ephemeral PostgreSQL database, full backend integration tests, and `go vet`; frontend CI passed on commit `7599c3c`, including dependency installation, design-token checks, TypeScript and production build. The V34 backend migration/API passed GitHub CI on commit `13d1a4d` (PostgreSQL migrations/integration tests and `go vet`); the V34 Address & Communication UI passed frontend CI on commit `4900836` (design-token checks, TypeScript and production build). V35 contact-details backend passed GitHub CI on commit `e2e0cac` (migrations, PostgreSQL integration tests and `go vet`); its frontend tab passed on `3375631` (design-token checks, TypeScript and production build). Parents & Siblings backend, shared-guardian query, revocation/re-link history integration tests passed backend CI on `610a6ee`; the complete frontend section passed design-token, TypeScript and production-build CI on `1a86c88`. V36 revised schema and repository passed backend CI on `853b3f8`, including the full migration chain, PostgreSQL integration tests and `go vet`; coverage includes medical version snapshots, append-only physical measurements, audit metadata privacy, database constraints and Teacher access denial. The profile and health frontend passed design-token, TypeScript and production-build CI on `17508c4`, including the version/measurement history UI and duplicate-save protection. User-local V32–V37 migration and backend test verification passed on 2026-11-14; frontend/browser acceptance remains pending.
- V37 Basic Details schema, repository, API routes and UI passed GitHub CI: backend on `def28e6` (full migration chain on ephemeral PostgreSQL, all backend/integration tests and `go vet`), frontend on `f4d53ce` (design-token checks, TypeScript and production build). The V37 integration test verifies optional profile persistence, same-school authorization, normalized registration-number uniqueness, immutable creator/identity, deletion protection and audit metadata that excludes demographic values. User-local V32–V37 migration and backend test verification passed on 2026-11-14; frontend/browser acceptance remains pending.
- **Application cutover remains in verification:** V32–V38 migrations and backend integration tests have passed in the user's local disposable `school_management_test` database; `go vet`, frontend typecheck and production build also passed for commit `9c07309`. Existing guardian profile editing has since been added through the existing `PATCH /school/guardians/:guardianID` route; that latest edit workflow and its follow-up status-preservation changes await a fresh local verification. The end-to-end browser workflow, role-by-role acceptance and document profile sections remain pending. The combined-admission and profile APIs are implemented and covered by backend tests, but the end-to-end browser workflow, frontend production build after this pull, and role-by-role acceptance remain pending. Document profile sections remain future implementation work.
- The persistent `school_management` application database has not been migrated by this work. Do not apply V32 or any pending migration to it without explicit approval and a verified backup.

## Staff Management — employee profile workflow implemented

- V28 adds the separate `staff_profiles` table. Full name, employee code, designation and one mobile number are required; email and joining date are optional. Employee codes are case-insensitively unique within each school.
- V29 makes employee inactivation disable the canonical staff principal, increment its token version and revoke active refresh sessions. V30 prevents any staff-login reactivation unless a complete employee profile exists with Active employment status.
- New staff creation requires employee details and creates the login account plus profile in one database transaction. Employee-code conflicts and profile failures cannot leave orphan login accounts.
- Staff listing includes employee details and a `profile_complete` indicator. Existing staff accounts remain intact and are shown as incomplete without fabricated profile data.
- Employee profiles can be edited independently of login credentials. Inactive employment disables login; returning employment to Active does not automatically enable the account. Explicit enablement with a temporary password requires a complete Active profile.
- The Staff Management UI provides the employee form, field-level validation, employee/profile listing, separate employment/login statuses, and hides login enablement for incomplete or inactive profiles.
- Profile changes do not delete academic assignments, attendance, timetable records or other history. No staff profile deletion operation is provided.
- Backend CI (fresh PostgreSQL migrations, full Go tests including PostgreSQL integration, and `go vet`) and frontend CI (TypeScript and Vite production build) are verified on the latest implementation commit. CI uses an ephemeral database; no persistent local database has been migrated by this work.

This file is the current implementation-status reference. Older checkpoint notes must not override it. Verification statements below distinguish repository inspection, user-reported local checks and real browser/database operations.

## Latest academic UI correction — section medium and date selection

- The shared date-picker parser now correctly recognizes ISO dates selected from the calendar, so selected academic-year start/end dates render visibly in the field. This fix is in commit `40a3d70506c6c1e07e181c40adedf0de5b5d45d2`.
- Migration V27 adds nullable section medium metadata restricted to `hindi` or `english`. Section names remain independently customizable. The section create/edit form exposes an optional Medium of instruction field, and the section list displays it.
- Backend persistence, validation, audit metadata, API handling and frontend form support have been updated. Frontend CI passed on the initial implementation commit; backend CI verification for the latest commits must be checked before local acceptance.
- Section-optional student enrollment/default-group behavior and stream metadata are design decisions still requiring implementation and verification. Do not treat those as delivered yet.

## Delivery scope and current priority

- **Phase 1:** Complete web product for Platform Admin, School Admin and school staff.
- **Immediate priority:** Complete the School Admin workspace and verify its end-to-end operational workflows.
- **Phase 2:** Parent and Driver experiences only after Phase 1 school web delivery.
- **Parent OTP/SMS:** Explicitly deferred. Do not implement parent OTP login, SMS delivery or parent-facing authentication now. Preserve existing guardian/parent relationship foundations without expanding this workstream until Phase 2 is approved.
- **Scale target:** 25–30 schools and 400–500 students per school; capacity is not yet load-tested.

## Current verification task — persistent authenticated sessions (implementation pushed; local acceptance pending)

The persistent-session/MFA policy has been implemented on this branch and pushed to GitHub. Backend GitHub Actions passed on code/test commit `b3af5904bd6e862aa261ee519a22effcacf2377a`, including application of all migrations to an ephemeral PostgreSQL database, backend tests with PostgreSQL integration, and `go vet`. Frontend CI passed on commit `6541584154c554e6ea662bdb9c78ac1ce27206a1`.

The frontend already restores a session at app startup by calling the refresh endpoint with the same-origin HttpOnly cookie; this was verified by source inspection, not by a fresh browser close/reopen test. Do not mark the task fully accepted until the user has reviewed and explicitly approved applying the pending migrations (V25–V30 at the time of this reconciliation) to the intended local database after a verified backup, and manually verified browser restart/session restoration, fresh-login MFA, background refresh, cross-tab account/logout behavior and security revocation. Disposable CI migrations do not change the user's local database.

- The refresh cookie is persistent with a 400-day Max-Age and is renewed on each successful refresh. Browser close/reopen and laptop restart restore the session while the browser retains the cookie; browser-enforced retention limits still apply.
- Platform Admin and School Admin must complete mandatory MFA on every fresh login. MFA must not be repeated during background access-token refresh or restoration of an existing valid session.
- The 30-minute administrative idle timeout and server-side absolute refresh-session expiry have been removed. Protected requests and refresh continue to validate the active session family.
- A session remains valid until the user manually logs out or a security/account event requires revocation (for example, password/security reset, account disablement or confirmed compromise).
- Manual logout must revoke the session family. Preserve opaque refresh-token rotation, replay detection/family revocation, transaction/concurrency protections, HttpOnly + Secure cookie policy, cross-tab logout synchronization, principal token-version checks and live database-backed RBAC authorization.
- Backend session lifecycle, persistent-cookie policy, regression/integration tests and backend/frontend security documentation have been updated. Automated browser close/reopen and interactive MFA acceptance remain pending locally.
- Continue development on the agreed `work` branch; keep `main` stable and merge only a completed, verified feature.

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
- Login lockout, IP rate limiting and breached-password checks.

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
- Timetable current-year lookup, period definitions and weekly section schedules, including active teacher-assignment validation, period-overlap prevention, class/teacher double-booking constraints, Teacher-scoped reads and atomic audit writes.
- School-scoped guardian management APIs and parent/guardian eligibility foundations.

## Frontend implementation present on the reviewed branch

- Shared login and authenticated application shell with role/permission-aware navigation.
- Platform Admin account list and school onboarding form.
- School Admin pages for staff accounts, roles/permissions, academic structure (including teacher-subject and class-in-charge assignment management), students/enrollments, attendance and timetable period/schedule management. Teacher accounts with `timetable.read` can view their own entries for the current academic year without being granted broad `academic_years.read` access.
- Separate page CSS, shared design tokens and Light/Dark theme foundations.
- Typed API modules and TanStack Query-based server state for school administration.
- Student profile summary is read-only; permanent student name editing is placed in the Basic Details section alongside date of birth, gender and other optional personal details. The former standalone “Edit student name” profile action has been removed; this UI adjustment awaits fresh local verification.

The presence of a page or API client does not mean its full browser workflow has been accepted. Continue testing against the real local API and verify every permission boundary. Do not present mock data or frontend-only checks as completed backend workflows.

## Recent verification evidence

### Timetable backend — implementation pushed; frontend pending
- Migration V26 adds school/academic-year-scoped timetable periods and weekly section entries. Composite foreign keys prevent cross-school/year mismatches; database triggers require an active teacher-subject assignment and reject break-period scheduling.
- Database safeguards prevent overlapping period definitions, duplicate section slots and teacher double-booking. A teacher assignment cannot be revoked while timetable entries still depend on it.
- Timetable period and entry APIs enforce live `timetable.read` / `timetable.manage` permissions, restrict Teacher reads to their own scheduled entries, and write audit records atomically with mutations.
- PostgreSQL integration tests verify the V26 schema and overlapping-period rejection. GitHub Actions backend run for `b52513d51c2bc8ba26d54f13c914ddac78bc371b` passed migration application, the full backend test suite including PostgreSQL integration, and `go vet` on commit `b52513d51c2bc8ba26d54f13c914ddac78bc371b`. The timetable current-year endpoint and frontend workspace have since been added; frontend TypeScript/Vite CI passed on commit `f448a871b574a60a6cc8da810adf9d2e648a3495`. The latest backend CI for the current-year endpoint is still running. Manual browser/role acceptance remains pending.


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
- Timetable implementation exists; real-data browser and role acceptance remains pending. Examinations and results have no delivered schema, API routes or frontend workflow yet; see [Examinations and Results — Domain Design Proposal](../product/EXAMINATIONS-RESULTS-DESIGN.md).
- Teacher/staff-specific operational screens and assignment-limited browser workflows beyond the new School Admin assignment-management screen.
- Exhaustive HTTP coverage with real JWT signature validation for every protected route.
- Production deployment hardening, least-privilege database grants, monitoring/alerting, backup/restore drills, penetration testing and load/capacity testing.
- Parent OTP/SMS delivery and Parent/Driver Phase 2 interfaces remain deferred and are not current work.

## Safe next sequence

Follow the mandatory [Phase 1 School Operations task-by-task build order](../product/PHASE1-WORKFLOW-ORDER.md). The immediate operational sequence is: verify school/academic foundations → complete/verify subject catalogue → create teacher/staff accounts → assign teachers to subjects/classes/sections → create students and enroll them → finish timetable acceptance using real data → complete attendance → design and implement examinations/results → notices and dashboard integration.

This sequence does not imply that existing modules must be rebuilt. Inspect current code and close missing implementation or acceptance gaps in place. Subject management already belongs to Academic Structure; do not create a duplicate subject module. Timetable implementation already exists and must be preserved while its real-data browser/role acceptance is completed. Student and teacher profile creation may be prepared in parallel after academic foundations, but downstream workflows must respect the dependency gates in the roadmap.

Before production acceptance, complete the pending persistent-session/MFA browser verification and local database adoption only after explicit approval. For each new business module, reconcile authorization against actual routes/repositories and record browser, responsive, accessibility, theme and session acceptance. Parent OTP/SMS and Parent/Driver Phase 2 work remain deferred until Phase 1 school web completion and explicit reprioritization.

## Git and database safety

- Develop on the single ongoing `work` branch. Keep `main` stable; merge from `work` only after feature completion and verification, then synchronize `work` from `main`.
- Do not stage unrelated helper files or commit secrets.
- Do not run migrations against persistent databases without explicit approval.
- Keep `.env`, passwords, JWT secrets, TOTP encryption keys and real student data out of Git.


## Student profile screen corrections — local verification pending

- The directory Edit action now opens the section-based Student Profile workflow; the legacy name-only edit modal has been removed from the page.
- Address reads now include the saved State, District and Block/Tehsil names in addition to hierarchy IDs. The UI can keep previously selected inactive locations visible instead of rendering an empty lower-level selector.
- Added PostgreSQL integration coverage for full address hierarchy read-back after locality deactivation.
- The design document now records the acceptance checklist and confirms Aadhaar remains intentionally excluded from the default profile; document upload/download remains deferred pending secure private storage and dedicated access controls.
- These changes are committed on `work`; run backend PostgreSQL integration tests, `go vet`, frontend typecheck and production build locally before browser acceptance. Do not apply migrations to the persistent application database for this work.


## Student Basic Details sensitive-read audit — 2026-11-14
- Follow-up audit found Address and Contact Detail GET methods were permission-protected but did not create read-audit events. Both now perform permission check, tenant-scoped read and section-only audit insert in one transaction. User ran both targeted PostgreSQL integration tests, the full backend test suite and go vet ./... against school_management_test; all passed. Windows emitted a post-test temporary executable unlink warning, but the targeted and full test commands returned success.

- Basic Details GET now runs its authorization, student lookup, sensitive-profile read and audit insert in one transaction. A successful read records `school.student.basic_details.read` with the student reference and section name only; profile values are never copied into read-audit metadata.
- Added PostgreSQL integration assertions for the read audit event and for absence of personal values in its metadata. User ran `go test -count=1 -p 1 ./internal/repository -run TestStudentBasicDetailsV37PostgresIntegration`, the full `go test -count=1 -p 1 ./...`, and `go vet ./...` against `school_management_test`; all passed.
- Contact Details and Address now have verified transactional read-audit coverage alongside Basic Details. Medical and Physical detail GET methods already use transactional section-only read audits, with PostgreSQL integration assertions for audit counts and absence of health values. Parents & Siblings guardian/list/link/related-student read-audit changes have now passed the user's targeted PostgreSQL integration test, full backend test suite and go vet ./... against school_management_test.
- Aadhaar remains excluded from the UI/API until its dedicated encryption/key-rotation implementation is complete.

## Student Management reference inventory update — 2026-11-14

- Recorded the complete November 13 MySmartSchool reference field inventory: combined admission/enrollment fields; Basic/Personal (including optional Aadhaar number and SR number); Address & Communication; Parents & Siblings; Medical; Physical; Class & Subject; Transport; App Permissions; Documents.
- Aadhaar card image/PDF and other document uploads remain deferred. Aadhaar number itself is intended as an optional field only; implementation is blocked until dedicated application-level encryption/key rotation, masked responses, least-privilege access and audit behavior are implemented. Never store or return the full number in plaintext; Aadhaar must not be mandatory for admission.
- Fixed the stale `updateStudent` reference in StudentManagementPage's busy state after removal of the legacy name-only edit modal. Local frontend typecheck/build must be rerun after pulling this fix.


## Student Parents & Siblings read-audit follow-up — 2026-11-14

- Review confirmed the Parents & Siblings repository read methods were permission-protected and school-scoped but did not create read-audit events.
- Added transactional read audits for guardian directory lists, individual guardian reads, a student's guardian links/history, and students sharing a guardian. Audit metadata contains only section and result count; it excludes guardian names, phone numbers, email addresses and relationship/contact values.
- Added PostgreSQL integration assertions to verify the four read-action families and that guardian contact values do not appear in audit metadata.
- User ran TestGuardianRepositoryPostgresIntegration, the full backend test suite and go vet ./... against school_management_test; all passed. Windows emitted a temporary executable unlink warning after the targeted test, but the test command itself returned success.


## Student directory/profile read-audit follow-up — 2026-11-14

- Review found ListStudents and GetStudent were school-scoped and permission-protected but did not emit read-audit events.
- Added transactional audit events for student directory reads (section and result count only) and individual student profile reads (section only). The list count query, result query, authorization and audit insert share one transaction.
- Added PostgreSQL integration assertions that both read actions are recorded and that student names/admission numbers do not appear in read-audit metadata.
- User verified `TestStudentProfileRepositoryPostgresIntegration`, the full backend test suite (`go test -count=1 -p 1 ./...`) and `go vet ./...` against `school_management_test`; all passed. A Windows temporary executable unlink warning appeared after the targeted test, but the test itself returned success.

## Student create-event audit data minimization — targeted verification passed

- Review found that the student-created audit event stored the raw admission number in JSON metadata. Changed it to record only the boolean `has_admission_number`; the student ID remains the audit resource reference.
- Added a PostgreSQL integration assertion that the create event exists, records the presence flag, and contains neither the student's name nor admission number.
- User pulled commit `69bf4d7` and ran `go test -count=1 -p 1 ./internal/repository -run "^TestStudentProfileRepositoryPostgresIntegration$"` against `school_management_test`; the test passed. Windows emitted a post-test temporary executable unlink warning, but the test command itself returned success.
- **Follow-up full verification (user-reported, 2026-11-14):** after pulling through the create-audit argument fix and status update, the user ran `go test -count=1 -p 1 ./...` with `DB_NAME=school_management_test` and `RUN_POSTGRES_INTEGRATION_TESTS=1`; every backend package passed, including repository and services. `go vet ./...` also completed without errors. Windows printed a post-test `services.test.exe` unlink warning, but the full test command returned success. This verifies the disposable test database and backend code only; it does not verify browser workflows or migrate the persistent `school_management` database.


## Student profile UI review — 2026-11-14

- Confirmed backend automatic roll-number allocation is implemented for combined admission, returning-student enrollment and promotion. Blank values are allocated transactionally after locking the destination section; manual roll-number overrides remain supported. The academic foundation PostgreSQL integration test includes assertions for automatic assignment and manual override, and the user-reported targeted test passed on 2026-11-14.
- Confirmed the promotion API and backend route exist, but the Student Management frontend does not yet expose a promotion action/workflow. This remains a frontend gap; do not treat promotion as end-to-end complete.
- Moved student address/medical profile tab styling into the shared modal stylesheet with explicit selected, hover, focus-visible and mobile states. Removed the duplicate page-level tab rules so styles are owned by the shared modal shell. This is a source-level styling correction only; browser rendering and crash-free interaction still require local browser acceptance.
- User's latest local frontend production build passed before this CSS-only follow-up. Re-run the production build after pulling the new commits; do not claim browser acceptance until manually exercised.


## Student promotion UI and modal tab styling follow-up — 2026-11-14

- Added a promotion action to active enrollment rows for users who have student-update, academic-year-read and class-read permissions. The dialog requires a destination active academic year, destination section, new admission number and date; roll number may be left blank for transactional automatic assignment or entered manually.
- Promotion calls the existing school-scoped backend route. On success, the student enrollment list/history query family is invalidated and the UI reports that the previous academic history is preserved.
- Student address and medical/physical profile tab styles now live in the shared modal stylesheet, with explicit active/pressed, hover, keyboard-focus and mobile layouts. Duplicate feature-page tab rules were removed. This keeps styling with the modal shell used by the profile editors.
- Review also found promotion audit metadata was storing the raw destination admission number and date. It now stores presence flags only, and the PostgreSQL integration test asserts that the raw values are absent.
- These frontend/backend follow-ups have been committed to the work branch. The user's earlier build/test output predates these changes. Run the targeted academic integration test, full backend tests, go vet and frontend production build after pulling; then manually test promotion, profile tabs, edit/view modals, permissions, responsive layout and refresh persistence. No browser acceptance or crash-free claim is made yet.

## Student Management end-to-end audit — 2026-11-14

- User-reported issues: Basic Details and Contact Details Save controls remain inactive after entry; required fields are not consistently identified; the State → District → Block/Tehsil → Village/City location selectors do not load expected data; profile tabs need consistent styling and the full Student Management workflow has not received browser acceptance.
- Source review confirms Basic Details and Contact Details Save controls are disabled while their prerequisite GET query is loading or has failed. Their input controls are not consistently disabled on query failure, so a user can enter values while Save remains disabled. Add explicit loading/error/retry states and prevent editing until existing values are safely loaded; do not bypass the query guard.
- The current status record says V32–V38 were verified in disposable school_management_test, while the persistent school_management application database was not migrated. Basic, contact, medical and address APIs depend on tables introduced by V34–V37. A stale application schema is therefore a strong candidate cause of the failed prerequisite reads and inactive Save controls; confirm the actual application migration version before making a definitive runtime diagnosis. Do not migrate the persistent database until a verified backup and explicit approval are in place.
- V40 now adds source_system/source_code metadata and seeds all 36 India State/UT entries from the Government of India's IGOD State/UT directory (checked 2026-11-14); the migration and its PostgreSQL integration test passed Backend CI. The frontend reads this data from the database rather than hard-coding it. District, Block/Tehsil and Village/City rows remain unseeded; the UI now explains when a reference level is empty and allows authorized school-owned additions. A versioned bulk import from the Ministry of Panchayati Raj LGD is still needed for the full national hierarchy. Do not invent district/locality data or add a runtime dependency on an unverified mirror.
- New UX acceptance rule: every Student Management field visibly says Required or Optional; conditional requirements state their condition; non-obvious fields may show accessible info icons explaining purpose, format, allowed values and restrictions. Help must work with hover, focus and touch. Field errors remain directly beneath their controls. The rule is recorded in the Student Management design and Frontend Architecture documents.
- Scope boundary: this audit/rebuild is limited to Student Management. Preserve Auth, RBAC, academic years, classes, sections and subjects; do not rewrite their migrations or APIs. Preserve student/enrollment/attendance/guardian history. Schema corrections must use additive forward migrations, not edits to applied migration history.
- Code changes are now committed on work: reusable Required/Optional/Required-when-used field labels, contextual help icons, loading/error/retry states for Basic, Contact, Address, Medical, Guardian, academic-year/section and history reads, location empty-state guidance, and V40 State/UT reference data. Frontend CI passed through commit 1ee4eecb. Backend CI passed V40 migration/integration at 0fa6fe5d and the gofmt-only test commit 58dba9fa; the latest frontend-only commit's Backend CI is still running. Browser acceptance and the persistent application-database migration are still pending. Do not apply V40 to school_management until a verified backup and explicit approval are in place.


## Student Management — latest verification (2026-11-14; commit aa8dc03b)

- V40 adds the 36 India State/UT entries from the Government of India IGOD directory as shared canonical location rows, with stable source-system/source-code identifiers and uniqueness constraints. The migration deliberately does not fabricate district, block/tehsil or village/city data; schools can add missing entries through the existing school-owned location workflow.
- V40 has a PostgreSQL integration test that verifies all 36 source-code/name pairs. GitHub Backend CI on `aa8dc03b` completed successfully, including the migration chain, PostgreSQL integration tests and `go vet`; Frontend CI on the same commit completed successfully, including TypeScript and the Vite production build.
- The latest student-profile safety fixes prevent Basic Details and Contact Details values from being edited or saved when their initial profile read fails. Retry controls remain available so the user can recover without accidentally overwriting existing values with empty defaults.
- Guardian management is school-scoped and uses live `guardians.read` / `guardians.manage` permission evaluation in the repository, in addition to route authentication. Guardian profile edits are shared identity updates and intentionally affect every linked student; the UI explicitly warns staff before/after this operation. Relationship type, primary-contact and emergency-contact flags are stored per student relationship. Revocation requires a reason and preserves history.
- Current branch head: `aa8dc03bfb14854184f0bf7a3efaf824fca04ee0`. Auth, RBAC, Class, Section and Subject modules were not changed by this Student Management UI safety pass.
- **Local V39–V40 migration application and browser-level acceptance remain pending.** CI success does not establish that the user's local persistent `school_management` database has these migrations. Do not migrate that database without explicit approval and a verified backup; use only `school_management_test` for local integration verification.

## Shared guardian update confirmation — 2026-11-14

- Added an explicit confirmation step before saving edits to a shared guardian profile. The confirmation explains that the guardian's contact/profile values are shared by every linked student, and the action label makes the multi-record scope clear.
- Validation still runs before confirmation; the update mutation remains protected by the existing live guardian-management permission and invalidates guardian/student query families after success. Relationship-specific values (relationship type, primary contact and emergency contact) remain on each student-guardian link, not on the shared guardian profile.
- This is a frontend safety improvement committed to the work branch; CI is running for the new commit. Browser-level acceptance is still pending.

## Student Basic Details date-range alignment — 2026-11-14

- The Date of birth picker now enforces both ends of the same range as the backend: no future date and no date earlier than 130 years before today. This prevents users from selecting values that the save API will reject and makes the field guidance actionable at entry time.
- Frontend CI must verify this follow-up; browser interaction and local environment acceptance remain pending.

## Student Parents & Siblings editability audit — 2026-11-14

- The student profile exposes Parents & Siblings as a dedicated section in the same student-record workflow. It loads guardian relationships and related students, supports creating/linking guardians, editing shared guardian profile fields with an all-linked-students confirmation, and revoking links with a required reason while preserving history. Parent OTP/login remains out of scope.
- A remaining functional gap was identified: once a guardian is linked, the student-specific relationship type and the Primary contact / Emergency contact flags are displayed but have no edit operation. These values belong to the student–guardian relationship, not the shared guardian profile. Add a separately authorized, audited relationship-update operation and UI before calling Parent Details editability complete. Do not work around this by editing the shared guardian profile or revoking/recreating the link, because that would unnecessarily alter history.
- The relationship-update gap is recorded for implementation; current browser acceptance of the Parents & Siblings workflow remains pending.

## Student-specific guardian relationship editing implemented — 2026-11-14

- Added `PATCH /school/student-guardian-links/:linkID`, protected by the existing live `guardians.manage` permission and school-scoped repository authorization.
- Staff can now edit Relationship Type, Primary Contact and Emergency Contact on an active student–guardian link without changing the shared guardian profile or revoking/recreating the relationship.
- Primary-contact selection is serialized by locking the owning student row; choosing a new primary clears the previous active primary only for that same student. Other students linked to the same guardian remain unchanged.
- The update is transactional, restricted to active students and active links, and writes an append-only audit event with before/after relationship values and contact flags. Revoked links cannot be edited; relationship history remains intact.
- Guardian relationship edit forms and action controls wrap responsively inside each guardian card on narrow screens.
- Frontend includes field-level relationship validation, per-student labels/help text, loading/error/success feedback and query invalidation. Backend handler/route tests and PostgreSQL integration coverage include permission enforcement, audit evidence, primary-contact exclusivity and isolation from another student's link.
- CI and browser acceptance are pending; do not mark Student Management complete until both are verified.

## Student guardian relationship database editability correction — 2026-11-14

- Backend CI exposed a schema/application mismatch: V12's relationship-history trigger treated `relationship_type` as immutable, so the new authorized relationship-edit repository operation was rejected by PostgreSQL even though its application validation allowed the change.
- Added forward migration V41 to permit edits to the relationship label while keeping student ID, guardian ID, school ownership, row identity, creation time and revoked-history fields immutable. The existing 1–40 character database constraint remains in force.
- V41 retains the same actor/school checks for revocation. The down migration restores the original V12 trigger behavior.
- The PostgreSQL integration test already exercises relationship-type changes, primary-contact exclusivity, cross-student isolation, revoked-link immutability, inactive-student restrictions and audit logging. CI must rerun with V41 before this correction is considered verified.
- This is an additive migration; do not edit V12 or apply V41 to the persistent `school_management` database without the user's explicit approval and a verified restorable backup.


## Student Management verification update — 2026-11-14

- Backend CI for commit 98c10e62 completed successfully: the full migration chain applied to an ephemeral PostgreSQL database, all backend tests including PostgreSQL integration tests passed, and go vet passed. The earlier V41 status entry above is therefore verified; its CI-pending wording is historical.
- The guardian relationship integration test now asserts primary-contact exclusivity against active links only. The include-history query verifies history visibility/count; it is not a historical snapshot of previous field values. A redundant second active-link read was removed.
- The frontend source audit confirms Basic Details and Contact Details Save controls intentionally remain disabled when the prerequisite sensitive-profile GET fails. Do not remove this guard or permit editing against empty fallback values. Retry must recover the read first.
- GET access for Basic Details, Contact Details and Address requires the existing students.read_sensitive permission; corresponding writes require students.update. Keep these existing live permission boundaries unchanged. If the UI still reports a failed read after the local database is current, capture the API error and diagnose the specific endpoint rather than weakening authorization.
- The known local adoption gap remains: the user's persistent school_management database was last reported at migration V24, while student address, contact, medical/physical and basic-details tables are introduced in V34–V37, guardian profile extensions in V38, State/UT reference data in V40 and relationship editability in V41. This is a likely explanation for failed profile reads on that database, not proof of its current version. Do not migrate it without explicit approval and a verified restorable backup.
- Current Student Management gaps: browser-level create/view/edit/save acceptance remains pending; the national District → Block/Tehsil → Village/City reference hierarchy still needs an authoritative LGD import; persistent database migration/adoption remains pending. State/UT rows are seeded by V40, but lower-level rows are deliberately not fabricated.
- Current work remains scoped to Student Management. Auth, RBAC, Class, Section and Subject are unchanged.


## Read-only database adoption preflight — 2026-11-14

- Added `backend/cmd/dbstatus`, a read-only command that reports the configured database name, golang-migrate version and dirty flag. It performs no schema or data writes and exits with an error for dirty or malformed migration state.
- Documented the command in the backend database guide. Use it to establish the actual persistent database version before discussing an upgrade; it does not replace a backup/restore rehearsal or the user's explicit approval.
- The new command is awaiting Backend CI verification. Do not run `go run ./cmd/migrate` against the persistent `school_management` database as part of this preflight.


## Address hierarchy terminology correction — 2026-11-14

- Corrected Student Address labels and help text: the database's `subdistrict` level is now presented as **Sub-District / Tehsil**, not Block / Tehsil. This avoids conflating a revenue sub-district (Tehsil/Taluk/Mandal) with a Development Block, which is a distinct LGD administrative entity.
- The current address path remains State/UT → District → Sub-District/Tehsil → Village/City. A future Development Block field, if required by school workflows, must be a separate optional reference relationship and must not be inserted into the revenue hierarchy.
- Documented official Ministry of Panchayati Raj / OGD LGD entry points for District, Sub-District and Village datasets and the requirement to verify actual official export schema before writing an importer. Community mirrors remain untrusted for production ingestion.
- The read-only `backend/cmd/dbstatus` command and its documentation passed Backend CI on commits a7ea3c3c / 22733318 (migration chain, PostgreSQL integration suite and go vet).
- The terminology change passed Frontend CI on commit 46fe22fe. The follow-up label-only correction and subsequent documentation commits are awaiting their own CI runs. Browser acceptance and the persistent database upgrade remain pending.


## LGD snapshot pre-import validation — 2026-11-14

- Added `backend/cmd/lgdvalidate`, a read-only standard-library CSV validator for a normalized all-India State → District → Sub-District → Village/City snapshot.
- The validator checks exact headers, required values, source-code/name limits, duplicate and conflicting hierarchy mappings, parent relationships, locality types, one-to-one two-character IGOD State/UT crosswalks, and minimum coverage counts. Leading-zero source codes are preserved as strings.
- It does not connect to PostgreSQL and cannot mutate data. The raw official OGD/LGD export schema has not yet been verified; the exact downloaded source file must be preserved and transformed to the documented normalized contract before validation. No community mirror is used as a production import dependency.
- Added unit tests for valid hierarchy parsing, malformed headers/crosswalks, duplicate or conflicting codes/parents, missing hierarchy consistency, and unsupported locality types. Backend CI is pending for the new command and tests.
- No LGD location rows have been imported. A database importer remains gated on a verified official snapshot, reviewed IGOD↔LGD State/UT crosswalk, validation report, and explicit apply mode.

- During source inspection on 2026-11-14, the OGD resource pages were returning sandbox/maintenance or no-result content in the page view, so the live raw export schema could not be independently inspected there. Dataset metadata pages still identify Ministry of Panchayati Raj as owner and describe monthly refreshes, but this is not sufficient to claim the current downloadable file schema is verified. Keep the normalized validator contract separate from raw-export parsing until an actual official file is obtained and inspected.


## Backend CI efficiency and validator test corrections — 2026-11-14

- Backend CI now skips documentation-only changes and uses a branch-scoped concurrency group with cancel-in-progress, preventing stale runs from consuming PostgreSQL runner capacity after new code is pushed.
- The first LGD validator CI run exposed test-fixture mistakes (short header, impossible missing-parent expectations in a denormalized ancestor-row format). The tests were corrected to check the actual contract: exact ten-column header, conflicting source-code/parent mappings, duplicate IGOD crosswalks, and BOM/leading-zero preservation.
- The corrected validator commits are undergoing Backend CI. Do not mark the tool verified until the latest branch-head run completes.

## LGD validator verification — 2026-11-14

- The corrected read-only validator and its test suite passed Backend CI on code commit `98c6acfb72` (workflow run `43593036081`). The ephemeral PostgreSQL migration chain, complete backend tests including PostgreSQL integration, and `go vet` all passed.
- The branch documentation-only HEAD is `ede9692470`; the CI workflow intentionally skips documentation-only changes, so the successful code commit remains the relevant verification target.
- This verifies the normalized CSV validator only. It does **not** verify a parser for the raw OGD/LGD export, import any locations, migrate the persistent local database, or replace browser acceptance.
- Next data gate: obtain and preserve an actual official LGD snapshot, verify its raw schema and source checksum, review the IGOD↔LGD crosswalk and row-count drift, then implement a staged importer with dry-run/report mode before any explicit database apply.

## Guardian directory least-privilege query — 2026-11-14

- Source review found that opening a read-only Parents & Siblings profile also fetched the full active guardian directory, even though the guardian picker is only used by users with `guardians.manage`.
- The directory query is now enabled only when the current user has `guardians.manage`. Read-only staff continue to load the student's linked guardian details through the existing `guardians.read` permission, without requesting unrelated guardian-directory records.
- This is a frontend least-privilege/data-minimization correction; it does not change backend permissions or allow/deny rules. Frontend CI passed on commit `bd9cec3b0f` (TypeScript/design-token checks and Vite production build). Backend CI for the same commit is running; the frontend build is verified, while browser-level role acceptance remains pending. Browser-level role acceptance remains pending.

## Guardian relationship concurrency hardening — 2026-11-14

- Guardian relationship revocation now resolves the owning student and locks that student row inside the same transaction before revoking the active link.
- This uses the same per-student serialization point as link creation and relationship edits, preventing revocation from racing with primary-contact selection.
- The lock intentionally does not require the student to remain active, preserving the existing ability to revoke a relationship for an inactive student.
- Added a PostgreSQL integration regression case that links a guardian, deactivates the student, revokes the relationship, and verifies the revoked relationship remains visible in history.
- Backend CI for commit `f902940b68` is running; the final result will determine whether this concurrency hardening is verified.


## Guardian relationship database invariants — 2026-11-14

- Backend CI for revocation serialization commit `f902940b68` passed: full migration chain, PostgreSQL integration tests (including inactive-student revocation/history), and `go vet`.
- Audit found that the application serialized primary-contact changes but PostgreSQL did not independently enforce the one-active-primary-per-student rule.
- Added V43 partial unique index `ux_student_guardians_active_primary_per_student`; migration preflights existing active primary duplicates and fails rather than silently changing records.
- Added PostgreSQL integration coverage proving a direct duplicate-primary insert is rejected, while existing repository behavior can safely transfer primary status.
- Multiple emergency contacts remain allowed intentionally. The V43 down migration drops only the index and preserves all relationship data.
- V43 migration and new constraint test are awaiting Backend CI. Persistent databases have not been migrated.


## Shared guardian profile propagation regression — 2026-11-14

- Added PostgreSQL integration coverage that edits a guardian's shared name, mobile and occupation after that guardian is linked to two students.
- The test verifies both students immediately read the updated shared profile while their independent relationship type, primary-contact and emergency-contact values remain unchanged.
- This complements existing tests for cross-school isolation, relationship edits, primary-contact uniqueness, revocation and history.
- Backend CI for commit `0efba97f51` is pending. No persistent database has been migrated.

## Student profile repository regression coverage — 2026-11-14

- Backend CI for commit `3f51290f2906262e85f4513525ad76c76dafbc46` (workflow run `43597056423`) passed: full migration chain on ephemeral PostgreSQL, all backend tests including PostgreSQL integration, and `go vet`.
- Added address save round-trip coverage that verifies updated current-address fields, preserves the independent permanent-address `same_as_current` setting, and checks the address-update audit event.
- Added contact save round-trip coverage that verifies primary phone, alternate phone and email persistence and checks the contact-update audit event.
- Existing V36 integration coverage verifies medical profile version snapshots, append-only physical measurement history, invalid measurement rejection, cross-school read denial, teacher health-data denial, and audit metadata that excludes private health values.
- Existing V37 integration coverage verifies basic-profile save/read, registration-number normalization/uniqueness, immutable identity/creator fields, cross-school read denial, and value-free audit metadata.
- Source review confirms address/contact/basic/medical/physical saves are separate transactional operations; all writes require live `students.update`, sensitive reads require `students.read_sensitive`, and each write locks the owning student row before mutation. Existing tenant and sensitive-data boundaries remain unchanged.
- No persistent database was migrated. Browser-level create/view/edit/save acceptance remains pending; do not claim full UI acceptance based on repository tests alone.

## Student Management view-switcher accessibility — 2026-11-14

- Student Directory / Enrollments switcher now uses a labelled button group with `aria-pressed`, matching its actual interaction model. It no longer advertises an incomplete ARIA tablist without tabpanel and keyboard-roving-focus behavior.
- Existing visual styling, permission-based visibility, and view-switching callbacks are unchanged.
- Frontend CI for commit `233fa27595dd2336edc23f21b7765a701152dbf2` (workflow run `43597474039`) passed: design-token/style checks, TypeScript project build, and Vite production build.
- Backend CI for the same commit (workflow run `43597474026`) was still running when this status entry was written.
- Browser-level Student Management acceptance remains pending; no claim of end-to-end browser verification is made.

## Student profile edit navigation — 2026-11-14

- Closing Basic Details, Address & Communication, Parents & Siblings, or Medical & Physical section now returns to the same student's profile instead of dropping the user back to the directory.
- This avoids repeated student searches when completing multiple profile sections in one session. Existing modal close/pending guards and permission checks are unchanged.
- Frontend CI for commit `478af7360832d5210d5a387e573f4839cde7d642` (workflow run `43597540743`) passed: `npm ci`, design-token/style checks, TypeScript project build, and Vite production build.
- Backend CI for this commit is queued/pending; no backend source changes were made.

## Student Management browser acceptance baseline — 2026-11-14

- Added a Playwright Chromium browser test and CI execution on Ubuntu 24.04 LTS. The runner is pinned to Playwright `1.56.1`; it is installed ephemerally in CI so the production dependency lockfile remains unchanged.
- Browser acceptance workflow run `43599136283` passed for commit `9a1d4f8b545a`: frontend style-token checks, TypeScript/Vite production build, Chromium installation, and the Student Management browser test.
- The browser test signs in with a test-only mocked staff session, admits a student with a blank roll number (automatic assignment path), verifies directory refresh and academic history, opens/closes Basic Details, opens Parents & Siblings, and checks that the same student profile remains available after closing each section. It also fails on uncaught browser exceptions.
- All `/api` calls in this browser test are intercepted with in-memory fixtures. It does not connect to or mutate the local/persistent school database; backend PostgreSQL integration tests remain the source of truth for persistence, tenant isolation and audit behavior.
- Failure screenshots and traces are retained as a seven-day GitHub Actions artifact. No generated test output is committed to the repository.
- Browser coverage is now in CI; manual acceptance against the user's configured local API/database remains a separate final verification step.

### Parent relationship browser coverage follow-up

- Expanded the Playwright test to return an existing linked guardian and verify the guardian name, relationship, primary-contact and emergency-contact labels render.
- The test edits the relationship from `Father` to `Legal Guardian`, verifies the PATCH payload updates only that student's relationship record, and confirms the updated label renders after query refresh.
- Frontend CI run `43599397403` passed for commit `43ea1aef0f5663053b806a393c2f3aec811cfcba`, including the expanded parent relationship browser flow.
- Guardian profile shared-field editing (which affects every linked student) remains separately protected by its confirmation flow; this smoke test deliberately does not mutate shared guardian details.

### Address, contact and physical-details browser coverage follow-up

- Expanded the same Chromium acceptance test to open Address & Communication, save a primary contact phone number, and verify the saved value in the mocked API state.
- Expanded Medical & Physical Details coverage: the test checks the restricted-health privacy notice, switches to Physical Details, confirms invalid height and missing measurement date errors appear together, then saves a valid height with a measurement date.
- Frontend CI run `43599830202` passed for commit `6732d1c8a5cfe42b44a0fd37b7b3cbc473d65d85`, including all expanded browser flows.
- These remain frontend-only mocked API checks. PostgreSQL integration tests separately verify persistence, tenant boundaries, permissions, history and audit behavior.


## Student Management address and medical browser coverage — 2026-11-14

- Expanded the Playwright Chromium workflow to exercise the Current Address location chain using test fixtures: Haryana → Ambala district → Ambala Sub-District/Tehsil → Naraingarh locality. The test verifies the saved locality and resolved State, District and Sub-District IDs in the mocked API state.
- Added Permanent Address coverage for the explicit “same as current” option and verifies the saved same_as_current flag.
- Added Medical Details save coverage for Blood Group, Allergies and Known Conditions, including the returned profile version increment. Existing physical-measurement range/date validation and save coverage remains in the same browser journey.
- The test still intercepts every API request and uses in-memory fixtures. It verifies frontend interaction and request/response handling only; it does not prove live database persistence or replace backend PostgreSQL integration tests.
- Frontend CI passed for commit f0267654b18c904359c12b2cdaf0ff45ba8f57d7 (workflow run 43600789669), including design-token checks, TypeScript, production build and Playwright Chromium.
- Backend CI for the same commit was still running at the time of this entry. Persistent database migration/adoption and manual browser acceptance against the configured local API remain pending.

## Student Management guardian permissions and promotion browser coverage — 2026-11-14

- Expanded guardian browser coverage for shared-profile editing: invalid email is rejected without saving; valid edits require explicit confirmation because contact/profile data affect all linked students; student-specific relationship, primary-contact and emergency-contact flags remain unchanged.
- Added guardian revocation browser coverage: a reason is mandatory, the active relationship disappears after revocation, and the revoked relationship remains available in history.
- Added view-only permission coverage: staff with student read and guardian read permissions but without update/archive/guardian-management permissions can view the student and linked guardian, but cannot see edit, archive, link or revoke actions. The test verifies no write request is sent.
- Added promotion browser coverage for required admission number validation, destination academic year/section selection, blank roll-number auto-assignment path, and preservation of the prior enrollment with promoted status alongside the new enrollment.
- Frontend CI passed for commit `e8b907abbb20` (run `43602032049`), including shared guardian update and revocation/history coverage.
- Frontend and Backend CI passed for commit `288a70a25440` (runs `43602633101` and `43602632967`), including student and guardian view-only permission coverage.
- Frontend and Backend CI passed for commit `1d739fcb0169` (runs `43603172771` and `43603172772`), including promotion validation and preserved enrollment history.
- Browser tests intercept all API requests and use in-memory fixtures; they do not write to the local or persistent school database. Backend PostgreSQL integration tests run separately against the ephemeral CI database. No persistent database migration was run.

## Student Management archive and guardian create/link browser coverage — 2026-11-14

- Added archive browser acceptance coverage: cancelling confirmation sends no request; a server conflict keeps the confirmation open and the student visible; retry success refreshes the directory to archived status.
- Added guardian create/link browser acceptance coverage: invalid email is rejected before create; a valid guardian contact is created, then linked to the student with a relationship and primary/emergency contact flags.
- Frontend CI passed for commit `5c555459a6f8` (run `43603791131`), including archive cancel/conflict/retry and archived directory refresh.
- Frontend CI passed for commit `196f3e41af15` (run `43603948048`), including guardian create/link validation and relationship flags.
- Backend CI passed for commit `5c555459a6f8` (run `43603791165`); the guardian create/link commit changes only frontend browser tests.
- Browser API requests are mocked and use in-memory fixtures; they do not mutate a real school database. Parent OTP/login remains deferred. No persistent database migration was run.

## Student name edit/profile state consistency — 2026-11-14

- Fixed a stale profile-state issue: after saving a student's name in Basic Details, the parent Student Management state now receives the updated name, so closing Basic Details returns to a profile showing the new name rather than the previous value.
- Preserved the successful name-save feedback while synchronizing the updated student prop.
- Added a browser assertion for the name update, success feedback, and profile description after returning from Basic Details.
- Frontend CI passed for commit `f5b9a2ff7396` (run `43604393646`).
- Browser test uses mocked API fixtures; this is not yet a live local-PostgreSQL/browser acceptance test.

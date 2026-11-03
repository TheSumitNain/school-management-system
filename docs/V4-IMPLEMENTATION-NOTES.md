# V4 Identity and RBAC Migration — Implementation Notes

## Status

- Design approval: received from the user (`MIGRATION V4 SCHEMA APPROVED`).
- Source baseline: commit `cb6af3b5abc5249fba420e2699d16e87b1d14664`.
- Database observed by the user: migration v3, `dirty = false`; 1 school, 1 staff user, 4 refresh sessions, 0 active sessions, 0 expired sessions.
- Username preflight: no case-insensitive duplicates and no usernames outside the approved lowercase ASCII format.
- Constraint/index inspection: completed against the development database.
- Migration v4 corrective review draft: `backend/migrations/000004_add_principal_rbac_foundation.up.sql`.
- Down migration: intentionally refuses rollback because v4 data may include platform and parent identities, roles, audit events, and security credentials.
- **Status update (2026-11-02):** The user later directly reported `school_management_test.schema_migrations` at cumulative `version=12`, `dirty=false`; V4 is therefore included in the applied V4–V12 sequence on that dedicated test database. Do not reapply V4 there. This does not prove every trigger edge case has a dedicated passing test, and it says nothing about the state of `school_management`.**

## Existing source behavior verified from the supplied archive

- Staff login still accepts `school_id + username + password`.
- `users` still has school-scoped username lookup and the repository maps the old unique-index name.
- JWT `sub` is still `users.id`; claims do not yet include `principal_type` or `token_version`.
- Refresh-session `Create` and `Rotate` still accept/return `models.User` and depend on `user_id`.
- Rotation serializes within a token family and across all token families for one user, detects replay, and revokes the affected family. Preserve these semantics during the principal refactor.
- Auth middleware validates the JWT but does not yet check principal status, token version, RBAC permissions, tenant/resource scope, teacher assignments, or parent-child links.

## V4 migration contents

The draft migration:

1. Creates `auth_principals` and backfills one `staff` principal for every existing user without changing `users.id`.
2. Adds a globally unique, case-insensitive username index and canonical lowercase ASCII username constraint.
3. Adds Platform Admin and Parent account subtype tables.
4. Seeds the platform-defined permission catalogue from `docs/PHASE1-RBAC.md`, including scope and a conservative explicit delegability allowlist.
5. Creates system roles for Platform Admin, School Admin, Teacher, and Parent, with conservative default permission bundles. It deliberately does not guess or assign a role to the existing staff account.
6. Adds role permissions, principal-role history, per-user allow/deny overrides, and bounded permission-delegation records.
7. Adds `principal_id` to refresh sessions, preserves `family_id` and token hashes, and replaces the staff-only FK with principal-aware constraints.
8. Adds encrypted TOTP credential storage, one-use recovery-code storage, parent OTP challenge storage, and append-only audit-event storage.
9. Adds database triggers for principal subtype consistency, role/permission scope, system-role protection, principal-role scope, overrides, delegation, refresh-session principal consistency, MFA/OTP principal type, and audit immutability.
   - Runtime role inserts are restricted to non-system, school-scoped staff custom roles and require an active staff creator in the same school. Migration-seeded system roles are inserted before the guard is installed. Role system classification and `created_by` are immutable after insertion. The trigger does not replace API enforcement of `roles.create_custom`.
10. Adds transitional insert triggers so the existing staff user/session creation SQL can populate `principal_id` while the Go service refactor is completed. These triggers do not constitute the final principal-aware application implementation.
11. Freezes staff school identity and role/permission identity-scope fields, synchronizes legacy staff status in both directions, validates assigning actors for staff role assignments, and checks active delegation ceilings for custom role permissions and user overrides. Parent role assignment is permitted only after mobile verification.

## Important blockers before applying v4

1. The user later reported the cumulative migration state at version 12, `dirty=false` on `school_management_test`. Re-run the migration/trigger integration suite against a clean disposable PostgreSQL 18 database after any SQL changes; do not reapply V4 to the already migrated test database.
2. The application code has since been refactored to principal-aware JWT claims, live principal/token-version checks and permission loading. `RequirePermission` exists, but business routes are not yet wired and custom-role grants/ALLOW overrides intentionally fail closed.
3. The existing staff account's intended role is not established by the database counts. Do not assign `school_admin` or another role by assumption. Confirm the account's intended role and create its principal-role assignment through an audited bootstrap operation.
4. The principal-aware login/JWT/session cutover is present in the current branch. Remaining work includes business route guards, resource/tenant scope enforcement, safe role/custom-role management, and end-to-end verification.
5. Database application-role grants for append-only `audit_events` must be configured. The application role must not own the audit table and must not have UPDATE/DELETE privileges.
6. Parent-child links, teaching assignments, and class-in-charge assignments remain deferred until the academic/student schema exists.
7. The down migration is intentionally forward-only. Use a verified backup/restore plan only if explicitly accepting loss of writes made after that backup.

## Required verification sequence

- Create a disposable PostgreSQL 18 database from a v3 backup/snapshot.
- Apply v4 there and inspect every constraint, index, trigger, seeded permission, and role bundle.
- Test existing staff login, new staff creation, refresh, replay detection, family logout, and concurrent cross-family rotation with the compatibility triggers.
- Test platform/parent subtype creation and refresh-session inserts.
- Test invalid subtype, mismatched role scope, cross-school staff-role assignment, unauthorized assigning actor, custom-role permission without an active delegation, invalid override, non-delegable permission delegation, unverified parent role assignment, invalid OTP principal, and audit UPDATE/DELETE rejection.
- Complete the Go principal/RBAC refactor and tests before production use.
- Take and verify a fresh backup, quiesce writes, apply the migration to the intended development database, and verify counts and migration state.

Never paste `.env` values, JWT secrets, passwords, OTPs, MFA secrets, or real student data into chat.
### C4 — Principal-role assignment authorization

- The assignment trigger now verifies that the target principal exists and is active before accepting any role assignment.
- Provisioning a School Admin requires an active Platform Admin who already holds the system `platform_admin` role and its active `platform.school_admins.create` grant.
- Routine staff role assignment requires an active same-school School Admin whose system-role bundle contains active `roles.assign`; an explicit DENY override for that permission blocks assignment.
- The existing same-school and principal/role scope checks remain in force. Custom-role grants remain fail-closed until delegation linkage is implemented.
- This is SQL source-level review only. PostgreSQL execution/integration tests have not been run, and migration v4 remains unapplied.
### C5 — User permission override authorization

- The override trigger now resolves the active school-scoped `permissions.override` permission and requires the grantor's active same-school School Admin system role to include that permission.
- An explicit DENY override for `permissions.override` on the grantor blocks the operation.
- Existing checks remain: grantor must be active staff in the target school; the permission being overridden must itself be active, school-scoped, and delegable; and an active `can_override_users` delegation ceiling must exist for that school/permission.
- Custom-role grants remain fail-closed until delegation linkage is modeled. This is SQL source-level review only; PostgreSQL execution/integration tests have not been run, and migration v4 remains unapplied.
### C6 — Delegation revocation is irreversible

- Added a dedicated `BEFORE UPDATE OF revoked_at` guard on `role_permission_delegations`.
- Setting `revoked_at` on an active row is allowed; once non-NULL, the value cannot be cleared or rewritten. A new delegation must be created to restore authority.
- The existing delegation validator still checks inserts and updates to school, permission, delegation capabilities, and grantor. This change does not add an authenticated `revoked_by` actor or independently authorize who may perform the initial revocation; the schema currently lacks that audit field/workflow.
- Source-level review only. PostgreSQL execution/integration tests have not been run, and migration v4 remains unapplied.
### C7 — Exact delegation linkage (fail-closed status retained)

- `role_permissions` and `user_permission_overrides` do not store the exact `role_permission_delegations.id` that authorized a custom-role grant or user override. Their write triggers only check that a matching delegation is active at write time.
- This means the database records alone cannot distinguish a grant created under an old delegation from one created under a later re-grant for the same school/permission.
- The current Go `LoadEffectivePermissions()` deliberately resolves only built-in system-role grants and explicit DENY overrides. It excludes all custom-role grants and all ALLOW overrides, preventing stale delegated grants from becoming effective after revocation/re-grant. This fail-closed behavior was verified in source.
- Explicit DENY overrides remain effective because they reduce access; user ALLOW overrides are ignored. Custom-role assignments may exist in data but their grants do not enter the effective permission set.
- **Do not enable custom-role permission grants or ALLOW overrides** until the schema stores an exact delegation ID on each applicable grant, enforces a composite FK tying delegation ID + school + permission together, and the resolver checks that exact delegation remains active. A re-grant must create a new delegation ID so old grants cannot revive.
- No code/schema change was required for this audit item; the existing resolver is intentionally fail-closed. Source review only; no PostgreSQL execution or Go test was run for this documentation-only audit. Migration v4 remains unapplied.
### C8 — Audit events: UPDATE, DELETE and TRUNCATE protection

- Existing row-level `BEFORE UPDATE OR DELETE` trigger rejects mutation of audit rows.
- Added a statement-level `BEFORE TRUNCATE` trigger using the same rejection function. PostgreSQL TRUNCATE does not fire row-level triggers, so the separate statement trigger is required.
- This does not make the table tamper-proof against its owner or a superuser. Before deployment, keep `audit_events` owned by a separate migration/administrative role; the application role must not own the table and should receive only the required `INSERT` and `SELECT` privileges, with `UPDATE`, `DELETE`, and `TRUNCATE` denied.
- Do not grant application role membership or ownership paths that let it assume the owner role. Review actual role inheritance and deployment grants once the production role names are defined.
- Source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C9 — Parent permission allowlist

- Replaced the Parent role's global-permission `CROSS JOIN` seed with an explicit allowlist of the seven approved read-only permissions: `children.list`, `children.profile.read`, `children.attendance.read`, `children.timetable.read`, `children.homework.read`, `children.results.read`, and `children.notices.read`.
- Added a migration assertion requiring exactly one global Parent system role and exactly those seven grants. Bidirectional `EXCEPT` checks detect both missing permissions and unexpected additional grants; the count check alone is not relied on for set equality.
- This prevents future additions to the global permission catalogue from being granted to Parent automatically. Parent-child link verification and resource-level authorization remain mandatory in the API.
- Source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C10 — School Admin exact permission set

- The School Admin seed already used the approved explicit 51-key allowlist, but its original migration assertion checked only the number of grants. A wrong set of 51 permissions could therefore pass.
- Kept the approved 51-key seed unchanged and strengthened the assertion with the same explicit expected-key array plus bidirectional `EXCEPT` checks. It now detects missing and unexpected grants as well as count drift.
- The assertion targets School Admin system roles only and confirms each such role is staff-principal, school-scoped, and tied to a non-NULL school. It does not change the permission bundle or authorize custom-role grants.
- Source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C11 — Teacher exact permission set

- Restricted the Teacher system-role seed to active catalogue entries in the school permission scope and to staff, school-scoped, non-NULL-school system Teacher roles.
- Added a migration assertion requiring exactly one Teacher system role per school and exactly the seven approved default keys: `students.read_basic`, `attendance.read`, `attendance.mark`, `attendance.update_same_day`, `timetable.read`, `homework.read`, and `notices.read`. Bidirectional `EXCEPT` checks detect missing and unexpected grants; a count check alone is not relied on.
- The bundle does not grant attendance access across all classes. API authorization must still require the exact class-in-charge assignment for attendance read/mark/same-day correction, and must enforce teaching-assignment scope and the approved basic-student field allowlist. Subject teaching assignment alone does not grant attendance access. Past attendance correction remains School Admin-only.
- This is source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C12 — Principal role assignment history is immutable

- Added a database guard for `principal_roles` so new assignment rows must start active, assignment identity/provenance fields (`principal_id`, `role_id`, `assigned_by`, and `assigned_at`) cannot be rewritten, and assignment rows cannot be deleted.
- Revocation is one-way: an active assignment may transition from `revoked_at IS NULL` to a non-NULL timestamp, but a revoked assignment cannot be reactivated or have its revocation timestamp rewritten. Re-granting a role requires a new assignment row, preserving the historical record.
- This guard protects history, not the authority to revoke: the application must still authorize the revoking actor and write an audit event. Deployment database privileges must also prevent untrusted direct writes.
- Source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C13 — Permission delegation history is immutable

- Added a guard requiring new delegation rows to start active, preserving the delegation ID, school, permission, capability flags, grantor, and creation timestamp after insertion, and rejecting deletion.
- The existing C6 guard continues to make revocation one-way. A revoked delegation cannot be reactivated or have its revocation timestamp rewritten; restoring authority requires a new delegation row with new provenance.
- This prevents a revoked delegation's historical school/permission/capability/grantor details from being rewritten or erased. The application must still authorize the revocation actor and write an audit event.
- Source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C14 — Platform permission catalogue entries cannot be deleted

- Added a database trigger rejecting deletion of any permission catalogue row, including currently unused permissions that would not yet be protected by role/override foreign keys.
- Existing identity-scope protection continues to make `permission_key`, `scope_type`, and `is_delegable` immutable. Obsolete permissions should be deactivated with `is_active`, preserving stable keys and historical meaning.
- This trigger does not authorize runtime permission creation or replace least-privilege database grants; deployment must keep catalogue writes under controlled migration/platform administration.
- Source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C15 — MFA recovery codes are one-use and immutable

- Added a database guard requiring newly issued recovery-code rows to start unused, preserving the code hash/principal/creation provenance, and rejecting row deletion.
- A recovery code may transition from unused to used once. A used code's timestamp cannot be cleared or rewritten, and its hash cannot be changed to make the credential reusable.
- Replacing a recovery-code set must insert new code rows; it must not reset or rewrite historical rows. The application must still consume a code atomically and audit recovery events.
- Source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C16 — Parent OTP challenge history is one-use and monotonic

- Added a guard requiring new OTP challenges to start unconsumed with zero attempts, preserving challenge identity/hash/expiry/creation fields, rejecting deletion, and preventing the attempt counter from decreasing.
- Once consumed, a challenge cannot be reactivated or have its consumption timestamp rewritten. The existing database check still caps attempts at five; the application must increment attempts atomically and consume/expire challenges as part of the OTP verification workflow.
- The partial unique index allows only one unconsumed challenge per parent. Because an expired row remains unconsumed until the application marks it consumed, the resend workflow must retire an expired challenge before inserting its replacement.
- Source-level review only. PostgreSQL execution/integration tests have not been run; migration v4 remains unapplied.

### C17 — Correct PL/pgSQL dollar-quote delimiters

A source-level syntax defect was found in five PL/pgSQL trigger functions: their function bodies used a single-dollar delimiter ($) rather than a valid PostgreSQL dollar-quote delimiter ($$). Corrected both the opening and closing delimiters for:

- prevent_permission_catalogue_delete
- guard_principal_role_history
- guard_mfa_recovery_code_history
- guard_parent_otp_challenge_history
- guard_role_permission_delegation_history

Only the delimiters changed; function logic was not modified. A complete source scan confirmed five corrected opening delimiters, five corrected closing delimiters, and no remaining instances of the malformed patterns. At the time of that source scan, PostgreSQL 18 parsing and migration execution had not yet been verified. The later user-reported cumulative test-database version 12 (`dirty=false`) supersedes the earlier 'V4 unapplied' status for `school_management_test`; dedicated trigger edge-case coverage still requires verification. Never apply migrations to `school_management` without separate explicit approval.


## 2026-11-03 Current-Branch Reconciliation

The historical source-behavior and blocker notes above describe an earlier pre-cutover snapshot. The current active branch now contains the principal-aware Go cutover:

- Staff login resolves a globally unique username to a staff principal; Platform Admin has a separate login service.
- Access JWTs carry principal identity/type and token version. Middleware reloads current principal state and effective permissions from PostgreSQL on every protected request.
- `RequirePermission` is implemented and has fail-closed middleware tests.
- Principal refresh-session repository/service code is wired into login, refresh and logout; the previously verified refresh-token concurrency/replay protections must remain covered.
- Mandatory MFA flows and login lockout/IP rate-limit components are present in the current source.
- The API router currently exposes authentication/MFA, `/auth/me`, `/health` and `/ready`; it does not yet expose Platform Admin school-onboarding or school business-module routes.
- The permission resolver intentionally fails closed for custom-role grants and user ALLOW overrides until exact delegation linkage is safe. This is a known functional limitation, not a completed custom-role workflow.

### Newly identified school-onboarding prerequisite

The V4 migration seeds school-scoped system roles (`school_admin` and `teacher`) only for school rows that exist when V4 executes. This gap is now addressed in source by additive migration V13 (`000013_provision_school_system_roles.up.sql`), which creates the exact approved system roles and permission bundles from a school-insert trigger while keeping direct runtime system-role creation/mutation blocked. Integration coverage is added in `school_system_role_provisioning_v13_integration_test.go`.

V13 is source-only and has not been applied or PostgreSQL-tested. **Do not implement the Platform Admin create-school/School Admin provisioning endpoint as complete until V13 is reviewed, applied to `school_management_test`, and its integration test passes.** V13 is forward-only; never edit an applied migration.

### Migration-state reconciliation still required

Migration-state clarification: the user directly reported `school_management_test.schema_migrations` at cumulative `version=12`, `dirty=false` on 2026-11-02. A dedicated V12 Parent–Guardian integration test run was reported PASS; a later verbose attempt skipped because the opt-in flag was omitted. This supersedes earlier statements in this document that V4/V12 were source-only at that time. Re-run integration tests with the required flag after relevant changes. Do not infer `school_management` state from the test database and do not apply migrations there without separate explicit approval.

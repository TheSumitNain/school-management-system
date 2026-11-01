# V4 Identity and RBAC Migration — Implementation Notes

## Status

- Design approval: received from the user (`MIGRATION V4 SCHEMA APPROVED`).
- Source baseline: commit `cb6af3b5abc5249fba420e2699d16e87b1d14664`.
- Database observed by the user: migration v3, `dirty = false`; 1 school, 1 staff user, 4 refresh sessions, 0 active sessions, 0 expired sessions.
- Username preflight: no case-insensitive duplicates and no usernames outside the approved lowercase ASCII format.
- Constraint/index inspection: completed against the development database.
- Migration v4 corrective review draft: `backend/migrations/000004_add_principal_rbac_foundation.up.sql`.
- Down migration: intentionally refuses rollback because v4 data may include platform and parent identities, roles, audit events, and security credentials.
- **Not applied. Not PostgreSQL-executed or integration-tested. Do not run yet.**

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

1. The SQL has not been run against PostgreSQL 18. Review and test it on a disposable database cloned from v3 before applying it to `school_management`.
2. The application code has not yet been refactored. The current JWT and middleware remain user-ID based and do not enforce RBAC or `token_version`. The migration alone does not deliver the approved authorization model.
3. The existing staff account's intended role is not established by the database counts. Do not assign `school_admin` or another role by assumption. Confirm the account's intended role and create its principal-role assignment through an audited bootstrap operation.
4. The app's migration cutover must update login to globally unique username + password, principal-aware JWT claims, DB-backed principal/token-version checks, principal-aware session rotation/revocation, and permission evaluation before the system is treated as RBAC-protected.
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

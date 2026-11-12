# Database and Migration Guide

## Database technology
PostgreSQL is accessed through pgx/pgxpool. Schema changes are maintained as versioned SQL migrations in `backend/migrations/`.

The initial deployment direction is one shared database and schema with strict school-level ownership enforced by the backend.

## Migration safety rules

1. Inspect the current migration files and their ordering before changing schema.
2. Prefer additive, reviewed migrations; include constraints and indexes that reinforce application invariants.
3. Test the full migration chain against a fresh disposable PostgreSQL database in CI.
4. Add PostgreSQL integration tests for repository behavior and constraints.
5. Never infer the version of a user's persistent database from CI.
6. Do not apply migrations to `school_management` or `school_management_test` without explicit user approval.
7. Never run destructive SQL as a cleanup shortcut.

## Domain areas represented in the schema

The migration history includes schools and identity/principal foundations, refresh sessions, RBAC/delegation/overrides, MFA and audit records, staff lifecycle, academic structure (including optional Hindi/English medium metadata on sections in V27), student/enrollment history, attendance, timetable periods and weekly schedules, and guardian relationships.

### Staff employee profiles (V28)

Employee details are stored separately from login credentials in `staff_profiles`, keyed by the staff principal and constrained to the same school as its `users` account. The profile stores employee code, full name, designation, one required mobile number, optional email and joining date, and Active/Inactive employment status. Employee codes are unique case-insensitively within a school.

V28 is additive: existing staff login accounts are not deleted, renamed, or assigned fabricated profile details. Existing accounts without a `staff_profiles` row are legacy profiles that must be completed through the Staff Management workflow.

Changing an employee profile to Inactive disables its canonical staff principal, increments its token version, and revokes active refresh sessions in the same transaction. V29 enforces the inverse guard on the auth_principals table: no code path can reactivate a staff principal while its profile is Inactive. Changing the profile back to Active does not automatically reactivate the login account; a School Admin must explicitly enable the account with a temporary password. Login enablement also requires a complete employee profile with Active employment status. The database rejects login reactivation while the employee profile remains Inactive.

New staff account and employee-profile creation occur in one repository transaction, so employee-code conflicts or profile validation failures cannot leave orphan login accounts. Existing staff accounts without profile rows remain visible as incomplete until an administrator completes their employee details.

The authoritative schema is the ordered SQL migration chain in `backend/migrations/`, not copied SQL excerpts in documentation. For current feature status, see [Project Status](../project/STATUS.md).

## Local database configuration

Use `backend/.env.example` as the configuration reference. Keep local `.env` files and all passwords, JWT secrets, TOTP encryption keys and other credentials out of Git.

## Required database verification

- Fresh-database up migration.
- Repository integration tests using isolated test data.
- Constraint/transaction/rollback behavior.
- Migration rollback or forward-recovery strategy where applicable.
- Tenant-boundary and assignment-scope negative tests.

GitHub Actions uses an ephemeral PostgreSQL database. Its success does not change or verify the schema version of any persistent local database.

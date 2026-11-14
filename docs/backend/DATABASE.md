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

Changing an employee profile to Inactive disables its canonical staff principal, increments its token version, and revokes active refresh sessions in the same transaction. V29 and V30 enforce the inverse guard on the auth_principals table: no code path can reactivate a staff principal unless a complete employee profile exists with Active employment status. Legacy profile-less accounts may continue using an already-active login, but must have their employee profile completed before a disabled login can be re-enabled. Changing the profile back to Active does not automatically reactivate the login account; a School Admin must explicitly enable the account with a temporary password. Login enablement also requires a complete employee profile with Active employment status. The database rejects login reactivation while the employee profile remains Inactive.

New staff account and employee-profile creation occur in one repository transaction, so employee-code conflicts or profile validation failures cannot leave orphan login accounts. Existing staff accounts without profile rows remain visible as incomplete until an administrator completes their employee details.

The authoritative schema is the ordered SQL migration chain in `backend/migrations/`, not copied SQL excerpts in documentation. For current feature status, see [Project Status](../project/STATUS.md).

### Student identity and year-scoped admission (V31–V32)

- `students` represents the permanent school-scoped identity. Its former `admission_number` column is retained only as nullable legacy data during the transition; V32 removes the school-wide uniqueness rule without deleting or rewriting existing values.
- `student_enrollments` owns the admission number and admission date for a specific academic year. V31 enforces paired nullability, non-empty values, and normalized uniqueness within school + academic year.
- A historical enrollment may remain NULL/NULL when its original year-specific admission identity cannot be proven. Do not copy a legacy student-level number into a year without evidence.
- V33 extends the existing immutable-history trigger: admission number/date cannot be changed after enrollment creation, even while active. The active-to-terminal status transition remains valid when identity fields are unchanged.
- New student creation and its first enrollment must be one transaction. Returning-student enrollment and promotion create only a new year-specific enrollment; promotion closes the previous enrollment without rewriting its identity.
- The V32 down migration refuses rollback when null or duplicate legacy student admission values exist. The V31 down migration refuses rollback once year-specific enrollment identities have been written.
- V34 adds a hierarchical `student_locations` catalogue with shared canonical rows and school-owned additions, normalized duplicate checks, parent-level and tenant-scope guards, and deactivate-not-delete lifecycle. `student_addresses` stores current/permanent address sections separately, supports same-as-current without copying values, and prevents cross-school location references.
- V34 rollback refuses to drop either table when address or location records exist. Do not seed guessed geography; import canonical reference data only from a reviewed authoritative source.
- V36 adds a one-to-one current `student_medical_details` profile plus append-only `student_medical_details_history` snapshots; each medical update preserves the prior version. `student_physical_details` is one-to-many and append-only so dated height/weight measurements and notes are never overwritten. All three tables enforce same-school composite foreign keys, bounded fields and immutable ownership/history.
- Medical/physical reads require `students.read_sensitive` and create an audit event; writes require `students.update` and create an audit event containing only version/presence metadata, never health values or notes. Teacher roles do not receive these restricted permissions. Rollback refuses while current, historical or measurement records exist.
- V35 adds optional student phone/email contact details in a separate one-row-per-student table. Phone/email values are not unique, shared household contact values are allowed, changes are audited without copying the PII values into audit metadata, and hard deletion is refused.
- V35 rollback refuses to drop contact rows. Contact details use the same `students.read_sensitive` / `students.update` live permission checks as the address section.
- V12 already enforces one active student/guardian link per pair with a partial unique index, while allowing later relationship periods after revocation; retain this behavior when changing guardian history.

## Local database configuration

Use `backend/.env.example` as the configuration reference. Keep local `.env` files and all passwords, JWT secrets, TOTP encryption keys and other credentials out of Git.

## Required database verification

- Fresh-database up migration.
- Repository integration tests using isolated test data.
- Constraint/transaction/rollback behavior.
- Migration rollback or forward-recovery strategy where applicable.
- Tenant-boundary and assignment-scope negative tests.

GitHub Actions uses an ephemeral PostgreSQL database. Its success does not change or verify the schema version of any persistent local database.

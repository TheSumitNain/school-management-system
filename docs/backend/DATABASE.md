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

The migration history includes schools and identity/principal foundations, refresh sessions, RBAC/delegation/overrides, MFA and audit records, staff lifecycle, academic structure, student/enrollment history, attendance and guardian relationships.

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

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

## Long-term database engineering standards

These rules apply to every new or modified domain, not only student and guardian data.

### Schema evolution
- Treat every applied migration as immutable. Correct deployed schema behavior with a new forward migration; never edit or renumber an applied migration.
- Keep each migration focused, ordered, reviewable and safe to retry only when its migration runner semantics explicitly support that behavior.
- Every up migration must have a deliberate down or forward-recovery strategy. A down migration must stop with a clear error rather than silently discard populated or newly meaningful data.
- Before a migration, inspect existing constraints, triggers, indexes, data shape and dependencies. Document compatibility assumptions and any manual deployment step.
- Prefer expand-and-contract changes for columns or tables used by deployed application versions; avoid a migration that requires an unsafe all-at-once application rollout.

### Integrity, tenancy and lifecycle
- Enforce invariants in PostgreSQL as well as in Go: NOT NULL, CHECK, UNIQUE/partial UNIQUE, foreign keys and composite school-scoped foreign keys where appropriate.
- Every tenant-owned relationship must prove school ownership at the database boundary; do not rely only on a client-supplied school ID or a prior application lookup.
- Use restrictive delete behavior for records with academic, financial, identity, guardian or audit history. Prefer explicit archive, deactivate, revoke or status transitions over hard deletion.
- Keep audit/history append-only where records establish who changed sensitive or historically meaningful data. Never place passwords, tokens, contact values, medical values or other unnecessary PII in audit metadata.
- Preserve historical facts. Do not rewrite old academic-year admission identities, guardian relationship periods or measurements to make current screens appear simpler.

### Transactions, concurrency and performance
- Put multi-row business operations in one transaction, including admission/enrollment creation, promotion, guardian linking, primary-contact changes and audit writes.
- Identify the row or key that serializes each competing operation; use appropriate row locks, unique constraints and deterministic lock ordering to prevent races and deadlocks.
- Add PostgreSQL integration tests for concurrent requests and rollback-on-failure whenever an invariant can be violated by two simultaneous operations.
- Add indexes for demonstrated filter, join, uniqueness and ordering patterns. Validate query plans for high-volume paths; avoid speculative duplicate indexes.
- Use bounded pagination and explicit ordering for list endpoints. Watch for N+1 queries, unbounded history reads and expensive wildcard searches as school data grows.
- Review query and index changes against realistic school sizes and expected multi-school growth; record the reason for non-obvious indexes.

### Verification and recovery
- For every schema change, test the complete migration chain on a fresh disposable PostgreSQL database and test upgrading a representative prior schema/data state.
- Test constraints, tenant isolation, transaction rollback, concurrency and the down/forward-recovery path that is safe for the data involved.
- CI must use an ephemeral database. Local integration tests must use a dedicated disposable test database, never a production or user data database by accident.
- Maintain documented backup frequency, retention, encryption/access controls and restore procedure for persistent environments. A backup is not verified until a restore has been rehearsed and checked.
- Before any persistent-database migration or recovery, identify the target database, inspect its migration state, verify a restorable backup and obtain explicit approval. Never delete a backup merely because the application currently starts.
- Record actual migration/test/restore evidence and outstanding gaps in project status; never infer local database health from a green CI run.

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
- V35 adds optional student contact details with normalized phone/email validation and audit metadata that records presence only. V36 adds restricted medical details with append-only prior-version snapshots and dated append-only physical measurements; health values are never copied into audit metadata.
- V37 adds optional `student_basic_details` (date/place of birth, gender, nationality, religion, social category, EWS status and school registration number), protected by `students.read_sensitive` / `students.update`. Aadhaar is intentionally excluded from the default profile. The V37 down migration refuses rollback when profile rows exist.
- V34 rollback refuses to drop either table when address or location records exist. Do not seed guessed geography; import canonical reference data only from a reviewed authoritative source.
- V36 adds a one-to-one current `student_medical_details` profile plus append-only `student_medical_details_history` snapshots; each medical update preserves the prior version. `student_physical_details` is one-to-many and append-only so dated height/weight measurements and notes are never overwritten. All three tables enforce same-school composite foreign keys, bounded fields and immutable ownership/history.
- Medical/physical reads require `students.read_sensitive` and create an audit event; writes require `students.update` and create an audit event containing only version/presence metadata, never health values or notes. Teacher roles do not receive these restricted permissions. Rollback refuses while current, historical or measurement records exist.
- V35 adds optional student phone/email contact details in a separate one-row-per-student table. Phone/email values are not unique, shared household contact values are allowed, changes are audited without copying the PII values into audit metadata, and hard deletion is refused.
- V35 rollback refuses to drop contact rows. Contact details use the same `students.read_sensitive` / `students.update` live permission checks as the address section.
- V12 already enforces one active student/guardian link per pair with a partial unique index, while allowing later relationship periods after revocation; retain this behavior when changing guardian history.

### Guardian relationship editability (V41)

V41 updates the existing guardian-history trigger so an active student–guardian link may change its relationship label while preserving immutable ownership and identity (id, school_id, student_id, guardian_id, created_at). Revoked relationship rows remain immutable, and revocation actor/reason validation remains enforced. The migration does not rewrite V12; its down migration restores the previous relationship-label immutability rule. PostgreSQL integration coverage verifies relationship edits, primary-contact exclusivity, cross-student isolation, revoked-link protection and audit behavior.

## India administrative-location reference data

The intended authoritative source for the address hierarchy is the Ministry of Panchayati Raj's Local Government Directory (LGD), published through the Government of India's Open Government Data platform. The LGD catalog describes revenue entities including districts, sub-districts and villages, and is updated monthly. Use the matching LGD revenue-entity resources; do not substitute local-body or panchayat rows for districts, tehsils or villages.

A future LGD import must:
- retain the official LGD code as the stable source identifier and record source system, source snapshot date and ingestion provenance;
- validate every parent-child edge (State → District → Sub-District → Village) before writing, including duplicate codes, missing parents and name/code drift;
- stage and review a complete snapshot before an atomic upsert; never partially replace the active hierarchy after a failed download or validation;
- deactivate or version entities removed/renamed upstream rather than hard-delete rows referenced by student addresses;
- preserve school-owned custom locations and distinguish them from canonical LGD rows;
- report source row counts, rejected rows and changes by level, and require human review for unexpected large changes;
- retain the exact downloaded source snapshot and its checksum for reproducibility, with required Government Open Data License attribution.

Do not build a runtime dependency on an unofficial mirror. Import only from a reviewed official LGD/OGD download or a verified official API response. The current V40 State/UT seed is separate; no lower-level national hierarchy has been fabricated.

### Revenue sub-districts versus development blocks

Do not treat a revenue Sub-District (commonly called a Tehsil/Taluk/Mandal, depending on the State/UT) as interchangeable with a Development Block. They are distinct administrative concepts and LGD publishes them as separate entity types. The student address locality chain currently models the revenue path State/UT → District → Sub-District → Village/City; the database level `subdistrict` represents only the revenue sub-district. Do not label it simply as Block or seed development-block records into it. If a future school workflow needs Development Block, add a separate optional relationship/reference level with its own LGD source codes and parent validation rather than inserting it into the revenue address chain. Urban localities may require a separately reviewed mapping strategy where LGD's village/revenue path does not directly represent a city address.

### Multiple-source location provenance (V42)

V42 adds `student_location_source_references` so a canonical location can retain multiple authoritative identifiers without overwriting V40's IGOD State/UT code. Source namespaces are entity-specific (for example, `LGD_STATE`, `LGD_DISTRICT`, `LGD_SUBDISTRICT`, `LGD_VILLAGE`) because codes from different LGD entity classes must not be assumed globally interchangeable. Each current source identifier is unique within its namespace; superseded identifiers remain as immutable history with their source name and snapshot date. School-owned/manual locations cannot receive canonical source references. The V42 down migration refuses to discard anything beyond the exact 36-row IGOD backfill.

An LGD importer must resolve parent references by source code, validate the entire staged hierarchy and expected row-count drift before writing, then apply the snapshot in one transaction. Match an existing canonical row by its current source reference first; use reviewed state-code crosswalks to attach LGD state identifiers to the existing IGOD State/UT rows. Never match records by name alone. When an upstream identifier or source name changes, supersede the old reference and add a new snapshot record; preserve the location row identity where the entity is demonstrably the same. If identity is ambiguous (split/merge/reorganization), require a reviewed mapping and retain old address references rather than guessing.

## Local database configuration

Use `backend/.env.example` as the configuration reference. Keep local `.env` files and all passwords, JWT secrets, TOTP encryption keys and other credentials out of Git.

## Required database verification

- Fresh-database up migration.
- Repository integration tests using isolated test data.
- Constraint/transaction/rollback behavior.
- Migration rollback or forward-recovery strategy where applicable.
- Tenant-boundary and assignment-scope negative tests.

GitHub Actions uses an ephemeral PostgreSQL database. Its success does not change or verify the schema version of any persistent local database.

## Read-only migration status check

From the `backend` directory, run `go run ./cmd/dbstatus` after loading the intended `.env` configuration. This command connects to the configured database and reports its database name, migration version and dirty flag. It performs no DDL or DML and never applies, rolls back or repairs migrations. A dirty state or malformed migration metadata is reported as an error and must be investigated before any migration attempt. Always verify `DB_NAME` before running it; do not paste credentials into command history or logs.

Official source entry points reviewed on 2026-11-14:
- OGD LGD catalog (Ministry of Panchayati Raj): https://data.gov.in/catalog/local-government-directory-lgd
- LGD District dataset: https://data.gov.in/resource/local-government-directory-lgd-districts
- LGD Sub-District dataset: https://ap.data.gov.in/resource/local-government-directory-lgd-sub-districts
- LGD Village dataset: https://ap.data.gov.in/datasets_webservices/datasets/7163864
- Ministry LGD hierarchy overview (shows revenue entities and development blocks as distinct branches): https://panchayat.gov.in/lgd/

The OGD catalog and its District resource describe monthly updates; record the exact source snapshot/update date actually downloaded, not the page's current crawl date. OGD pages expose download/API entry points, but automated access and the current export schema must be verified against the actual official resource response before implementing a parser. The official catalog page itself is the source-of-truth discovery point; do not assume a community API mirrors the current official dataset.

# Backend Testing

## Local checks

From `D:\school-management-system\backend`:

```cmd
go test -count=1 -p 1 ./...
go vet ./...
```

Sequential package execution (`-p 1`) is preferred in the reported Windows environment to avoid parallel test-binary cleanup warnings.

## PostgreSQL integration tests

Integration tests are opt-in and require:
- `RUN_POSTGRES_INTEGRATION_TESTS=1`
- a dedicated disposable/test database, normally `school_management_test`
- the complete migration chain applied to that test database.

Never point integration tests at production or a persistent user database without explicit approval. Do not infer a local database's migration version from CI.

## Current automated coverage areas

- Authentication, JWT/session validation, refresh rotation/replay/revocation and concurrency.
- Password policy, staff/Platform Admin password changes and account lifecycle.
- MFA enrollment, TOTP/recovery flows, lockout/auditing and encryption-key rotation.
- Live permission resolution, role/delegation/override workflows and route permission gates.
- School onboarding and platform/staff account lifecycle.
- Selected student, enrollment, attendance and guardian repository/API boundaries.
- PostgreSQL migration and integration behavior.

## Known test gaps

- Exhaustive real-JWT HTTP coverage for every protected route.
- Complete cross-school/resource/assignment/field-level authorization tests across all modules.
- Additional database outage, concurrency and rollback cases.
- Migration rollback/forward-recovery verification.
- Browser end-to-end tests (frontend not yet implemented).
- Load, penetration, deployment and disaster-recovery testing.

## Verification evidence

The current project status records the latest known local and CI verification. CI applies migrations to ephemeral PostgreSQL; it does not touch persistent local databases. A passing test suite is not a production-readiness certification.

Add regression tests for every security or data-isolation fix. Never record a test as passed until the command or workflow has actually completed.

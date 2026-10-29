# Testing Documentation

## 1. Purpose

This document describes the testing strategy for the School Management System.

Testing must verify application correctness, security, database behavior, and reliability as the system grows.

The project is in its initial development stage. Tests documented as future requirements must not be treated as completed tests.

## 2. Current Testing Status

The backend has automated Go tests for selected packages and components.

Current test areas include:

- Password-service behavior.
- Authentication-service behavior.
- JWT access-token generation and validation.
- Authentication login-handler behavior.
- API server behavior and selected configuration validation.

Previously verified commands:

```bash
go test -count=1 ./...
go vet ./...
```

The test suite passed at the time of the last recorded verification. These historical results do not establish that the latest code and documentation changes have been verified.

The current test coverage is not a guarantee that all application behavior is tested. Refresh-token service and handler unit tests have been added in the current review workspace, but PostgreSQL repository integration tests, concurrent refresh rotation/replay tests, RBAC, tenant isolation, and most school business modules remain incomplete. Run the full suite before treating the new changes as verified.

Run the commands again after code or dependency changes before treating the current code as verified.

## 3. Testing Principles

- Test observable behavior rather than implementation details alone.
- Cover successful operations and expected failures.
- Test security boundaries explicitly.
- Keep tests repeatable and independent where practical.
- Use isolated test data and avoid depending on production data.
- Test database constraints as well as application-level validation.
- Do not claim production readiness based only on unit-test success.
- Add regression tests when fixing a bug.
- Distinguish automated tests from manual verification.
- Record test results only after the commands have actually completed.

## 4. Test Categories

### 4.1 Unit tests

Unit tests verify individual functions and business rules without requiring a complete running application.

Current examples include password-service tests, authentication-service tests using fake repositories, JWT token-service tests, and login-handler tests.

Future unit tests should cover:

- Input validation.
- User and school status rules.
- Authorization decisions.
- Business rules for attendance and other modules.
- Error handling.
- Edge cases and boundary values.

### 4.2 Repository and database integration tests

Integration tests verify behavior against an actual PostgreSQL database.

They should cover:

- Successful inserts, reads, updates, and deletes.
- Foreign-key constraints.
- Unique constraints and duplicate records.
- Transaction rollback.
- Query filtering and pagination.
- Database errors.
- Migration behavior.

Tests that require PostgreSQL should use a dedicated test database or isolated test schema. They must not modify production data.

A dedicated automated PostgreSQL integration-test setup remains future work unless separately verified.

### 4.3 HTTP tests

HTTP tests should exercise the Gin router and verify:

- HTTP status codes.
- Response bodies and content types.
- Request validation.
- Error responses.
- Health and readiness behavior.
- Login-handler behavior.
- Authentication and authorization when those routes are implemented.

### 4.4 Security tests

Security tests must verify that unauthorized access is denied and sensitive data is not exposed.

The security test plan is detailed in `SECURITY.md`.

### 4.5 Load and performance tests

Load tests should use realistic request patterns and representative data volumes.

Measure:

- Request throughput.
- Response latency, including p95 and p99.
- Error rate.
- CPU and memory usage.
- PostgreSQL connection usage.
- Slow queries and lock contention.

The target of 25–30 schools and approximately 10,000–15,000 students is a planning target. No concurrency or capacity guarantee should be made without reproducible load-test results.

## 5. Running Backend Tests

Run commands from the backend directory:

```bat
cd /d D:\school-management-system\backend
```

Run all Go tests:

```bash
go test -count=1 ./...
```

The `-count=1` flag disables reuse of cached test results for this run.

Run static analysis:

```bash
go vet ./...
```

Format Go source files when necessary:

```bash
gofmt -w cmd internal
```

Review formatting changes before committing them. Run `gofmt` only on Go source directories and review the resulting diff before staging files.

## 6. Current Test Areas

### Password service

Verify:

- Valid passwords can be hashed.
- Password verification succeeds for the matching password.
- An incorrect password fails verification.
- Passwords outside the supported byte-length range are rejected.
- Whitespace-only passwords are rejected.
- Hashing does not expose the original password.

### Authentication service

Verify:

- A user can be created when the required school and account conditions are satisfied.
- Inactive or suspended schools cannot authenticate users.
- Disabled users cannot authenticate.
- Invalid credentials are rejected.
- Repository errors are handled safely.
- Duplicate usernames are handled correctly by the repository/database boundary.

### Refresh-token lifecycle

The current working changes add unit and handler tests for:
- Storing only a SHA-256 hash of a generated refresh token.
- Configurable refresh expiry and expiry validation.
- Refresh-token rotation and invalid-token responses.
- Logout forwarding the token for revocation.
- Refresh-token reuse error handling.

Before production, add PostgreSQL integration tests for transaction atomicity, family revocation on replay, inactive-user/school revocation, and concurrent rotation. Use a dedicated test database and do not run integration tests against production data.

### JWT token service

The token service has an initial automated test suite.

Verify:

- Valid access tokens can be generated and validated.
- Invalid tokens are rejected.
- Expired tokens are rejected.
- Invalid signatures are rejected.
- Issuer and required claims are validated.
- Invalid token configuration is rejected where covered by the implementation.

The exact cases covered must be checked against the actual test files.

### Login handler

The login handler has an initial automated test suite.

Verify:

- Valid requests return the expected success response.
- Invalid credentials return an unauthorized response.
- Malformed requests return an appropriate client error.
- Internal token-generation failures do not expose sensitive internal details.
- Password hashes and other sensitive fields are not included in responses.

The exact cases covered must be checked against the actual test files.

## 7. Health and Readiness Tests

### Health endpoint

`GET /health`

Verify:

- A successful response is returned while the API is running.
- The status code and response body match the implementation.

### Readiness endpoint

`GET /ready`

Verify:

- The endpoint succeeds when the database is available.
- The endpoint returns the intended unavailable status when the database check fails.
- Database failure information is not exposed inappropriately.

Re-run these tests whenever server initialization or database connection handling changes.

## 8. Future Authentication and Authorization Tests

Once the remaining HTTP authentication flow is implemented, test:

- Registration validation and secure provisioning.
- Login with valid and invalid credentials.
- Access-token signature and expiry validation.
- Refresh-token rotation.
- Reuse of an old refresh token.
- Refresh-token revocation after logout.
- Disabled-user behavior.
- Missing or malformed authorization headers.
- Invalid token type and claims.
- Concurrent refresh requests.
- Rate limiting for repeated failed login attempts.

Once authentication middleware and RBAC are implemented, test that unauthenticated requests are denied and every role can perform only its authorized operations.

These are required future tests, not evidence that the corresponding functionality currently exists.

## 9. Multi-Tenant Isolation Tests

Cross-school data leakage is a critical risk for this product.

Create test data for at least two schools and verify:

- A user from School A cannot read School B's records.
- A user from School A cannot update or delete School B's records.
- List endpoints return only authorized school records.
- Related-record lookups cannot bypass school restrictions.
- Client-supplied school identifiers cannot override the authenticated school context.
- Administrative operations enforce the intended scope.

Repeat these checks for every school-owned module as it is introduced.

Automated tenant-isolation tests remain future work until the corresponding protected endpoints and authorization mechanisms exist.

## 10. Database Migration Tests

For each migration:

1. Apply it to a clean test database.
2. Verify that the expected tables, constraints, and indexes exist.
3. Verify that the application can use the resulting schema.
4. Test rollback behavior where the down migration is safe and supported.
5. Test the migration against a database with the expected previous schema.
6. Check how failures are reported and recovered from.

Do not test destructive rollback procedures against production data.

## 11. Test Data and Environment

- Keep test credentials separate from development and production credentials.
- Never commit secrets to test configuration.
- Use deterministic test fixtures where practical.
- Clean up test data safely.
- Avoid tests that depend on execution order.
- Use a dedicated database for tests that perform writes.
- Ensure tests do not accidentally connect to production.
- Avoid printing tokens, password hashes, or credentials in test output.

## 12. Before Committing Changes

From the backend directory:

```bash
go test -count=1 ./...
go vet ./...
```

From the repository root:

```bash
git diff --check
git status --short
```

Inspect the changed files and verify that no secrets, generated binaries, or unintended files are included.

For changes involving database migrations, run the relevant migration and database tests as well.

Documentation-only changes do not replace the need to verify the code changes already pending in the working tree.

## 13. Before Production Release

The release checklist should include:

- [ ] All Go tests pass.
- [ ] Static analysis passes.
- [ ] Relevant database integration tests pass.
- [ ] HTTP integration tests pass.
- [ ] Authentication and authorization tests pass.
- [ ] Cross-school isolation tests pass.
- [ ] Database migrations have been tested.
- [ ] Dependency and vulnerability checks have been run.
- [ ] Load tests have been run for the expected workload.
- [ ] Backup restoration has been tested.
- [ ] Monitoring and error reporting have been reviewed.

A checked item must be supported by an actual verification result.

## 14. Maintenance

Whenever a bug is fixed or functionality changes:

1. Add or update the relevant tests.
2. Run the affected tests.
3. Run the full backend test suite when appropriate.
4. Update `PROGRESS.md` with verified results.
5. Update this document if the testing process or test coverage changes.

Keep testing evidence separate from planned work.

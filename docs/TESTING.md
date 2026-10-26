
# Testing Strategy

## 1. Purpose

Testing helps ensure the School Management System is correct, secure, reliable and maintainable.

Every feature must be tested before it is considered complete.

## 2. Types of Testing

### Unit Tests

- Test individual functions and business rules.
- Cover validation, calculations and permission decisions.
- Keep tests independent wherever practical.

### Integration Tests

- Test interactions between the API, PostgreSQL and other components.
- Verify database constraints and transaction behavior.
- Test migrations against a test database.

### API Tests

- Test successful requests and invalid requests.
- Verify HTTP status codes and response formats.
- Test missing, malformed, expired and revoked authentication tokens.
- Verify validation and error handling.

### Authorization Tests

- Verify that each role can perform only permitted actions.
- Test that users cannot access another school's records.
- Test that teachers are restricted to assigned classes and subjects.
- Test that parents can access only their verified linked children.
- Test suspended-school access restrictions.

### Frontend Tests

- Test important React components and user interactions.
- Verify loading, success, empty and error states.
- Check form validation and protected navigation.
- Never treat frontend restrictions as a substitute for backend authorization.

### End-to-End Tests

- Test important user workflows across the frontend, backend and database.
- Add these tests incrementally as complete modules become available.

## 3. Go Backend

- Use Go's standard `testing` package.
- Run `go test ./...` regularly.
- Use table-driven tests where they improve clarity.
- Keep tests repeatable and independent of production data.
- Run tests after meaningful changes.

## 4. PostgreSQL Testing

- Use a separate test database.
- Never run automated tests against production data.
- Verify migrations on a clean database.
- Test primary keys, foreign keys, unique constraints and required fields.
- Test rollback behavior for failed transactions.
- Clean up test data safely.

## 5. React Frontend Testing

- Use a suitable React testing library.
- Test important components and form behavior.
- Test API loading, success and failure states.
- Check that users cannot access restricted pages through ordinary navigation.
- Verify usability at common desktop and mobile screen sizes.

## 6. Security Test Cases

At minimum, verify:

- Invalid credentials and login lockout.
- MFA verification and recovery.
- Access-token expiry and revocation.
- Refresh-token rotation and reuse prevention.
- Logout and session invalidation.
- Role and permission enforcement.
- Cross-school data isolation.
- Parent-child relationship restrictions.
- Invitation expiry and reuse prevention.
- Suspended-school restrictions.
- Input validation and rate limiting.

## 7. Performance Testing

- Establish realistic expected workloads before production.
- Test concurrent requests and common database queries.
- Measure response times, error rates and resource usage.
- Test pagination and large list endpoints.
- Investigate slow queries and missing indexes.
- Increase load gradually and document the results.

The initial target is approximately 25–30 schools with 400–500 students per school. Actual capacity must be verified through testing rather than assumed.

## 8. Test Data and Secrets

- Use synthetic student, parent and staff data.
- Never commit passwords, tokens, database credentials or production secrets.
- Keep local test configuration separate from production configuration.
- Ensure test reports do not expose sensitive information.

## 9. Definition of Done

A feature is complete only when:

- Its expected behavior is implemented.
- Relevant validation and authorization are enforced on the backend.
- Appropriate automated tests pass.
- Database changes use version-controlled migrations.
- Documentation is updated where necessary.
- No secrets or real student data have been committed.

## 10. Current Status

Testing strategy documented. Specific test tools and automated test coverage will be introduced incrementally during implementation.



## 11. Implemented Automated Tests

The following API tests are currently implemented in `backend/cmd/api/main_test.go`:

### Health Endpoint

- Verify that `GET /health` returns HTTP `200 OK`.
- Verify the JSON content type.
- Verify the expected JSON response body.

### Readiness Endpoint

- Verify that `GET /ready` returns HTTP `200 OK` when the database ping succeeds.
- Verify that `GET /ready` returns HTTP `503 Service Unavailable` when the database ping fails.
- Verify the expected JSON response bodies for both scenarios.

### Test Implementation

- Tests use Go's standard `testing` package.
- HTTP requests are tested using `net/http/httptest`.
- A fake database implementation simulates successful and failed database pings.
- API tests do not require a live PostgreSQL connection for these scenarios.

### Current Verification

Run the following command from the `backend` directory:

```bash
go test ./...
```

**Current status:** All existing Go packages compile, and the implemented API tests pass.

### Remaining Test Coverage

The following areas still require dedicated tests:

- Environment configuration and invalid configuration values.
- PostgreSQL connection failures and connection-pool behavior.
- Database migrations and rollback behavior.
- HTTP server shutdown behavior.
- Request timeouts and API error handling.
- Authentication, authorization, and school-level tenant isolation when implemented.
- Frontend components and end-to-end workflows when implemented.

These are planned test areas, not completed test coverage.

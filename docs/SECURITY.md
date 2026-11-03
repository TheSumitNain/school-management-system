# Security Documentation

## 1. Purpose

This document describes the current security foundation, security decisions, known gaps, and production security requirements for the School Management System.

The application is designed to support multiple schools. Student, parent, teacher, and school information must be protected against unauthorized access.

**Important:** This document distinguishes implemented controls from planned controls. A documented requirement does not mean the corresponding feature has already been implemented or verified for production.

## 2. Current Implementation Status

### Implemented foundation

- PostgreSQL database connection uses application configuration and a connection pool.
- User records store password hashes rather than plaintext passwords.
- Password hashing uses Go's bcrypt implementation with cost 12.
- The current password implementation rejects passwords shorter than 12 bytes, longer than 72 bytes, whitespace-only values, and entries in a small local common-password denylist.
- Password verification is implemented.
- Authentication service logic checks whether the associated school and user are active.
- JWT access-token generation and validation are implemented.
- JWT configuration includes issuer and expiry settings, with signing-secret validation.
- `POST /auth/login` is implemented.
- Successful login returns an access token, a refresh token, and a limited user DTO.
- Invalid credentials return a generic unauthorized response.
- JWT authentication middleware validates Bearer access tokens.
- `GET /auth/me` is protected by authentication middleware and returns the authenticated user's ID and school ID.
- Refresh-session database migrations include token hashes, expiry, revocation, and refresh-token family metadata.
- Login issues an opaque, cryptographically random refresh token and stores only its SHA-256 hash.
- `POST /auth/refresh` rotates refresh tokens.
- Reuse of a revoked refresh token triggers revocation of active sessions in its token family.
- `POST /auth/logout` revokes all active refresh sessions in the supplied token family; repeated logout is idempotent and unknown tokens are a no-op.
- HTTP server timeouts, graceful shutdown, health checks, and readiness checks are implemented.
- Automated tests cover the implemented authentication foundation and service-level token lifecycle behavior.

These features establish an initial authentication foundation. They do not mean the complete application has been security-certified or is ready for production use.

### Not yet implemented or fully verified

- Immediate revocation of already-issued access JWTs.
- Complete tenant/resource/assignment/field-level enforcement across school business services and repository queries.
- Safe custom-role grants and user ALLOW overrides (currently fail closed pending exact delegation-ID linkage).
- Staff password change/reset workflows and immediate invalidation of already-issued access tokens after administrative security changes.
- Compromised-password intelligence checks and comprehensive production abuse monitoring.
- Session invalidation after password changes, password resets, or other administrative security events.
- Complete current-user profile retrieval from `GET /auth/me`.
- Secure school and user provisioning workflows.
- Comprehensive security audit logging and monitoring.
- Production TLS and deployment-specific security configuration.
- Broader PostgreSQL repository integration tests beyond the existing refresh-session lifecycle coverage.
- Additional concurrent refresh-token rotation edge cases and broader race-condition testing.
- A clean, independent end-to-end logout and refresh-after-logout verification.
- Full production security testing, deployment review, and incident-response procedures.

The authentication middleware currently protects `GET /auth/me`. Every additional protected route must explicitly enforce authentication and authorization.

## 3. Password Security

### Current implementation

The password service uses bcrypt with a cost factor of 12. Passwords are hashed before storage, and password verification compares a supplied password against the stored hash.

The current implementation enforces an input range of 12–72 bytes, rejects whitespace-only passwords, and checks a deliberately small local common-password denylist. This is not a breached-password intelligence service.

Bcrypt has a 72-byte input limit. Any future password-policy change must handle Unicode and byte length correctly and must not silently truncate passwords.

### Production requirements

Before public production deployment:

- Finalize and document the password policy.
- Enforce the intended minimum of 12 characters for administrator accounts, with clearly defined Unicode and byte-length handling.
- Consider compromised-password checks and a secure password-reset workflow.
- Never log plaintext passwords or expose them in error messages.
- Never store plaintext passwords.
- Evaluate whether bcrypt remains appropriate or whether Argon2id is preferable for the deployment environment.
- Implement a safe migration and rehash strategy if the hashing algorithm or policy changes.
- Revoke appropriate sessions after password changes and password resets.

The current minimum is 12 bytes and maximum is 72 bytes. The local common-password denylist is intentionally limited; a breached-password intelligence check and password-reset lifecycle remain outstanding.

## 4. Authentication and Token Security

### Access-token implementation

The token service generates and validates JWT access tokens. The login handler issues an access token after successful credential verification.

The authentication middleware:

1. Reads the `Authorization` header.
2. Requires the Bearer authentication scheme and a token.
3. Validates the token through the token-validation service.
4. Extracts the principal ID from the JWT subject.
5. Reloads canonical principal subtype, status, token version, school/account state and effective permissions from PostgreSQL.
6. Rejects inactive or mismatched identities and places canonical principal context and permissions in the Gin request context.
7. Allows the request to continue only after successful authentication.

Authentication failures return `401 Unauthorized`. If the middleware has no token validator configured, it returns `500 Internal Server Error` to indicate an internal configuration problem.

`GET /auth/me` returns the authenticated principal ID/type and staff-only school/username context. Protected requests reload current principal status, token version and permissions. The endpoint does not provide a complete user profile or prove resource-level tenant isolation.

The middleware establishes token-based identity. It does not independently guarantee that a user remains active in the database on every request, that the user has permission to perform an operation, or that all database queries are restricted to the correct school.

### Refresh-token lifecycle

The refresh-token HTTP lifecycle is implemented with the following behavior:

- `POST /auth/login` issues an access token and a refresh token.
- `POST /auth/refresh` validates a refresh token and rotates it.
- `POST /auth/logout` revokes all active refresh sessions in the supplied token family; repeated logout is idempotent and unknown tokens are a no-op.
- Refresh tokens are opaque, cryptographically random values separate from JWT access tokens.
- Only a SHA-256 hash of a refresh token is stored in the database.
- Refresh-token expiry is configurable through `REFRESH_TOKEN_EXPIRY_DAYS`, with a default of 7 days and an allowed range of 1–30 days.
- Refresh rotation revokes the presented session and creates its replacement within a database transaction.
- Refresh rotation and logout acquire the same transaction-scoped PostgreSQL advisory lock for the token family. Rotation additionally acquires a per-user transaction advisory lock to serialize concurrent rotations across that user's token families, then performs a fresh row-locked lookup.
- Reuse of a revoked refresh token triggers revocation of the remaining active sessions in its token family.
- Refresh processing checks user and school status.

The implemented service behavior and opt-in PostgreSQL repository integration tests provide a foundation for refresh-token rotation and replay protection. Existing integration tests cover selected refresh-session lifecycle scenarios, including concurrent refresh behavior. Broader race-condition coverage and independent end-to-end HTTP logout verification remain outstanding.

### Logout limitations

Logout revokes every active refresh session in the supplied token family. Repeated logout is idempotent and unknown tokens are a no-op. It does **not** immediately invalidate access JWTs that have already been issued. Those tokens remain valid until they expire unless an additional revocation mechanism is introduced.

Before production deployment, decide whether short access-token lifetimes are sufficient or whether immediate revocation is required. Also define session invalidation behavior after password changes, account disabling, and administrative security events.

### Client-side token handling

The current HTTP contract returns access and refresh tokens in JSON.

Clients must not expose tokens in application logs, URLs, analytics events, or error reports. Browser integration must carefully evaluate token storage and the risks of cross-site scripting (XSS).

If cookies are used, configure appropriate `HttpOnly`, `Secure`, and `SameSite` attributes and implement suitable CSRF protections. The transport and storage approach must be reviewed before production browser deployment.

### Remaining requirements

- Keep access tokens short-lived and document the configured lifetime.
- Validate token signature, issuer, expiry, and relevant claims.
- Keep signing keys and secrets outside source control.
- Define a key-rotation strategy.
- Apply authentication middleware to every route requiring authentication.
- Continue testing concurrent refresh/logout ordering and transaction behavior against PostgreSQL, including database failure cases.
- Test refresh-token replay and logout independently through the HTTP API.
- Define how disabled users, inactive schools, and revoked sessions affect already-issued access tokens.
- Return generic authentication errors where detailed responses could reveal sensitive information.
- Never log access tokens, refresh tokens, passwords, signing keys, or database credentials.

## 5. Multi-Tenant Isolation

The application is designed to serve multiple schools using a shared PostgreSQL database and shared schema.

Each school-owned record must be associated with the correct school. Users must only access data that their authenticated identity and permissions authorize.

### Required controls

- Derive the authenticated identity and authorized school context from trusted server-side authentication context.
- Never trust a request-supplied `school_id` as proof of authorization.
- Scope every school-owned read, update, delete, and list query to the authorized school.
- Verify school ownership when accessing related records.
- Enforce appropriate database foreign keys and uniqueness constraints.
- Add automated tests proving that a user from one school cannot access another school's records.
- Review background jobs, exports, reports, and administrative operations for tenant leaks.
- Ensure cross-school access is possible only through explicitly authorized platform-level workflows.

Tenant isolation must be enforced in backend services and repository queries, not just in the frontend.

**Current status:** Authentication middleware loads canonical school context from the database-backed principal and validates current status/token version. Complete tenant-isolation enforcement across business resources remains pending because the school business APIs and their resource-scope filters are not yet implemented. A school_id column or valid token alone does not prove tenant isolation.

## 6. Authorization and Roles

The core principal/RBAC schema, built-in role bundles, live system-role permission resolver and RequirePermission middleware are implemented. Custom-role grants and user ALLOW overrides remain disabled until exact delegation-ID linkage is enforced. Full route/resource/tenant/assignment authorization remains pending. Phase 1 web roles are Platform Admin, School Admin and Teacher; Parent and Driver are Phase 2.

The implementation must enforce:

- One-school-only School Admin and staff accounts; separate accounts for a person employed by multiple schools.
- Globally unique, case-insensitive staff usernames. The current schema/login does not yet meet this target.
- School Admin staff provisioning with temporary password and forced first-login change; staff self-change requires current password.
- Admin-initiated same-school staff password reset without old password, with session/access revocation, forced next-login change, and audit logging.
- School Admin-created custom roles limited to the platform permission catalogue and an explicit delegation ceiling.
- Multiple roles per staff account within its one school; per-user allow/deny overrides; explicit deny beats role allow; default deny.
- Separate teacher teaching and class-in-charge assignments, field-level student data restrictions, and exact class/section scoping.
- Parent OTP identity linked to verified children; parent can access multiple linked children across schools, with server-side link verification on every request.
- Server-side permission evaluation on every protected request so changes apply to existing sessions on the next request.
- Cross-school, cross-class, cross-subject, field-level, and parent-child access-denial tests.

The backend must enforce permissions independently of the frontend. Hiding a button or route in React is not an authorization control.

The backend must enforce permissions independently of the frontend. Hiding a button or route in React is not an authorization control.

The current authentication middleware establishes identity only. It does not prove that a user is authorized to perform every operation or access every resource within the user's school.

## 7. Database Security

### Current foundation

- Database access is configured through application configuration.
- PostgreSQL connections use a connection pool.
- The schema includes foreign-key and data-integrity constraints.
- The user schema stores password hashes rather than plaintext passwords.
- Database migrations are tracked in the repository.

### Production requirements

- Use a dedicated database role with only the permissions required by the application.
- Never run the production application as a PostgreSQL superuser.
- Keep database credentials outside source control.
- Restrict database network access.
- Configure encrypted database connections appropriate to the deployment environment. Production configuration must require certificate verification where specified by the application configuration.
- Back up the database and periodically test restoration.
- Protect backups and exported data.
- Apply schema migrations through a controlled deployment process.
- Avoid exposing raw database errors to API clients.
- Review database privileges and tenant-related constraints before release.

Local development configuration is not proof of production database security.

## 8. Configuration and Secrets

- Keep local environment files such as `.env` out of Git.
- Provide safe example configuration without real credentials.
- Use environment variables or an appropriate secret-management system for production secrets.
- Validate required configuration at startup.
- Never commit database passwords, JWT signing keys, API keys, or private certificates.
- Rotate any secret that is accidentally exposed.
- Keep development and production credentials separate.
- Verify that local environment files are ignored and are not tracked by Git.

## 9. HTTP and API Security

Before production release:

- Use HTTPS for external application traffic.
- Configure trusted proxies explicitly when operating behind a reverse proxy.
- Validate and limit request sizes.
- Validate all input on the server.
- Configure CORS for known frontend origins rather than allowing every origin by default.
- Add rate limits, especially for login, refresh, password-reset, and future OTP endpoints.
- Avoid leaking stack traces or internal errors.
- Use secure headers and appropriate cookie settings if cookies are used for tokens.
- Ensure HTTP timeouts are appropriate for the deployment.
- Review file uploads, exports, and external integrations before exposing them.
- Test authentication endpoints against brute-force attempts, replay attempts, and abuse scenarios.

The existing HTTP timeouts, configuration validation, and graceful shutdown are only part of the broader security requirements. Identifier-based five-failure/15-minute lockout and IP-based login rate limiting are implemented in the current branch. Broader abuse monitoring and production alerting remain pending.

## 10. Development Seed Command

Location: `backend/cmd/devseed/`

The development seed command is intended for preparing test data for local development and is guarded by the development environment.

Security requirements:

- Do not use this command as a production user-provisioning mechanism.
- Do not put real credentials in source code or documentation.
- Keep seed credentials in environment variables.
- Verify the environment guard and production deployment separation before release.
- Ensure production workflows cannot accidentally create development test accounts.

The command's existence does not replace a secure school-administration or user-provisioning workflow.

## 11. Logging, Monitoring, and Auditing

Production requirements:

- Record security-relevant events such as login failures, account disabling, role changes, session revocation, and administrative actions.
- Never log passwords, raw tokens, signing keys, or database credentials.
- Avoid unnecessarily logging student or parent personal information.
- Define log retention and access policies.
- Monitor repeated failed authentication attempts and suspicious activity.
- Add audit records for sensitive administrative changes.
- Ensure logs do not become a source of cross-school data leakage.
- Define monitoring alerts and an incident-response process.

A complete security audit and monitoring system is not yet implemented.

## 12. Testing Requirements

Security testing must cover at least:

- Password hashing and verification.
- Invalid credentials and disabled users.
- Inactive or suspended schools.
- JWT signature validation, expiry, issuer, and claims.
- Authentication middleware behavior for valid and invalid tokens.
- Missing and malformed authorization headers.
- Invalid identity claims.
- Protected identity endpoint success and authentication failures.
- Refresh-token creation, rotation, expiry, revocation, and replay.
- Logout followed by refresh-token reuse.
- Concurrent refresh attempts and transaction behavior.
- Broader PostgreSQL repository integration tests for session lifecycle failure modes and additional repository behavior.
- Insufficient role permissions.
- Cross-school read and write attempts.
- SQL query parameterization.
- Configuration and secret-handling failures.
- Rate limiting and abuse protection.
- Production database permissions and backup restoration.

Tests should be automated where practical and run before changes are merged. Passing unit tests alone does not prove production security or complete tenant isolation.

## 13. Production Readiness Checklist

Before serving real schools and student data, verify:

- [ ] Authentication endpoints and middleware behavior are tested.
- [ ] Authentication middleware is applied to every route requiring authentication.
- [x] Selected refresh-token rotation, replay handling, logout revocation, expiry, and concurrent-refresh scenarios are covered by opt-in PostgreSQL repository integration tests.
- [ ] Broader concurrency/failure-mode and independent HTTP end-to-end verification is complete.
- [ ] Access-token expiry and revocation strategy is documented and implemented as required.
- [ ] RBAC permissions are defined and enforced server-side.
- [ ] Tenant isolation is verified for all school-owned resources.
- [ ] Password policy is finalized and implemented.
- [ ] MFA is implemented for privileged administrator accounts.
- [ ] Password reset and administrative session invalidation are implemented.
- [ ] Production TLS is configured and tested.
- [ ] Production secrets are securely managed.
- [ ] Database access follows least privilege.
- [ ] Rate limiting and authentication abuse protection are enabled.
- [ ] Sensitive information is excluded from logs and API errors.
- [ ] Security tests and dependency checks run in the release workflow.
- [ ] Backups and restoration procedures are tested.
- [ ] Monitoring and incident-response procedures are documented.
- [ ] The development seed command is excluded from production workflows.
- [ ] Deployment configuration and production security controls have been reviewed.

## 14. Maintenance

Whenever security behavior changes:

1. Update this document to reflect actual implementation.
2. Add or update automated tests.
3. Update `PROGRESS.md` and `CHANGELOG.md` when appropriate.
4. Record important architecture or security decisions in `ARCHITECTURE.md` or a dedicated decision record.
5. Never mark a security requirement complete based only on documentation.
6. Reassess this checklist before each production release.


## 15. Approved Phase 1 Account Security Rules (Implementation Pending)

### Staff account isolation

Every School Admin/staff account is tied to exactly one school. A staff account cannot be reused across schools. If a person works at multiple schools, each school provisions a separate account with separate credentials and sessions. Staff usernames are globally unique and case-insensitive. The current principal-aware schema enforces globally unique, case-insensitive staff usernames and the staff login resolves by username without a school ID.

### Password change and reset

Staff self-service password change must verify the current password. School Admin can reset a staff member's password without knowing the old password, but only for staff in the same school. Reset must revoke every refresh session for the target account and invalidate already-issued access tokens, require a password change at next login, and create an audit event. The password itself must never be logged or retrievable by administrators.

### Teacher and parent data boundaries

A subject-teaching assignment does not grant attendance access. Subject teachers see only name, roll number, class, and section for assigned classes/sections, plus permitted teaching-related data. Class-in-charge teachers may access attendance (same-day correction within configured cutoff), basic student details, timetable, homework, and applicable notices only for the exact class/section assigned. Past attendance correction is School Admin-only.

Parent uses registered mobile + OTP and can access multiple officially linked children across schools. The parent dashboard's child selection is not an authorization grant; each API request verifies the relationship. Parent data access is read-only in Phase 1, and students have no separate login accounts.

### Immediate enforcement

Role/permission changes, staff password resets, account disabling, and school suspension must affect subsequent protected requests. The access-token revocation mechanism is not yet implemented; design it before claiming immediate invalidation. Do not rely on refresh-session revocation alone to invalidate an already-issued access JWT.

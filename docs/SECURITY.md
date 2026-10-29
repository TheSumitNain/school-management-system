# Security Documentation

## 1. Purpose

This document describes the current security foundation, security decisions, known gaps, and production security requirements for the School Management System.

The application is intended to support multiple schools. Student, parent, teacher, and school information must be protected against unauthorized access.

**Important:** This document distinguishes implemented controls from planned controls. A documented requirement does not mean the corresponding feature has already been implemented.

## 2. Current Implementation Status

### Implemented foundation

- PostgreSQL database connection uses application configuration.
- User records include a password-hash field rather than a plaintext-password field.
- Password hashing uses Go's `bcrypt` implementation with cost 12.
- Password hashing rejects passwords shorter than 8 bytes or longer than 72 bytes.
- Whitespace-only passwords are rejected.
- Password verification is implemented.
- Authentication service logic checks whether the associated school is active.
- Authentication service logic checks whether the user is active and whether the supplied credentials are valid.
- JWT access-token generation and validation are implemented in the token service.
- JWT configuration includes issuer, expiry settings, and secret validation.
- `POST /auth/login` is implemented and has been manually tested.
- Login responses return an access token and a limited user DTO rather than the password hash.
- Invalid credentials return a generic unauthorized response.
- JWT authentication middleware is implemented and tested.
- The middleware validates Bearer access tokens and extracts the authenticated user's ID and school ID from validated token claims.
- `GET /auth/me` is implemented as a protected endpoint and returns the authenticated user's ID and school ID.
- Automated tests cover valid authentication, missing or malformed authorization headers, invalid tokens, invalid claims, and the protected identity endpoint.
- Refresh-session storage has database migrations for token hash, expiry, revocation, and rotation-family metadata.
- Login issues an opaque refresh token and stores only its SHA-256 hash.
- Refresh rotates the token atomically and checks user/school status.
- Reuse of a revoked refresh token revokes active sessions in its token family.
- Logout revokes the supplied refresh token; access JWTs are not immediately revoked.
- The API server has configured HTTP timeouts and graceful shutdown behavior.
- Health and readiness endpoints are implemented.

These controls form an initial foundation. They do not mean that production authentication or the entire application has been security-certified.

### Not yet implemented

- Immediate access-token revocation; logout currently revokes refresh sessions, while issued stateless access tokens remain valid until expiry.
- Registration or administrative user-provisioning HTTP workflow.
- Session invalidation after password changes, password resets, or other administrative security events.
- Full current-user profile retrieval from `GET /auth/me`.
- Role-based access control (RBAC).
- Complete tenant-isolation enforcement across business queries.
- Production TLS and deployment-specific security configuration.
- Rate limiting and abuse protection for authentication endpoints.
- Complete security audit logging and monitoring.

The authentication middleware currently protects `GET /auth/me`. Other protected routes must explicitly apply the middleware when implemented.

## 3. Password Security

### Current implementation

The password service uses bcrypt with a cost factor of 12.

Passwords are hashed before storage. Password verification compares the supplied password with the stored hash.

The current implementation enforces an 8–72-byte input range. Because bcrypt has a 72-byte input limit, future password-policy changes must preserve correct handling of that limit.

### Production requirements

Before public production deployment:

- Establish and document a final password policy.
- Consider a minimum password length of 12 characters, with clear Unicode and byte-length handling.
- Consider breached-password checks and appropriate password-reset workflows.
- Never log plaintext passwords or include them in error messages.
- Never store plaintext passwords.
- Review whether bcrypt remains appropriate or whether Argon2id is preferable for the deployment environment.
- If the hashing algorithm or policy changes, provide a safe migration and rehash strategy.

The proposed minimum of 12 characters is a production requirement to evaluate; it is not the current implementation's minimum.

## 4. Authentication and Token Security

### Current implementation

- The authentication service verifies credentials and checks school and user status.
- The token service generates and validates JWT access tokens.
- JWT configuration validates the signing secret and access-token expiry settings.
- The login handler returns an access token following successful authentication.
- The login endpoint has been manually tested with valid and invalid credentials.
- Authentication middleware validates Bearer access tokens for `GET /auth/me`.
- The middleware rejects missing or malformed authorization headers and invalid tokens.
- The middleware rejects token claims containing an invalid user ID or non-positive school ID.
- The protected identity endpoint returns the user ID and school ID from validated token claims.

The implemented login and identity endpoints form an initial access-token flow. They are not a complete authentication-session lifecycle.

### Refresh-token handling

Refresh tokens are opaque cryptographically random values, separate from JWT access tokens. The database stores a SHA-256 hash rather than the raw refresh token. Refresh rotation revokes the presented session and creates its replacement in one database transaction. A previously revoked token triggers revocation of remaining active sessions in the same token family.

The current HTTP contract returns refresh tokens in JSON. Clients must avoid exposing them to scripts or logs and should prefer a secure, appropriately protected storage mechanism. Production browser integration should review HttpOnly/Secure/SameSite cookie options together with CSRF protections before choosing a cookie-based transport.

Logout invalidates the refresh token only. Already-issued stateless access JWTs remain valid until their short expiry; immediate access-token revocation is not implemented.

PostgreSQL repository integration tests, concurrent rotation tests, and broader abuse/rate-limit testing remain required before production deployment.

### Authentication middleware behavior

The middleware:

1. Reads the `Authorization` header.
2. Requires the Bearer authentication scheme and a token.
3. Validates the token through the token-validation service.
4. Extracts the user ID from the token subject and the school ID from the token claims.
5. Rejects invalid identity identifiers.
6. Stores the validated user and school identifiers in the Gin request context.
7. Allows the request to continue only after successful authentication.

Authentication failures return `401 Unauthorized`. If the middleware is configured without a token validator, it returns `500 Internal Server Error` to indicate an internal configuration problem.

The middleware establishes the identity represented by a validated token. It does not independently implement role-based authorization, database-backed user-status rechecks on every request, or complete tenant isolation.

### Remaining requirements

- Use a cryptographically secure method to generate refresh tokens and identifiers.
- Keep access tokens short-lived.
- Use a separate, longer-lived refresh-session mechanism.
- Validate token type, signature, expiry, issuer, and relevant claims.
- Store refresh-token hashes rather than raw refresh tokens.
- Rotate refresh tokens safely and revoke the previous session token.
- Revoke refresh sessions on logout.
- Define how access-token revocation or short token lifetimes will handle logout.
- Prevent concurrent refresh requests from reusing the same refresh token.
- Keep signing keys and secrets outside source control.
- Never log access tokens, refresh tokens, passwords, or secret values.
- Return generic authentication errors where detailed responses could reveal sensitive information.
- Apply authentication middleware to every route that requires an authenticated user.
- Implement server-side authorization checks for protected operations.
- Decide how disabled users and revoked sessions should be handled during the lifetime of an access token.

The refresh-token HTTP lifecycle, logout behavior, and complete protected-route enforcement remain pending. The final token lifetimes, claims, and key-rotation strategy must be reviewed before production deployment.

## 5. Multi-Tenant Isolation

The application is designed for multiple schools.

Each school-owned record must be associated with the correct school, and users must only access data they are authorized to see.

### Required controls

- Derive the user's authenticated identity and authorized school context from a trusted server-side authentication context.
- Do not trust a request's `school_id` as authorization.
- Scope every school-owned read, update, delete, and list query to the authorized school.
- Verify school ownership when accessing related records.
- Enforce appropriate database foreign keys and uniqueness constraints.
- Add tests proving that a user from one school cannot access another school's records.
- Review background jobs, exports, reports, and administrative operations for tenant leaks.

Tenant isolation must be enforced in backend services and repository queries, not just in the frontend.

**Status:** The authentication middleware extracts the school ID from validated token claims, but the overall tenant-isolation mechanism is still pending. The existence of a `school_id` field, a foreign key, or a validated token alone does not prove that tenant isolation is complete.

## 6. Authorization and Roles

Role-based access control (RBAC) is planned but not yet implemented as a complete system.

Before release, define:

- Supported roles and their permitted actions.
- Which roles can create, read, update, or delete each resource.
- How school administrators are provisioned.
- How users are assigned, changed, disabled, and removed.
- How permissions are checked for every protected operation.
- How permission changes take effect for users with existing access tokens.

The backend must enforce permissions independently of the frontend. Hiding a button or route in React is not an authorization control.

The current authentication middleware establishes identity only. It does not establish that the user is authorized to perform every operation or access every resource associated with the token's school.

## 7. Database Security

### Current foundation

- Database access is configured through application configuration.
- The database connection is managed through a connection pool.
- The schema includes foreign-key and data-integrity constraints.
- The user schema stores password hashes, not plaintext passwords.

### Production requirements

- Use a dedicated database role with only the permissions the application requires.
- Do not run the production application using a PostgreSQL superuser.
- Keep database credentials outside source control.
- Restrict database network access.
- Configure encrypted database connections where required by the deployment environment.
- Back up the database and periodically test restoration.
- Protect backups and exported data.
- Apply schema migrations through a controlled deployment process.
- Avoid exposing raw database errors to API clients.

Local development configuration is not proof of production database security.

## 8. Configuration and Secrets

- Keep local environment files such as `.env` out of Git.
- Provide safe example configuration without real credentials.
- Use environment variables or an appropriate secret-management system for production secrets.
- Validate required configuration at startup.
- Do not commit database passwords, JWT signing keys, API keys, or private certificates.
- Rotate any secret that is accidentally exposed.
- Keep development and production credentials separate.

Verify that the local environment file is ignored and is not tracked by Git before pushing project changes.

## 9. HTTP and API Security

Before production release:

- Use HTTPS for external application traffic.
- Configure trusted proxies explicitly when operating behind a reverse proxy.
- Validate and limit request sizes.
- Validate input on the server.
- Configure CORS for known frontend origins rather than allowing every origin by default.
- Add appropriate rate limits, especially for authentication and OTP-related operations.
- Avoid leaking stack traces or internal errors.
- Use secure headers and appropriate cookie settings if cookies are used for tokens.
- Ensure HTTP timeouts are appropriate for the deployment.
- Review file uploads, exports, and any external integrations before exposing them.

The existing server timeouts and configuration validation are only part of the broader security requirements.

## 10. Development Seed Command

Location: `backend/cmd/devseed/`

The development seed command creates or prepares a test user for local development and is guarded by the development environment.

Security requirements:

- Do not use the command as a production user-provisioning mechanism.
- Do not put real credentials in source code or documentation.
- Keep seed credentials in environment variables.
- Verify the environment guard and production deployment separation before release.
- Ensure production workflows cannot accidentally create development test accounts.

The command's existence does not replace a secure school-administration or user-provisioning workflow.

## 11. Logging, Monitoring, and Auditing

Production requirements:

- Record security-relevant events such as login failures, account disabling, role changes, and administrative actions.
- Never log passwords, raw tokens, signing keys, or database credentials.
- Avoid unnecessarily logging student or parent personal information.
- Define log retention and access policies.
- Monitor repeated failed authentication attempts and suspicious activity.
- Add audit records for sensitive administrative changes.
- Ensure logs do not become a source of cross-school data leakage.

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
- Refresh, rotation, and revocation once those flows exist.
- Insufficient role permissions.
- Cross-school read and write attempts.
- SQL query parameterization.
- Configuration and secret-handling failures.
- Rate limiting and abuse protection.
- Production database permissions and backup restoration.

Tests should be automated where practical and run before changes are merged.

## 13. Production Readiness Checklist

Before serving real schools and student data, verify:

- [ ] Authentication endpoints and current authentication middleware behavior are tested.
- [ ] Authentication middleware is applied to every route that requires authentication.
- [ ] Full token lifecycle and logout behavior are implemented and tested.
- [ ] RBAC permissions are defined and enforced server-side.
- [ ] Tenant isolation is verified for all school-owned resources.
- [ ] Password policy is finalized and documented.
- [ ] Production TLS is configured and tested.
- [ ] Production secrets are securely managed.
- [ ] Database access follows least privilege.
- [ ] Rate limiting and abuse protection are enabled.
- [ ] Sensitive information is excluded from logs and API errors.
- [ ] Security tests and dependency checks run in the release workflow.
- [ ] Backups and restoration procedures are tested.
- [ ] Monitoring and incident-response procedures are documented.
- [ ] The development seed command is excluded from production workflows.

## 14. Maintenance

Whenever security behavior changes:

1. Update this document to reflect actual implementation.
2. Add or update automated tests.
3. Update `PROGRESS.md` and `CHANGELOG.md` when appropriate.
4. Record important architecture or security decisions in `ARCHITECTURE.md` or a dedicated decision record.
5. Never mark a security requirement complete based only on documentation.

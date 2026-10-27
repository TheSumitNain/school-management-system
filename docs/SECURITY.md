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
- Refresh-session storage has a database migration with token-hash, expiry, and revocation fields.
- The API server has configured HTTP timeouts and graceful shutdown behavior.

These controls form an initial foundation. They do not mean that production authentication or the entire application has been security-certified.

### Not yet implemented as a complete HTTP security flow

- JWT access-token issuance and validation for this project.
- Refresh-token issuance, rotation, and revocation workflows.
- Login, logout, and registration HTTP handlers.
- Authentication middleware for protected routes.
- Role-based access control (RBAC).
- Complete tenant-isolation enforcement across business queries.
- Production TLS and deployment-specific security configuration.
- Rate limiting and abuse protection for authentication endpoints.
- Security audit logging and monitoring.

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

Authentication HTTP endpoints and JWT flows are not yet implemented in this project's current API.

The following are requirements for the eventual implementation:

- Use a cryptographically secure method to generate tokens and identifiers.
- Keep access tokens short-lived.
- Use a separate, longer-lived refresh-session mechanism.
- Validate token type, signature, expiry, and relevant claims.
- Store refresh-token hashes rather than raw refresh tokens.
- Rotate refresh tokens safely and revoke the previous session token.
- Revoke refresh sessions on logout.
- Define how access-token revocation or short token lifetimes will handle logout.
- Prevent concurrent refresh requests from reusing the same refresh token.
- Keep signing keys and secrets outside source control.
- Never log access tokens, refresh tokens, passwords, or secret values.
- Return generic authentication errors where detailed responses could reveal sensitive information.

The exact token lifetimes, signing algorithm, claims, and key-rotation strategy must be finalized before implementing the production flow.

## 5. Multi-Tenant Isolation

The application is designed for multiple schools.

Each school-owned record must be associated with the correct school, and users must only access data they are authorized to see.

### Required controls

- Derive the user's authorized school context from a trusted server-side authentication context.
- Do not trust a request's `school_id` as authorization.
- Scope every school-owned read, update, delete, and list query to the authorized school.
- Verify school ownership when accessing related records.
- Enforce appropriate database foreign keys and uniqueness constraints.
- Add tests proving that a user from one school cannot access another school's records.
- Review background jobs, exports, reports, and administrative operations for tenant leaks.

Tenant isolation must be enforced in backend services and repository queries, not just in the frontend.

**Status:** The overall tenant-isolation mechanism is still pending. The existence of a `school_id` field or a foreign key alone does not prove that tenant isolation is complete.

## 6. Authorization and Roles

Role-based access control (RBAC) is planned but not yet implemented as a complete system.

Before release, define:

- Supported roles and their permitted actions.
- Which roles can create, read, update, or delete each resource.
- How school administrators are provisioned.
- How users are assigned, changed, disabled, and removed.
- How permissions are checked for every protected operation.

The backend must enforce permissions independently of the frontend. Hiding a button or route in React is not an authorization control.

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

The existing server timeouts and trusted-proxy configuration are only part of the broader security requirements.

## 10. Logging, Monitoring, and Auditing

Production requirements:

- Record security-relevant events such as login failures, account disabling, role changes, and administrative actions.
- Never log passwords, raw tokens, signing keys, or database credentials.
- Avoid unnecessarily logging student or parent personal information.
- Define log retention and access policies.
- Monitor repeated failed authentication attempts and suspicious activity.
- Add audit records for sensitive administrative changes.
- Ensure logs do not become a source of cross-school data leakage.

A complete security audit and monitoring system is not yet implemented.

## 11. Testing Requirements

Security testing must cover at least:

- Password hashing and verification.
- Invalid credentials and disabled users.
- Inactive or suspended schools.
- Token expiry, validation, refresh, and revocation once token flows exist.
- Missing authentication on protected endpoints.
- Insufficient role permissions.
- Cross-school read and write attempts.
- SQL query parameterization.
- Configuration and secret-handling failures.
- Rate limiting and abuse protection.
- Production database permissions and backup restoration.

Tests should be automated where practical and run before changes are merged.

## 12. Production Readiness Checklist

Before serving real schools and student data, verify:

- [ ] Authentication endpoints and middleware are implemented and tested.
- [ ] Token lifecycle and logout behavior are implemented and tested.
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

## 13. Maintenance

Whenever security behavior changes:

1. Update this document to reflect actual implementation.
2. Add or update automated tests.
3. Update `PROGRESS.md` and `CHANGELOG.md` when appropriate.
4. Record important architecture or security decisions in `ARCHITECTURE.md` or a dedicated decision record.
5. Never mark a security requirement complete based only on documentation.
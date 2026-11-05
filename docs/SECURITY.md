# Security Documentation

## 1. Purpose

This document describes the current security foundation, security decisions, known gaps, and production security requirements for the School Management System.

The application is designed to support multiple schools. Student, parent, teacher, and school information must be protected against unauthorized access.

**Important:** This document distinguishes implemented controls from planned controls. A documented requirement does not mean the corresponding feature has already been implemented or verified for production.

## 2. Current Implementation Status

### Implemented foundation

- PostgreSQL access uses application configuration and a connection pool; schema changes are versioned through additive migrations.
- Passwords are bcrypt-hashed (cost 12), require at least 12 Unicode characters and at most 72 UTF-8 bytes, reject invalid UTF-8, whitespace-only and local common-password entries, and are checked against the Have I Been Pwned Pwned Passwords range service using k-anonymity.
- Staff usernames are globally unique and case-insensitive. Staff and Platform Admin login use database-backed five-failure/15-minute lockout and atomic IP rate limiting. Login fails closed if the login-security store or IP limiter is unavailable.
- Platform Admin and School Admin require TOTP MFA. MFA failure accounting is principal-wide across replacement challenges and challenge purposes: five failures trigger a 15-minute lockout, failures are audited, and successful MFA verification resets the counter.
- Access JWTs carry canonical principal identity, token version and refresh-family ID. Protected requests reload principal status, current MFA enrollment state, school status and live permissions from PostgreSQL.
- Refresh sessions use opaque random tokens stored as SHA-256 hashes, rotation, replay/family revocation, per-family and per-principal transaction locking, and per-device 30-minute administrative idle timeout for Platform Admin and School Admin.
- Live RBAC includes system roles, custom roles, exact-delegation-bound custom-role grants, user ALLOW/DENY overrides, staff role assignment/revocation and route-level `RequirePermission` enforcement. V22 adds database and repository protections against self role/permission escalation.
- Staff account creation, listing, update, disable/re-enable, current-password change, School Admin password reset, forced temporary-password change, token-version invalidation and audit events are implemented.
- Academic foundations include school-scoped student/enrollment/attendance and guardian APIs. Teacher enrollment roster responses are restricted to assigned sections and approved basic fields (student name, roll number, class and section). Teacher-only principals cannot use custom roles or permission overrides to obtain sensitive student-profile or guardian-contact access; a principal who also holds the School Admin system role follows the School Admin authorization path.
- Parent authentication and guardian matching require active linked schools and official active guardian/student relationships. Parent OTP delivery and Phase 2 Parent/Driver workflows remain intentionally paused.
- GitHub Actions run [41961583328](https://github.com/TheSumitNain/school-management-system/actions/runs/41961583328) on commit `7071d7913ab62ff525637e2af0f4d3718b9f57b8` applied migrations V1–V22 to ephemeral PostgreSQL 18, passed the full backend suite including PostgreSQL integration tests, and passed `go vet ./...`. CI does not modify either persistent local database.

### Not yet implemented or fully verified

- Complete tenant/resource/assignment/field-level enforcement across every current and future school business module. Implemented endpoints have targeted scope checks, but the entire product surface is not yet complete.
- Complete current-user profile retrieval beyond the identity context returned by `GET /auth/me`.
- Immediate revocation of already-issued ordinary-staff access JWTs on family logout; refresh sessions are revoked and access tokens otherwise expire naturally. Token-version changes invalidate access tokens on subsequent requests.
- Exhaustive end-to-end HTTP tests using real JWT signature validation for every protected route, plus additional concurrency and database-failure cases.
- Production database least-privilege grants, TLS/proxy/deployment hardening, monitoring, alerting, backup and recovery procedures.
- Production penetration testing, capacity/load testing and incident-response readiness.

All registered protected routes use the shared authentication middleware and route-level `RequirePermission` checks where an operation permission is required. The middleware enforces per-device 30-minute idle timeout for Platform Admin and active School Admin sessions. Future routes must follow the same pattern.


## 3. Password Security

### Current implementation

The password service uses bcrypt with a cost factor of 12. Passwords are hashed before storage, and password verification compares a supplied password against the stored hash.

The current implementation enforces an input range of 12–72 bytes, rejects whitespace-only passwords, checks a local common-password denylist, and checks the Pwned Passwords range API using k-anonymity. The check is applied to first School Admin passwords during school onboarding, new staff initial passwords, temporary passwords when re-enabling staff, self-service password changes, School Admin resets, and first Platform Admin bootstrap passwords. The application sends only the first five hexadecimal characters of the SHA-1 digest; the remaining suffix is compared locally. The complete password and complete hash are never sent. Range responses are cached in memory for one hour, and zero-count padding records are ignored. The check fails closed: if the service is unavailable, password creation/change/reset returns a temporary service-unavailable response instead of silently bypassing the policy.

Bcrypt has a 72-byte input limit. Any future password-policy change must handle Unicode and byte length correctly and must not silently truncate passwords.

### Production requirements

Before public production deployment:

- Finalize and document the password policy.
- Maintain the k-anonymity breached-password check and monitor its availability; retain a clear, fail-closed recovery path for temporary provider outages.
- Never log plaintext passwords or expose them in error messages.
- Never store plaintext passwords.
- Evaluate whether bcrypt remains appropriate or whether Argon2id is preferable for the deployment environment.
- Implement a safe migration and rehash strategy if the hashing algorithm or policy changes.
- Revoke appropriate sessions after password changes and password resets.

The current minimum is 12 Unicode code points (characters) and maximum is 72 UTF-8 bytes to respect bcrypt's input limit. Invalid UTF-8 is rejected. School onboarding, staff account creation/re-enable, staff self-service password changes, same-school School Admin password resets, and the one-time Platform Admin bootstrap all run the breached-password check before persistence. New staff accounts default to a temporary-password state; the next authenticated request is restricted to password change and identity inspection until replacement. Password changes/resets increment the principal token version and revoke all refresh sessions in the same transaction.

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

Access JWTs issued through login, MFA completion, and refresh are bound to the refresh-session family using the `sid` claim. The V21 `last_activity_at` timestamp is maintained per family. Every protected request checks and touches only that device's active family for Platform Admin and active system-role School Admin principals. An idle family at or beyond 30 minutes is revoked and audited; another device's family remains independent. Refresh rotation checks the same timeout inside the existing family/principal-locked transaction and preserves the activity timestamp rather than resetting it. Ordinary staff and parent sessions are not subject to this administrative idle timeout. The middleware still does not replace resource-level tenant and assignment checks in each business repository.

### TOTP encryption key storage and rotation

TOTP authenticator secrets are encrypted at rest with AES-256-GCM. The active encryption key is supplied as Base64-encoded 32-byte material through `TOTP_ENCRYPTION_KEY`; `TOTP_ENCRYPTION_KEY_VERSION` is a positive version number stored alongside each encrypted credential. Do not reuse the JWT signing key or Parent OTP HMAC key for TOTP encryption.

The versioned key ring supports:
- The current key encrypts newly enrolled TOTP credentials.
- `TOTP_ENCRYPTION_PREVIOUS_KEYS` optionally holds comma-separated `version:Base64` entries for old keys. Every previous key must decode to exactly 32 bytes and have a distinct positive version different from the current version.
- A previous key is used only to decrypt credentials bearing its version. After a successful TOTP verification, the service encrypts the same secret with the current key and attempts an optimistic database update matching the principal, old key version, and old ciphertext. A concurrent update cannot be overwritten by a stale rotation.
- Key rotation requires no schema migration. Existing credentials are upgraded lazily as administrators successfully complete MFA enrollment, login, or recovery-code regeneration. A credential that has not been used remains encrypted with its previous key.

Operational rotation procedure:
1. Generate a cryptographically random 32-byte key, Base64-encode it, and store it in the deployment secret manager. Never commit real key material.
2. Increase `TOTP_ENCRYPTION_KEY_VERSION` and set the new `TOTP_ENCRYPTION_KEY`. Keep the old version/key in `TOTP_ENCRYPTION_PREVIOUS_KEYS`.
3. Deploy the key-ring-capable application and the complete key configuration to every API instance as one coordinated rollout. Do not run old single-key application instances alongside instances that have started encrypting with the new key.
4. Keep every previous key available while any stored credential still uses that version. Check progress with this read-only query:

   ```sql
   SELECT encryption_key_version, COUNT(*)
   FROM principal_mfa_credentials
   WHERE method = 'totp'
   GROUP BY encryption_key_version
   ORDER BY encryption_key_version;
   ```

5. Remove an old key only after its credential count is zero **and** backups that may contain credentials encrypted with that key have expired or have been re-encrypted under an approved backup process. Retain old keys securely for the required backup-retention period.

If a required previous key is missing, affected administrators cannot complete TOTP verification with their authenticator secret. Do not delete or replace key material as a troubleshooting shortcut. Treat the key ring as production-critical secret material and test restore procedures before deployment.

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

Logout revokes every active refresh session in the supplied token family. Repeated logout is idempotent and unknown tokens are a no-op. Logout alone does **not** immediately invalidate access JWTs that have already been issued. Staff password changes and administrator password resets do immediately invalidate existing access JWTs by incrementing the principal's database-backed token version; they also revoke every active refresh session for that principal.

Before production deployment, decide whether short access-token lifetimes are sufficient or whether immediate revocation is required. Also define session invalidation behavior after password changes, account disabling, and administrative security events.

### Client-side token handling

The current HTTP contract returns access and refresh tokens in JSON.

Clients must not expose tokens in application logs, URLs, analytics events, or error reports. Browser integration must carefully evaluate token storage and the risks of cross-site scripting (XSS).

If cookies are used, configure appropriate `HttpOnly`, `Secure`, and `SameSite` attributes and implement suitable CSRF protections. The transport and storage approach must be reviewed before production browser deployment.

### Remaining requirements

- Keep access tokens short-lived and document the configured lifetime.
- Validate token signature, issuer, expiry, and relevant claims.
- Keep signing keys and secrets outside source control.
- Define and follow the versioned TOTP encryption-key rotation and backup-retention procedure above.
- Apply authentication middleware to every route requiring authentication.
- Continue testing concurrent refresh/logout ordering and transaction behavior against PostgreSQL, including database failure cases. V21 integration tests now cover admin idle expiry during refresh, per-device family isolation, audit creation, and the ordinary-staff exemption.
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

**Current status:** Authentication middleware loads canonical school context and live principal state. Implemented student/enrollment/attendance/guardian repositories enforce school and teacher-assignment scope on their supported routes, including an assigned-section teacher roster. Complete tenant-isolation enforcement across every business resource remains pending; a school_id column or valid token alone does not prove tenant isolation.

## 6. Authorization and Roles

The core principal/RBAC schema, built-in role bundles, live permission resolver and RequirePermission middleware are implemented. V16 binds custom-role grants and user ALLOW overrides to the exact active delegation ID. V22 adds repository and database guards against self role assignment, self permission overrides, and self-removal of DENY overrides. The role/delegation/override HTTP workflows, same-school staff role administration, and account lifecycle routes have PostgreSQL integration coverage. GitHub Actions run [41962468469](https://github.com/TheSumitNain/school-management-system/actions/runs/41962468469) on commit `41e6e1a3f51c19143977b50f1e24bc57efd6ab59` applied V1–V22 to ephemeral PostgreSQL 18, passed the backend suite including PostgreSQL integration, and passed `go vet ./...`. The suite includes regression coverage proving a Teacher with a delegated guardian custom role is still denied guardian and sensitive student-profile access. CI does not modify either persistent local database. Complete authorization for all future business resources and every assignment/field-level boundary remains pending. Phase 1 web roles are Platform Admin, School Admin and Teacher; Parent and Driver are Phase 2.

The implementation must enforce:

- One-school-only School Admin and staff accounts; separate accounts for a person employed by multiple schools.
- Globally unique, case-insensitive staff usernames. The current principal-aware schema and username-only staff login enforce this boundary.
- New staff rows default to a temporary-password state; first-login change, same-school School Admin reset, staff create/list/update/disable/re-enable, token-version invalidation and audit logging are implemented.
- Admin-initiated same-school staff password reset without old password, with session/access revocation, forced next-login change, and audit logging.
- Custom-role grants are bound to the exact active delegation ID; the database guard and live resolver enforce the delegation ceiling. Role/delegation/override HTTP workflows are implemented and covered by targeted integration tests.
- Multiple roles per staff account and per-user allow/deny overrides are supported; ALLOW overrides require their exact active delegation, explicit DENY removes access, and default is deny. Self-targeted role/override changes are rejected at repository and database layers.
- Teacher subject/class and class-in-charge assignments have separate scope checks. Enrollment roster returns only assigned sections and approved basic student fields. Teacher-only principals are blocked from guardian records and sensitive student profiles even if a custom role grants those permissions; complete field-level and resource coverage remains ongoing.
- Parent data foundations preserve verified guardian-child links across schools. Parent auth/matching now requires active linked schools and active official relationships; production OTP/SMS and Phase 2 Parent endpoints remain paused.
- Server-side permission evaluation on every protected request so changes apply to existing sessions on the next request.
- Cross-school, cross-class, cross-subject, field-level, and parent-child access-denial tests.

The backend must enforce permissions independently of the frontend. Hiding a button or route in React is not an authorization control.

The backend must enforce permissions independently of the frontend. Hiding a button or route in React is not an authorization control.

The current authentication middleware establishes identity, enforces administrative session activity and MFA enrollment, and resolves live role/override permissions on each protected request. Registered staff, role/delegation, academic, student/enrollment, guardian, attendance, onboarding and password-reset routes use route-level RequirePermission checks. Repository-level school and assignment filters exist for implemented resources, but full coverage across future modules and every field-level boundary remains necessary.

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

The existing HTTP timeouts, configuration validation, and graceful shutdown are only part of the broader security requirements. Identifier-based five-failure/15-minute lockout, IP-based login rate limiting, fail-closed behavior when either control is unavailable, and principal-wide five-failure/15-minute MFA lockout across replacement challenges are implemented. MFA failures are audited. Broader abuse monitoring and production alerting remain pending.

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


## 15. Approved Phase 1 Account Security Rules — Implementation Status

### Staff account isolation

Every School Admin/staff account is tied to exactly one school. A staff account cannot be reused across schools. If a person works at multiple schools, each school provisions a separate account with separate credentials and sessions. Staff usernames are globally unique and case-insensitive. The current principal-aware schema enforces globally unique, case-insensitive staff usernames and the staff login resolves by username without a school ID.

### Password change and reset

Staff self-service password change must verify the current password. School Admin can reset a staff member's password without knowing the old password, but only for staff in the same school. Reset must revoke every refresh session for the target account and invalidate already-issued access tokens, require a password change at next login, and create an audit event. The password itself must never be logged or retrievable by administrators.

### Teacher and parent data boundaries

A subject-teaching assignment does not grant attendance access. Subject teachers see only name, roll number, class, and section for assigned classes/sections, plus permitted teaching-related data. Class-in-charge teachers may access attendance (same-day correction within configured cutoff), basic student details, timetable, homework, and applicable notices only for the exact class/section assigned. Past attendance correction is School Admin-only.

Parent uses registered mobile + OTP and can access multiple officially linked children across schools. The parent dashboard's child selection is not an authorization grant; each API request verifies the relationship. Parent data access is read-only in Phase 1, and students have no separate login accounts.

### Immediate enforcement

Role/permission changes and staff password changes/resets take effect on subsequent protected requests through live database permission loading and `auth_principals.token_version` checks. Staff password changes/resets increment token version and revoke active refresh sessions. Account disabling and school suspension are rejected by the live principal loader. Staff disable/re-enable workflows are implemented; full school suspension operations and every business-resource response remain subject to continued audit.

## 2026-11-03 implementation update

- V15 adds `users.must_change_password`: existing staff accounts retain their current state, while new staff accounts default to a required first-login password change.
- Staff password change verifies the current password and policy-checks the replacement. In one transaction it clears the temporary flag, increments `auth_principals.token_version`, revokes all active refresh sessions, and appends an audit event.
- School Admin password reset is exposed at `POST /school/staff/:principalID/password/reset`. The route requires `users.password_reset`; the repository independently verifies an active same-school School Admin, an active target staff account in the same active school, and the non-denied permission grant. Reset sets `must_change_password=TRUE`, increments token version, revokes refresh sessions, and audits the action.
- Temporary-password staff access is restricted on each protected request to `POST /auth/password/change` and `GET /auth/me`; refresh rotation does not remove the restriction because the flag is reloaded from PostgreSQL.
- V16 binds custom-role permission grants and user permission overrides to exact delegation IDs. A revoked delegation cannot reactivate an old grant after a new delegation is issued; explicit rebinding to the new ID is required.
- GitHub Actions on branch `fix/v4-auth-security-hardening`, commit `524d795`, applied V1–V16 to a fresh ephemeral PostgreSQL 18 database, passed `go test -count=1 -p 1 ./...` with `RUN_POSTGRES_INTEGRATION_TESTS=1`, and passed `go vet ./...`. The user's dedicated `school_management_test` database was not modified.

## 2026-11-04 implementation update

- V21 adds `refresh_sessions.last_activity_at`, initializes existing rows from `created_at`, and defaults new refresh sessions to the database clock.
- Access tokens issued by staff/Platform Admin login, MFA completion, and refresh include the stable refresh-family UUID in `sid`.
- All registered protected API routes share the middleware wired to the principal authorization repository. Platform Admin and staff with an active system `school_admin` role are checked against the exact session family on every protected request.
- The 30-minute administrative idle timeout is per device/session family. Activity updates and refresh rotation serialize on the same PostgreSQL advisory family lock. An expired family is revoked and recorded in `audit_events`; another device family remains active. Refresh rotation does not reset `last_activity_at`. Parent and ordinary staff sessions are exempt.
- PostgreSQL integration coverage verifies per-device isolation, timeout revocation/audit, refresh-time idle expiry, and ordinary-staff exemption.
- New staff initial passwords, staff re-enable temporary passwords, self-service password changes, School Admin resets, and first Platform Admin bootstrap passwords are checked against the Pwned Passwords range API. The implementation uses k-anonymity (only a five-character SHA-1 prefix is sent), ignores zero-count padding records, caches range responses, and fails closed with HTTP 503 when the provider is unavailable.
- GitHub Actions at commit `2daefb4` applied migrations V1–V21 to an ephemeral PostgreSQL database, passed backend tests including PostgreSQL integration, and passed `go vet ./...`. This does not modify the user's local database or production database.
## 2026-11-05 implementation update

- V22 prevents a principal from assigning a role or granting/removing a permission override on their own principal through repository methods and database triggers. Direct-SQL integration assertions cover the trigger boundary.
- Administrative MFA failure counters are principal-wide across replacement challenges and challenge purposes. Five failures in a rolling 15-minute window lock the principal for 15 minutes; failed attempts are appended to the audit trail, and successful MFA verification/enrollment/recovery reset clears the counter.
- Staff login fails closed if the username lockout store is unavailable; staff and Platform Admin HTTP login also fail closed if the IP rate limiter is missing.
- Parent mobile lookup, parent principal loading, guardian matching and pre-account registration exclude inactive/suspended schools. Parent OTP/SMS delivery and Phase 2 endpoints remain paused.
- Teacher enrollment roster includes student name, roll number, class and section only within assigned teaching/class-in-charge sections.
- GitHub Actions run 41961583328 on commit 7071d7913ab62ff525637e2af0f4d3718b9f57b8 passed migrations V1–V22, the full backend suite including PostgreSQL integration, and go vet ./... on ephemeral PostgreSQL 18. No persistent user database was changed by CI.

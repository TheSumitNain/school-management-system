
# Security Design

## 1. Purpose

Security is a core requirement of the School Management System because it handles school, staff, parent and student information.

Security controls must be enforced by the backend, not only by the frontend.

## 2. Authentication

- Platform Admin and School Admin must log in using email and password.
- Multi-factor authentication (MFA) is mandatory for every admin login.
- There must be no public Platform Admin registration.
- School Admin accounts must be created by an authorized Platform Admin.
- Teacher accounts must be created or invited by authorized school staff.
- Parent authentication will use mobile OTP in a later phase.
- All authentication endpoints must be protected against abuse.

## 3. Password Security

- Minimum password length: 12 characters.
- Passphrases must be allowed.
- Do not require routine password changes without a security reason.
- Block common and known-compromised passwords where practical.
- Store passwords using a suitable password-hashing algorithm, such as Argon2id, with appropriate parameters.
- Never store plaintext passwords.
- Password reset tokens must be single-use and expire after 30 minutes.
- Password reset must revoke existing sessions and require MFA at the next admin login.

## 4. Token and Session Security

- Access tokens must expire after 15 minutes.
- Refresh tokens must expire after 7 days.
- Refresh token rotation is required.
- A refresh token must not be reusable after rotation or revocation.
- Logout must revoke the relevant refresh session and prevent further use of the associated access token according to the token-revocation design.
- Password changes and account deactivation must invalidate existing sessions.
- Admin sessions must expire after 30 minutes of inactivity.
- Token secrets must be supplied through secure configuration and never committed to Git.

## 5. Login Protection

- Five consecutive failed login attempts must trigger a 15-minute temporary lock.
- Apply rate limits to accounts and IP addresses.
- Use generic authentication errors that do not reveal whether an account exists.
- Reset the consecutive-failure counter after a successful login.
- Apply rate limits and security logging to MFA verification and recovery.

## 6. MFA Recovery

- Backup codes must be single-use.
- MFA recovery must use an approved, verified recovery process.
- Recovery must revoke existing sessions and create an audit event.
- Platform Admin recovery requires additional verification.
- MFA must not be bypassed through an ordinary password reset.

## 7. Authorization and Roles

Initial roles:

- Platform Admin
- School Admin
- Teacher
- Parent

Additional roles, such as Student, Driver and other School Staff, may be introduced later.

Rules:

- Check permissions on every protected API request.
- Never rely on frontend route protection alone.
- Never trust role, user ID or school ID supplied by the client without server-side verification.
- Enforce the minimum permissions required for each role.
- Role or permission changes must take effect on the next request, including for existing sessions.

## 8. School Data Isolation

- A user must only access school data they are authorized to access.
- School-owned records must be scoped to the correct school.
- A teacher may access only assigned classes and subjects.
- A parent may access only their verified linked children.
- A parent may have multiple linked children, and a child may have multiple verified guardians.
- Driver access, when introduced, must be limited to assigned routes and permitted data.
- Cross-school access must be covered by automated authorization tests.
- Client-supplied identifiers must never be treated as proof of authorization.

## 9. Account Lifecycle

- School statuses: `pending_setup`, `active` and `suspended`.
- Pending or suspended schools must not access normal school operations.
- Suspending a school must preserve its data.
- Only a Platform Admin may reactivate a suspended school.
- Invitations must be single-use, expire after 24 hours and be stored as hashes.
- Sending a new invitation must invalidate the previous invitation.
- School Admins must not be able to elevate their own role.

## 10. Audit Logging

Record security-sensitive events, including:

- Permission and role changes.
- MFA recovery or reset.
- Password reset and account recovery.
- School suspension and reactivation.
- Relevant authentication failures and account lockouts.
- Other sensitive administrative actions.

Audit records must not contain passwords, raw tokens or other secrets.

Audit retention and access policies must be finalized before production.

## 11. Application and Database Security

- Validate all incoming data on the backend.
- Use parameterized SQL queries.
- Keep secrets out of source control and logs.
- Use HTTPS in production.
- Apply least-privilege database permissions.
- Protect backups and test restoration.
- Avoid unnecessary collection of personal information.
- Do not use real student data in routine development or testing.
- Use secure error handling that does not expose internal details.

## 12. Testing Requirements

Before production, test:

- Invalid, expired and revoked tokens.
- Refresh token rotation and reuse.
- Logout and session invalidation.
- MFA verification and recovery.
- Login lockout and rate limits.
- Role-based access restrictions.
- Cross-school data access attempts.
- Parent-child relationship authorization.
- Suspended school access.
- Password reset and invitation token reuse.

## 13. Open Decisions

- Final MFA delivery and authenticator method.
- Secure initial Platform Admin setup procedure.
- Recovery process if an admin loses access to their registered email.
- Teacher account activation method if email and mobile are unavailable.
- Audit retention period.
- Final tenant-isolation enforcement, including whether PostgreSQL Row-Level Security will be used.
- Production secrets management and deployment configuration.

## 14. Current Status

### Documented

- Authentication, password and token security requirements.
- MFA, login protection and account recovery requirements.
- Role-based authorization and school-level data isolation requirements.
- Account lifecycle, audit logging and database security requirements.
- Security testing requirements and open design decisions.

### Implemented Foundation

- Go + Gin backend scaffold.
- PostgreSQL connection setup.
- Initial `schools` table migration with school status constraints.
- Basic API health and database readiness endpoints.
- HTTP server timeouts and graceful shutdown handling.
- Automated tests for API health and database readiness behavior.

These foundation features do not implement authentication or authorization.

### Not Yet Implemented

- Authentication endpoints and secure password hashing.
- Access and refresh token lifecycle and revocation.
- MFA and login rate limiting.
- RBAC and permission enforcement.
- School-level tenant authorization and data isolation.
- Account lockout, password reset and invitation workflows.
- Security audit logging.
- Automated security and cross-school authorization tests.
- Production HTTPS and secrets management configuration.

### Production Readiness

The security requirements are documented, but the security implementation is incomplete. The application must not be considered production-ready for real school or student data until the required controls are implemented, reviewed and tested.

Open decisions in Section 13 must be resolved at the appropriate implementation stage.
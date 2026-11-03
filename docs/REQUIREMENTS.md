# Product Requirements

## 1. Product Overview

Build a commercial, multi-tenant School Management System that allows multiple schools to manage their academic and administrative activities through one centralized platform.

The application will initially use a modular monolith architecture with a shared PostgreSQL database and shared schema. School-level data isolation must be enforced by the backend.

## 2. Initial Target Users and Scale

- 25–30 schools
- Approximately 400–500 students per school
- Approximately 10,000–15,000 students across all schools

These figures are initial planning targets, not verified system capacity limits.

## 3. Technology Stack

- Frontend: React, TypeScript and Vite
- Backend: Go and Gin
- Database: PostgreSQL
- Database access: pgx / pgxpool
- Database migrations: golang-migrate
- Authentication: JWT access tokens and opaque refresh tokens
- Password hashing: bcrypt

## 4. User Groups and Delivery Phases

**Phase 1 — complete web product:**
- Platform Admin
- School Admin
- School staff, including Teachers and approved school-scoped operational roles as their modules are delivered.

**Phase 2 — mobile applications/interfaces after Phase 1 web completion:**
- Parent
- Driver and other approved operational user groups.
- Student access is not currently approved; no student login is included in Phase 1.

These are planned user groups, not proof of implemented account types. Complete role storage, RBAC, permission management, resource scope and authorization enforcement must be implemented and tested before any role is treated as operational.

### Role responsibilities

- **Platform Admin:** Manage platform-level school onboarding and authorized platform operations.
- **School Admin:** Manage their own school's settings, users and authorized school operations.
- **Teacher:** One-school staff account. Access is restricted by separately managed teaching assignments and class-in-charge assignments, with field-level limits on student data.
- **Parent (Phase 2):** Intended to access officially linked children across schools through a read-only mobile experience. Authentication and OTP provider are deferred until Phase 2 planning.

School Admin creates staff accounts and provides a temporary initial password. Staff usernames are globally unique and case-insensitive. A person working at multiple schools receives separate accounts at each school. School Admin provisions staff and may reset their passwords without the old password; staff self-service changes require the current password.

School Admin may create school-scoped custom roles from the platform permission catalogue, within an explicit delegation ceiling. Clerk/Manager, Librarian, Gate Keeper, Helper, Bus Conductor and other operational roles are future scope, not Phase 1 implementations. Student login accounts are excluded from Phase 1.

See `docs/PHASE1-RBAC.md` for the approved detailed role-permission matrix and authorization rules.

## 5. Initial Product Scope

The system will be developed incrementally. The following describes the intended product scope, not a claim that all modules are implemented.

### 5.1 Foundation

- Application configuration
- Database connectivity
- Health and readiness endpoints
- User authentication and session management
- Role-based authorization
- School-level data isolation
- Input validation and consistent error responses
- Automated testing
- Security logging and auditability
- Reliable server startup and shutdown

### 5.2 School Management

- School creation, activation and deactivation
- School Admin invitation and account setup
- User account management
- Class and section management
- Academic year management
- School-specific settings

### 5.3 Student Management

- Student admission and profile
- Student enrollment
- Class and section assignment
- Historical enrollment records
- Student status and relevant history

Student identity should be modeled separately from enrollment or class placement so that academic history can be preserved when a student changes classes or academic years.

### 5.4 Academic Management

- Subject management
- Teacher assignment
- Class and subject assignments
- Timetable management
- Academic year configuration

### 5.5 Attendance Management

- Daily class/section-wise attendance
- Attendance summaries
- Attendance correction with authorization
- Attendance history
- Configurable same-day attendance cutoff

Initial attendance statuses:

- Present
- Absent
- Late
- Leave

Assigned teachers and School Admins are intended to mark attendance according to the finalized permission rules. Past attendance corrections are intended to be restricted to School Admins.

### 5.6 Future Parent Mobile Features (Phase 2 — Deferred)

- Parent account management
- Verified parent–student relationships
- Access to linked children's basic profiles
- Daily attendance visibility
- Timetable access
- Homework and assignment access
- Examination results and report cards
- Notices and announcements

The future Parent experience is intended to be read-only and may include linked children's basic profiles, attendance, timetable, homework, published results and notices. One Parent identity may access multiple officially verified children, including across schools. Parent login, OTP provider, account activation and recovery decisions will be made when Phase 2 begins. Do not implement or extend Parent authentication while the Parent workstream is paused.

## 6. Phase 1 Exclusions and Future Scope

The following are excluded from the Phase 1 web delivery, but may be considered after the complete web product is stable:

- Parent and Driver mobile applications/interfaces (Phase 2).
- Fee collection and payment gateway.
- Live bus GPS tracking.
- Advanced analytics and AI features.
- Microservices architecture.

Transport administration may be planned as a school web module if approved; live GPS tracking remains separate future scope.

## 7. Core Business Rules

- A school must not access another school's private records.
- Users must only access features and records permitted by their role and permissions.
- Protected operations must be authorized by the backend.
- The frontend must not be treated as a security boundary.
- School and user status must be checked where required for authentication and authorization.
- Client-supplied identifiers must not be trusted as proof of access rights.
- Cross-school access must be prevented by backend-enforced authorization and appropriate database queries.
- Important data changes must preserve data integrity.
- Student and enrollment history must be retained according to approved retention rules.
- Sensitive operations must be auditable.
- Permission changes must apply to subsequent protected API requests, including requests made using existing sessions.
- Passwords and raw refresh tokens must not be stored in plaintext.
- Security-sensitive errors must not expose secrets or internal implementation details.
- Platform Admin operations and school-level operations must remain appropriately separated.

## 8. Authentication and Account Security Requirements

### 8.1 Implemented Foundation

The current backend includes:

- Current implemented username/password login scoped to a school ID (target staff login will use globally unique username + password; migration pending)
- bcrypt password hashing and verification
- JWT access-token generation and validation
- Bearer-token authentication middleware
- Protected `GET /auth/me` endpoint
- Cryptographically random opaque refresh tokens
- Hashed refresh-token storage
- Refresh-token rotation
- Refresh-token revocation on logout
- Refresh-token family revocation when a revoked token is replayed

These are initial authentication capabilities. They do not establish that every production security requirement has been completed.

### 8.2 Security Requirements Still Pending

- Mandatory MFA for Platform Admin and School Admin
- Login rate limiting
- Five failed login attempts followed by a 15-minute lockout, subject to the finalized implementation and recovery rules
- Security-event logging and audit trails
- Administrator password policy with a minimum of 12 characters
- Common and compromised-password checks
- Secure password-reset and account-recovery workflows
- Parent mobile OTP authentication
- Invitation expiry and secure School Admin activation
- Immediate access-token invalidation strategy where required
- Complete session-management verification

The intended administrator idle timeout is 30 minutes. Its enforcement must be implemented and tested before it is treated as operational.

## 9. Multi-Tenant Data Isolation

The initial architecture uses one shared database and schema for multiple schools.

Every school-owned record must be associated with the appropriate school, directly or through a verified relationship.

For every protected request, the backend must:

1. Authenticate the caller where authentication is required.
2. Determine the caller's authorized school and role.
3. Verify the requested operation against applicable permissions.
4. Confirm that requested resources belong to the permitted school.
5. Reject unauthorized cross-school access.
6. Apply these restrictions consistently to reads, writes, deletes, exports and related operations.

Tenant isolation must be verified through automated tests, including attempts by one school's users to access another school's resources.

**Status:** The shared-database approach is selected, but comprehensive tenant authorization and cross-school isolation tests are pending.

## 10. Development Approach

- Deliver the complete web product first: Platform Admin, School Admin and school staff.
- Defer Parent, Driver and other mobile-first experiences until the Phase 1 web product is complete.
- Inspect the existing repository and reconcile the approved schema before each module; do not start a parallel rewrite.
- Implement one module at a time.
- Document requirements before implementing each major module.
- Use version-controlled database migrations.
- Write and run tests for implemented features.
- Verify school-level isolation and authorization.
- Update project documentation and progress after meaningful changes.
- Distinguish implemented functionality from requirements that are still pending.
- Do not claim production readiness based solely on passing unit tests.
- Review code and database changes before committing them.
- Keep secrets and local environment files out of version control.

## 11. Open Decisions

The following details require further planning before the relevant modules are implemented:

- Final database schema and relationships
- Academic year transition rules
- Admission number format
- Teacher account activation and recovery method
- Parent OTP provider
- Attendance correction and retention policy
- Deployment, backup and disaster recovery strategy
- Exact relational schema/migration strategy for the approved identity model (global Platform Admin, one-school staff accounts, cross-school parent identity)
- Parent OTP provider, OTP expiry/attempt/resend limits, and guardian-match edge cases
- Staff global-username migration and target login API contract
- School onboarding and deactivation workflow
- Audit-log retention and access policy
- Immediate access-token revocation approach
- Invitation expiry and recovery policies
- Production monitoring and incident-response procedures

## 12. Current Status

### 12.1 Documented Requirements

- Initial product scope and target users
- Technology stack and planned user roles
- Phase 1 web requirements for Platform Admin, School Admin and school staff; future Phase 2 Parent/Driver mobile requirements
- Features excluded from the initial scope
- Core business and security rules
- Multi-tenant architecture decision
- Open planning decisions

### 12.2 Implemented Foundation

- Go + Gin backend scaffold
- Environment-based application configuration
- PostgreSQL connection pool
- Initial `schools` table migration
- `users` and `refresh_sessions` migration
- Refresh-token family metadata migration
- API health and database-readiness endpoints
- HTTP server timeouts and graceful shutdown handling
- Initial user and school repository operations
- bcrypt password hashing and verification
- Principal-aware staff login with globally unique username lookup
- Separate Platform Admin authentication
- Principal-based JWT access tokens with token-version claims
- Authentication middleware that reloads principal status, token version and current permissions
- Fail-closed `RequirePermission` middleware
- MFA enrollment, challenge verification and recovery-code workflows
- Staff failed-login lockout and login IP rate-limit components
- Principal-aware refresh-token issuance, rotation, replay detection and logout revocation
- PostgreSQL integration coverage for refresh-session, MFA, login security, student/guardian and Parent/Guardian foundation areas (with opt-in test execution)
- Unit, handler, middleware and service tests for implemented behavior

These implementations form an initial foundation; they do not mean that all authentication, authorization, tenant-isolation or production-security requirements have been met.

### 12.3 Not Yet Implemented or Fully Verified

- Complete route-level RBAC for all business APIs, custom-role grants and delegation-bound ALLOW overrides
- Comprehensive school/resource/teacher-assignment authorization and automated cross-school isolation tests
- Complete append-only audit-event writes for all sensitive business operations
- Full staff password lifecycle, administrator idle timeout and final password-policy verification
- Full school onboarding, School Admin provisioning and school lifecycle workflows
- Complete staff account, role and assignment management workflows
- Student admission, academic-year enrollment and class/section placement/history
- Academic setup, attendance, timetable, homework, examinations, results, notices and reporting APIs
- React frontend, API integration, Platform Admin dashboard, School Admin dashboard and school-staff workflows
- Parent authentication, Parent mobile app and Driver mobile app (Phase 2; intentionally deferred and Parent work paused)
- V13 new-school system-role provisioning migration review/application and PostgreSQL integration verification
- Independent HTTP-level logout/session-revocation and full migration/forward-recovery verification
- End-to-end, production deployment, backup/recovery, security and load testing
- Production deployment, monitoring, backup and recovery safeguards
- Load and performance testing

### 12.4 Capacity Note

The initial target of 25–30 schools and approximately 10,000–15,000 students is a planning objective, not a tested capacity guarantee.

Load testing, security testing, operational validation and production monitoring will be required before making capacity commitments.

## 13. Definition of Completion

A major module should not be marked complete until:

- Its requirements and expected behavior are documented.
- Its implementation matches the intended architecture.
- Relevant input validation and error handling are present.
- Authentication and authorization are enforced where required.
- School-level isolation is tested where applicable.
- Unit and relevant integration tests pass.
- Database migrations are reviewed and verified where applicable.
- API and security documentation are updated.
- Known limitations are recorded in the project progress documentation.

Requirements and implementation status must be updated after the relevant code and tests have been reviewed.


## 14. Approved Phase 1 Identity, Roles and Permission Requirements

The Phase 1 role and permission direction is approved. Detailed permission identifiers and the default role matrix are maintained in [`PHASE1-RBAC.md`](PHASE1-RBAC.md).

### 14.1 Account boundaries

- Phase 1 web accounts are Platform Admin, School Admin and school staff (including Teachers and approved school-scoped roles).
- Platform Admin has global platform scope and provisions School Admin accounts.
- Every School Admin and staff account belongs to exactly one school. No staff account can access multiple schools. A person working at multiple schools must have separate accounts, passwords, and sessions.
- Staff usernames are globally unique and case-insensitive. Duplicate usernames are not permitted across different schools.
- School Admin creates staff accounts and provides a temporary initial password. Staff must change it at first login. Self-service password change requires the current password.
- School Admin can reset a same-school staff password without knowing the old password. Reset must revoke all target sessions/access, force password change at next login, and be audited.
- School Admin cannot create another School Admin or a Platform Admin.
- Parent mobile access is Phase 2, after web completion. The intended model is one verified parent identity linked to multiple officially recorded children across schools; authentication is undecided and no Parent auth work is active in Phase 1. No student login accounts are included in Phase 1.

### 14.2 Teacher assignment and data limits

Teaching assignments and class-in-charge assignments are distinct. A subject teacher sees only student name, roll number, class, and section for assigned classes/sections, plus permitted teaching-related information for the assigned subject/class. A teaching assignment alone does not grant attendance access.

A class-in-charge teacher may access attendance (including same-day correction within the school cutoff), basic student details, timetable, homework, and applicable notices only for the exact class/section assigned as in-charge. Past attendance correction is School Admin-only. Neither assignment grants access to other classes or sensitive student fields by default.

### 14.3 Custom roles and permission behavior

School Admin may create school-scoped custom roles from platform-defined permissions, but only within an explicit authorized delegation ceiling. Multiple roles per staff account are supported within its one school. Role permissions combine with per-user allow/deny overrides; explicit deny overrides role allow, authorized explicit allow can add a permission otherwise absent, and default is deny. Overrides never bypass tenant/resource/assignment/field-level checks.

Permission evaluation is server-side on every protected request so changes take effect on the next request, including existing sessions.

### 14.4 Future Parent mobile experience (Phase 2)

After OTP verification and guardian matching, the parent sees the list of all officially linked children. Selecting a child shows that child's permitted profile, attendance, timetable, homework, published results/report cards, and applicable notices. The backend verifies the parent-child link on every request; a selected child ID from the client is not proof of authorization. Parent access is intended to be read-only in Phase 2. No Parent dashboard or Parent mobile implementation is part of Phase 1.

### 14.5 Implementation status

These are approved product requirements, not implemented functionality. Current code still has school-scoped username uniqueness and school-ID + username + password login. The database and API migration must be designed additively and safely before these target account rules can be treated as implemented.

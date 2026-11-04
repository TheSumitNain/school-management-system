# API Documentation

## 1. Overview

This document describes the School Management System backend API, its conventions, implemented endpoints, authentication behavior, and planned functionality.

The backend uses Go, Gin, and PostgreSQL.

### Current Implementation Status

- GET /health and GET /ready are implemented.
- Staff POST /auth/login resolves a globally unique username; no school ID is accepted.
- Platform Admin POST /auth/platform-admin/login uses a separate email/password flow.
- MFA enrollment, verification and recovery-code routes are implemented for privileged accounts.
- POST /auth/refresh and POST /auth/logout use principal-bound refresh sessions with rotation and token-family replay/revocation protections.
- GET /auth/me validates the JWT and reloads current principal identity, status, token version and effective permissions.
- RequirePermission middleware is enforced on school staff, role-management, and academic management routes; effective permissions are reloaded for every protected request.
- POST /platform/schools creates a school and its first School Admin transactionally, protected by platform.schools.create and platform.school_admins.create.
- V13 provisions the fixed School Admin and Teacher system roles for newly created schools.
- V14 contains a source-only one-time first Platform Admin bootstrap migration and CLI; it is not yet PostgreSQL-verified or approved for application.
- Academic Year, Class, Section, Subject, teacher-subject assignment, class-in-charge assignment, student admission/profile, enrollment, promotion, enrollment-history, and guardian contact/link APIs are implemented with school-scoped repository authorization, audited writes, and PostgreSQL integration coverage. Attendance, timetable, examinations, results, notices, parent authentication/OTP, and frontend workflows remain pending.

**Important:** An implemented authentication endpoint does not mean that the full authorization system or every production security control is implemented. Verify code and tests before treating a feature as complete.

## 2. API Conventions

### 2.1 Base URL

Local development:

`http://localhost:8081`

### 2.2 Content Type

Requests containing JSON should use:

`Content-Type: application/json`

JSON responses are returned by the implemented endpoints.

### 2.3 Response Format

Successful responses use JSON. For example:

```json
{
  "status": "success"
}
```

Error responses should not expose internal errors, database details, secrets, or sensitive information.

### 2.4 Identifiers

Database identifiers are numeric IDs in the current implementation.

### 2.5 Multi-Tenant Architecture

The application is designed to support multiple schools within a shared PostgreSQL database.

School-specific records are associated with a `school_id`.

Academic management endpoints derive the school from the authenticated staff principal and verify the matching live permission in PostgreSQL. They do not accept a client-provided school ID. Other resource families must follow the same rule before being considered complete.

## 3. Implemented Endpoints

### 3.1 Health Check

**Endpoint:** `GET /health`

**Purpose:** Confirms that the API process is responding. It does not check database readiness.

Example request:

```http
GET /health
```

Successful response:

**HTTP status:** `200 OK`

```json
{
  "status": "success",
  "message": "School Management System API is running"
}
```

### 3.2 Readiness Check

**Endpoint:** `GET /ready`

**Purpose:** Checks whether the application can reach its PostgreSQL database.

Example request:

```http
GET /ready
```

Successful response:

**HTTP status:** `200 OK`

```json
{
  "status": "success",
  "message": "API and database are ready"
}
```

If the database ping fails or times out, the endpoint returns:

**HTTP status:** `503 Service Unavailable`

```json
{
  "status": "error",
  "message": "Database is not ready"
}
```

The database ping uses a two-second timeout.

### 3.3 Login

**Endpoint:** `POST /auth/login`

**Purpose:** Authenticates a staff account by its globally unique username and returns principal-based access and refresh tokens. School selection is deliberately not supplied by the login client.

Request body:

```json
{
  "username": "devtest",
  "password": "your-test-password"
}
```

The values are illustrative placeholders, not actual credentials.

#### Successful response

**HTTP status:** `200 OK`

```json
{
  "status": "success",
  "access_token": "<access-token>",
  "refresh_token": "<refresh-token>",
  "token_type": "Bearer",
  "principal": {
    "principal_id": 1,
    "principal_type": "staff",
    "status": "active",
    "school_id": 1,
    "username": "devtest"
  }
}
```

The actual user ID and username depend on the authenticated account. The response must not contain a password or password hash.

#### Error responses

| HTTP status | Meaning |
|---|---|
| `400 Bad Request` | Malformed JSON or missing/blank username or password. |
| `401 Unauthorized` | Invalid username or password. |
| `403 Forbidden` | User account or school is inactive. |
| `500 Internal Server Error` | Unexpected internal failure, such as access-token or session issuance failure. |

#### Authentication notes

- The login request does not accept a client-supplied school ID; the backend resolves the globally unique staff username and its school.
- The authentication service checks the canonical principal, staff account and school status.
- Passwords are verified against stored password hashes.
- Password hashing currently uses bcrypt.
- A successful login returns an access token and an opaque refresh token.
- Access-token expiry is controlled by the JWT configuration.
- Refresh-token expiry is controlled by `REFRESH_TOKEN_EXPIRY_DAYS` (default: 7 days; allowed range: 1–30 days).
- Refresh tokens are random opaque values; their SHA-256 hashes are persisted in PostgreSQL.
- Login does not, by itself, establish that RBAC or complete tenant-isolation enforcement is implemented.

### 3.4 Refresh Session

**Endpoint:** `POST /auth/refresh`

**Purpose:** Exchanges a valid refresh token for a new access token and a replacement refresh token.

Request body:

```json
{
  "refresh_token": "<refresh-token>"
}
```

#### Successful response

**HTTP status:** `200 OK`

The response contains a new `access_token`, a replacement `refresh_token`, `token_type: "Bearer"`, and the principal DTO, using the same response structure as login.

A successful rotation replaces the previous refresh session with a new one. Do not continue using the old refresh token after receiving a successful response.

#### Error responses

| HTTP status | Meaning |
|---|---|
| `400 Bad Request` | Malformed request body or missing/blank `refresh_token`. |
| `401 Unauthorized` | Invalid, expired, revoked, or replayed refresh token. |
| `403 Forbidden` | User account or school is inactive. |
| `500 Internal Server Error` | Unexpected internal failure. |

#### Rotation and replay detection

- Refresh tokens are opaque and are not JWTs.
- A refresh token is intended to be used once successfully.
- Reusing a previously rotated token triggers token-family replay handling.
- Replay handling revokes active sessions in the affected token family.
- Refresh rotation and logout serialize on the same family-scoped PostgreSQL transaction advisory lock. Rotation also acquires a per-user advisory lock to serialize concurrent rotations across that user's token families.
- Concurrent refresh requests may cause one request to be rejected and can require the user to log in again.
- Selected transaction and concurrency behavior is covered by opt-in PostgreSQL repository integration tests. Broader database-failure and interleaving coverage remains required before production release.

### 3.5 Logout

**Endpoint:** `POST /auth/logout`

**Purpose:** Revokes all active refresh sessions in the token family associated with the supplied refresh token.

Request body:

```json
{
  "refresh_token": "<refresh-token>"
}
```

#### Successful response

**HTTP status:** `200 OK`

```json
{
  "status": "success",
  "message": "Logged out successfully"
}
```

#### Error responses

| HTTP status | Meaning |
|---|---|
| `400 Bad Request` | Malformed request body or missing/blank `refresh_token`. |
| `500 Internal Server Error` | Unexpected failure while revoking the session. |

Logout resolves the token family and revokes every active refresh session in that family. Repeated logout is idempotent. An unknown or malformed token hash is treated as a no-op by the repository, while the HTTP handler still validates that a nonblank token was supplied.

Refresh rotation and logout use the same transaction-scoped PostgreSQL advisory lock for a family. Rotation additionally acquires a per-user transaction advisory lock, serializing concurrent rotations across different families for the same user. This ensures that when a rotation detects an inactive user and revokes all of that user's active refresh sessions, another concurrent rotation cannot leave a replacement active afterward. If refresh obtains the family lock first, logout waits and then revokes the replacement session; if logout obtains it first, the subsequent refresh sees the revoked token and is rejected. Tests cover these documented scenarios, but not every possible database failure or interleaving.

Existing stateless JWT access tokens are not immediately invalidated by logout and can remain usable until their configured expiry. Immediate access-token revocation is not implemented.

### 3.6 Authenticated Identity

**Endpoint:** `GET /auth/me`

**Purpose:** Validates the Bearer access token and returns the authenticated user's identity identifiers.

This endpoint uses JWT authentication middleware.

#### Request header

```http
Authorization: Bearer <access-token>
```

Use the access token returned by a successful login or refresh operation.

#### Successful response

**HTTP status:** `200 OK`

```json
{
  "status": "success",
  "principal_id": 10,
  "principal_type": "staff",
  "school_id": 2,
  "username": "devtest"
}
```

The IDs are illustrative examples.

#### Error responses

| HTTP status | Meaning |
|---|---|
| `401 Unauthorized` | Authorization header is missing or malformed, token is invalid or expired, or identity claims are invalid. |
| `500 Internal Server Error` | Authentication middleware has an internal configuration problem, such as a missing token validator. |

#### Authentication notes

- The middleware validates the access token and reloads current principal and permission state before the handler runs.
- The principal ID is extracted from the token subject. The middleware reloads the current principal subtype, status, token version, and staff school from PostgreSQL.
- The live principal type and token version must match the JWT claims.
- Effective permissions are resolved from live RBAC tables on every protected request; the current resolver intentionally ignores custom-role grants and user ALLOW overrides until exact delegation linkage is implemented.
- The endpoint returns principal identity context, not the full user profile. Complete resource-level authorization and tenant isolation are not yet implemented.
- The endpoint does not, by itself, prove complete tenant isolation for other resources.

### 3.7 Academic Management

All routes in this section require a Bearer access token for an active staff principal. The school ID is derived from the authenticated principal; clients must not send or choose a `school_id`. Each route enforces its live permission in middleware, and the repository independently checks the actor, permission, and school scope. Successful create/update/revoke operations write an audit event in the same database transaction.

#### Academic years

| Method | Endpoint | Permission |
|---|---|---|
| GET | `/school/academic-years` | `academic_years.read` |
| POST | `/school/academic-years` | `academic_years.manage` |
| PATCH | `/school/academic-years/:yearID` | `academic_years.manage` |

Create/update request:

```json
{
  "name": "2026-27",
  "starts_on": "2026-04-01",
  "ends_on": "2027-03-31",
  "status": "active",
  "is_current": true
}
```

Dates use `YYYY-MM-DD`; status is `planned`, `active`, or `closed`. Only an active year can be current, and a school can have only one current year. Closed academic years cannot be edited or reopened through this API.

#### Classes and sections

| Method | Endpoint | Permission |
|---|---|---|
| GET | `/school/classes` | `classes.read` |
| POST | `/school/classes` | `classes.manage` |
| PATCH | `/school/classes/:classID` | `classes.manage` |
| GET | `/school/sections` | `classes.read` |
| POST | `/school/sections` | `classes.manage` |
| PATCH | `/school/sections/:sectionID` | `classes.manage` |

Class request:

```json
{
  "name": "LKG",
  "display_order": 10,
  "is_active": true
}
```

Class names are configurable school data, not a fixed enum. Names such as Pre-Nursery, Play Group, Nursery, LKG, UKG, Class 1–12, or future custom names are supported. Deactivation uses `is_active: false`; class rows are not physically deleted.

Section request:

```json
{
  "academic_year_id": 1,
  "class_id": 1,
  "name": "A",
  "display_order": 1,
  "is_active": true
}
```

Sections are tied to one class and one academic year. The list endpoint optionally accepts `?academic_year_id=1`. Updates preserve the section's original class and academic-year identity. A new/reactivated section requires an active class and a non-closed academic year in the same school.

#### Subjects

| Method | Endpoint | Permission |
|---|---|---|
| GET | `/school/subjects` | `subjects.read` |
| POST | `/school/subjects` | `subjects.manage` |
| PATCH | `/school/subjects/:subjectID` | `subjects.manage` |

Request:

```json
{
  "name": "Mathematics",
  "subject_code": "MATH",
  "is_active": true
}
```

Subject names and optional codes are unique within a school after trimming and case normalization. Deactivation is soft; historical assignment rows remain.

#### Teacher-subject assignments

| Method | Endpoint | Permission |
|---|---|---|
| GET | `/school/teacher-assignments` | `teacher_assignments.read` |
| POST | `/school/teacher-assignments` | `teacher_assignments.manage` |
| DELETE | `/school/teacher-assignments/:assignmentID` | `teacher_assignments.manage` |

Create request:

```json
{
  "academic_year_id": 1,
  "section_id": 1,
  "subject_id": 1,
  "teacher_principal_id": 25
}
```

The list endpoint optionally accepts `academic_year_id` and `section_id` query parameters. PostgreSQL verifies that the teacher is an active same-school staff principal with the Teacher system role, and that the section, class, subject, and academic year are eligible. DELETE revokes the assignment; it does not delete history or permit reactivation.

#### Class-in-charge assignments

| Method | Endpoint | Permission |
|---|---|---|
| GET | `/school/class-incharge-assignments` | `class_incharge_assignments.read` |
| POST | `/school/class-incharge-assignments` | `class_incharge_assignments.manage` |
| DELETE | `/school/class-incharge-assignments/:assignmentID` | `class_incharge_assignments.manage` |

Create request:

```json
{
  "academic_year_id": 1,
  "section_id": 1,
  "teacher_principal_id": 25
}
```

The list endpoint optionally accepts `academic_year_id`. At most one active class-in-charge assignment can exist per section. PostgreSQL enforces the same-school active Teacher identity and non-closed academic-year constraints. DELETE performs a one-way revocation and preserves the assignment history.

Academic create/update conflicts return `409 Conflict`; invalid input or database-enforced academic eligibility failures return `400 Bad Request`; missing school-scoped resources return `404 Not Found`; missing permissions or invalid assignment actors return `403 Forbidden`.


#### Student enrollment, promotion, and history

| Method | Endpoint | Permission |
|---|---|---|
| GET | `/school/enrollments` | `students.read_basic` |
| POST | `/school/enrollments` | `students.create` |
| GET | `/school/students/:studentID/enrollments` | `students.read_basic` |
| POST | `/school/enrollments/:enrollmentID/promote` | `students.update` |

List current/historical enrollment rows for the school, optionally filtered by `academic_year_id`:

```http
GET /school/enrollments?academic_year_id=1
```

Create an initial enrollment for an existing active student:

```json
{
  "student_id": 42,
  "academic_year_id": 1,
  "section_id": 7,
  "roll_number": "12",
  "enrolled_at": "2028-04-01"
}
```

The section determines its class; clients cannot supply a class ID or school ID. The student must be active and belong to the authenticated school. The academic year must be active, and the section must be active and belong to that year and school. Enrollment dates use `YYYY-MM-DD`. A student can have only one enrollment row per academic year; duplicate admission to the same year returns `409 Conflict`.

Read a student's complete school-scoped enrollment history:

```http
GET /school/students/42/enrollments
```

Promote an actively enrolled student into a different active academic year and destination section:

```http
POST /school/enrollments/15/promote
Content-Type: application/json

{
  "academic_year_id": 2,
  "section_id": 9,
  "roll_number": "08",
  "enrolled_at": "2029-04-01"
}
```

Promotion is one transaction: the source enrollment transitions from `active` to `promoted` with its end date, and a new enrollment row is inserted for the destination year. If either operation or its audit write fails, the entire transaction rolls back. Enrollment-time class and section names are snapshotted by PostgreSQL and are never rewritten when catalogue names later change. Enrollment history cannot be deleted or re-homed. A destination year must differ from the source year, and the promotion date cannot precede the original enrollment date.

These endpoints manage placement for an existing student record. Student profile/admission creation and guardian contact/link management are provided by separate APIs; see the student profile and Guardian Contacts sections. No public student self-registration is provided.


## 4. HTTP Status Codes

The following status codes are used by implemented endpoints or may be used by future endpoints.

| Status code | Meaning |
|---|---|
| `200 OK` | Request completed successfully. |
| `201 Created` | Resource created successfully; intended for applicable future endpoints. |
| `204 No Content` | Request succeeded without a response body; intended for applicable future endpoints. |
| `400 Bad Request` | Invalid request data or validation failure. |
| `401 Unauthorized` | Authentication is missing or invalid, or credentials are rejected. |
| `403 Forbidden` | Account or school is inactive for the current authentication endpoints; future authorization checks may also use this status. |
| `404 Not Found` | Resource does not exist; intended for applicable endpoints. |
| `409 Conflict` | Request conflicts with existing data; intended for applicable endpoints. |
| `500 Internal Server Error` | Unexpected server-side failure. |
| `503 Service Unavailable` | A required dependency is unavailable, such as the database during readiness checks. |

Not every status code listed above is currently returned by an implemented endpoint.

## 5. Authentication Endpoints

### 5.1 Implemented Endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| Method | Endpoint | Purpose |
|---|---|---|
| POST | /auth/login | Staff username/password login; privileged staff continue through MFA. |
| POST | /auth/platform-admin/login | Platform Admin email/password login; always starts mandatory MFA. |
| POST | /auth/mfa/enroll/verify | Complete first-time TOTP enrollment using a challenge token and code. |
| POST | /auth/mfa/verify | Verify a TOTP login challenge and issue the session. |
| POST | /auth/mfa/recovery-codes/regenerate/start | Start authenticated recovery-code regeneration challenge. |
| POST | /auth/mfa/recovery-codes/regenerate | Verify challenge/code and issue replacement recovery codes. |
| POST | /auth/refresh | Rotate a principal-bound refresh token. |
| POST | /auth/logout | Revoke active refresh sessions in the supplied token family. |
| GET | /auth/me | Return live principal identity context after token and database validation. |
| POST | /platform/schools | Create a school and its initial School Admin; requires both platform school-create permissions. |

The one-time first Platform Admin bootstrap is a manually invoked CLI command, not a public HTTP endpoint. There is no public registration endpoint. RequirePermission and live permission resolution are implemented, but complete tenant/resource/assignment authorization and custom-role delegation remain incomplete.

### 5.2 Planned Authentication Endpoints and Controls

| Endpoint or capability | Purpose | Status |
|---|---|---|
| Staff password change/reset | Secure self-service and same-school administrative password lifecycle. | Pending |
| Exact delegation linkage | Bind custom-role grants and user ALLOW overrides to the precise delegation record. | Pending |
| Immediate access-token revocation | Invalidate already-issued access tokens after password reset, disablement or other security events. | Pending |
| Business API route guards | Add authentication, permission and resource-scope checks to each school module route. | Pending |

School and user provisioning must be restricted to authorized workflows. Do not expose unrestricted public registration.

### 5.3 Access Token

The token service generates and validates JWT access tokens.

Configuration includes:

- JWT signing secret.
- JWT issuer.
- Configurable access-token expiry.

The signing secret must be configured securely through environment variables and must not be committed to source control.

The intended authorization header is:

```http
Authorization: Bearer <access-token>
```

The authentication middleware protects GET /auth/me, the authenticated MFA recovery-code regeneration start route, and POST /platform/schools. The school-onboarding route applies both required live platform permissions. Every future protected route must explicitly add authentication, the relevant permission checks, and resource-scope enforcement. Complete tenant/resource/assignment authorization remains pending.

### 5.4 Refresh Token

Refresh tokens are opaque random values rather than JWTs. The backend stores their SHA-256 hashes in the `refresh_sessions` table.

The implemented workflow supports issuance, rotation, revocation, and replay detection through the authentication endpoints. The configured lifetime is controlled by `REFRESH_TOKEN_EXPIRY_DAYS`.

Opt-in PostgreSQL repository integration tests cover refresh-session persistence and lifecycle scenarios, including concurrent refresh behavior. Additional HTTP-level end-to-end logout verification and broader database failure/concurrency coverage remain required before production readiness can be claimed.

## 6. Authorization and Tenant Isolation

The core principal/RBAC schema and permission resolver are implemented in the current branch. Protected requests reload live principal state and permissions; RequirePermission is used on the Platform Admin school-onboarding route. Built-in system-role grants and explicit DENY overrides are resolved. Custom-role grants and user ALLOW overrides intentionally fail closed until exact delegation-ID linkage is added. Full tenant/resource/assignment/field-level authorization remains pending for the school business APIs.

Approved Phase 1 roles are Platform Admin (global), School Admin (one school), and Teacher (one school). Parent and Driver are Phase 2 mobile roles; Parent-specific implementation is currently paused. Student login is not part of Phase 1.

Staff login now accepts a globally unique, case-insensitive username and password; it does not accept school_id. Staff accounts remain tied to exactly one school. Platform Admin uses a separate email/password login followed by mandatory TOTP MFA. School Admin also requires TOTP MFA.

### Tenant-Isolation Requirements

- Every school-specific resource must be associated with the appropriate school.
- Authenticated users must not access another school's data without explicit authorization.
- School ownership must be checked by server-side operations.
- A client-provided `school_id` must never be trusted as authorization by itself.
- Database queries and service-layer operations must consistently enforce tenant boundaries.
- Cross-school access attempts must be covered by automated tests.

These are required design rules, not evidence that complete tenant isolation is already implemented.

The current `/auth/me` endpoint returns the school ID from validated token claims. This does not replace server-side ownership checks on school-specific resource endpoints.

## 7. API Versioning and Route List

The currently registered routes are:

| Method | Endpoint | Authentication |
|---|---|---|
| GET | /health | Not required |
| GET | /ready | Not required; checks database readiness |
| POST | /auth/login | Staff credentials; no token required |
| POST | /auth/platform-admin/login | Platform Admin credentials; no token required; MFA follows |
| POST | /auth/mfa/enroll/verify | One-time challenge token and TOTP code |
| POST | /auth/mfa/verify | One-time challenge token and TOTP/recovery code |
| POST | /auth/mfa/recovery-codes/regenerate/start | Bearer access token required |
| POST | /auth/mfa/recovery-codes/regenerate | One-time challenge token and TOTP code |
| POST | /auth/refresh | Refresh token in request body |
| POST | /auth/logout | Refresh token in request body |
| GET | /auth/me | Bearer access token required |
| POST | /platform/schools | Bearer token plus platform.schools.create and platform.school_admins.create |

The current routes do not use an `/api/v1` prefix. Versioned routes may be introduced through an explicit routing and migration decision.

Do not assume `/api/v1/auth/login` or `/api/v1/auth/me` is currently available.

## 8. Local Development

### 8.1 Configuration

Use `backend/.env.example` as the reference for environment variables.

Create and maintain a local `backend/.env` for development configuration. Keep local secrets out of Git and public documentation.

### 8.2 Start the API

From the backend directory:

```bash
go run ./cmd/api
```

The default development port is `8081`, unless overridden through configuration.

### 8.3 Check Health and Readiness

Health:

```http
GET http://localhost:8081/health
```

Readiness:

```http
GET http://localhost:8081/ready
```

### 8.4 Test Login

Send a `POST` request to:

`http://localhost:8081/auth/login`

Use a valid development user's `school_id`, `username`, and `password`.

Do not put real passwords or access tokens into source files, documentation, Git commits, or public issue reports.

### 8.5 Test Authenticated Identity

Send a `GET` request to:

`http://localhost:8081/auth/me`

Include:

```http
Authorization: Bearer <access-token>
```

A valid token should return the authenticated user's ID and school ID. A missing, malformed, or invalid token should return `401 Unauthorized`.

## 9. Testing

Run the backend test suite from the backend directory:

```bash
go test -count=1 ./...
```

Run static analysis from the backend directory:

```bash
go vet ./...
```

Previously reported test coverage includes:

- Configuration validation.
- Password hashing and verification.
- Login success and failure.
- JWT generation and validation.
- Authentication middleware handling of valid and invalid authentication.
- Missing and malformed authorization headers.
- Invalid or missing token claims.
- Authenticated identity endpoint behavior.
- Health and readiness behavior.
- Refresh-token rotation, revocation, and replay handling at the tested service/handler levels.

The exact coverage should be rechecked against the current tests before a release.

Additional tests required include:

- PostgreSQL-backed repository integration tests for refresh-session persistence and rotation are implemented as opt-in tests; run them only against the dedicated test database.
- Additional concurrent refresh/replay race interleavings and database-failure scenarios beyond the existing integration-test coverage.
- Logout using an independent session and repeated logout behavior.
- Token expiry and invalid-token handling.
- Request validation and database error handling.
- RBAC and permission enforcement.
- Cross-school access prevention.
- Login rate limiting, account lockout, MFA, and password-reset behavior when implemented.

Only functionality supported by the current code and relevant tests should be marked complete.

## 10. Security Notes

- Never commit `.env` files or secrets to source control.
- Use a strong, private JWT signing secret.
- Use HTTPS in production.
- Never store plaintext passwords.
- Use the existing bcrypt password-hashing service.
- Do not return password hashes or sensitive internal error details.
- Keep development-only tools separate from production application entry points.
- Do not expose unrestricted user registration.
- Apply authentication middleware to protected endpoints.
- Implement authorization checks before exposing role-restricted functionality.
- Enforce school-level tenant isolation on the server.
- Validate request input and return appropriate HTTP status codes.
- Review database permissions, connection settings, migrations, logging, and deployment configuration before production release.
- Do not treat the current authentication foundation as proof that the full application is production-ready.


### 3.9 Guardian Contacts and Student Relationships

Guardian records are school-scoped contact records, not authenticated Parent accounts. These APIs do not create parent principals, enable the parent portal, verify mobile ownership, send OTPs, or initiate SMS delivery.

All guardian routes require the dedicated `guardians.read` or `guardians.manage` permission. Neither permission is included in the built-in School Admin or Teacher role. V19 makes these two permissions explicitly delegable; a Platform Admin must grant a school-specific delegation with `can_assign_to_roles=true`. The School Admin can then create a custom role such as **Guardian Manager**, grant only the required guardian permission(s) using the exact delegation ID, and assign that role to the designated staff. The system-role permission bundles remain immutable. Teacher's `students.read_basic` permission does not grant guardian access.

| Method | Endpoint | Permission | Purpose |
|---|---|---|---|
| GET | `/school/guardians` | `guardians.read` | Search/list guardian contacts |
| POST | `/school/guardians` | `guardians.manage` | Create a guardian contact |
| GET | `/school/guardians/:guardianID` | `guardians.read` | Read one guardian contact |
| PATCH | `/school/guardians/:guardianID` | `guardians.manage` | Update contact details/status |
| GET | `/school/students/:studentID/guardians` | `guardians.read` | List linked guardians |
| POST | `/school/students/:studentID/guardians` | `guardians.manage` | Link a guardian to a student |
| POST | `/school/student-guardian-links/:linkID/revoke` | `guardians.manage` | Revoke a relationship with a reason |

Guardian list supports `q`, `status`, `limit`, and `offset`; page size defaults to 50 and is capped at 100. Search matches name, mobile number, and email. Guardian creation accepts `full_name`, optional `mobile_number`, optional `email_address`, and optional `status` (defaults to active). Updates accept the same fields and require an explicit status. Guardian contact values are school-scoped and are not globally unique.

Link request:

```json
{
  "guardian_id": 42,
  "relationship_type": "mother",
  "is_primary_contact": true,
  "is_emergency_contact": true
}
```

A student can have multiple guardians and one guardian can be linked to multiple students in the same school. The API rejects inactive or cross-school records and duplicate active pairs. Selecting a new primary contact clears the previous active primary contact for that student within the same transaction. Relationship history is retained; it is never physically deleted or reactivated. Revocation requires a non-empty reason and is recorded in the audit trail.

Use `GET /school/students/:studentID/guardians?include_history=true` to include revoked relationships. Without that query parameter, only active relationships are returned. All repository operations independently validate the authenticated staff principal, current permission, and school ownership; clients cannot submit a school ID.

## 11. Documentation Maintenance

Update this document whenever an API endpoint, request or response schema, status code, authentication flow, or authorization rule changes.

Keep this document consistent with:

- `README.md`
- `PROGRESS.md`
- `CHANGELOG.md`
- `docs/ARCHITECTURE.md`
- `docs/DATABASE.md`
- `docs/HANDOFF.md`
- `docs/REQUIREMENTS.md`
- `docs/SECURITY.md`
- `docs/TESTING.md`

Clearly distinguish implemented functionality from planned functionality. Do not mark a feature complete until its implementation and required tests have been reviewed.


## 12. Approved Phase 1 Authorization and Account API Requirements (Planned)

This section describes target API behavior only. Existing endpoints above remain the implemented contract until the migration is designed, coded, tested, and documented as implemented.

### 12.1 Staff login target

Staff accounts use globally unique, case-insensitive usernames. The target login identifies one staff account using username + password; it must not require a school ID to disambiguate duplicate usernames because duplicates are prohibited globally. Each staff account remains tied to exactly one school. Current `POST /auth/login` still requires `school_id`; changing that contract is pending.

### 12.2 Staff password lifecycle target

- School Admin creates a staff account with a temporary initial password.
- Staff must change the temporary password at first login.
- Self-service password change requires the current password.
- School Admin can reset a same-school staff password without the old password.
- Admin reset revokes all target sessions, invalidates outstanding access tokens, requires password change at next login, and is audited.

### 12.3 Parent authentication and child selection target

Parent authentication uses registered mobile number + OTP. After successful OTP verification and guardian matching, one parent account can access all officially linked children, including children across schools. A child-list endpoint and selected-child context may support the dashboard, but every child-data endpoint must independently verify the current parent-child relationship. Parent access is read-only in Phase 1; no student login API is planned.

### 12.4 Permission and resource-scope enforcement

Every protected operation must check the effective permission and resource scope. Teacher teaching assignments and class-in-charge assignments are distinct; teaching assignment alone does not grant attendance access. Subject teachers receive only basic student fields (name, roll number, class, section) for assigned class/subject. Class-in-charge teachers may access attendance, same-day corrections within cutoff, basic details, timetable, homework, and applicable notices only for their assigned class/section. Past attendance correction is School Admin-only. Parent endpoints verify child links on every request.

School Admin may create school-scoped custom roles only from platform-defined permissions within an explicit delegation ceiling. Explicit user deny overrides role allow; explicit allow is valid only when the assigning admin can delegate that permission. Permission changes take effect on the next protected request, including existing sessions.


## 6. Platform Admin onboarding and MFA routes

### Platform Admin login

POST /auth/platform-admin/login accepts email and password. On valid credentials it returns HTTP 202 with status=mfa_required and a short-lived challenge token. The client must complete TOTP enrollment or verification before a session is issued. A successful MFA completion returns the normal principal-based access/refresh token response. Platform Admin MFA cannot be bypassed by using the staff login route.

### MFA verification

POST /auth/mfa/enroll/verify and POST /auth/mfa/verify accept challenge_token and code. Enrollment completion returns recovery codes once; clients must display and securely store them because they are not returned again. Recovery-code regeneration first requires an authenticated start request, then POST /auth/mfa/recovery-codes/regenerate with challenge_token and code.

### Platform Admin school onboarding

POST /platform/schools requires a Bearer access token and both live permissions platform.schools.create and platform.school_admins.create. The JSON body contains name, code, school_admin_username and school_admin_initial_password. School, initial School Admin principal/account and role assignment are created transactionally. The endpoint returns the school and new School Admin principal ID/username, never the password or password hash.

The initial Platform Admin itself is created separately by the one-time bootstrap CLI after the V14 migration has been reviewed and applied to the dedicated test database. Do not run that CLI against a database until the operator has explicitly selected and verified the target.


### 3.8 Student Admission and Basic Profile

Profile read routes require \`students.read_sensitive\`, not the Teacher role's limited \`students.read_basic\` permission. This prevents the basic teacher permission from exposing the full school student directory or unrestricted profile records. All routes require an authenticated staff principal and derive the school from that principal. No endpoint accepts a client-supplied `school_id`. Repository authorization independently checks the current permission and school ownership. Student identity is separate from academic-year enrollment: creating a student does not automatically place the student into a class or section. Use the enrollment endpoint after creating the profile.

The current V11 schema stores only admission number, full name, status, and timestamps. This API intentionally does not invent date-of-birth, gender, address, medical, family, or other sensitive profile fields. Those require a separately reviewed schema and field-level permission design.

| Method | Endpoint | Permission | Purpose |
|---|---|---|---|
| GET | `/school/students` | `students.read_sensitive` | Search/list school students |
| POST | `/school/students` | `students.create` | Admit/create student identity |
| GET | `/school/students/:studentID` | `students.read_sensitive` | Read one student |
| PATCH | `/school/students/:studentID` | `students.update` | Update student name |
| POST | `/school/students/:studentID/archive` | `students.archive` | Soft-archive student |

Create request:

```json
{
  "admission_number": "ADM-2026-001",
  "full_name": "Student Name"
}
```

The admission number is unique within its school after trimming and case-folding. The API returns `409 Conflict` for a duplicate. Names and admission numbers are trimmed and validated against the existing V11 column lengths.

List supports optional `q`, `status`, `limit`, and `offset` query parameters. Search matches admission number or full name; status may be `active`, `inactive`, or `archived`. The default page size is 50 and the maximum is 100. The response includes a total count and pagination values. The basic directory does not join guardian contacts or sensitive student data.

Update request:

```json
{
  "full_name": "Updated Student Name"
}
```

Admission number is immutable through the profile update endpoint to preserve the school's stable identity reference. Archived students cannot be updated or archived again. Archiving is a status transition, not physical deletion; enrollment and guardian relationship history is retained. Student creation, update, and archive each write an audit event in the same transaction as the data change.

After admission, create the student's academic placement using `POST /school/enrollments`. Promotion creates a new enrollment row and preserves the previous academic-year history; it does not create another student identity.

Guardian contact and student-guardian relationship APIs are implemented below. Parent account creation, OTP verification, and SMS delivery remain paused.


## Student Attendance (V20)

- GET `/school/attendance?academic_year_id=1&section_id=2&date=2026-11-04` lists attendance within the caller's permitted school and section scope. Filters are optional; results are capped at 1,000 records.
- POST `/school/attendance` records one status for an enrollment and date. JSON: `{"enrollment_id":12,"attendance_date":"2026-11-04","status":"present","remarks":"optional"}`.
- PATCH `/school/attendance/:attendanceID` updates same-day status/remarks and requires `attendance.update_same_day`.
- PATCH `/school/attendance/:attendanceID/correct` corrects historical status/remarks and requires `attendance.correct_past`. JSON: `{"status":"late","remarks":"optional"}`.
- Status values: `present`, `absent`, `late`, `excused`. Only one record is allowed per enrollment/date.
- School Admin access is school-scoped. Teachers are restricted to sections with an active class-in-charge assignment; subject-only assignments do not grant attendance access.
- Same-day changes require `attendance.update_same_day`; past-date entries/corrections require `attendance.correct_past`. Future dates and dates outside the enrollment period are rejected. Every write creates an audit event.

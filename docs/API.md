# API Documentation

## 1. Overview

This document describes the School Management System backend API, its conventions, implemented endpoints, authentication behavior, and planned functionality.

The backend uses Go, Gin, and PostgreSQL.

### Current Implementation Status

- `GET /health` is implemented.
- `GET /ready` is implemented.
- `POST /auth/login` is implemented.
- `POST /auth/refresh` is implemented.
- `POST /auth/logout` is implemented.
- JWT access-token generation and validation are implemented.
- JWT authentication middleware protects `GET /auth/me`.
- `GET /auth/me` returns the authenticated user's ID and school ID, not the full user profile.
- Internal user creation is available through the service layer and a development seed command.
- Public user registration is not implemented.
- Refresh tokens use opaque random values, database-backed rotation and revocation, and token-family replay detection.
- Role-Based Access Control (RBAC), complete tenant-isolation enforcement, MFA, login rate limiting, and account lockout are not yet complete.

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

Future protected endpoints must verify that the authenticated user is authorized to access the requested school's data. A client-provided `school_id` must not, by itself, be treated as proof of authorization.

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

**Purpose:** Authenticates a user belonging to a school and returns an access token and a refresh token.

Request body:

```json
{
  "school_id": 1,
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
  "user": {
    "id": 1,
    "school_id": 1,
    "username": "devtest",
    "status": "active"
  }
}
```

The actual user ID and username depend on the authenticated account. The response must not contain a password or password hash.

#### Error responses

| HTTP status | Meaning |
|---|---|
| `400 Bad Request` | Malformed JSON, missing required fields, or invalid `school_id`. |
| `401 Unauthorized` | Invalid username or password. |
| `403 Forbidden` | User account or school is inactive. |
| `500 Internal Server Error` | Unexpected internal failure, such as access-token or session issuance failure. |

#### Authentication notes

- The login request identifies the school using `school_id`.
- The authentication service checks the user and school.
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

The response contains a new `access_token`, a replacement `refresh_token`, `token_type: "Bearer"`, and the basic user DTO, using the same response structure as login.

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
  "user_id": 10,
  "school_id": 2
}
```

The IDs are illustrative examples.

#### Error responses

| HTTP status | Meaning |
|---|---|
| `401 Unauthorized` | Authorization header is missing or malformed, token is invalid or expired, or identity claims are invalid. |
| `500 Internal Server Error` | Authentication middleware has an internal configuration problem, such as a missing token validator. |

#### Authentication notes

- The middleware validates the access token before the handler runs.
- The user ID is extracted from the token subject and the school ID from token claims.
- Invalid identity identifiers are rejected.
- The endpoint returns IDs from validated claims; it does not retrieve the full user profile from the database.
- The endpoint does not yet perform role or permission checks.
- The endpoint does not, by itself, prove complete tenant isolation for other resources.

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
| `POST` | `/auth/login` | Authenticate a user and issue access and refresh tokens. |
| `POST` | `/auth/refresh` | Rotate a refresh token and issue a new access token. |
| `POST` | `/auth/logout` | Revoke all active refresh sessions in the supplied token family. |
| `GET` | `/auth/me` | Return identity identifiers from a validated access token. |

Internal user creation is available through the service layer and a development seed command. There is no public registration endpoint.

RBAC and complete tenant-isolation enforcement are separate, pending capabilities.

### 5.2 Planned Authentication Endpoints and Controls

| Endpoint or capability | Purpose | Status |
|---|---|---|
| `POST /auth/register` | Controlled user provisioning, if approved by the final onboarding policy. | Planned |
| Password reset | Secure, one-use password reset workflow. | Planned |
| MFA | Mandatory MFA for Platform Admin and School Admin. | Planned |
| Login rate limiting and account lockout | Reduce brute-force login attempts. | Planned |
| Immediate access-token revocation | Invalidate an access token before expiry. | Planned |

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

The authentication middleware currently protects `GET /auth/me`.

The presence of middleware does not automatically protect other routes. Every protected route must explicitly apply the required middleware. Role-based authorization and complete tenant isolation remain pending.

### 5.4 Refresh Token

Refresh tokens are opaque random values rather than JWTs. The backend stores their SHA-256 hashes in the `refresh_sessions` table.

The implemented workflow supports issuance, rotation, revocation, and replay detection through the authentication endpoints. The configured lifetime is controlled by `REFRESH_TOKEN_EXPIRY_DAYS`.

Opt-in PostgreSQL repository integration tests cover refresh-session persistence and lifecycle scenarios, including concurrent refresh behavior. Additional HTTP-level end-to-end logout verification and broader database failure/concurrency coverage remain required before production readiness can be claimed.

## 6. Authorization and Tenant Isolation

RBAC and complete tenant-isolation enforcement are pending.

The current user schema associates a user with a school through `school_id`; the current users schema does not yet provide the complete role and permission system.

Future authorization must define and enforce permissions for supported roles, which may include:

- Platform Admin.
- School Admin.
- Teacher.
- Parent.
- Student.
- Driver.
- Other approved school staff.

The final role list and permission matrix must be documented before RBAC can be considered complete.

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
| `GET` | `/health` | Not required |
| `GET` | `/ready` | Not required; checks database readiness |
| `POST` | `/auth/login` | Not required |
| `POST` | `/auth/refresh` | Refresh token in request body |
| `POST` | `/auth/logout` | Refresh token in request body |
| `GET` | `/auth/me` | Bearer access token required |

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

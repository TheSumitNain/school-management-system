# API Documentation

## 1. Overview

This document describes the School Management System backend API, its conventions, implemented endpoints, authentication behavior, and planned API functionality.

The backend is built using Go, Gin, and PostgreSQL.

### Current Implementation Status

- Health check endpoint implemented.
- Readiness endpoint implemented.
- Login endpoint implemented.
- JWT access token generation and validation services implemented.
- User creation is available internally through the service layer and a development-only seed command.
- Public user registration endpoint is not implemented.
- Refresh token and logout endpoints are not implemented.
- Protected profile endpoint and authentication middleware are not implemented.
- Role-Based Access Control (RBAC) and tenant-isolation middleware are pending.

**Important:** Planned functionality described in this document must not be treated as implemented functionality.

## 2. API Conventions

### 2.1 Base URL

Local development:

`http://localhost:8081`

### 2.2 Content Type

Endpoints that accept or return JSON use:

`Content-Type: application/json`

### 2.3 Response Format

Successful responses should use a consistent JSON structure where applicable.

Example:

```json
{
  "status": "success"
}
```

Error responses should avoid exposing internal errors, database details, secrets, or sensitive information.

### 2.4 Identifiers

Database identifiers are represented as numeric IDs in the current implementation.

### 2.5 Multi-Tenant Architecture

The application is designed to support multiple schools within a shared PostgreSQL database.

School-specific records are associated with a `school_id`.

Future protected endpoints must verify that the authenticated user is authorized to access the requested school's data. A client-provided `school_id` must not, by itself, be considered proof of authorization.

## 3. Implemented Endpoints

### 3.1 Health Check

**Endpoint:** `GET /health`

**Purpose:** Checks whether the API process is responding.

Example:

```http
GET /health
```

Expected successful response:

```json
{
  "status": "ok"
}
```

### 3.2 Readiness Check

**Endpoint:** `GET /ready`

**Purpose:** Checks whether the application is ready to serve requests, including its database readiness.

Example:

```http
GET /ready
```

Expected successful response:

```json
{
  "status": "ready"
}
```

If the application is not ready, the endpoint returns an unsuccessful HTTP response.

### 3.3 Login

**Endpoint:** `POST /auth/login`

**Purpose:** Authenticates a user belonging to a school and returns a JWT access token.

**Request body:**

```json
{
  "school_id": 1,
  "username": "devtest",
  "password": "your-test-password"
}
```

The credentials above are illustrative placeholders, not actual credentials.

#### Successful response

**HTTP status:** `200 OK`

Example:

```json
{
  "status": "success",
  "access_token": "<access-token>",
  "token_type": "Bearer",
  "user": {
    "id": 1,
    "school_id": 1,
    "username": "devtest",
    "status": "active"
  }
}
```

The actual user ID and username depend on the authenticated account.

The response must not include the user's password or password hash.

#### Error responses

| HTTP Status | Meaning |
|---|---|
| `400 Bad Request` | Request body is malformed or input validation fails. |
| `401 Unauthorized` | Credentials are invalid, or authentication fails. |
| `500 Internal Server Error` | An unexpected internal error occurs, such as an access-token issuance failure. |

Inactive users or schools are currently handled by the authentication service as invalid credentials, so these cases generally result in `401 Unauthorized`.

#### Authentication notes

- The login request identifies the school using `school_id`.
- The authentication service checks that the school and user are eligible to authenticate.
- Passwords are verified against stored password hashes.
- Passwords are hashed using bcrypt when users are created.
- A successful login returns an access token.
- The configured access-token expiration applies.
- The current login endpoint does not return a refresh token.
- A successful login does not imply that protected API routes or RBAC have been implemented.

## 4. HTTP Status Codes

The following status codes are used or expected by the API.

| Status Code | Meaning |
|---|---|
| `200 OK` | Request completed successfully. |
| `201 Created` | A resource was created successfully; to be used by applicable future endpoints. |
| `204 No Content` | Request succeeded without a response body; to be used where appropriate. |
| `400 Bad Request` | Invalid request data or validation failure. |
| `401 Unauthorized` | Authentication is missing or invalid, or credentials are rejected. |
| `403 Forbidden` | Authenticated user lacks permission; to be used by protected endpoints when authorization is implemented. |
| `404 Not Found` | Requested resource does not exist; to be used by applicable endpoints. |
| `409 Conflict` | Request conflicts with existing data; to be used where appropriate. |
| `500 Internal Server Error` | Unexpected server-side failure. |
| `503 Service Unavailable` | Application or a required dependency is unavailable, where applicable. |

Not every status code listed above is currently returned by an implemented endpoint.

## 5. Authentication Endpoints

### 5.1 Implemented

Currently implemented:

- `POST /auth/login`

The backend also has internal services for user creation and access-token generation and validation.

The development seed command is intended for local development and is not a public registration endpoint.

### 5.2 Planned Endpoints

The following endpoints are planned but are **not implemented yet**.

| Method | Endpoint | Purpose | Status |
|---|---|---|---|
| `POST` | `/auth/register` | Controlled user provisioning or registration, depending on the final authorization rules. | Planned |
| `POST` | `/auth/refresh` | Issue a new access token using a valid refresh session. | Planned |
| `POST` | `/auth/logout` | Revoke the relevant refresh session and handle access-token revocation according to the final design. | Planned |
| `GET` | `/auth/me` | Return the authenticated user's profile. | Planned |

The exact registration policy must be finalized before exposing a public registration endpoint. School and user provisioning must be restricted to authorized workflows.

### 5.3 Access Token

The current token service generates and validates JWT access tokens.

The token configuration includes:

- JWT signing secret.
- JWT issuer.
- Configurable access-token expiration.

The signing secret must be configured securely through environment variables and must not be committed to source control.

The current configuration validates the secret length and access-token expiration settings.

The access token is intended to be sent in the HTTP `Authorization` header:

```http
Authorization: Bearer <access-token>
```

**Important:** The header format is documented for future protected endpoints. Authentication middleware and protected profile routes are not implemented yet.

### 5.4 Refresh Token

The database includes a `refresh_sessions` table, but refresh-token issuance, rotation, and revocation are not yet exposed through HTTP endpoints.

The refresh-token workflow must be implemented and tested before refresh sessions are considered operational.

## 6. Authorization and Tenant Isolation

Role-Based Access Control (RBAC) and tenant-isolation enforcement are pending.

The database currently associates users with schools through `school_id`. The current users schema does not yet contain a role column.

Future authorization must define and enforce permissions for the supported roles, such as:

- School administrator.
- Teacher.
- Parent.
- Student.
- Driver.
- Other approved school staff.

The final role list and permission matrix must be documented before RBAC implementation is considered complete.

### Tenant-Isolation Requirements

- Every school-specific resource must be associated with the appropriate school.
- Authenticated users must not access another school's data without explicit authorization.
- School ownership must be checked on server-side operations.
- Clients must not be trusted to authorize themselves by supplying a `school_id`.
- Database queries and service-layer operations must consistently enforce tenant boundaries.
- Cross-school access attempts must be covered by automated tests.

These are required design rules; they should not be interpreted as proof that complete tenant-isolation middleware already exists.

## 7. API Versioning

The current implemented routes are:

- `GET /health`
- `GET /ready`
- `POST /auth/login`

These routes currently do not use an `/api/v1` prefix.

For future public API expansion, versioned routes such as `/api/v1/...` may be introduced through an explicit routing and migration decision.

Do not assume that `/api/v1/auth/login` is currently available.

## 8. Local Development

### 8.1 Configuration

Use `backend/.env.example` as the reference for required environment variables.

Create and maintain a local `backend/.env` for development configuration.

The local `.env` file contains environment-specific values and secrets. It must not be committed to Git or shared publicly.

### 8.2 Start the API

From the backend directory:

```bash
go run ./cmd/api
```

The default development port is `8081`, unless overridden through configuration.

### 8.3 Check API Health

```http
GET http://localhost:8081/health
```

### 8.4 Check Readiness

```http
GET http://localhost:8081/ready
```

### 8.5 Test Login

Send a `POST` request to:

`http://localhost:8081/auth/login`

Use a valid development user's `school_id`, `username`, and `password`.

Do not put real passwords or access tokens into source files, documentation, Git commits, or public issue reports.

## 9. Testing

The backend test suite can be run from the backend directory:

```bash
go test -count=1 ./...
```

Tests should cover the following areas as the API grows:

- Configuration validation.
- Password hashing and verification.
- Login success and failure.
- JWT generation and validation.
- Token expiration and invalid-token handling.
- Request validation.
- Database error handling.
- Authentication middleware.
- RBAC and permission enforcement.
- Tenant isolation and cross-school access prevention.
- Refresh-token rotation and revocation.
- Logout behavior.
- Health and readiness behavior.

Only functionality covered by implemented code and relevant tests should be marked complete in project documentation.

## 10. Security Notes

- Never commit `.env` files or secrets to source control.
- Use a strong, private JWT signing secret.
- Use HTTPS in production.
- Do not store plaintext passwords.
- Use bcrypt password hashing through the existing password-hashing service.
- Do not return password hashes or sensitive internal error details in API responses.
- Keep development-only tools separate from production application entry points.
- Do not expose unrestricted user registration.
- Apply authentication and authorization middleware before exposing protected endpoints.
- Enforce school-level tenant isolation on the server.
- Validate request input and return appropriate HTTP status codes.
- Review database permissions, connection settings, migrations, logging, and deployment configuration before production release.

## 11. Documentation Maintenance

Update this document whenever an API endpoint, request or response schema, status code, authentication flow, or authorization rule changes.

Keep API documentation consistent with:

- `README.md`
- `PROGRESS.md`
- `CHANGELOG.md`
- `docs/ARCHITECTURE.md`
- `docs/DATABASE.md`
- `docs/HANDOFF.md`
- `docs/REQUIREMENTS.md`
- `docs/SECURITY.md`
- `docs/TESTING.md`

Clearly distinguish implemented functionality from planned functionality.

Do not mark a feature complete until its implementation and required tests have been reviewed.

# API Documentation

## 1. Overview

This document describes the current HTTP API for the School Management System.

**Backend stack:** Go, Gin, PostgreSQL

**Default local API address:** `http://localhost:8081`

**Current implementation status:** Initial API foundation with health and readiness endpoints. Authentication, authorization, and school-specific business endpoints are not yet exposed through HTTP routes.

## 2. API Conventions

- API requests and responses use HTTP.
- JSON is the expected format for JSON request and response bodies.
- HTTP status codes indicate the outcome of a request.
- Database readiness is checked separately from basic application health.
- Protected routes and tenant-specific data access will be introduced after authentication and authorization are implemented.

## 3. Implemented Endpoints

### 3.1 Health Check

**Endpoint:** `GET /health`

**Purpose:** Checks whether the API process is running and able to respond.

**Authentication:** Not required.

**Expected successful status:** `200 OK`

**Response example:**

```json
{
  "status": "success",
  "message": "School Management System API is running"
}
```

This endpoint is intended to provide a basic application health signal. It should not be treated as proof that PostgreSQL is available.

### 3.2 Readiness Check

**Endpoint:** `GET /ready`

**Purpose:** Checks whether the application is ready to serve requests, including whether its database connection is available.

**Authentication:** Not required.

**Expected outcomes:**

- `200 OK` when the database readiness check succeeds.
- `503 Service Unavailable` when the database readiness check fails.

The exact response body should follow the implementation in the current backend source code.

This endpoint can be used by deployment and monitoring systems to determine whether the application is ready to receive traffic.

## 4. HTTP Status Codes

The current health and readiness endpoints use the following status codes:

| Status | Meaning |
|---|---|
| `200 OK` | The requested health or readiness check succeeded. |
| `503 Service Unavailable` | The readiness check failed because a required dependency is unavailable. |

Future endpoints may also use standard HTTP status codes such as:

- `400 Bad Request` for invalid request input.
- `401 Unauthorized` for missing or invalid authentication.
- `403 Forbidden` when an authenticated user lacks permission.
- `404 Not Found` when the requested resource does not exist.
- `409 Conflict` for a conflicting operation.
- `422 Unprocessable Content` when adopted by the API's validation conventions.
- `500 Internal Server Error` for unexpected server failures.

These future status-code conventions are guidance and do not imply that corresponding handlers have already been implemented.

## 5. Authentication Endpoints — Planned

The following endpoints are not yet implemented as HTTP routes in this project:

| Method | Planned endpoint | Purpose |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Register a user within an authorized school context. |
| `POST` | `/api/v1/auth/login` | Authenticate a user and issue tokens. |
| `POST` | `/api/v1/auth/refresh` | Renew an access token using a valid refresh session. |
| `POST` | `/api/v1/auth/logout` | Revoke the relevant refresh session. |
| `GET` | `/api/v1/auth/me` | Return the authenticated user's permitted profile information. |

These paths are proposals for the future API structure, not currently available routes.

Registration must not allow an unauthenticated client to create an administrator for an arbitrary school. The eventual registration flow must define a secure school-provisioning and account-creation process.

## 6. Authorization and Tenant Isolation — Planned

The system is intended to serve multiple schools.

Future protected endpoints must:

1. Authenticate the request.
2. Identify the authenticated user and their authorized school context.
3. Verify the user's permissions for the requested operation.
4. Restrict database queries and mutations to the authorized school.
5. Avoid trusting a client-supplied `school_id` as proof of authorization.

Role-based access control (RBAC) and tenant-isolation middleware are not yet implemented in the current HTTP API.

## 7. API Versioning

The proposed convention for future business and authentication endpoints is:

`/api/v1/...`

The existing `/health` and `/ready` endpoints remain at their current paths.

Versioning conventions may be finalized before the API expands.

## 8. Local Development

Start the API from the backend directory:

```bash
go run ./cmd/api
```

The default local server address is:

`http://localhost:8081`

Check application health:

```bash
curl http://localhost:8081/health
```

Check application readiness:

```bash
curl http://localhost:8081/ready
```

These commands assume the required environment configuration is present and PostgreSQL is available for the readiness check.

## 9. Testing

Run backend tests from the backend directory:

```bash
go test -count=1 ./...
```

Run static analysis:

```bash
go vet ./...
```

Endpoint-specific integration tests should verify successful responses, failure behavior, HTTP status codes, and database-unavailable scenarios.

## 10. Security Notes

- Do not return password hashes, refresh-token hashes, database credentials, or signing secrets in API responses.
- Authentication and authorization must be enforced on the server.
- Do not rely on frontend route protection as an authorization boundary.
- Validate all client-provided input.
- Avoid exposing internal database errors or sensitive implementation details to clients.
- Production deployment must use an appropriately secured network connection and production-ready configuration.

See `SECURITY.md` for the broader security requirements.

## 11. Documentation Maintenance

When adding or changing an endpoint:

1. Update this document with its actual method, path, purpose, authentication requirements, status codes, and response format.
2. Add or update tests.
3. Update `PROGRESS.md` and `CHANGELOG.md` when appropriate.
4. Ensure the documentation accurately reflects implemented behavior rather than planned functionality.

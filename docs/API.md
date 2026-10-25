
# API Design

## 1. Purpose

This document defines the initial API conventions for the School Management System.

The frontend will communicate with the backend through HTTP APIs returning JSON.

## 2. API Versioning

- API routes will use the `/api/v1` prefix.
- Breaking API changes should be introduced through a planned versioning strategy.
- Existing API behavior must not be changed unexpectedly.

## 3. Naming Conventions

- Resource names should generally be plural.
- Resource paths should use lowercase letters.
- Use HTTP methods according to the operation being performed.

Examples:

- `GET /api/v1/students`
- `GET /api/v1/students/:id`
- `POST /api/v1/students`
- `PATCH /api/v1/students/:id`
- `DELETE /api/v1/students/:id`

These are illustrative conventions, not implemented endpoints.

## 4. HTTP Methods

- GET: Retrieve resources.
- POST: Create a resource or perform a defined action.
- PATCH: Update selected fields.
- PUT: Replace a resource when full replacement is required.
- DELETE: Delete a resource only when the business rules permit it.

## 5. JSON Format

- Request and response bodies should use JSON where appropriate.
- JSON field names should use `snake_case`.
- Dates should use the ISO 8601 format.
- Timestamps should include timezone information, preferably UTC.

Example:

    {
      "student_id": 123,
      "first_name": "Amit"
    }

## 6. Response Conventions

Successful responses should return an appropriate HTTP status code.

Common status codes:

- 200 OK: Request completed successfully.
- 201 Created: Resource created successfully.
- 204 No Content: Request succeeded without a response body.
- 400 Bad Request: Invalid request format or parameters.
- 401 Unauthorized: Authentication is missing or invalid.
- 403 Forbidden: Authenticated user lacks permission.
- 404 Not Found: Resource does not exist or must not be disclosed.
- 409 Conflict: Request conflicts with existing data.
- 422 Unprocessable Content: Request data fails validation.
- 429 Too Many Requests: Rate limit exceeded.
- 500 Internal Server Error: Unexpected server failure.

## 7. Error Responses

Errors should follow a consistent JSON structure.

Example:

    {
      "error": {
        "code": "VALIDATION_ERROR",
        "message": "The request data is invalid."
      }
    }

- Do not expose stack traces, database details, secrets or internal implementation details.
- Validation errors may include safe field-level details.
- Authentication errors should not reveal whether a particular account exists.

## 8. Validation

- Validate request bodies, path parameters and query parameters on the backend.
- Never rely only on frontend validation.
- Reject invalid identifiers, unsupported values and malformed requests.
- Enforce business rules before writing data to the database.

## 9. Authentication and Authorization

- Protected endpoints must require valid authentication.
- Authorization must be checked on every protected request.
- The backend must enforce role permissions and school-level data access.
- Never trust a client-provided school identifier or user role without server-side verification.
- Sensitive endpoints must apply appropriate rate limits and audit logging.

Detailed rules will be defined in the security documentation.

## 10. Pagination and Filtering

List endpoints should support pagination where appropriate.

- Use `page` and `page_size` as the initial pagination convention.
- Apply a reasonable maximum page size.
- Validate sorting and filtering parameters.
- Avoid returning an unlimited number of records by default.

Exact limits will be finalized during implementation and testing.

## 11. Database and Transaction Safety

- Use parameterized database queries.
- Return appropriate conflict errors for duplicate or conflicting operations.
- Use transactions when multiple database operations must succeed or fail together.
- Do not expose database implementation details in API responses.

## 12. API Documentation and Testing

- Document implemented endpoints, request fields, responses and authorization requirements.
- Test successful requests, invalid inputs and unauthorized access.
- Test cross-school access restrictions.
- Keep API documentation synchronized with implementation.

## 13. Current Status

Initial API conventions documented. Detailed endpoint specifications will be added as each module is designed and implemented.

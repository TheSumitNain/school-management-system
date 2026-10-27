# System Architecture

## 1. Purpose

This document describes the architecture of the School Management System, its current implementation, and the planned direction for future development.

The system is intended to serve multiple schools through a shared application platform.

### Target scale

- 25–30 schools.
- Approximately 400–500 students per school.
- Approximately 10,000–15,000 students in total.

These figures are planning targets, not verified capacity or load-test results.

## 2. Technology Stack

| Layer | Technology | Status |
|---|---|---|
| Frontend | React, TypeScript, Vite | Planned |
| Backend | Go, Gin | Initial foundation implemented |
| Database | PostgreSQL | Implemented locally |
| Database connectivity | pgx connection pool | Implemented |
| Database migrations | Go migration command and SQL migration files | Implemented |
| Password security | bcrypt password hashing and verification | Implemented |
| Authentication service | User creation and credential authentication | Implemented |
| JWT access tokens | Access-token generation and validation service | Implemented and tested |
| Login API | `POST /auth/login` | Implemented and manually tested |
| Refresh-token HTTP flow | Refresh-token issuance, rotation, and revocation | Pending |
| Authentication middleware | Protected-route authentication | Pending |
| Authorization | Role-based access control (RBAC) | Pending |
| Testing | Go unit tests and Go vet | Initial tests implemented |

## 3. High-Level Architecture

The planned system uses a React frontend, a Go/Gin API, and PostgreSQL.

```mermaid
flowchart TD
    U[School Users] --> F[React Frontend]
    F --> A[Go API - Gin]
    A --> H[Health and Readiness]
    A --> M[Authentication Handler]
    M --> S[Service Layer]
    S --> R[Repository Layer]
    R --> D[(PostgreSQL)]
    A -. Future protected routes .-> MW[Authentication Middleware]
    MW -. Future authorization checks .-> RBAC[RBAC and Tenant Isolation]
    RBAC -.-> S
```

This is a logical architecture diagram. It distinguishes implemented foundations from planned protected-route authorization. The complete frontend and all future business modules are not yet implemented.

## 4. Architectural Style

The backend follows a modular-monolith direction.

A modular monolith keeps the application in one deployable backend while separating responsibilities into packages and modules.

This is the initial approach because the current target scale does not, by itself, require a microservices architecture.

### Benefits

- Simpler deployment and local development.
- Easier debugging for a small development team.
- Clear boundaries between HTTP handlers, business logic, and database access.
- Ability to scale the application without introducing distributed-system complexity too early.

Modules can be separated into independent services later if measured operational needs justify that change.

## 5. Backend Responsibilities

The backend uses Go and Gin.

### API and server layer

Location: `backend/cmd/api/`

Responsibilities include:

- Starting the HTTP server.
- Loading application configuration.
- Establishing database connectivity.
- Registering routes.
- Configuring HTTP server timeouts.
- Handling graceful shutdown.
- Exposing health and readiness checks.
- Registering the implemented login endpoint.

### Configuration layer

Location: `backend/internal/config/`

Responsibilities include loading and validating application configuration.

The current configuration includes the application environment and port, JWT issuer, JWT expiry settings, and JWT secret validation.

Secrets and environment-specific values must not be hardcoded in source code. Local secrets must remain outside version control.

### Database layer

Location: `backend/internal/database/`

Responsibilities include PostgreSQL connection-pool initialization and database connectivity.

### Models

Location: `backend/internal/models/`

Responsibilities include application data structures, such as the current user model.

Models must not expose sensitive fields, such as password hashes, in API JSON responses.

### Repository layer

Location: `backend/internal/repository/`

Responsibilities include database queries and persistence operations.

Repositories should use parameterized SQL queries and return errors that the service layer can handle safely.

### Handler layer

Location: `backend/internal/handlers/`

Responsibilities include HTTP request parsing, request validation, calling the appropriate service, and returning HTTP responses.

The current authentication handler implements `POST /auth/login`. Successful responses return an access token and a limited user DTO rather than exposing the password hash.

### Service layer

Location: `backend/internal/services/`

Responsibilities include business rules and coordination between repositories.

The authentication service contains user-creation and credential-authentication logic, including school and user status checks.

The token service generates and validates JWT access tokens. These services provide an initial authentication foundation, not a complete production authentication and authorization system.

### Migrations

Location: `backend/migrations/`

Responsibilities include versioned database schema changes.

The migration command is located at `backend/cmd/migrate/`.

Current migrations establish the initial schools, users, and refresh-session schema. Consult `DATABASE.md` for the exact schema and migration history.

### Development seed command

Location: `backend/cmd/devseed/`

A development-only command creates or prepares a test school user for local development. It is guarded by the development environment and uses environment-provided credentials.

It must not be treated as a production user-provisioning mechanism. Its environment guard and deployment separation must be reviewed before production deployment.

## 6. Request Flow

### Current login flow

1. The client sends `POST /auth/login` with `school_id`, `username`, and `password`.
2. Gin routes the request to the authentication handler.
3. The handler validates the request structure.
4. The authentication service verifies the school, user status, and password.
5. The token service generates an access token.
6. The handler returns the token, token type, and limited user information.
7. Invalid credentials return a generic unauthorized response; malformed requests return a client error.

The current login endpoint and its supporting authentication and token services have been implemented and tested. This does not establish that every authentication-related security requirement has been completed.

### Intended protected business-request flow

1. The client sends an HTTP request with its access token.
2. Gin matches the request to a route.
3. Authentication middleware validates the token and establishes the trusted identity context.
4. Authorization checks whether the user may perform the requested operation.
5. Tenant-isolation checks ensure that the operation is limited to the authorized school.
6. The service layer applies business rules.
7. The repository executes the required database operations.
8. The API sends an appropriate status code and response.

Authentication middleware, RBAC, tenant-isolation enforcement, and protected business endpoints remain pending implementation.

## 7. Multi-School Architecture

### Initial strategy: shared database

The initial design uses a shared PostgreSQL database for multiple schools.

School-owned data must include a school relationship where appropriate. The backend must enforce the authorized school context in every relevant query and mutation.

A `school_id` field or database foreign key alone does not guarantee tenant isolation.

### Required tenant-isolation rules

- Every school-owned resource must be associated with the correct school.
- Authenticated requests must use a trusted server-side school context.
- Client-provided school identifiers must not grant access by themselves.
- Repository queries must restrict access to the authorized school.
- Related records must be checked for school ownership.
- Automated tests must verify that users cannot read or modify another school's data.

Tenant-isolation enforcement remains a required future implementation task. The current login request includes a school identifier for credential authentication; that identifier must not be treated as sufficient authorization for subsequent school-owned operations.

### Separate database per school

A separate database for each school is not the initial strategy.

It would add operational work for connection management, migrations, monitoring, backup, restoration, and reporting across schools.

Revisit this decision only if concrete requirements justify a different isolation model.

## 8. Database Design

PostgreSQL is the primary relational database.

The schema should use:

- Primary keys for record identity.
- Foreign keys for valid relationships.
- Unique constraints for business rules.
- Check constraints for supported values.
- Appropriate indexes for frequent queries.
- Transactions for operations that must succeed or fail together.
- Timestamps where creation and update history is needed.

Relationships should be represented through normalized tables unless a documented requirement justifies another approach.

JSON/JSONB may be appropriate for genuinely flexible attributes, but it should not replace relational tables for core entities that need referential integrity, joins, and consistent querying.

The exact schema and migration history are documented in `DATABASE.md`.

## 9. Authentication and Authorization

The current authentication foundation includes:

- A `schools` table and a `users` table associated with a school.
- Password hashing and verification using bcrypt.
- Service-level checks for school and user status.
- JWT access-token generation and validation.
- Application configuration for JWT issuer, expiry, and secret validation.
- An implemented `POST /auth/login` endpoint.
- A `refresh_sessions` table intended for refresh-session management.
- Unit tests for the token service and login handler.

The following are not yet complete:

- A public or administrative user-registration HTTP workflow.
- A complete refresh-token HTTP flow, including issuance, rotation, and revocation.
- Logout and access-token revocation endpoints.
- Authentication middleware for protected routes.
- A protected current-user endpoint such as `GET /auth/me`.
- RBAC and role enforcement.
- Complete tenant-isolation enforcement.
- Comprehensive security and integration testing.

These features must be implemented and tested before the system can be considered ready for real school users.

See `SECURITY.md` for security requirements.

## 10. Planned Functional Modules

The following are planned product areas, not a statement that their APIs or user interfaces already exist:

- School administration and configuration.
- User and role management.
- Students and parent relationships.
- Teachers and class assignments.
- Classes, sections, and subjects.
- Attendance.
- Timetable and academic scheduling.
- Student and parent dashboards.
- Transport and bus tracking.
- Notifications and reports.

Fee and payment functionality is outside the initial implementation scope unless explicitly added to the requirements later.

## 11. Reliability and Performance

The application currently has an initial HTTP server, PostgreSQL connection foundation, health and readiness endpoints, and initial authentication functionality.

Future production work should include:

- Query and index review.
- Pagination for large result sets.
- Appropriate connection-pool configuration.
- Request timeouts and graceful shutdown.
- Monitoring of errors, latency, database connections, and resource usage.
- Backup and restoration testing.
- Integration tests and migration tests.
- Load tests using realistic school data and representative workloads.

No fixed concurrency or performance capacity should be claimed without reproducible load-test results.

## 12. Deployment Direction

The intended production topology is:

- A React frontend hosted through an appropriate web-serving platform.
- A Go API service.
- A PostgreSQL database with restricted network access.
- HTTPS for external traffic.
- Environment-specific configuration and managed secrets.
- Automated testing and controlled database migrations.
- Monitoring, backups, and an operational recovery procedure.

The production deployment environment has not yet been finalized.

The development seed command must be excluded from production workflows or otherwise made unavailable to production deployments.

## 13. Architecture Decision Principles

Architecture decisions should prioritize:

1. Data isolation and security.
2. Correctness and maintainability.
3. Clear module and layer responsibilities.
4. Simplicity appropriate to the team and target scale.
5. Measurable performance.
6. Avoiding unnecessary infrastructure complexity.

Significant changes should be recorded in this document or a dedicated architecture decision record, and reflected in `PROGRESS.md` where appropriate.

## 14. Maintenance

Update this document when the technology stack, request flow, database strategy, module boundaries, or deployment approach changes.

Keep planned architecture separate from verified implementation. For current tasks and outstanding work, see `PROGRESS.md` and `HANDOFF.md`.

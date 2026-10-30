# System Architecture

## 1. Project Overview

The School Management System is a production-oriented, multi-school application designed to manage school operations through a centralized platform.

The system follows a **modular monolith architecture**. The initial implementation uses a shared PostgreSQL database and shared schema for all schools. School-level data isolation must be enforced by the backend.

### Initial scale target

- 25–30 schools
- Approximately 400–500 students per school
- Default academic classes from Class 1 to Class 12, customizable by each school
- Centralized platform administration
- Role-based access control (RBAC)

These are initial design targets, not verified production capacity figures.

## 2. Technology Stack

| Layer | Technology | Purpose |
|---|---|---|
| Frontend | React | User interface |
| Frontend language | TypeScript | Type safety |
| Frontend tooling | Vite | Development and build tooling |
| Backend | Go | API and business logic |
| HTTP framework | Gin | Routing and HTTP middleware |
| Database | PostgreSQL | Persistent application data |
| Database driver/pool | pgx / pgxpool | Database connectivity |
| Authentication | JWT access tokens | API authentication |
| Refresh sessions | Opaque refresh tokens | Session renewal and revocation |
| Password hashing | bcrypt | Password hash storage |
| Database migrations | SQL migration files | Versioned schema changes |
| Version control | Git and GitHub | Source control |

## 3. Architectural Style

The application uses a modular monolith.

The backend is deployed as one application, while responsibilities are separated into packages such as handlers, services, repositories, models, middleware, configuration, and database infrastructure.

```text
school-management-system/
├── backend/
│   ├── cmd/
│   │   └── api/
│   ├── internal/
│   │   ├── config/
│   │   ├── database/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   ├── models/
│   │   ├── repository/
│   │   └── services/
│   ├── migrations/
│   ├── .env.example
│   └── go.mod
├── frontend/
├── docs/
├── README.md
├── PROGRESS.md
└── CHANGELOG.md
```

This tree describes the intended organization. The actual repository contents remain the source of truth for which files and modules currently exist.

### Responsibilities

- **Handlers:** Validate HTTP input, call services, and format HTTP responses.
- **Services:** Implement business rules and coordinate operations.
- **Repositories:** Handle database queries and persistence.
- **Models:** Represent application data structures.
- **Middleware:** Apply request-level concerns, including access-token validation.
- **Configuration:** Load and validate application settings.
- **Database infrastructure:** Manage database connections and readiness checks.
- **Migrations:** Track database schema changes in version control.

Handlers should not contain complex business logic or perform unnecessary direct database operations.

## 4. High-Level Architecture

```text
┌───────────────────────────────┐
│        React Frontend         │
│       TypeScript + Vite       │
└──────────────┬────────────────┘
               │ HTTP / JSON
               ▼
┌───────────────────────────────┐
│         Go + Gin API          │
├───────────────────────────────┤
│ Routes and HTTP Handlers      │
│ Authentication Middleware     │
│ Services / Business Rules     │
│ Repositories / Data Access     │
└──────────────┬────────────────┘
               │
               ▼
┌───────────────────────────────┐
│          PostgreSQL           │
│ Shared Database and Schema    │
│ Schools, Users, Sessions      │
└───────────────────────────────┘
```

The diagram shows the architectural responsibilities. It does not imply that every planned module has already been implemented.

## 5. Multi-Tenant Database Strategy

The initial system uses:

- One PostgreSQL database
- One shared schema
- Multiple schools represented by school records
- School-specific users and future school-owned resources
- Backend-enforced school-level data isolation

A separate database for each school is not part of the initial architecture.

### Tenant isolation requirements

Every school-owned resource must be associated with the appropriate school, directly or through a verified relationship.

For protected requests, the backend must:

1. Authenticate the user.
2. Identify the authenticated user's school and applicable role.
3. Check the user's permissions for the requested operation.
4. Ensure that requested resources belong to the permitted school.
5. Reject unauthorized cross-school access.
6. Apply the same restrictions to reads, creates, updates, deletes, exports, and related operations.

The backend must not trust a client-supplied `school_id` as proof of authorization.

Tenant isolation and comprehensive RBAC enforcement remain implementation requirements until their code and tests are completed.

## 6. Authentication Architecture

The initial authentication foundation is implemented in the Go backend.

### Implemented components

- Username/password login scoped to a school ID
- Password hashing using bcrypt
- JWT access-token generation and validation
- Bearer-token authentication middleware
- Protected `GET /auth/me` endpoint
- Opaque refresh tokens generated using cryptographically secure randomness
- Refresh-token hashes stored in the database rather than raw refresh tokens
- Refresh-token rotation
- Family-wide refresh-token revocation on logout
- Refresh-token family revocation when a revoked token is replayed
- Shared PostgreSQL transaction advisory locking for refresh rotation and logout within a token family, plus per-user locking to serialize rotations across a user's token families
- Configurable refresh-token expiry, with a default of seven days and a supported range of 1–30 days

These features form the initial authentication foundation. They do not mean that all production security controls are complete.

### Authentication flow

```text
Login
  │
  ▼
Validate credentials and account status
  │
  ▼
Issue access token and refresh token
  │
  ├── Access token ──► Protected API requests
  │
  └── Refresh token ─► POST /auth/refresh
                           │
                           ▼
                     Rotate refresh token
                           │
                           ▼
                  Return new token pair
```

### Refresh-token security

Refresh tokens are opaque random values. The server stores a hash of each token and tracks its session family.

When a valid refresh token is used, the server rotates it. Reuse of a revoked token triggers revocation of active sessions in the same family. Rotation and logout acquire the same transaction-scoped PostgreSQL advisory lock derived from the family ID. Rotation also acquires a per-user transaction advisory lock, serializing concurrent rotations across that user's token families. Rotation performs a fresh row-locked session lookup after acquiring both locks.

Logout revokes all active refresh sessions in the supplied token family; an unknown token is a no-op and repeated logout is safe. Because access tokens are currently JWTs, revoking a refresh session does not automatically invalidate an already-issued access token. Immediate access-token revocation is a separate security requirement.

### Current authentication endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | API health check |
| GET | `/ready` | API and database readiness check |
| POST | `/auth/login` | Authenticate a user |
| POST | `/auth/refresh` | Rotate a refresh token |
| POST | `/auth/logout` | Revoke all active refresh sessions in the supplied token family |
| GET | `/auth/me` | Return authenticated user identifiers |

The exact response schemas and error behavior are documented in `docs/API.md`. In the current backend source, route registration is performed in `backend/cmd/api/main.go`; a separate routing package is not yet part of the implemented structure.

## 7. Authorization and RBAC

Authentication answers **who the user is**. Authorization answers **what that user may do**.

Planned roles include:

- Platform Admin
- School Admin
- Teacher
- Parent
- Future roles such as Clerk/Manager, Librarian, Gatekeeper, Helper, Bus Conductor, and others as requirements are finalized

The system is intended to support role-based default permissions and configurable individual permissions managed by authorized administrators.

### Required authorization behavior

- Platform-level permissions must be distinct from school-level permissions.
- School administrators must not access another school's data.
- Teachers must be restricted to their assigned responsibilities.
- Parents must only access children officially linked to their accounts.
- Permission changes must affect subsequent protected API requests, including requests made using an existing session.
- Sensitive operations must be checked on the backend.

**Status:** Full RBAC, permission management, and comprehensive authorization enforcement are pending. Authentication middleware alone is not a substitute for authorization.

## 8. Database and Migration Architecture

Database schema changes are maintained as versioned SQL migration files in `backend/migrations/`.

The current migration sequence is:

```text
000001_create_schools_table.up.sql
000001_create_schools_table.down.sql

000002_create_auth_tables.up.sql
000002_create_auth_tables.down.sql

000003_add_refresh_token_families.up.sql
000003_add_refresh_token_families.down.sql
```

The migrations establish the initial schools, authentication tables, and refresh-token family metadata.

Migration files are the authoritative source for the exact schema, column definitions, constraints, and indexes.

### Database principles

- Use migrations for schema changes.
- Keep credentials and environment-specific secrets out of source control.
- Use parameterized database queries.
- Apply database constraints where appropriate.
- Add indexes based on query patterns and measured performance.
- Use transactions for operations that require atomicity.
- Validate database readiness separately from basic API health.
- Verify migrations and rollback behavior in an appropriate test environment.

Opt-in PostgreSQL repository integration tests and concurrency-focused refresh-session tests exist. Broader transaction-failure and concurrency interleaving coverage remains outstanding.

## 9. School Onboarding

The intended onboarding workflow is:

1. Platform Admin creates or approves a school.
2. The system provisions the School Admin account or invitation.
3. The School Admin verifies the invitation and establishes a password.
4. The School Admin configures school details and academic structure.
5. Authorized school staff create and manage teacher accounts.
6. Additional users receive only the permissions appropriate to their responsibilities.

Public Platform Admin registration is not intended to be available.

The complete onboarding workflow, invitation security, mandatory administrator MFA, and related audit controls are not yet fully implemented.

## 10. Planned Functional Modules

The architecture is designed to accommodate these modules incrementally:

1. Authentication and session management
2. RBAC and authorization
3. School onboarding and settings
4. Academic years, classes, and sections
5. Student profiles and enrollment
6. Teacher profiles and assignments
7. Attendance
8. Timetable
9. Homework and assignments
10. Examinations and report cards
11. Notices and announcements
12. Parent accounts and linked children
13. Fees and payments
14. Additional operational modules based on finalized requirements

These are planned capabilities, not claims of current implementation.

## 11. API Design Principles

The backend exposes HTTP APIs using JSON.

API handlers should:

- Validate request bodies and path/query parameters.
- Return consistent response structures.
- Use appropriate HTTP status codes.
- Avoid exposing internal errors, database details, or secrets.
- Apply authentication and authorization where required.
- Avoid trusting user-controlled identifiers for access decisions.
- Keep API documentation synchronized with actual handler behavior.

API versioning, such as `/api/v1`, must not be assumed to exist until it is explicitly implemented.

## 12. Configuration and Environment Management

Application configuration is loaded from environment variables.

`backend/.env.example` documents the expected development configuration. Actual local secrets must remain in an untracked environment file or a suitable secret-management system.

Important configuration includes:

- `APP_ENV`
- `APP_PORT`
- Database connection settings
- Database SSL mode
- Refresh-token expiry configuration
- Authentication signing secret and other secrets required by the current implementation

Production database connections must use verified TLS settings. Development settings must not be copied blindly into production.

The application should fail safely when required production configuration is missing or invalid.

## 13. Operational Reliability

The API foundation includes:

- Configurable application environment
- A default API port of `8081`
- HTTP server timeouts
- Graceful server shutdown
- Disabled Gin trusted proxies by default
- Database connection pooling
- A health endpoint
- A readiness endpoint that checks database connectivity

The readiness endpoint uses a bounded database ping. These measures provide a foundation for reliability, but they do not constitute a complete production deployment, monitoring, backup, or disaster-recovery setup.

## 14. Security Architecture and Outstanding Controls

The following controls are planned or still require completion and verification:

- Complete RBAC and permission enforcement
- Cross-school authorization and isolation tests
- Mandatory MFA for Platform Admin and School Admin
- Login rate limiting and failed-login lockout
- Comprehensive security-event and audit logging
- Strong administrator password policy, including compromised-password checks
- Secure password-reset and account-recovery workflows
- Parent mobile OTP authentication
- Immediate access-token invalidation strategy where required
- Broader PostgreSQL repository integration coverage beyond the existing refresh-session lifecycle tests
- Additional concurrent refresh-token rotation and replay edge cases
- Independent logout and session-revocation verification
- Production secret management, monitoring, backup, and recovery procedures

Do not treat the current authentication foundation as production-ready until the required controls have been implemented, tested, and reviewed.

## 15. Testing Strategy

Testing should cover multiple layers:

- Unit tests for services and validation
- Handler tests for HTTP status codes and response bodies
- Repository integration tests against PostgreSQL for the refresh-session lifecycle (opt-in; broader repository coverage remains pending)
- Migration and rollback verification
- Authentication and refresh-session lifecycle tests
- Concurrent token rotation and replay tests
- RBAC and permission-change tests
- Cross-school access-denial tests
- Frontend tests
- End-to-end tests for important user journeys
- Load and performance tests against the intended school and student scale

Current test commands are documented in `docs/TESTING.md`.

Previously reported successful backend checks include `go test -count=1 ./...` and `go vet ./...` after the refresh-token implementation. The PostgreSQL repository integration tests require explicit opt-in and a dedicated test database; do not infer that they ran during an ordinary `go test ./...` invocation. These results do not prove that authorization, full tenant-isolation, end-to-end, or all concurrency scenarios have passed.

## 16. Development and Documentation Principles

- Build on the existing repository; do not recreate the project from scratch.
- Implement one well-defined feature at a time.
- Keep handlers, services, repositories, and models appropriately separated.
- Add tests for new behavior and security-sensitive changes.
- Update API, database, security, testing, and progress documentation as implementation changes.
- Keep migration files under version control.
- Review the diff before committing.
- Never commit secrets, local environment files, or production credentials.
- Verify the application and relevant tests before marking a feature complete.

## 17. Current Project Status

The project has an initial Go/Gin API foundation, PostgreSQL connectivity, initial database migrations, and a working foundation for authentication and refresh-session management.

The next major implementation stages are:

1. Continue refresh-session verification for database failures and additional refresh/logout interleavings; independently verify the HTTP logout flow.
2. Implement RBAC and enforce authorization on protected endpoints.
3. Implement and test school-level tenant isolation.
4. Complete secure school onboarding and administrator MFA.
5. Continue into the academic and student-management modules.

Refer to `PROGRESS.md`, `docs/HANDOFF.md`, and `docs/TESTING.md` for the latest recorded implementation status and continuation instructions.

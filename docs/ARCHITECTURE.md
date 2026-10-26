
# System Architecture

## 1. Purpose

Define the initial architecture for a commercial, multi-tenant School Management System.

## 2. Technology Stack

- Frontend: React, TypeScript and Vite
- Backend: Go and Gin
- Database: PostgreSQL
- Version control: Git
- Remote repository: Private GitHub repository

## 3. Architecture Style

Use a modular monolith initially.

The application will be deployed as a manageable frontend and backend rather than starting with microservices. Internal modules must have clear responsibilities and boundaries.

Potential backend modules:
- Configuration
- Database
- Authentication
- Authorization
- Schools
- Users
- Audit logging

Modules will be introduced when needed, not all implemented immediately.

## 4. Multi-Tenancy

The application must support multiple schools.

Initial design direction:
- Shared PostgreSQL database.
- School-owned records are associated with a school identifier where applicable.
- Backend authorization enforces school-level isolation.
- Client-provided school IDs must never be trusted without validation.
- Platform-level operations are separated from school-level operations.

The final schema and tenant-enforcement approach must be reviewed before implementing business modules.

## 5. Backend Principles

- Separate HTTP handlers, business logic and data access where appropriate.
- Keep functions focused and readable.
- Document exported Go identifiers and explain non-obvious security decisions.
- Handle errors explicitly; do not silently ignore failures.
- Validate input at API boundaries.
- Use context-aware database operations.
- Never expose internal errors, credentials or secrets to API clients.
- Keep configuration outside source code.

## 6. Frontend Principles

- Use TypeScript for application code.
- Separate reusable UI, pages, API communication and feature logic.
- Keep authentication state and error handling consistent.
- Never rely on frontend checks as the security boundary.
- Do not store server secrets in frontend environment variables.

## 7. Database Principles

- Use version-controlled schema migrations.
- Enforce data integrity with appropriate database constraints.
- Use parameterized queries.
- Plan indexes around actual query patterns.
- Never use production student data for ordinary local testing.
- Backups and restore procedures must be addressed before production deployment.

## 8. Quality and Testing

- Add tests with each implemented feature.
- Test authorization boundaries and cross-school access.
- Run formatting, tests and relevant static checks before committing.
- Document manual verification when automation is not yet available.
- Do not mark a feature complete until its expected behavior is verified.

## 9. Security

- Keep credentials and secrets out of Git.
- Enforce authentication and authorization on protected backend routes.
- Apply finalized token, MFA, password and session policies at the appropriate implementation stages.
- Log security-relevant events without logging passwords, tokens or MFA codes.
- Review dependencies and deployment configuration before release.

## 10. Documentation and Change Management

Maintain:
- README.md
- PROGRESS.md
- docs/ARCHITECTURE.md
- docs/REQUIREMENTS.md
- docs/DATABASE.md
- docs/API.md
- docs/SECURITY.md
- docs/TESTING.md
- CHANGELOG.md

Update relevant documentation when behavior or decisions change.

## 11. Deployment Readiness

Before production release, verify:
- Environment configuration and secret management
- Database migrations and backups
- Automated test results
- Authentication and tenant isolation
- Logging and monitoring
- Graceful shutdown and health checks
- HTTPS and deployment security
- Recovery and rollback procedures

## 12. Current Status

### Implemented Foundation

- Go + Gin backend scaffold.
- PostgreSQL connection setup.
- Initial `schools` database migration.
- `GET /health` and `GET /ready` endpoints.
- HTTP server timeouts and graceful shutdown handling.
- Automated tests for health and database readiness behavior.

### Architecture Decisions

- Modular monolith as the initial application architecture.
- React + TypeScript + Vite frontend.
- Go + Gin backend.
- PostgreSQL as the database.
- Shared database as the initial multi-tenant direction.

### Still To Be Finalized or Implemented

- Detailed module boundaries and dependencies.
- School-level tenant authorization and isolation.
- Authentication and role-based access control.
- Business modules and their database relationships.
- Production deployment topology, HTTPS, monitoring, backups and recovery procedures.

The architecture remains a work in progress. Planned decisions must be validated before they are treated as implemented behavior.
# Product Requirements

## 1. Product Overview

Build a commercial, multi-tenant School Management System that allows multiple schools to manage their academic and administrative activities through one platform.

## 2. Initial Target Users

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

## 4. Initial User Roles

- Platform Admin
- School Admin
- Teacher
- Parent

Student, Driver and additional Staff access may be introduced in later phases.

These are planned product roles. Role storage, role-based access control and permission enforcement must be implemented and tested before these roles are treated as operational.

## 5. Initial Product Scope

The system will be developed incrementally.

### Foundation

- Application configuration
- Database connectivity
- Health and readiness endpoints
- Authentication
- Role-based authorization
- School-level data isolation
- Input validation and consistent error responses
- Automated testing

### School Management

- School creation and activation
- User account management
- Class and section management
- Academic year management

### Student Management

- Student admission and profile
- Student enrollment
- Class and section assignment
- Historical enrollment records

### Academic Management

- Subject management
- Teacher assignment
- Class and subject assignments
- Timetable management

### Attendance Management

- Daily attendance
- Attendance summaries
- Attendance correction with authorization
- Attendance history

### Parent Features

- Parent account management
- Verified parent–student relationships
- Child profile and attendance visibility
- Notices and timetable access

## 6. Excluded From Initial Scope

The following features are not part of the initial implementation:

- Fee collection and payment gateway
- Live bus tracking
- Native Android or iOS applications
- Advanced analytics and AI features
- Microservices architecture

These may be considered in future releases.

## 7. Core Business Rules

- A school must not access another school's private records.
- Users must only access features and records permitted by their role.
- Protected operations must be authorized by the backend.
- Important data changes must preserve data integrity.
- Student and enrollment history must be retained according to approved retention rules.
- Sensitive operations must be auditable.
- The frontend must not be treated as a security boundary.
- School and user status must be checked where required for authentication and authorization.
- Cross-school access must be prevented by backend-enforced authorization and appropriate database queries.

## 8. Development Approach

- Implement one module at a time.
- Document requirements before implementing each major module.
- Use version-controlled database migrations.
- Write and run tests for implemented features.
- Verify school-level isolation and authorization.
- Update project documentation and progress after meaningful changes.
- Distinguish implemented functionality from requirements that are still pending.
- Do not claim production readiness based solely on passing unit tests.

## 9. Open Decisions

The following details require further planning before the relevant modules are implemented:

- Final database schema and relationships
- Academic year transition rules
- Admission number format
- Teacher account activation and recovery method
- Parent OTP provider
- Attendance correction and retention policy
- Deployment, backup and disaster recovery strategy
- Final role-permission matrix
- School onboarding and deactivation workflow
- Audit-log retention and access policy

## 10. Current Status

### Documented

- Initial product scope and target users.
- Technology stack and initial user roles.
- Core school, student, academic, attendance and parent features.
- Features excluded from the initial scope.
- Core business rules and open planning decisions.

### Implemented Foundation

- Go + Gin backend scaffold.
- Environment-based application configuration.
- PostgreSQL connection pool.
- Initial `schools` table migration.
- `users` and `refresh_sessions` database migration.
- API health and database-readiness endpoints.
- HTTP server timeouts and graceful shutdown handling.
- User and school repository operations.
- bcrypt password hashing and verification.
- Initial authentication service.
- JWT access-token generation and validation.
- Initial `POST /auth/login` endpoint.
- Unit tests for implemented configuration, password, authentication, token and login-handler behavior.

These implementations form an initial foundation; they do not mean that the complete authentication or authorization requirements have been met.

### Not Yet Implemented or Fully Verified

- Complete refresh-token issuance, rotation and session lifecycle.
- HTTP logout endpoint and complete session-revocation workflow.
- Authentication middleware and protected profile endpoint.
- Public registration and production school onboarding.
- Role storage, RBAC and permission enforcement.
- School-level tenant authorization and automated cross-school isolation tests.
- Complete school and user management workflows.
- Student admission and enrollment.
- Academic, attendance and parent-facing modules.
- Frontend dashboards and user workflows.
- PostgreSQL repository integration tests and migration tests.
- Production deployment and operational safeguards.

### Capacity Note

The initial target of 25–30 schools and approximately 10,000–15,000 students is a planning objective, not a tested capacity guarantee. Load testing, security testing and production validation will be required before making capacity commitments.

Requirements will be refined before each major module is implemented. The implementation status should be updated only after the relevant code and tests have been reviewed.

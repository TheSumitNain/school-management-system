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

## 4. Initial User Roles

- Platform Admin
- School Admin
- Teacher
- Parent

Student, Driver and additional Staff access may be introduced in later phases.

## 5. Initial Product Scope

The system will be developed incrementally.

### Foundation

- Application configuration
- Database connectivity
- Health-check endpoint
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

## 8. Development Approach

- Implement one module at a time.
- Document requirements before implementing each major module.
- Use version-controlled database migrations.
- Write and run tests for implemented features.
- Verify school-level isolation and authorization.
- Update project documentation and progress after meaningful changes.

## 9. Open Decisions

The following details require further planning before the relevant modules are implemented:

- Final database schema and relationships
- Academic year transition rules
- Admission number format
- Teacher account activation and recovery method
- Parent OTP provider
- Attendance correction and retention policy
- Deployment, backup and disaster recovery strategy

## 10. Current Status

Initial product scope documented. Detailed requirements will be refined before each module is implemented.
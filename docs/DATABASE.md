# Database Design

## 1. Database Technology

- Database: PostgreSQL
- Initial approach: Shared database for multiple schools
- Schema changes must be managed through version-controlled migrations.

## 2. Multi-School Data Isolation

The database must support multiple schools within the same application.

- Each school will have a unique identifier.
- School-owned records must be associated with the correct school wherever applicable.
- Backend queries and authorization must prevent access to another school's data.
- Client-provided school identifiers must not be trusted without authorization checks.
- Platform-level records must be distinguishable from school-level records.

The final tenant-isolation strategy must be reviewed before business modules are implemented.

## 3. Data Integrity

- Use primary keys to identify records.
- Use foreign keys to enforce valid relationships.
- Use unique constraints where duplicate values are not allowed.
- Use NOT NULL constraints for required fields.
- Use transactions for operations that must succeed or fail together.
- Use appropriate indexes based on actual query patterns.

## 4. Relationship Design

Relationships must represent real school business rules.

Examples:
- A school can have multiple users.
- A school can have multiple classes and sections.
- A student can have enrollment records across academic years.
- Teachers can be assigned to specific classes and subjects.
- A parent can be linked to multiple students.
- A student can have multiple verified guardians.

The exact cardinality, constraints and table structure will be documented before implementing each relevant module.

## 5. Student and Enrollment History

Student identity and academic enrollment should be modeled separately.

- Student records represent the student.
- Enrollment records represent the student's placement in a particular academic year, class and section.
- Historical enrollment must not be overwritten when a student moves to another class or academic year.
- The system must preserve historical data according to the approved retention policy.

## 6. Subject and Teacher Assignments

- Subjects may vary by class and academic year.
- Teacher assignments must represent the classes and subjects a teacher is authorized to teach.
- The design must support different subject combinations for different academic requirements.
- Flexible JSON fields must not replace relational tables when data needs reliable relationships, constraints or querying.

## 7. Migrations

- Every schema change must be version-controlled.
- Existing migrations must not be silently edited after they have been applied to shared environments.
- Migrations must have a documented and tested application process.
- Rollback behavior must be considered for each change.

## 8. Security and Privacy

- Never commit database passwords or production credentials.
- Use parameterized queries.
- Grant the application database account only the permissions it needs.
- Avoid storing unnecessary personal information.
- Do not use real student data in ordinary development or testing.
- Backups must be protected and restoration must be tested before production use.

## 9. Decisions Pending

- Final table list and column definitions
- Primary and foreign key strategy
- Tenant-isolation enforcement details
- Academic year and enrollment rules
- Soft deletion versus permanent deletion
- Audit history and retention periods
- Backup and restore procedures
- Whether PostgreSQL Row-Level Security will be used in addition to backend authorization

## 10. Current Status

### Implemented

- PostgreSQL connection configured and verified.
- Version-controlled migration tooling configured.
- Initial `schools` table migration created and applied.
- `schools` table columns, primary key, unique constraint and status CHECK constraint verified.
- Migration version `1` verified with `dirty = false`.

### Pending

- Design and implement remaining business tables.
- Finalize and implement school-level tenant isolation.
- Add automated database integration tests.
- Establish and test production backup and restoration procedures.

The current database schema is an initial foundation. It does not yet represent the complete School Management System database.

## 11. Implemented Database Objects

### 11.1 Schools Table

The first database migration creates the `schools` table.

Migration files:

- `backend/migrations/000001_create_schools_table.up.sql`
- `backend/migrations/000001_create_schools_table.down.sql`

Current columns:

| Column | Type | Constraints / Default |
|---|---|---|
| `id` | BIGINT | Generated identity, primary key |
| `name` | VARCHAR(150) | NOT NULL |
| `code` | VARCHAR(50) | NOT NULL, UNIQUE |
| `status` | VARCHAR(20) | NOT NULL, default `pending_setup` |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

Allowed status values:

- `pending_setup`
- `active`
- `suspended`

The database enforces these allowed values through a CHECK constraint.

**Important:** The `updated_at` default sets the initial timestamp. It does not automatically update the value when a row changes. Automatic timestamp updates must be implemented explicitly in application code or through a database trigger.

### 11.2 Migration Tracking

Database migrations are managed using `golang-migrate`.

The migration runner is located at:

`backend/cmd/migrate/main.go`

Run it from the backend directory:

```cmd
cd /d D:\school-management-system\backend
go run ./cmd/migrate
```

The migration tracking table is `schema_migrations`. The initial migration was verified at version `1` with `dirty = false`.

Do not run destructive rollback migrations against production without an approved backup and recovery plan.

## 12. Current Implementation Status

- [x] PostgreSQL connection established and verified
- [x] Migration tooling configured
- [x] Initial `schools` table migration created and applied
- [x] Table columns and constraints verified
- [ ] Remaining business tables designed and implemented
- [ ] Tenant-isolation strategy implemented and tested
- [ ] Automated database integration tests
- [ ] Production backup and restoration procedures tested

The `schools` table is the initial database foundation, not the completed school-management schema.

## 13. Next Database Design Tasks

Before implementing student-related tables, define and review:

1. School users and role relationships
2. Academic years
3. Classes and sections
4. Student identity and enrollment history
5. Subjects and teacher assignments
6. Parent/guardian-to-student relationships
7. Required foreign keys, unique constraints and indexes
8. School-level tenant isolation and authorization rules

Each new schema change must be added as a new version-controlled migration.
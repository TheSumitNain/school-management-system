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

Initial database design principles documented. The final relational schema will be developed incrementally after the business rules are reviewed.
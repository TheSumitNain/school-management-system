# Student Management — Domain and Database Design

**Status:** Approved design baseline; implementation not yet complete  
**Last reconciled:** 2026-11-13  
**Scope:** Phase 1 School Admin student admission, profiles, academic-year enrollment, curriculum selection and related school workflows.

This document defines the intended domain model and safe implementation sequence. It is not evidence that the schema, API or UI has already been implemented. The ordered migration chain and automated tests remain authoritative for actual implementation.

## 1. Product decisions

The following decisions are approved:

- Admission uses one guided, combined workflow. Student identity and first academic-year enrollment are created together in one transaction.
- Student profile information is organized into independently editable sections, not one long mandatory form.
- A student may be saved with the minimum required identity details. Other profile sections can remain incomplete and be completed later.
- A student has one permanent school-scoped identity. Each academic year has its own enrollment record and its own admission number.
- A returning student is not created again. A new academic year creates a new enrollment for the existing student.
- Admission number is an attribute of an academic-year enrollment, not the permanent student identity.
- Historical enrollments, attendance, guardian relationships and student IDs must be preserved.
- Parent OTP, SMS delivery, parent login and parent-facing applications remain Phase 2 and are not part of this implementation.

## 2. Audit findings in the current implementation

The reviewed `work` branch at `d38ae1ecc0f47384aabdcee8816319d39df1df9e` contains a useful foundation, but it does not yet implement the approved student model.

| Area | Existing state | Required evolution |
|---|---|---|
| Student identity | V11 `students` stores `admission_number` and `full_name`; admission number is unique per school | Keep stable student identity; phase admission number out of the permanent identity |
| Guardian relationships | V11 `guardians` and `student_guardians` with same-school composite foreign keys | Reuse; add profile workflow and field-level permissions without enabling parent authentication |
| Academic year/class/section | V17 school-scoped years, classes and year-specific sections | Preserve existing model and constraints |
| Enrollment | V17 `student_enrollments` is unique per student/year and preserves class/section snapshots; it has no admission number | Add year-specific admission number/date and safe legacy transition |
| Promotion | Repository creates destination enrollment and closes source enrollment transactionally | Extend to capture destination admission number; preserve source row |
| Attendance | V20 records reference enrollment; repository currently reads `students.admission_number` | Read admission number from the matching enrollment |
| Subjects | School-level `subjects`; teacher assignments currently validate active subject/section/year | Add class/year curriculum and student enrollment subject selection |
| Student profile | Repository exposes only name, admission number and status | Add independently stored profile sections |
| Addresses/locations | No dedicated student address/location hierarchy found in reviewed migrations | Add normalized address and reusable hierarchical location model |
| Medical/physical/documents | No dedicated student profile tables found in reviewed migrations | Add separate, permission-restricted profile sections |

The current student list search and attendance projection also read admission number from `students`. All such readers and writers must be migrated together; changing only the table definition would break existing behavior.

## 3. Target domain model

### 3.1 Permanent student identity

`students` remains the stable identity for one student within one school. It must not be duplicated for each academic year.

Target responsibilities:

- Stable primary key and immutable `school_id`.
- Minimum identity fields required to create the student.
- Name and core identity fields that do not change merely because the student is promoted.
- Profile completeness/status metadata where useful.
- Created/updated timestamps and audit linkage.

Do not use student name, date of birth, Aadhaar or another personal attribute as a unique identity key. Siblings and unrelated students can share names or dates of birth. Duplicate detection should be a review workflow using carefully chosen matching signals, not an unsafe database uniqueness rule.

Sensitive identity fields should be collected only when there is a defined school need, permission model, retention rule and secure handling path. Aadhaar must not become a required field by default.

### 3.2 Academic-year enrollment

`student_enrollments` is the source of truth for each student's admission to a particular academic year.

Target fields include:

- `school_id`, `student_id`, `academic_year_id`.
- `admission_number` and `admission_date` for that academic year.
- `class_id`, `section_id`, optional `roll_number`.
- Existing class/section name snapshots, enrollment status, enrolled/ended dates and audit timestamps.

Preserve the existing same-school composite foreign keys and the one-enrollment-per-student-per-academic-year rule unless a later approved business rule requires a different model. Admission numbers must be unique within a school and academic year after trimming and case folding. Roll number remains unique within the applicable active section according to the existing constraint.

Admission number and academic placement must not be rewritten on a closed historical enrollment. A promotion or next-year admission creates a new enrollment. The source enrollment is closed through the existing one-way lifecycle transition.

### 3.3 Profile sections

Keep independently maintained profile sections separate from the enrollment record:

- Basic identity: name, date of birth, gender and approved optional demographic fields.
- Address and communication: current and permanent address, phone/email where appropriate, and same-as-current behavior.
- Parents/guardians: reuse the existing guardian and relationship model; support multiple guardians and relationship/contact flags.
- Medical details: separate restricted record; capture only approved fields and expose only to authorized roles.
- Physical details: separate profile section with explicit field definitions and access rules.
- Documents: metadata, document type, issue/expiry dates where relevant, verification status and an opaque secure-storage reference. Do not store public file URLs or filesystem paths in API responses.

A section may be absent or incomplete. Its absence must not prevent basic student creation or ordinary student directory use.

### 3.4 Address and location hierarchy

Use normalized location records with a parent relationship and a controlled location type, allowing a hierarchy such as State → District → Block/Tehsil → Village/City. Do not hard-code a fixed list of Indian states or assume every deployment uses the same geography.

The design must support:

- School-owned location additions where canonical reference data is unavailable.
- Parent/child validation and duplicate prevention within the appropriate owner and parent scope.
- Active/inactive status; referenced locations are deactivated, never hard-deleted.
- Searchable, dependent selectors with server-side validation.
- Free-text house/building, street, locality/mohalla, landmark and PIN/postal code fields.
- Separate current and permanent addresses, with an explicit same-as-current option.
- School-scoped reporting by village/city.

Canonical shared reference data may be introduced later only with a reliable source, update/maintenance strategy and explicit ownership rules. Do not seed a guessed or incomplete geography catalogue.

### 3.5 Curriculum and student subject selection

The existing `subjects` table remains the school subject catalogue. Add a year/class curriculum layer rather than treating every school subject as valid for every class.

The target model should represent:

- A curriculum definition scoped to school, academic year and class.
- Curriculum subject rows linked to the school subject catalogue.
- Compulsory versus optional subjects.
- Optional subject groups with explicit selection rules (for example, choose one from a group).
- Student subject selections linked to the specific `student_enrollments` row.

Teacher subject assignments must be checked against the applicable class/year curriculum, not merely against an active subject and section. Historical student selections must remain unchanged if a future curriculum is edited. Changes to a curriculum should be versioned or represented by new effective records rather than silently rewriting past selections.

The exact rules for optional groups (minimum/maximum choices, mutually exclusive choices, and whether selection can change mid-year) must be explicit in the domain service and covered by database/application tests before enabling the UI.

## 4. Combined admission and profile workflow

### 4.1 Initial admission

1. Open the guided admission workflow.
2. Enter the minimum basic identity details.
3. Select the academic year, class and section; capture the admission number/date and optional roll number.
4. Optionally complete address, guardian, medical, physical and document sections.
5. Review and submit.

The backend creates the student and first enrollment in one database transaction. If either insert or required audit write fails, the transaction rolls back. The UI must not create a permanent student first and then leave the user to discover that the required initial enrollment failed.

The minimum required fields should be small and documented. Optional sections can be skipped and completed from the student's profile later.

### 4.2 Returning student / new academic year

1. Find the existing student using identity/profile search and prior enrollment history.
2. Select the destination academic year, class and section.
3. assign a new admission number for that academic year and capture its admission date.
4. Create a new enrollment; do not create a second student identity.
5. Preserve prior enrollment, attendance and subject-selection history.

Promotion and transfer are distinct business actions. Promotion creates the next-year enrollment and closes the source enrollment using the approved promotion lifecycle. A within-year section transfer must be explicitly designed against the existing immutable enrollment identity rule; do not mutate a historical enrollment's section in place. If transfer requires a replacement enrollment or a separate placement-history model, define its attendance/date semantics before implementation.

### 4.3 Student profile

Use a profile page with clearly named sections and separate edit actions. Show academic history as a year-by-year timeline/list, including each year's admission number, class, section, roll number and status. Clearly distinguish incomplete profile sections from enrollment status.

The directory should support year-scoped admission-number search and filters, alongside student name/status search. A student without a current-year enrollment must not be silently represented as enrolled in the current year.

## 5. Authorization, privacy and audit

- Every query and mutation is school-scoped at the repository/database boundary.
- Keep live backend RBAC checks; frontend permission checks only control visibility.
- Use least-privilege permissions for basic identity, sensitive student profile, guardian contacts, medical information and document access. Do not assume that one broad student permission is sufficient for every sensitive section.
- Teacher roster responses remain assignment-scoped and must not expose medical, guardian-contact, document or other sensitive profile fields.
- Medical and document access must be limited to explicitly authorized roles and audited.
- Audit admission creation, enrollment creation, admission-number changes where allowed, profile-section changes, guardian relationship changes, document operations, promotion, transfer and archive actions. Avoid copying unnecessary sensitive values into audit metadata.
- Do not enable parent login, OTP, SMS or parent-facing access as part of this work.

## 6. Migration and legacy-data transition

The migration must be additive and staged. Do not drop or rename the existing `students.admission_number` in the first migration.

### Stage A — Expand

1. Add enrollment-level `admission_number` and `admission_date` fields.
2. Initially allow historical rows to remain without a year-specific number where the old data cannot prove the correct mapping.
3. Add a normalized unique partial index scoped by `school_id` and `academic_year_id` for non-null enrollment admission numbers.
4. Add database/application guards so every newly created enrollment requires a valid non-empty admission number and date, even while legacy historical rows may remain null.
5. Keep the old student-level value as a deprecated legacy value during compatibility rollout.

### Stage B — Reconcile

1. Inventory every student and enrollment, including students without an enrollment.
2. Backfill a legacy admission number only where the intended academic-year enrollment is unambiguous and the mapping is supported by available records.
3. Do not infer a year from today's current academic year, a student's creation timestamp or a guessed convention.
4. Preserve unmatched/ambiguous legacy values for manual reconciliation; do not fabricate historical admission numbers.
5. Record counts and reconciliation outcomes, and test duplicate collisions before enforcing uniqueness.

### Stage C — Switch all readers and writers

Update student create/list/search, enrollment create/list/promote, attendance projections, API DTOs, frontend forms, directory filters, tests and documentation. New admission numbers are written only to enrollments. Existing API consumers must be migrated deliberately; do not leave two independently writable sources of truth.

### Stage D — Contract

Only after repository-wide search, data reconciliation, compatibility checks and production-readiness review:

- Remove deprecated student-level admission-number reads/writes.
- Drop the old column and old school-wide unique index in a separate reviewed migration.
- Retain any required legacy mapping/audit evidence according to the data-retention policy.

Do not combine expand, data reconciliation, application cutover and destructive contract into one migration.

## 7. API/service boundaries

The exact route naming may follow existing router conventions, but the service contract should distinguish:

- Combined initial admission (student + first enrollment transaction).
- Student identity/profile read and update.
- Independent profile-section read/update operations.
- Student directory search with academic-year context.
- Enrollment history and year-scoped enrollment operations.
- Promotion and transfer commands with explicit source/destination context.
- Curriculum management and enrollment subject selection.

Do not expose direct database-shaped updates that allow a client to change school ownership, enrollment identity, historical class/section snapshots or closed enrollment records. Validate all referenced IDs against the authenticated school and selected academic year.

## 8. Required test and acceptance gates

### Database and repository

- Fresh migration chain succeeds on a disposable PostgreSQL database.
- Existing data remains intact through the expand migration.
- Same admission number may be reused in different academic years but not twice within one school/year.
- Case/whitespace variants collide under normalized uniqueness.
- New enrollment cannot be created without a valid admission number/date.
- Student and first enrollment creation is atomic, including audit failure rollback.
- Duplicate student/year enrollment remains rejected.
- Cross-school student, year, class, section, subject, guardian and location references are rejected.
- Historical enrollment identity and closed history remain immutable.
- Promotion creates a new year record without changing source-year attendance or subject selections.
- Legacy null/ambiguous admission-number rows remain queryable and are reported for reconciliation.

### Authorization and privacy

- School A cannot read or mutate School B records by changing IDs.
- Basic roster-only Teachers receive only approved roster fields.
- Teacher subject/class scope is enforced server-side.
- Medical, guardian contacts and documents require their explicit permissions.
- Sensitive profile reads and writes produce appropriate audit events.

### Frontend and end-to-end

- A new student can be saved with minimum details and profile completion can continue later.
- Combined admission either creates both student and first enrollment or creates neither.
- Returning student admission creates a new enrollment, not a duplicate student.
- Admission number is displayed and searched in the selected academic-year context.
- Current-year directory and historical enrollment views are clearly distinguished.
- Every invalid field shows its validation message directly beneath that field.
- Forms preserve safe entered values after recoverable validation/server errors.
- Responsive, keyboard-accessible profile sections and dependent location selectors are verified in supported browser engines.
- Verify the complete flow through browser → API → PostgreSQL → RBAC, not by frontend appearance alone.

## 9. Explicit non-goals for this workstream

- Parent OTP, SMS delivery, parent login and parent app.
- Payment gateway or fee collection.
- Native parent application.
- Live bus GPS.
- Advanced AI analytics.
- Speculative student transport/app-permission modules without a Phase 1 workflow.

## 10. Implementation order

1. Complete schema and code-path inventory, including every read/write of `students.admission_number`.
2. Finalize the legacy-data reconciliation query/report and migration constraints.
3. Add and test the enrollment admission-number expand migration.
4. Implement backend combined admission and enrollment-level admission-number contracts.
5. Switch attendance and all other dependent readers/writers.
6. Implement profile-section storage, authorization and APIs.
7. Implement location hierarchy and address workflows.
8. Implement curriculum and enrollment subject selections.
9. Build the combined admission and sectioned profile UI.
10. Run full integration, tenant-isolation, RBAC and real-browser end-to-end acceptance.

A phase is complete only when code, migrations, automated tests and applicable browser acceptance pass and the actual result is recorded in Project Status.

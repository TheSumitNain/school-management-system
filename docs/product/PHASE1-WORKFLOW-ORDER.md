# Phase 1 School Operations — Task-by-Task Build Order

**Status:** Approved implementation sequence  
**Last reconciled:** 2026-11-13  
**Scope:** School Admin and assignment-scoped Teacher workflows in the Phase 1 web application.

This is the execution checklist for creating school data in the order that makes later workflows usable. It is a dependency order, not a claim that every listed item is still unimplemented. Before starting a task, inspect the current backend, frontend, migrations, tests and browser behavior; complete or verify existing work instead of rebuilding it.

## Operating rule

A task is not complete merely because a database table, API route or frontend page exists. Mark it complete only after the backend contract, school/role/resource authorization, frontend workflow, automated tests and applicable real-browser acceptance are verified. Record implementation and acceptance separately in [Project Status](../project/STATUS.md).

## Ordered work plan

### Task 0 — Verify school and academic foundations

**Must exist first**
- A school tenant created through the supported Platform Admin-assisted onboarding path.
- School Admin account and effective permissions.
- At least one usable academic year.
- Required classes and sections for that academic year.

**Actions**
1. Verify school onboarding and School Admin access against the real API/database.
2. Create or confirm the academic year and its start/end dates and status.
3. Create/confirm the school's classes in the intended display order (for example Nursery, LKG, UKG, Class 1 onward).
4. Create/confirm sections under the correct class and academic year.
5. Check that selectors display actual class names and sections, never database IDs or order/index values.

**Gate:** School Admin can view and manage the correct academic year, classes and sections; tenant boundaries and duplicate rules are enforced.

### Task 1 — Create and configure subjects

**Depends on:** Task 0.

**Actions**
1. Create the school's subject catalogue (for example English, Mathematics, Hindi, Science, Social Studies).
2. Define subject name/code and any required subject metadata.
3. Map each subject to the applicable classes/grades; do not assume every subject is taught in every class.
4. Support edit and active/inactive status, duplicate prevention and safe handling of subjects already referenced by assignments or timetable entries.
5. Verify that inactive subjects cannot be selected for new assignments or timetable entries.

**Implementation note:** Subject management already belongs to the existing Academic Structure area. Extend and verify that workflow there; do not create a second, competing Subject Management page or duplicate subject model.

**Gate:** Correct subjects are available for the intended classes and can be selected by teacher-assignment workflows.

### Task 2 — Create teacher/staff records and accounts

**Depends on:** Task 0 and the staff/role permission foundation. Subject catalogue may be prepared in parallel, but assignments wait for Task 3.

**Actions**
1. Create each teacher as a school-scoped staff record/account with the Teacher role and only the permissions needed for the job.
2. Capture required identity/contact, employee/staff code, designation, joining date and employment status fields according to the approved data model.
3. Enforce globally unique case-insensitive staff usernames and strict school ownership.
4. Support safe edit, deactivate/reactivate and password-reset flows; prevent deactivation when active dependencies require an explicit reassignment or safe resolution.
5. Keep non-teaching staff creation available through the same staff foundation, but do not grant Teacher-only access by default.

**Gate:** A teacher can sign in, sees only permitted school data, and has no class/subject access until explicit assignments grant it.

### Task 3 — Assign teachers to subjects, classes and sections

**Depends on:** Tasks 0, 1 and 2.

**Actions**
1. Assign a teacher to one or more active subjects that are valid for the selected class.
2. Scope assignments to the correct school and academic year, and to class/section where the product model requires it.
3. Assign class-in-charge responsibility separately from subject teaching responsibility.
4. Prevent duplicate, invalid or inactive teacher/subject/class assignments.
5. Provide clear edit/reassignment behavior and block removal/deactivation when dependent timetable entries exist until those dependencies are resolved.
6. Enforce the same assignment limits in APIs and repositories; frontend filtering is not authorization.

**Gate:** The School Admin can see an accurate assignment list, and a Teacher's API responses are restricted to their own assigned classes/subjects and permitted operations.

### Task 4 — Combined student admission and first enrollment

**Depends on:** Task 0. Teacher assignments may be completed before or after student admission, but a valid enrollment must exist before operational class workflows.

**Actions**
1. Use the combined guided admission workflow to create the permanent student identity and first academic-year enrollment in one backend transaction.
2. Require only the minimum basic identity fields needed to create the student; allow address, guardian, medical, physical and document sections to be completed later.
3. Assign an admission number to the enrollment for the selected academic year. Do not treat admission number as a permanent student identity or enforce school-wide uniqueness across all years.
4. Capture guardian/contact details as student relationship data where needed for school administration. This does not enable parent login, OTP or SMS.
5. Support student search, profile sections, academic history and school-/role-/field-level authorization.

**Gate:** Combined admission either creates both the student and first enrollment or neither; the profile can be completed incrementally; year-specific admission-number uniqueness is enforced.

### Task 5 — Returning-student enrollment, promotion and transfer

**Depends on:** Tasks 0 and 4.

**Actions**
1. Find the existing permanent student identity and select the destination academic year, class and section.
2. Create a new enrollment with a new admission number for that academic year; never create a duplicate student profile for a returning student.
3. Enforce one enrollment per student per academic year and validate that the selected section belongs to the selected class, school and year.
4. Preserve prior enrollment, attendance and subject-selection history. Promotion creates a new year record and closes the source enrollment through the approved lifecycle.
5. Design within-year section/class transfer explicitly; do not rewrite immutable historical enrollment identity.
6. Ensure class rosters and teacher roster endpoints show only correctly enrolled students within the caller's assignment scope.

**Gate:** Returning students retain one permanent identity, receive a new admission number each academic year, and historical enrollments remain intact.

### Task 6 — Finish and accept the timetable workflow

**Depends on:** Tasks 0, 1, 2, 3 and 5.

**Actions**
1. Configure period definitions and break periods for the school timetable.
2. Create weekly timetable entries for the correct academic year, class/section, subject and assigned teacher.
3. Validate teacher-subject eligibility and section enrollment context.
4. Prevent period overlap, class double-booking, teacher double-booking and scheduling during breaks at the database/API boundary.
5. Verify School Admin create/edit/remove permissions and Teacher read-only, assignment-limited views.
6. Test empty, partially configured and fully populated weekly schedules in the browser.

**Important:** Timetable backend and frontend work already exists. Preserve it. First verify and close its outstanding integration/browser acceptance gaps; do not rewrite completed timetable foundations.

**Gate:** A real school timetable can be configured using real teachers, subjects and sections, with conflicts rejected by the server/database.

### Task 7 — Complete attendance using real enrollment and rosters

**Depends on:** Tasks 0, 2, 3, 4 and 5. Timetable integration may improve navigation but must not be an unnecessary hard dependency for recording attendance.

**Actions**
1. Load the correct class/section roster for the selected date and academic year.
2. Restrict Teachers to assigned classes/sections; allow School Admin according to permissions.
3. Record, edit and review attendance with duplicate/date/roster safeguards and audit history.
4. Verify holidays, empty rosters, late changes and correction behavior according to approved attendance rules.

**Gate:** Attendance is recorded only for eligible enrolled students and authorized class scopes.

### Task 8 — Examinations, marks and results

**Depends on:** Tasks 0, 1, 3, 4 and 5.

**Design reference:** [Examinations and Results — Domain Design Proposal](EXAMINATIONS-RESULTS-DESIGN.md). This is a proposal, not an approved grading policy or permission to add schema. Resolve the listed product decisions before creating migrations.

**Actions**
1. Configure exam types/terms and examination sessions for an academic year.
2. Define exam schedule and class/subject applicability.
3. Enter and validate marks only for eligible students and assigned subjects.
4. Define grading, absent/exempt handling, totals, percentages and result calculation rules before implementation.
5. Publish/revise results with audit history and generate printable report cards.
6. Verify that teachers can enter marks only within their assignments and School Admin permissions control finalization/publication.

**Gate:** Results are reproducible from stored marks and versioned grading rules; no silent recalculation or cross-school access.

### Task 9 — Notices, communication and operational dashboard completion

**Depends on:** Core school identity, staff, academic setup and the relevant target audience/permissions.

**Actions**
1. Complete school notices/announcements with audience and role scope.
2. Show useful operational summaries based on real API data (students, staff, attendance, timetable and examinations as those workflows become available).
3. Provide meaningful empty states and links to the next setup task rather than fake/demo metrics.
4. Verify responsive layouts, accessibility, themes, session behavior and permission boundaries.

**Gate:** The School Admin can understand school setup/completion and daily operations from real data.

## Dependency overview

```text
School onboarding + School Admin
              |
Academic year + Classes + Sections
              |
          Subjects
              |
       Teacher accounts
              |
 Teacher ↔ Subject/Class assignments
              |
      Student admission
              |
 Student ↔ Academic year/Class/Section
              |
           Timetable
              |
          Attendance

Examinations & Results: after Subjects + Teacher assignments + Student enrollment
Notices & Dashboard: progressively after real operational data is available
```

Student profile creation and teacher setup can be prepared in parallel after the academic foundation, but do not start timetable acceptance until real subject, teacher-assignment and section data are available.

## Definition of done for every task

- Approved requirements and data ownership are documented; no speculative schema is added.
- Additive migration(s), constraints and indexes are reviewed where schema changes are needed.
- APIs enforce authentication, live permissions, tenant ownership, resource scope and assignment scope.
- Backend validation is authoritative; frontend validation is for usability, not security.
- Frontend has meaningful labels/placeholders, field-level errors, loading/empty/error/success states and safe destructive-action confirmation.
- Automated unit, handler, repository and PostgreSQL integration tests cover success, validation, conflicts and cross-tenant/unauthorized access.
- TypeScript check and production build pass for frontend changes; Go tests and `go vet` pass for backend changes.
- Browser acceptance is recorded for the actual role(s), including responsive and theme behavior.
- Documentation and [Project Status](../project/STATUS.md) are updated with evidence; incomplete acceptance remains explicitly pending.

## Explicitly out of this sequence

- Parent OTP/login, OTP/SMS delivery and Parent dashboard.
- Driver interface/mobile app.
- Fees/payment gateway, native mobile apps, live bus GPS and advanced AI analytics.
- Self-service school registration until the Phase 1 school web product is complete and separately approved.

Do not create placeholder routes, fake data or speculative migrations for deferred work.

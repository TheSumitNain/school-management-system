# Student Management — Domain and Database Design

**Status:** Approved design baseline; core Student Management implementation and mocked browser acceptance are substantially verified in CI; authenticated live acceptance against the local API/database remains outstanding.  
**Last reconciled:** 2026-11-15  
**Scope:** Phase 1 School Admin student admission, profiles, academic-year enrollment, curriculum selection and related school workflows.

This document defines the intended domain model and safe implementation sequence. Implementation notes below distinguish completed code paths from work still awaiting local verification or future phases. The ordered migration chain and automated tests remain authoritative for actual implementation.

## 1. Product decisions

The following decisions are approved:

- Admission uses one guided, combined workflow. Student identity and first academic-year enrollment are created together in one transaction.
- Student profile information is organized into independently editable sections, not one long mandatory form.
- A student may be saved with the minimum required identity details. Other profile sections can remain incomplete and be completed later.
- A student has one permanent school-scoped identity. Each academic year has its own enrollment record and its own admission number.
- A returning student is not created again. A new academic year creates a new enrollment for the existing student.
- Admission number is an attribute of an academic-year enrollment, not the permanent student identity.
- Roll numbers are scoped to an active section/enrollment, not the permanent student identity. The system assigns the next available numeric roll number automatically by default; an authorized School Admin may override it manually. Allocation must be transaction-safe under concurrent admissions, consider prior numeric roll numbers in that section so numbers are not recycled after withdrawal/promotion, and reject active duplicate roll numbers in the same section. Returning students and section changes receive a destination-section roll number; historical enrollment roll numbers remain preserved.
- Historical enrollments, attendance, guardian relationships and student IDs must be preserved.
- Parent OTP, SMS delivery, parent login and parent-facing applications remain Phase 2 and are not part of this implementation.

## 2. Audit findings in the current implementation

The current `work` branch contains the approved core Student Management model and its major profile workflows. The latest frontend/browser checkpoint `2b7e6c38f34f1ba28ca38dd3f3e527392b77fd26` passed Frontend CI (production build and all 11 Playwright tests). The latest backend security-test checkpoint `15820d22d0efb0493a2c45c3a6e014def008eff7` passed Backend CI (complete migrations, PostgreSQL integration tests and `go vet ./...`) on 2026-11-15, including cross-school read/write denial for student identity, Basic Details, contact, address, medical and physical profile data; enrollment history/create/promotion; location directory tenant boundaries; guardian ownership/relationship isolation; explicit cross-school guardian-link revoke denial; archived-student attendance rejection; independent `guardians.read` plus `students.read_basic` route permission enforcement for related-student results; and attendance route permission alternatives for read, mark, same-day update and past-date correction. Browser acceptance now also covers optional curriculum subject-group minimum/maximum boundaries and clear exact-one-subject guidance, alongside full Parent Details browser reload and shared guardian profile propagation across linked students while preserving per-student relationship flags. Browser API calls are mocked; authenticated live acceptance against the user's local API/database remains a separate required gate.

| Area | Existing state | Required evolution |
|---|---|---|
| Student identity | V11 `students` stores `full_name`; V32 makes the legacy `admission_number` nullable and removes school-wide uniqueness while preserving old values | Keep stable student identity; current-year display admission number is projected from enrollment |
| Guardian relationships | V11 `guardians` and `student_guardians` with same-school composite foreign keys; staff-managed create/edit/link/revoke/history, live permissions, shared guardian profile fields and revocation audit are implemented | Preserve relationship history, tenant isolation and privacy-safe audit metadata; parent authentication remains deferred |
| Academic year/class/section | V17 school-scoped years, classes and year-specific sections | Preserve existing model and constraints |
| Enrollment | V31 adds year-specific `admission_number`/`admission_date` and normalized school+year uniqueness; V33 makes both immutable. Backend now auto-assigns the next numeric roll number when omitted for combined admission, returning enrollment and promotion. It locks the destination section row before allocation; explicit manual overrides remain supported, and the active section-scoped unique index remains the final duplicate guard. Integration assertions cover initial auto-numbering, incrementing, promotion allocation and manual override; PostgreSQL integration suite and `go vet ./...` passed in GitHub Actions on 2026-11-14 | Preserve one enrollment per student/year and keep legacy unknown rows NULL/NULL. Browser acceptance remains required before marking the UI workflow complete |
| Promotion | Repository and frontend guided workflow create the destination enrollment with a new year-specific admission number/date and close the source transactionally; automated integration and Playwright coverage exist | Complete authenticated live acceptance and verify preserved history |
| Attendance | V20 records reference enrollment; attendance projections now read `student_enrollments.admission_number` | Keep year-scoped display and historical NULL compatibility |
| Subjects | V44 class/year curriculum versions, compulsory/optional subjects, selection limits, enrollment-specific choices and append-only selection history are implemented and covered by PostgreSQL integration tests | Preserve version/history semantics and complete authenticated live acceptance |
| Student profile | V37 optional one-to-one `student_basic_details` is implemented with restricted reads and privacy-safe audit; name and profile sections are independently editable | Complete authenticated live verification of load, validation, save and reload |
| Addresses/locations | V34 adds `student_addresses` and tenant-aware `student_locations`, with parent-level, ownership, duplicate and deactivate-not-delete rules | Complete local verification; extend the reviewed IGOD State/UT snapshot with an authoritative LGD import for districts/subdistricts/localities |
| Medical/physical/documents | V36 medical profile version history and append-only dated physical measurements are implemented with restricted reads/writes and privacy-safe audit. Document upload/download remains deferred pending private storage and document-specific controls | Complete authenticated live verification of medical/physical history; keep documents deferred |

Student directory/profile search now uses the current academic year's enrollment admission number; attendance projections use the matching enrollment. New student creation and first enrollment share one transaction. Legacy student-level values remain available only as compatibility data when there is no current-year enrollment; no ambiguous value has been copied into an enrollment.

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

- Basic identity: permanent student name is edited inside the Basic Details section (not as a separate action on the profile summary), alongside an independently editable optional basic-details record containing date/place of birth, gender, nationality, religion, social category, EWS status and school registration number. These fields are not required for initial admission.
- Aadhaar is intentionally not part of the default student profile and must not be added as a required field. Any future national-ID collection needs a separately approved purpose, legal/privacy review, least-privilege access and retention policy.
- Address and communication: current and permanent address, phone/email where appropriate, and same-as-current behavior.
- Parents/guardians: reuse the existing guardian and relationship model; support multiple guardians and relationship/contact flags.
- Medical details: one optional restricted current profile with blood group, allergies, known medical conditions, ongoing medication notes and emergency medical notes. Every change snapshots the previous version into append-only history. These are school-entered facts, not diagnoses or automated health assessments.
- Physical details: dated append-only measurement records for height (cm), weight (kg) and concise vision/physical notes. New measurements never overwrite earlier measurements; the profile shows the latest record and retains prior records. The system must not infer health status from measurements.
- Basic, address/contact and medical/physical profile reads use the existing restricted `students.read_sensitive` permission; updates use `students.update`. These permissions are not delegated to Teacher roles. Read and write operations are audited. Basic-profile audit metadata records field-presence flags only, never demographic values; health values and notes are excluded from audit metadata.
- Documents: metadata, document type, issue/expiry dates where relevant, verification status and an opaque secure-storage reference. Do not store public file URLs or filesystem paths in API responses. Document upload/download stays deferred until private storage and document-specific access controls are reviewed.

A section may be absent or incomplete. Its absence must not prevent basic student creation or ordinary student directory use.

V38 extends each school-owned guardian contact with optional alternate mobile, occupation, education and workplace/organization fields. These are contact/profile attributes on the guardian identity (shared when that guardian is linked to multiple siblings), while relationship type and primary/emergency flags remain specific to each student link. Student current/permanent address remains the authoritative household address; it is not duplicated onto each guardian record. Existing guardian records remain valid with these fields NULL. Parent login/OTP remains deferred.

The existing V11 `guardians` and `student_guardians` model is now exposed in a staff-managed Parents & Siblings profile section. Staff can create guardian contact records, edit the shared guardian profile (with changes reflected for every linked student), link multiple guardians with relationship/primary/emergency flags, and revoke links with a reason while retaining history. The existing V12 active-only unique index permits a later re-link as a new relationship period without overwriting the revoked row. The related-student list is derived only from active shared guardian links and is labelled “Other students sharing a guardian”; it must not be presented as proof of a biological sibling relationship. This work does not enable parent login or OTP; parent authentication remains deferred.

### 3.4 Address and location hierarchy

Use normalized location records with a parent relationship and a controlled location type, allowing the address hierarchy State → District → Sub-District/Tehsil → Village/City. Development Block is a separate administrative dimension and must not be treated as a synonym or substituted for Sub-District/Tehsil. Do not hard-code a fixed list of Indian states or assume every deployment uses the same geography.

V34 implements the first address/location foundation:

- `student_locations` supports shared canonical rows (`school_id IS NULL`) and school-owned additions, with State → District → Subdistrict/Tehsil → Locality (Village/City) parent validation. The current schema does not yet model Development Blocks or village-to-block crosswalks.
- Normalized duplicate checks apply within owner/parent scope, and active canonical entries cannot be duplicated by school-owned additions (or vice versa).
- Ownership and hierarchy cannot be rewritten. Locations are deactivated rather than deleted; parents with active descendants cannot be deactivated.
- `student_addresses` stores current and permanent addresses separately, supports same-as-current without copying data, and enforces same-school/global location ownership.
- Searchable dependent selectors and school-owned additions are available in the Address & Communication UI. Free-text house/street, area/mohalla, landmark and PIN/postal code are stored separately.
- A saved address can retain a previously selected inactive locality; it cannot be newly selected for another address.

V35 adds optional student primary phone, alternate phone and email in a separate one-row-per-student contact table. Phone/email values are not unique because households may share contact channels; updates are audited without putting the values into audit metadata.

The frontend does not hard-code geography options. V40 adds a database-backed snapshot of all 36 India State/UT entries from the Government of India's IGOD State/UT directory (source codes are retained as source_system=IGOD/source_code). School-created State/UT entries must use exactly two ASCII uppercase letters as the ISO country code; byte-length alone is not sufficient. District, Sub-District/Tehsil and Village/City rows can be added by a school when absent. Development Blocks are not currently representable in `student_locations`; do not import or label a Block as a Sub-District. A reviewed schema change and separate village-to-block mapping are required before LGD Block data can be imported. Any future bulk import must retain source codes, source date, parent relationships and a repeatable update strategy.

LGD source acquisition and import rules:
- Official source: Ministry of Panchayati Raj LGD portal (`https://lgdirectory.gov.in/`) and its [Download Directory](https://lgdirectory.gov.in/downloadDirectory.do) workflow. The portal exposes state-wide and entity-level downloads for districts, sub-districts, villages and development blocks, with date-scoped modification downloads. Its report generation uses a human CAPTCHA and offers PDF, HTML, XLS and ODT formats; a staff member must obtain the source snapshot through the official UI. Do not automate, bypass or attempt to solve the CAPTCHA.
- LGD assigns unique location codes. Preserve each source code, source level, source name, parent reference/mapping, active/effective status and source snapshot date as provided. Keep the original downloaded artifact unchanged and record its retrieval date, selected state/entity scope, report date range, format and SHA-256 checksum before transformation.
- Do not treat a Development Block as a Sub-District/Tehsil. The official hierarchy distinguishes State, District, Block, Sub-District and Village; village-to-block coverage can be a separate mapping. The address selector may use State → District → Sub-District → Village/City, but an import must preserve the separate block relationship rather than flattening or silently substituting it.
- Before import, inspect the actual report headers and publish a reviewed field crosswalk. Reconcile duplicate/missing LGD codes, parent references, inactive/merged/split entities, state-code crosswalks and row counts. Run a dry-run report first; never infer parentage from matching names alone. Version every snapshot/import so later LGD changes can be applied as reviewed deltas without deleting historical address references.
- V40 currently seeds only the 36 State/UT display names from IGOD; V42 stores versioned source identifiers and backfills those IGOD references. The official [LGD portal](https://lgdirectory.gov.in/) describes its directory as the authoritative standardized source for administrative units and exposes distinct citizen lookup pages for Districts, Sub-Districts, Villages and Development Blocks. The landing page's observed summary on 2026-11-15 showed 790 districts, 7,112 sub-districts, 7,328 Development Blocks and 677,660 villages; these are a dated discovery snapshot, not hard-coded validation thresholds.
- The official [Download Directory](https://lgdirectory.gov.in/downloadDirectory.do) page has been inspected. Its form offers a state-wise export mode and separate options for all districts, all sub-districts, all villages, all Development Blocks, and all Development Blocks with covered villages; it also offers date-scoped modification reports and PDF/HTML/XLS/ODT output. Report generation requires a human CAPTCHA. The workflow is confirmed, but no actual report has yet been generated/downloaded, so exact export headers, row counts and source-file checksums remain unverified. Obtain exports through the official UI without automating or bypassing CAPTCHA, preserve the original bytes, and record state/entity scope, report date range, format, retrieval date and SHA-256 before transformation. The official [OGD India LGD catalog](https://www.data.gov.in/catalog/local-government-directory-lgd) reports an update date of 2026-11-13 and displays a ZIP Download action and Catalog API entry; the archive target was not resolved during this review, so treat it as a second official acquisition path to inspect manually, not as an already obtained dataset.
- The current read-only `backend/cmd/lgdvalidate` tool is only a preliminary validator: its ten-column normalized schema models State → District → Sub-District → locality. It does not represent Development Blocks as a separate dimension or validate village-to-block mappings. Do not treat its current PASS as approval for a full LGD import. After inspecting the real source export, extend the normalized schema and validator to validate District → Development Block separately from District → Sub-District and Village ↔ Development Block relationships, including multi-district/multi-block exceptions and unmapped entities.
- Before any future import, publish a reviewed source-field crosswalk, reconcile counts, duplicate/missing codes, parentage and inactive/merged/split entities, run a read-only dry-run, and review its report. Keep source IDs as strings to preserve leading zeroes. Any database import must be a separately reviewed, idempotent, append-history-aware operation and must not run against the user's persistent database without explicit approval and a verified restorable backup.

- OGD catalog discovery page: `https://data.gov.in/catalog/local-government-directory-lgd`; use it as a secondary official publication/discovery surface, not as a substitute for verifying the current LGD source snapshot.


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
3. Select the academic year, class and section; capture the admission number/date. The system assigns the next available numeric roll number in that section by default; an authorized School Admin may enter a manual override.
4. Optionally complete address, guardian, medical, physical and document sections.
5. Review and submit.

The backend creates the student and first enrollment in one database transaction. If either insert or required audit write fails, the transaction rolls back. The UI must not create a permanent student first and then leave the user to discover that the required initial enrollment failed.

The minimum required fields should be small and documented. Optional sections can be skipped and completed from the student's profile later.

### 4.2 Returning student / new academic year

1. Find the existing student using identity/profile search and prior enrollment history.
2. Select the destination academic year, class and section.
3. Assign a new admission number for that academic year and capture its admission date. Assign the next available numeric roll number in the destination section by default, with an authorized School Admin manual override.
4. Create a new enrollment; do not create a second student identity.
5. Preserve prior enrollment, attendance and subject-selection history.

Promotion and transfer are distinct business actions. Promotion creates the next-year enrollment and closes the source enrollment using the approved promotion lifecycle. A within-year section transfer must be explicitly designed against the existing immutable enrollment identity rule; do not mutate a historical enrollment's section in place. If transfer requires a replacement enrollment or a separate placement-history model, define its attendance/date semantics before implementation.

### 4.3 Student profile

The directory opens a student profile detail view with permanent identity and academic history shown year by year. Each history entry displays that year's admission number/date, class, section, roll number, status and enrollment lifecycle dates. Launch independently editable sections (Basic Details, Address & Communication, Parents & Siblings, Medical & Physical, Documents) from the profile; the student name belongs in Basic Details, while the profile summary remains a read-only overview. Do not turn the profile into one long mandatory form. Clearly distinguish incomplete profile sections from enrollment status.

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


## 9. Student profile screen audit — 2026-11-15

### Latest verification reconciliation

- Latest Frontend CI run [43719842826](https://github.com/TheSumitNain/school-management-system/actions/runs/43719842826) passed on code commit `2b7e6c38f34f1ba28ca38dd3f3e527392b77fd26`; Vite production build and all 11 Playwright tests passed, including cross-student profile isolation, Parent Details full browser reload, shared guardian profile propagation across two linked students, and optional-subject minimum/maximum validation.
- Latest Backend CI run [43723595815](https://github.com/TheSumitNain/school-management-system/actions/runs/43723595815) passed on backend security-test commit `15820d22d0efb0493a2c45c3a6e014def008eff7`; the full migration chain, PostgreSQL integration suite and `go vet ./...` passed, including cross-school student identity/basic/contact/address/medical/physical read-write denial, enrollment history/create/promotion, location tenant boundaries, guardian ownership/relationship isolation, cross-school guardian-revoke denial, archived-student attendance rejection, independent related-students route permission checks and attendance route permission alternatives.
- These are separate verification checkpoints: the frontend/browser workflow verifies the unchanged UI at `2b7e6c38f34f1ba28ca38dd3f3e527392b77fd26`, while the later backend workflow verifies additional repository-level tenant-isolation and route-permission tests at `b0a3605f72b214101dc8137bfafda047b51a5ce6`. No frontend source changed in the backend-only commits.
- Guardian form validation now allows over-limit values to reach application validation so each field can show its explicit length error. Do not reintroduce HTML `maxLength` truncation on those fields.
- Guardian revocation audit has one authoritative writer: the V12 database trigger, updated by V45 to add only the boolean `has_revocation_reason`. The repository must not insert a duplicate audit event. Free-text revocation reasons must remain out of audit metadata.
- These automated checks do not complete the manual/live acceptance checklist below. Playwright uses mocked APIs; local authenticated acceptance remains pending and must respect the persistent-database approval/backup safety gate.

- The latest browser-test work adds explicit window-focus background-refetch checks for unsaved drafts in Basic Details, Parent Details, Address, Contact, Medical and Physical forms. The test waits beyond the relevant query stale windows, verifies that the API is read again, and asserts that the user's draft remains visible when the persisted response is unchanged. Test checkpoint: `eaed7705bcf14f156a423a157f5f2adebfc5181c`; Frontend CI [43712030788](https://github.com/TheSumitNain/school-management-system/actions/runs/43712030788) passed all 8 Playwright tests, including these draft checks.
- The app-wide QueryClient focus policy remains unchanged. The browser regression deterministically invalidates the app's actual QueryClient query families, waits for active profile queries to refetch, and verifies unchanged unsaved drafts remain intact. This exercises the same invalidation/refetch path used by successful mutations without introducing extra focus-triggered reads.
- The first run exposed a name-save feedback race: the success message was cleared when the same student's updated name changed a synchronization-effect dependency. The reset now depends on student identity only; source fix `2419d61299a878669392f69f44549a8e45f7ff9a`. The fix is covered by the passing frontend checkpoint above.

## 9. Student profile screen audit — 2026-11-14

The latest screen/code review found two concrete UX gaps and records the current disposition:

- The directory row's **Edit** action previously opened a legacy modal that submitted only the permanent student name. It now opens the Student Profile section hub so Basic Details, Address & Communication, Medical & Physical, and permission-authorized Parents & Siblings remain independently editable in line with the approved section-based design. This does not combine unrelated records into one unsafe update request.
- Address reads already returned the selected hierarchy IDs, but the frontend location pickers only listed active locations. If a previously saved school-owned state/district/sub-district/locality was later deactivated, its ID could be present while its option was absent, making the lower hierarchy appear blank. Address reads now also return the ancestor names, and the frontend retains a visible inactive option for an existing saved selection. Deactivation still removes the location from new active selections and does not rewrite historical addresses.
- A PostgreSQL integration assertion covers saving a complete State → District → Sub-District/Tehsil → Village/City path, deactivating the saved locality, and reading all saved IDs and names back. The test fixture uses a Tehsil label for the sub-district level; it does not claim to model a Development Block.
- Aadhaar is intentionally not a default student field and must not be required. The UI/API/schema currently omit it by design. Any request to collect an Aadhaar number or upload an Aadhaar image must first distinguish the intended data, establish a lawful school purpose, retention/deletion rules, masked display, encryption/private storage, and dedicated least-privilege access. Do not add a plaintext Aadhaar field or public document URL.
- Student document upload/download remains deferred until private storage, malware scanning, opaque references, retention and document-specific access controls are implemented.

Automated frontend coverage now exercises the archived-state rule independently of missing permissions: the mocked staff principal has `students.update` and `students.archive`, archives a student after a conflict/retry, reopens the archived profile, confirms saved Basic Details hydrate, and verifies that edit/save controls remain unavailable. This is a mocked API workflow only; the checklist below still requires live authenticated API/database acceptance.

### Required acceptance checks before Student Management is called complete

- [ ] Directory View and Edit actions open the correct profile workflow; no legacy name-only edit route remains reachable.
- [ ] Basic Details: existing values hydrate, each field validates, save/reopen retains values, and archived students are read-only.
- [ ] Basic Details sensitive reads require `students.read_sensitive` and create a transactional audit event containing only the section name and student reference; no personal field values are copied into read-audit metadata.
- [ ] Student directory and individual profile reads require `students.read_sensitive` and create transactional audit events containing only section/result-count metadata and the student reference; student names and admission numbers must not be copied into read-audit metadata.
- [ ] Address & Communication sensitive reads require `students.read_sensitive` and create transactional, section-only audit events for address and contact details; audit metadata must never contain phone numbers, email addresses or address text.
- [ ] Address & Communication: current/permanent/contact tabs hydrate independently; every address text field and all four location levels load; inactive saved locations remain visible; same-as-current works; save/reopen retains values.
- [ ] Parents & Siblings: section visibility follows guardians.read; create/edit/link/revoke follows guardians.manage; revoked history is retained; related students require basic-read permission.
- [ ] Parents & Siblings reads (guardian directory, guardian profile, student guardian links/history, and students sharing a guardian) require the applicable live permissions and write transactional audit events containing only section/result-count metadata; no guardian contact values, names or relationship values are copied into read-audit metadata.
- [ ] Promotion: only active students may be promoted; the permanent student row is locked before the source enrollment so archive/promotion operations use a consistent lock order.
- [ ] Archive: all active enrollments transition to withdrawn with a valid end date in the same transaction; enrollment and attendance history remain readable; archived students cannot receive new attendance marks, including a mark racing with the archive transaction.
- [ ] Medical & Physical: read/write permissions, version/measurement history, loading/error states and archived-state behavior are verified.
- [ ] All list/profile/API actions enforce live permissions and same-school isolation; frontend visibility is never treated as authorization.
- [ ] Responsive layout, keyboard/focus behavior, loading/empty/error states, field-level validation and save feedback are checked at desktop and narrow viewport sizes.
- [ ] Backend PostgreSQL integration tests, frontend typecheck/build, and manual browser acceptance all pass.

Until these checks are completed, Student Management remains **in progress**, not production-ready.


## 10. MySmartSchool reference field inventory and Aadhaar handling

The November 13, 2026 reference-screen review captured the following target inventory. Preserve the combined admission workflow and section-based profile layout already approved:

- **Combined admission / yearly enrollment:** academic session, class, section, optional roll number (auto-assign when blank if the section policy supports it), admission number, admission date; link to permanent student identity.
- **Basic / personal details:** student full name, date of birth, gender, nationality, place of birth, religion, social category, EWS status, Aadhaar number (optional), school SR/registration number, student photo (photo upload deferred with documents).
- **Address & communication:** correspondence/current address, permanent address, same-as-current, phone/mobile and email.
- **Parents & siblings:** father and mother names, guardian relationships and contact details, sibling links.
- **Medical details** and **Physical details** as separate profile sections.
- **Class & subject:** year-specific class/section/roll, compulsory and optional subject selections, academic history.
- **Transport**, **App permissions**, and **Documents** as separately permission-controlled sections; document uploads are deferred to the secure document-storage phase.

### Aadhaar number field requirements

- Aadhaar is optional and must never block admission, enrollment, school services, or profile completion. UIDAI says school admission cannot be denied for want of Aadhaar and its official mythbusters state Aadhaar is not mandatory for school admission or CBSE examinations.
- Do not store the 12-digit number as plaintext, return it in ordinary student profile responses, include it in audit metadata/logs, or expose it to roles that only need ordinary student details.
- Before enabling entry, implement dedicated application-level authenticated encryption using a separately managed key (not the TOTP/MFA key), key versioning/rotation, strict input validation, last-four-only display, explicit reveal only if justified and separately permission-gated, audited access, and clear/delete behavior. The field must be blankable and never required.
- Until that backend/key-management work and its permission/integration tests are complete, do not ship a frontend field that transmits or persists the full number insecurely.

Official references: UIDAI circular, 5 September 2018, “No denial of admission in schools for want of Aadhaar”; UIDAI Aadhaar Mythbusters, question on school admission.
## 11. Student Management form clarity and validation standard

Every Student Management form must make field expectations understandable before submission:

- Show an explicit **Required** or **Optional** marker beside every editable field label. An asterisk may supplement the text but must not be the only indication. Conditional requirements must say when they become required (for example, measurement date is required only when a physical measurement is being recorded).
- Add a keyboard- and touch-accessible information icon only where a field has non-obvious meaning, format, allowed values, length/range, privacy implications or a dependency. Its help text must explain what to enter and any restrictions in plain language. Hover-only tooltips are not sufficient; focus/click must expose the same help.
- Place examples in placeholders or short helper text, but never use a placeholder as the only label or as a substitute for validation/help.
- Show every relevant validation error next to its field and associate it with the control using accessible descriptions. Keep a form-level message only for non-field/server errors and connect it to the affected fields where possible. Map server-side uniqueness/conflict responses to the specific field whose value caused the conflict (for example, a duplicate school registration number); preserve the entered values and let the user correct and retry without losing the rest of the form.
- Required/optional labels, frontend validation, API validation, repository rules and database constraints must agree. Do not mark a field optional in the UI if the server or database requires it, and do not make a genuinely optional profile field block admission or profile saving.
- Save controls must expose why they are unavailable: loading, failed prerequisite read, missing permission, invalid input or an in-flight save. Failed prerequisite reads must offer a retry and must not silently allow an empty form to overwrite existing data.
- Text length limits must count Unicode code points/characters consistently across frontend, API, repository validation and PostgreSQL VARCHAR(n); never compare UTF-8 byte length to a database character limit. Include Hindi and other multilingual names/address values in regression tests.
- Do not use native HTML maxLength as the enforcement mechanism for fields whose limits are Unicode code-point based: browser maxLength counts UTF-16 code units. Validate with Array.from(value).length (or an equivalent code-point counter), show the error directly beneath the affected control, associate it with aria-describedby, and keep repository validation authoritative. Apply this consistently to medical and physical notes as well as identity, address, contact and guardian text.
- Preserve entered values after recoverable server errors. Do not clear forms or close dialogs on a failed save.

Apply this standard consistently to initial admission, returning enrollment, promotion, Basic Details, Address & Communication, Parents & Siblings, Medical & Physical and school-owned location creation.


## Backend input-validation regression coverage — 2026-11-15

- Shared student profile text normalization has unit coverage for nil/blank inputs, whitespace trimming, exact Hindi Unicode-code-point limits and over-limit rejection. This helper is used by Basic Details, Address, Contact and Health persistence.
- Student Contact Details PostgreSQL integration now asserts that a 31-code-point phone value and an email containing a display name are rejected as invalid input before any database write.
- Backend CI [run 43712579620](https://github.com/TheSumitNain/school-management-system/actions/runs/43712579620) passed the full migration chain, PostgreSQL integration suite and `go vet ./...` on test commit `5a031c694f839338d4e766f5f6a72fcaf525dbea`.


- Follow-up browser coverage explicitly edits the shared guardian's full name, checks the 200-character Unicode boundary, saves the change and verifies it after reopening Parent Details. Frontend CI [run 43712924033](https://github.com/TheSumitNain/school-management-system/actions/runs/43712924033) passed all 8 Playwright tests on follow-up checkpoint `fc43c199b48a26b08c15a33e86ff28115c8b244c`; Backend CI [run 43713194771](https://github.com/TheSumitNain/school-management-system/actions/runs/43713194771) also passed the full migration chain, PostgreSQL integration suite and `go vet` on the latest checkpoint `7e78fd01bc88277d420b276b3ecb82f42eba1753`.


- Browser regression coverage now checks Unicode length rejection for the per-student relationship label and editing both primary/emergency flags to false. The test verifies the outgoing PATCH payload and that the values persist after the shared guardian profile edit and reopening Parent Details. Frontend CI [run 43713791921](https://github.com/TheSumitNain/school-management-system/actions/runs/43713791921) passed all 8 Playwright tests; Backend CI [run 43713791860](https://github.com/TheSumitNain/school-management-system/actions/runs/43713791860) passed full migrations, PostgreSQL integration and `go vet` on `c4a221ad24048333bafc2dda2f52d9c9cc8542eb`.


- Cross-student browser coverage: Student A's unsaved Basic Details draft and field validation error are verified not to appear when switching to Student B; Student B's own saved basic details load. Frontend CI [run 43715113384](https://github.com/TheSumitNain/school-management-system/actions/runs/43715113384) passed all 9 Playwright tests, and Backend CI [run 43715113361](https://github.com/TheSumitNain/school-management-system/actions/runs/43715113361) passed full migrations, PostgreSQL integration and `go vet` on `433461bec48f502765b97464ba6e188a9324d687`. Authenticated live acceptance remains outstanding.


- Parent Details cross-student isolation is also covered: Student A's unsaved new-guardian name/relationship and field-level validation error do not appear when opening Student B's Parent Details. Latest checkpoint `b6d14c6ed4030724857838ef8cebc131041219ae`: Frontend CI [run 43716057611](https://github.com/TheSumitNain/school-management-system/actions/runs/43716057611) passed all 10 Playwright tests; Backend CI [run 43716057614](https://github.com/TheSumitNain/school-management-system/actions/runs/43716057614) passed migrations, PostgreSQL integration and `go vet`. Authenticated local acceptance remains outstanding.


- Final Parent Details draft-isolation test checkpoint `b6d14c6ed4030724857838ef8cebc131041219ae` passed Frontend CI [run 43716057611](https://github.com/TheSumitNain/school-management-system/actions/runs/43716057611) with all 10 Playwright tests and Backend CI [run 43716057614](https://github.com/TheSumitNain/school-management-system/actions/runs/43716057614) with migrations, PostgreSQL integration and `go vet`. The test-only selector corrections did not change application source. Authenticated local acceptance remains outstanding.

## Guardian email validation regression coverage — 2026-11-15

- Backend repository integration tests reject malformed guardian email values, including display-name and comma-separated multiple-address inputs, as well as over-limit Unicode values.
- Parent Details Playwright coverage applies the same malformed email cases to both shared guardian profile edits and new guardian creation. It verifies inline errors and that invalid submissions do not change the mocked persisted guardian profile or create a guardian.
- Checkpoint `da85edc9f020c7db2d0b474c4855e3be7249bf80`: Frontend CI [run 43724412256](https://github.com/TheSumitNain/school-management-system/actions/runs/43724412256) passed production build and all 11 Playwright tests; Backend CI [run 43724412200](https://github.com/TheSumitNain/school-management-system/actions/runs/43724412200) passed migrations, PostgreSQL integration tests and `go vet ./...`. Authenticated local API/database acceptance remains outstanding.


## Promotion and attendance history integrity — 2026-11-16

- A promotion may not close its source enrollment before attendance already recorded for that enrollment. The backend rejects such a retroactive promotion; the source enrollment and destination-year history remain unchanged.
- PostgreSQL integration coverage verifies the rejection using a destination academic-year date that otherwise passes normal year-boundary validation. Backend CI [run 43794753873](https://github.com/TheSumitNain/school-management-system/actions/runs/43794753873) passed migrations, PostgreSQL integration and `go vet ./...` on checkpoint `60674df6b419a5504d4d11813f7354ecbe3058ae`.

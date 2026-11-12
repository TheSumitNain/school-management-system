# Examinations and Results — Domain Design Proposal

**Status:** Proposed design; no implementation or schema changes are authorized by this document alone.  
**Scope:** Phase 1 School Admin and assignment-scoped Teacher workflows.  
**Parent access:** Out of scope; Parent OTP/SMS and parent-facing result access remain deferred.

## 1. Purpose and boundaries

Provide a school-configurable examination workflow that uses the existing academic-year, class, section, subject, student-enrollment and teacher-subject-assignment foundations.

The module must not introduce a second student, subject, class, section or academic-year model. All examination records belong to one school and one academic year. Student marks must be tied to the student's immutable enrollment record for that year, not merely to a mutable student profile.

This proposal does not define a universal school grading policy. Schools must be able to configure their own exam structure and grading rules.

## 2. Proposed workflow

1. **Exam setup:** A School Admin creates an examination session for an academic year (for example, Unit Test 1, Half-Yearly or Annual). The name and exam category/type are school-configurable.
2. **Assessment schedule:** The School Admin adds subject assessments for the applicable class/section, with assessment date, maximum marks and pass marks. A session may cover multiple classes, sections and subjects.
3. **Teacher mark entry:** An assigned Teacher sees only assessments for the teacher's active subject/section assignments and enters marks for eligible students in the corresponding active enrollment.
4. **Review:** The School Admin reviews completeness, invalid values and missing/absent/exempt entries before finalization.
5. **Calculation:** Totals, percentages and grades are calculated from stored assessment marks using the grading-rule version attached to the result.
6. **Publication:** An authorized School Admin publishes results. Corrections after publication create a new revision with an audit trail; previously published results remain reproducible.
7. **Report card:** The application renders a printable report card from the published result revision and its grading-rule snapshot.

## 3. Proposed domain records

The following are conceptual entities, not approved table names or a migration specification.

- **Exam session:** school, academic year, configurable name/type, description, lifecycle status and audit timestamps.
- **Exam assessment:** exam session, class/section, subject, assessment date, maximum marks, pass marks and the teacher-subject assignment that authorizes mark entry.
- **Assessment mark:** assessment, student enrollment, status (for example, present, absent or exempt), marks when applicable, remarks and created/updated actor metadata.
- **Grading scheme and version:** school-owned scheme with immutable versions containing ordered grade bands and any configured grade labels/remarks.
- **Result revision:** exam session/scope, calculation inputs, grading-scheme version, computed totals/percentage/grade, lifecycle state, revision number, publication metadata and audit actor.
- **Audit history:** append-only events for configuration, mark entry/change, finalization, publication and post-publication revision.

Use composite foreign keys and repository-level checks to enforce school and academic-year scope. Where a record references a section, subject, enrollment or teacher assignment, the database must prevent cross-school or cross-year combinations.

## 4. Mark-entry and calculation rules

- Maximum marks must be greater than zero; pass marks must be between zero and maximum marks.
- A numeric mark must be non-negative and no greater than the assessment maximum.
- An absent or exempt status must be explicit; do not encode it as zero or a blank numeric mark.
- An exempt assessment must be excluded from applicable totals and denominators. An absent assessment must follow the school's configured result policy; it must not silently become exempt or a pass.
- Store the raw assessment mark/status as the source of truth. Calculated totals, percentages and grades must be reproducible from the exact included assessments and grading-rule version.
- Define rounding and decimal precision explicitly in the grading configuration before implementation. Do not rely on language/runtime defaults.
- Do not silently change historical results when a grading scheme or assessment configuration is edited. Published results retain their calculation inputs and rule version.
- Incomplete marks must be visible as incomplete; the system must not publish a misleading partial result unless an explicitly configured policy allows it.

## 5. Lifecycle and change control

Proposed lifecycle:

- Exam session: Draft → Scheduled → In progress → Completed → Published.
- Assessment configuration is editable before mark entry begins, subject to validation. Once marks exist, changes that affect eligibility or calculation must require an explicit controlled revision.
- Mark entry is editable while the session is open. Closing mark entry prevents ordinary teacher edits.
- Finalization validates that every eligible enrollment has an explicit mark status.
- Publication is a distinct permission-controlled action.
- Post-publication correction creates a new result revision; retain the previous published revision and identify who changed what and why.

The exact status names and transition permissions should be confirmed during API design.

## 6. Authorization and tenant isolation

Use the existing permission vocabulary where suitable: `exams.read`, `exams.manage`, `results.read`, `results.enter`, `results.update` and `results.publish`. Confirm the live permission registry and role grants before wiring routes.

- Every request resolves the actor's school from the authenticated principal. Never trust a school ID supplied by the browser as an authorization boundary.
- School Admin actions require the corresponding live school-scoped permission.
- Teachers may read and enter/update marks only for assessments whose section/subject/teacher combination is covered by their active assignment. A client-side filter is not sufficient.
- Student rows must be drawn from the correct enrollment for the assessment's academic year and section. Historical enrollment must not be rewritten.
- Reads, writes, exports and report-card generation must enforce the same school/resource/assignment checks.
- Publish/finalize operations must be atomic with audit writes. A failed audit write must roll back the protected mutation.

## 7. Frontend workflow expectations

- Add one Examinations & Results area to the existing School Admin workspace; do not create a duplicate academic catalogue.
- Provide clear empty states that direct the administrator to configure the academic year, subjects, teacher assignments and enrollment first.
- Assessment forms must use shared controls, theme tokens and the shared date picker; show field-level validation below each invalid field.
- Mark-entry grids must make student identity, admission number, enrollment/class context, maximum marks and status explicit. Support keyboard navigation and save feedback without losing unsaved entries.
- Clearly distinguish Draft, In progress, Completed and Published states; do not communicate state by color alone.
- Provide responsive layouts and accessible controls. Printable report cards must use real published data, never mock metrics or sample student records.

## 8. Required verification

Backend tests must cover:
- Same-school and same-academic-year constraints, including cross-tenant IDs.
- Enrollment eligibility and teacher-assignment scope on every mark read/write.
- Invalid maximum/pass marks, negative marks, marks above maximum, absent/exempt handling and incomplete sessions.
- Duplicate mark submissions, concurrent edits and closed-session behavior.
- Deterministic totals/percentages/grades, rounding and grading-version retention.
- Publication, post-publication revision and atomic append-only audit behavior.
- Permission denial, inactive/revoked teacher assignments and school isolation.

Frontend acceptance must cover School Admin setup/review/publication and Teacher assignment-limited mark entry, including loading, empty, validation, error, success, keyboard, responsive and theme states.

CI must apply migrations to a fresh disposable PostgreSQL database and run the full backend suite, integration tests, `go vet`, frontend TypeScript checks, style checks and production build. Local persistent databases must not be migrated without the user's explicit approval and a verified backup.

## 9. Product decisions to confirm before implementation

The following must be configurable or explicitly approved rather than guessed:

1. Whether each exam session contains one or multiple assessment components per subject (for example, written, practical, oral and internal assessment).
2. Whether absent marks count as zero for pass/final-grade calculations, and how exemptions affect minimum-subject requirements.
3. Whether the school needs grade-only results, marks plus grades, or both.
4. Whether result publication is whole-session, class/section-wise or student-wise.
5. Required decimal precision and rounding rule.

**Recommended default:** support multiple assessment components, configurable absent/exempt policies, marks plus configurable grades, class/section-wise publication and a school-configurable rounding rule. This is a recommendation only, not an approved product policy.

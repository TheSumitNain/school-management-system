# Staff Qualifications & Documents — Backend Contract

**Status:** V57 schema and school-scoped GET/POST/PATCH APIs for qualifications, certifications and experience, with frontend create/edit integration, are committed on `work`. Browser acceptance covers create, validation and edit flows for all three record types. The dedicated verification workflow remains intentionally unavailable pending a separate permission decision; private document upload/download remains gated on a private storage adapter. CI verification for the latest Staff Management checkpoint is recorded in `docs/project/STATUS.md`; do not infer local or persistent database migration state from CI.
**Reviewed branch:** `work`
**Review date:** 2026-11-21

This contract extends the existing school-scoped `staff_profiles` foundation. It does not change Academic, Student Management, Settings, authentication, or the existing employee-account lifecycle.

## 1. Existing baseline verified

- `staff_profiles` is keyed by `principal_id` and includes `school_id`, with a composite foreign key to the same-school staff user.
- Existing profile fields are employee code, full name, designation, mobile, optional email/joining date, and employment status.
- Employee codes are unique case-insensitively within a school.
- Existing account and profile mutations use transactions and append audit events.
- V57 adds school-scoped education, certification and experience tables. GET/POST/PATCH APIs and the Staff Management profile tab support listing, creating and editing these records; verification is a separate deferred workflow.
- No private staff-document storage/upload/download implementation was found during this source review. Do not implement a public URL, web-root upload, or unauthenticated download.

## 2. Qualifications & Experience data model

Use separate normalized record types so education, professional certifications, and employment history are not overloaded into one generic row.

### 2.1 Staff education

Proposed table: `staff_qualifications`

| Field | Rule |
|---|---|
| id | Generated primary key |
| school_id | Required; derived from authenticated staff context |
| staff_principal_id | Required; same-school staff profile FK |
| qualification_type | Required; controlled values: school, diploma, undergraduate, postgraduate, doctorate, other |
| qualification_name | Required; trimmed, 1–160 chars |
| specialization | Optional; max 160 chars |
| institution | Required; max 200 chars |
| board_or_university | Optional; max 200 chars |
| completion_year | Optional; valid year within a bounded historical/future range |
| score_value | Optional; decimal, non-negative |
| score_scale | Optional; percentage, cgpa, gpa, marks, grade; required when score is present |
| credential_reference | Optional; max 120 chars; do not assume globally unique |
| verification_status | Required; pending, verified, rejected; default pending |
| verification_notes | Optional; max 1000 chars; restricted to verifier/admin access |
| verified_by_principal_id / verified_at | Nullable; set only by verification action |
| created_at / updated_at | System-managed timestamps |

### 2.2 Certifications and training

Proposed table: `staff_certifications`

Fields: id, school_id, staff_principal_id, certificate_name (required, 1–180), issuing_organization (required, max 200), issue_date (optional), expiry_date (optional and not earlier than issue_date), credential_id (optional, max 120), verification_status (pending/verified/rejected; default pending), verification_notes (restricted), verified_by_principal_id, verified_at, created_at, updated_at.

### 2.3 Previous experience

Proposed table: `staff_experience`

Fields: id, school_id, staff_principal_id, employer_name (required, max 200), designation (required, max 120), employment_type (required controlled value: permanent, contract, part_time, temporary, internship, other), start_date (required), end_date (optional and not before start_date), is_current (required; when true end_date must be null), description (optional, max 2000), created_at, updated_at.

Experience rows are self-reported employment history, not verified facts. Do not imply verification unless an explicit verification workflow is added.

### 2.4 Shared invariants

- Each child table has a composite FK `(staff_principal_id, school_id)` to `staff_profiles(principal_id, school_id)`; never accept school_id from request JSON.
- Index by `(school_id, staff_principal_id, created_at)`.
- Use soft archival or a status transition when records have verification/audit history; do not hard-delete verified records.
- All mutations and corresponding audit events are in the same transaction.
- Audit metadata may include record type, record id, changed field names and verification outcome. Never log document bytes or unnecessary personal data.
- Prevent cross-school reads and writes at both handler authorization and repository query boundaries.
- Qualification mismatch is advisory only; do not block teaching assignments without an explicitly approved school policy.

## 3. API contract

Routes are authenticated School Admin endpoints. School ownership comes only from the authenticated principal context.

| Method | Route | Permission | Purpose |
|---|---|---|---|
| GET | `/school/staff/:principalID/qualifications` | `users.read` | List education records |
| POST | `/school/staff/:principalID/qualifications` | `users.update` | Create education record |
| PATCH | `/school/staff/:principalID/qualifications/:recordID` | `users.update` | Update an education record |
| POST | `/school/staff/:principalID/qualifications/:recordID/verify` | Not implemented | Future verification route; requires a separate permission decision |
| GET | `/school/staff/:principalID/certifications` | `users.read` | List certifications |
| POST | `/school/staff/:principalID/certifications` | `users.update` | Create certification |
| PATCH | `/school/staff/:principalID/certifications/:recordID` | `users.update` | Update a certification |
| GET | `/school/staff/:principalID/experience` | `users.read` | List previous experience |
| POST | `/school/staff/:principalID/experience` | `users.update` | Create experience |
| PATCH | `/school/staff/:principalID/experience/:recordID` | `users.update` | Update experience |
| GET | `/school/staff/:principalID/audit-events` | `school.audit.read` | Read up to 100 staff-scoped audit events |

Create/update requests must reject unknown enum values, malformed dates, overlong strings, invalid score/scale combinations, and cross-school target IDs. Return stable field-level validation errors. Return a generic 403 response for unauthorized or cross-school targets without disclosing target details.

### 3.1 School-wide qualification and certification report (planned; not implemented)

The profile APIs above are intentionally staff-scoped. The Reports workspace must **not** call either profile endpoint once per employee. Before adding the school-wide Qualifications & Certifications report, implement one aggregate read endpoint:

| Method | Route | Permission | Purpose |
|---|---|---|---|
| GET | `/school/staff/qualifications-report` | `users.read` | Paginated school-scoped education and certification report |

Contract:
- Derive `school_id` exclusively from authenticated context; never accept a school identifier from query parameters.
- Support optional `record_type=qualification|certification|all`, `verification_status=pending|verified|rejected`, and bounded `limit` / `offset` (default 50, maximum 200).
- Return a single normalized row shape with record type, record ID, staff principal ID, employee code, staff display name, record display name, institution/issuing organization, completion/issue/expiry dates where applicable, and verification status. Do not return credential references, verification notes, contact details, or document/storage metadata in this report.
- Return total counts by record type and verification status for the same filter scope, plus pagination metadata. Counts and rows must be computed in a bounded number of SQL statements, not per-staff queries.
- Implement school scoping in every repository query. Join each record to `staff_profiles` on both principal and school IDs. Do not include archived/inactive staff unless an explicit filter is later approved.
- Use stable ordering (record type, staff name, record ID); validate filters and pagination strictly.
- Enforce live `users.read` permission at the route and repository/service boundary. A report permission must not imply verification or mutation authority.
- Add handler, repository and PostgreSQL integration tests for empty results, mixed record types, filters, pagination, count consistency, same-school staff joins, cross-school isolation and missing permission.
- CSV export must use only returned authorized report fields and the existing formula-injection-safe CSV helper.

This endpoint is a prerequisite for the frontend report. Do not substitute client-side N+1 requests or fabricate totals. No schema migration is expected for this read-only report.

The existing School Admin system-role permission bundle is immutable. Qualification, certification and experience read/create/update operations use the existing `users.read` and `users.update` permissions, enforced by route middleware and repository-level School Admin checks. Verification remains intentionally unavailable: do not reuse `users.update` as verification authority or silently expand the School Admin permission bundle.

## 4. Documents — storage gate before implementation

The document metadata API and file transport must not be shipped until a private storage adapter is selected and tested. ₹0 budget means do not assume a paid cloud bucket or scanning service.

Minimum acceptable storage decision:
- Private storage outside the public web root; no guessable/public object URLs.
- Random opaque storage key, never a client-supplied path or filename.
- Server-side content-type and file-signature validation; allow-list formats and a documented size limit.
- Filename sanitization for display only; never use the submitted filename as a filesystem path.
- Authenticated download handler checks current live permission, school ownership, staff ownership and document state on every request.
- Storage adapter must prevent path traversal and symlink escape if local disk is selected.
- Upload writes to a temporary object, validates fully, then atomically marks metadata available; failed writes leave no active metadata.
- No document bytes, secrets, signed URLs or sensitive contents in logs/audit metadata.
- Verification status and document deletion/archive are audited; retention and backup treatment are documented.
- Malware scanning is desirable, but if no scanner is available, restrict formats and state the residual risk rather than claiming files are scanned.

After the storage decision, proposed metadata table: `staff_documents` with id, school_id, staff_principal_id, category, display_name, original_filename (sanitized), detected_mime_type, size_bytes, opaque_storage_key, issue_date, expiry_date, verification_status, verification_notes, uploaded_by_principal_id, verified_by_principal_id, verified_at, archived_at, created_at, updated_at. Do not store file bytes in this table.

Proposed routes:
- `GET /school/staff/:principalID/documents` — list metadata only, permission `staff.documents.read`.
- `POST /school/staff/:principalID/documents` — authenticated multipart upload, permission `staff.documents.manage`.
- `GET /school/staff/:principalID/documents/:documentID/download` — authorized streaming download, permission `staff.documents.read`.
- `PATCH /school/staff/:principalID/documents/:documentID` — metadata/verification state, permission `staff.documents.manage`.
- `POST /school/staff/:principalID/documents/:documentID/archive` — audited archive, permission `staff.documents.manage`.

No route may return a public storage URL. The download endpoint must stream the private object after authorization and set safe content-disposition/nosniff headers.

## 5. Verification required before marking complete

Backend:
- V57 migration up/down must be tested on a disposable PostgreSQL database only; the source commit alone is not proof of runtime migration success.
- Unit tests for validation, enum/date/score combinations and service errors.
- PostgreSQL integration tests cover create/list/update, transactional audit, staff-scoped audit history and cross-school denial. Dedicated verification remains pending behind a separate permission decision.
- Authorization tests for missing permission, DENY override, School Admin delegation and out-of-school target.
- Document tests for MIME spoofing, size limit, path traversal, unauthorized download, cross-school access, failed upload cleanup and archive audit.

Frontend:
- Real API loading, empty, error/retry, permission-denied and success states.
- Field-level validation below each field; preserve non-secret values after recoverable errors.
- Verification status and verifier notes shown only to authorized roles.
- No mock upload, fake verification or placeholder success.
- Responsive layout and keyboard/screen-reader coverage.

## 6. Explicit non-actions in this checkpoint

- V57 migration source is committed but has not been applied to any user database.
- No local or persistent database was changed.
- No document upload/download code was added.
- No Academic, Student Management or Settings files were changed.
- No qualification-based assignment blocking was introduced.

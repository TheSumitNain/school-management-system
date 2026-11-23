# Staff Bulk Import — architecture gate and implementation contract

**Status:** Design gate documented; import write path is not implemented.  
**Date:** 2026-11-23  
**Branch:** `work`

## 1. Why the current schema cannot yet satisfy the approved product workflow

The approved Staff Management product contract separates an employee profile from a login account. A staff member may exist in the school directory without receiving system credentials.

The current database does not support that distinction:

- `staff_profiles.principal_id` is the primary key and a foreign key to `users(principal_id, school_id)`.
- The existing create-with-profile workflow creates a login account and profile together in one transaction.
- The staff directory is currently sourced from `users` with a left join to `staff_profiles`.
- The current profile model is limited to employee code, full name, designation, mobile, email, joining date and employment status. It does not currently represent Staff Type, Department, Employment Type, date of birth, gender, photograph, current/permanent address, emergency contact, reporting manager or branch/work location.

Consequently, a bulk import that uses the current create endpoint would force every imported employee to have a login username and password. Generating temporary passwords, inventing usernames, or silently provisioning login identities would violate the approved separation of employment and access.

## 2. Safe decision

Do not implement an import endpoint by looping over the existing create endpoint. Do not generate credentials, create default passwords, or treat login provisioning as an import side effect.

Do not add a database migration or alter existing profile/account relationships without explicit review and approval. No persistent database has been changed for this design.

## 3. Required schema/API design before full import
The full identity transition, dependent-table compatibility map, sensitive personal-details boundary, expand/contract rollout and approval gates are documented in [Staff Identity and Profile Schema Design](./STAFF-IDENTITY-SCHEMA-DESIGN.md). That design remains proposal-only and does not authorize database changes.


The target design must support an employee profile with its own stable staff identifier and school ownership, independent of an optional auth principal. Login association should be optional and separately managed. Existing staff profiles and their principal/account relationships must be preserved through a reviewed backfill/compatibility migration.

Before implementation, design and review:

1. A school-scoped staff identity/profile key independent of `users`, with school-scoped employee-code uniqueness.
2. Optional, unique login-principal association with same-school enforcement.
3. Staff Type, Department, Employment Type and Branch/Work Location only after their controlled values/master-data contracts are defined; do not create duplicate free-text master data.
4. Define approved personal/emergency-contact fields (DOB, gender, addresses, emergency contact and photograph) with explicit field-level access, retention and audit requirements before storing them.
5. Compatibility for existing attendance, leave, qualifications, certifications, experience, timetable and assignment foreign keys that currently refer to principals.
6. Audit and lifecycle behavior for employees without login accounts and for later account linking/unlinking.
7. A migration/backfill and rollback/restore plan. Apply it to persistent databases only after explicit approval and verified backup/restore.

## 4. Bulk import acceptance contract

Once the identity model is approved, the import workflow will:

- Accept CSV first; XLSX support is a separate parser dependency decision.
- Provide a downloadable, versioned template and explicit column mapping.
- Require Staff Type, Full Name, Employee Code, Designation and Mobile. Optional fields must match the final approved schema.
- Validate every row before any write; report row number, field, issue and correction guidance.
- Detect case-insensitive duplicate employee codes within the file and existing school records.
- Never import passwords, session tokens, roles or permission overrides.
- Show a preview and exact create/skip/error counts before explicit confirmation.
- Use a single all-or-nothing transaction for the confirmed valid batch. If any row fails at commit time, roll back the whole batch and report the row-level conflict; do not partially create staff.
- Audit the batch and each created employee without recording personal secrets.
- Recheck live `users.create`/staff-create permission and authenticated school scope on the server. Never accept a client-supplied school ID.
- Be idempotent against duplicate submission (request/batch key) and enforce a bounded row count/file size.

## 5. Interim capability

The current application can only create a staff profile together with a login account. The existing Add Staff wizard always collects a username and initial password; it does not yet implement the roadmap's explicit Create Login Account: Yes/No decision. This is a known product/schema gap, not an approved permanent requirement. Until the identity model is reviewed and approved, do not claim that profile-only single or bulk onboarding is supported, and do not create placeholder accounts or credentials to simulate it. A profile-only bulk import and the profile-only single-create path are both blocked by the schema gate above. The existing account-plus-profile workflow remains available as an interim capability.

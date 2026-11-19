# Phase 1 Web RBAC — Roles, Accounts and Scope

**Decision status:** Approved product rules.  
**Implementation status:** Authentication and core RBAC workflows are implemented; complete resource-level enforcement across every business module remains pending. See [Project Status](../project/STATUS.md).

## Phase 1 delivery boundary

Phase 1 is the school web application: Platform Admin, School Admin, Teachers and approved school staff. The immediate delivery priority is completing the school dashboard and its operational workflows.

Parent is not an active Phase 1 login role. **Do not implement Parent OTP authentication, OTP/SMS delivery or parent-facing login flows now.** Parent authentication and interfaces are deferred to Phase 2, after the Phase 1 school web product is complete. Existing guardian relationship foundations may be retained without expanding this workstream.

## Authorization principles

1. Default deny.
2. Permissions define what an actor may do; tenant ownership and resource assignments define which records they may access.
3. The backend enforces every protected request. Frontend navigation and hidden controls are not security controls.
4. Role and permission changes take effect on the next protected request.
5. Permission overrides never bypass school, resource, assignment, field-level or account-status boundaries.
6. Security-sensitive changes and authorization failures are auditable without logging secrets.

## Roles

| Role | Scope | Account rule |
|---|---|---|
| Platform Admin | Global platform | Manages platform operations and provisions schools/School Admins; global scope does not automatically grant access to school academic records. |
| School Admin | Exactly one school | Manages the assigned school only. Cannot create/promote Platform Admins or another School Admin. |
| Teacher | Exactly one school | Access is limited by permissions and exact teaching/class-in-charge assignments. |
| Parent | Phase 2; verified linked children | Intended read-only access to officially linked children, potentially across schools. Not an active Phase 1 workflow. |

## Staff account rules

- Each staff account belongs to exactly one school. A person employed by multiple schools receives separate accounts.
- Staff usernames are globally unique and case-insensitive.
- School Admin provisions staff with an initial temporary password.
- Staff self-service password change requires the current password.
- School Admin may reset a staff password within the same active school without the old password.
- School Admin accounts are provisioned by Platform Admin.

## Teacher data scope

- A class-in-charge teacher may access attendance and necessary management data only for the assigned class/section.
- A subject teacher may access limited teaching data only for assigned classes/sections and subjects.
- The approved minimal roster fields are student name, roll number, class and section.
- Teacher access does not imply access to full student profiles, family/guardian contacts or financial data.
- A frontend must never rely on client-side filtering to enforce these boundaries.

## Parent and student rules — Phase 2 only

- No separate student login accounts or student passwords are part of Phase 1.
- Parent OTP authentication and OTP/SMS delivery are deferred. Do not implement them during the current school-dashboard workstream.
- Parent identity is intended to be linked only after OTP verification and a matching official guardian relationship when Phase 2 is approved.
- One parent may have multiple linked children, including across schools; both parents may have separate verified accounts for one child.
- Every parent request must re-check the current authenticated parent-child relationship server-side.
- Parent-facing interfaces and login workflows remain paused until Phase 1 school web delivery is complete.

## Permission catalogue and delegation

- Permission identifiers are platform-defined and stable.
- Custom roles may be assembled only from catalogue permissions the actor is authorized to delegate.
- Custom-role grants and user ALLOW/DENY overrides must retain the exact delegation authorization.
- DENY takes precedence over ALLOW.
- Role/permission changes are resolved live from the database on protected requests.

## Implementation caveat

These are approved rules, not proof of complete enforcement across all modules. Every new route and repository operation must add explicit tenant/resource/assignment/field checks and corresponding positive and negative tests.
## School Admin permission-management experience

- The School Admin can create school-specific custom roles and grant only permissions that the Platform Admin has actively delegated to the school for role assignment.
- Built-in system roles (including the standard Teacher role) are protected; do not expose direct editing of their permission bundles. Use a school custom role or a justified individual staff override when an authorized exception is needed.
- Staff Management supports assigning/removing roles and individual permission overrides. Overrides are limited by active, exact delegation, require a reason, and follow DENY-over-ALLOW precedence. They never bypass school ownership, resource assignments, field-level controls or account status.
- A class-in-charge assignment is a resource-scope relationship, not a blanket permission. It must restrict class/section data to the exact active assignment. Subject teachers remain limited to their assigned subjects/classes/sections.
- The product should eventually provide a clear School Admin access matrix showing, per module and action, what each role can view/create/edit/delete, plus a preview of effective access for a selected staff member. This is a usability enhancement; do not imply that the current UI already offers a complete matrix.
- Keep medical, guardian-contact, document and other sensitive student fields separately protected. A roster permission must not silently imply full Student Profile access.

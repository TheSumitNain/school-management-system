# Staff Substitutions — backend contract

## Lifecycle and eligibility

- A substitution is school-scoped and requires both the original teacher and substitute to be active staff principals with active Teaching profiles and active same-school staff accounts.
- The original recurring timetable entry is not rewritten. A dated substitution is a separate, auditable record.
- A staff account cannot be disabled while it is the original teacher or substitute on any active substitution dated today or later in Asia/Kolkata.
- The School Admin must first cancel or reassign every affected active substitution. Past-dated active rows do not block account disable because their coverage date has elapsed.
- This rule is enforced by a database trigger on canonical principal status changes, not only by the UI or repository. It therefore also covers direct SQL and the staff-profile inactivation trigger.
- The repository returns ErrStaffProfileHasActiveSubstitutions for the known lifecycle conflict so the UI can explain why disable/inactivation is blocked. Other database failures remain errors and must not be reported as successful disable operations.

## Concurrency guarantees

- New substitution inserts first take the same refresh-principal advisory locks used by account/profile lifecycle operations, in ascending principal-ID order, then lock both canonical principal rows in that order. They re-check that both canonical principals remain active staff before the V60 profile/assignment validators run. This preserves the established advisory → principal → profile lock order and prevents profile-inactivation deadlocks.
- Disabling a principal takes the same database row lock. If substitution creation wins, the disable trigger sees the committed active substitution and rejects the disable; if disable wins, the later substitution insert sees the disabled principal and is rejected.
- The trigger lock order is deterministic for both teachers to avoid opposite-direction teacher-pair deadlocks.
- The V70 migration is additive and has a down migration that restores the previous V62 substitute-only trigger. No existing substitution rows are deleted or rewritten.

## Audit and verification

- Substitution create/cancel operations retain their existing append-only audit events and transaction boundaries.
- PostgreSQL integration coverage asserts that both the original teacher and substitute cannot be disabled while a future active substitution exists, and that their canonical account status remains active after the rejected operation.
- Full backend CI must apply the complete migration chain, run all Go/PostgreSQL integration tests, and run go vet before this change is considered verified.

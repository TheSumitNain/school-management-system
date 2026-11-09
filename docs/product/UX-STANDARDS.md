# Product UX and Usability Standards

**Status:** Mandatory for every current and future user-facing workflow.

These standards exist so school owners, administrators, teachers and other in-scope users can complete routine work without repeatedly contacting the support team. The interface must explain what the user is doing, what information is needed, what went wrong and how to recover.

## 1. Interaction choice

- Use the shared frontend `Modal` for focused create, edit, update, reset and multi-field tasks that would otherwise insert a form into a list or shift the page layout.
- Keep the underlying page and list stable while the dialog is open.
- Use a clear confirmation dialog for destructive or security-sensitive actions. Explain the consequence before the user confirms.
- Do not use a modal for simple search, filtering, navigation or quick reversible inline actions where inline interaction is clearer.
- Do not create page-specific modal overlays. Use the shared component and consistent actions.
- Avoid modal chains and repeated questions. Keep related steps together where practical.

## 2. Field-level validation

- Every field has a visible, specific label and clear required/optional indication.
- Provide examples and helper text for non-obvious formats, length limits and constraints.
- On submit, validate all fields and display every detected error together directly below its corresponding field.
- A generic top/bottom error box must never be the only indication of field validation errors.
- Associate each field message through `aria-describedby`, set `aria-invalid` when invalid and use actionable wording.
- Clear the relevant field error when the field is corrected. Do not interrupt typing with premature validation.
- Preserve safe entered values after validation or recoverable failures. Clear password values when security-sensitive behavior requires it.
- Map structured server validation errors to their matching fields. Use a form-level alert only for non-field/server failures; a summary may supplement but never replace field errors.

## 3. Dialog accessibility and behavior

- Use a meaningful task title and description; identify the selected record where applicable.
- Provide keyboard focus management, focus restoration, Escape behavior, visible close/cancel action, background scroll locking and responsive sizing.
- Prevent dismissal or duplicate submissions while a mutation is pending if doing so could leave an ambiguous result.
- Respect light/dark themes, keyboard-only interaction, screen readers, touch targets and reduced motion.

## 4. Guidance and recovery

- Every page explains its purpose and primary next action in plain language.
- Empty states explain why the list is empty and how to get started.
- Search/filter no-result states suggest a useful adjustment.
- Permission-denied states explain which access is missing and whom to contact when that information is available.
- Errors explain what happened in user language and give a recovery action: correct a field, retry, change a filter or contact the responsible administrator.
- Do not expose raw SQL, stack traces, internal codes, credentials, tokens or other implementation details.
- Explain important consequences such as session revocation, account deactivation, password change requirements and retained archive records.

## 5. Consistent action states

- Show a clear pending state and disable duplicate submissions.
- Confirm successful completion in plain language and explain the next logical step.
- Keep feedback near the affected field, form, record or workflow.
- Handle loading, background refresh, empty, validation, API/server, network/offline, permission, success and session-expiry states.
- Do not rely only on color or transient toast notifications.

## 6. Review gate

Before a page or workflow is considered complete, review every user action and form against this document. Record intentional exceptions with their usability reason. Verify desktop/tablet/mobile, keyboard and screen-reader operation, both themes, failed requests, retries and duplicate-click behavior. Run the production TypeScript/Vite build and relevant tests. A page is not complete merely because its API call succeeds.

## 7. Correct use of inline interaction

Inline controls remain appropriate for high-frequency, low-complexity actions such as attendance status selection per student, search, filters, pagination and quick reversible toggles. Do not force users through a modal for every click. The goal is clarity and predictable behavior, not maximizing modal usage.


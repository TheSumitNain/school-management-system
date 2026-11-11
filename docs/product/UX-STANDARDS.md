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
- Every editable text, email, password, search, number and other value-entry field must provide a meaningful placeholder that shows the expected kind or format of data (use realistic, non-sensitive examples such as `e.g. ADM-2026-001`). Do not use placeholders as labels or repeat generic text such as `Enter value`.
- Required dropdowns must show a task-specific prompt such as `Select class` or `Choose teacher`; optional filters must clearly communicate their unfiltered state (for example, `All statuses`).
- Date controls must explain what date is expected through a contextual prompt or adjacent helper text. For native date/time/number controls where browsers may hide or ignore placeholders, provide visible helper text with a concrete format/example.
- Optional fields must say they are optional in the label or nearby helper text and, where useful, show an example placeholder.
- Keep placeholders concise and context-specific, preserve persistent labels, and ensure placeholder/helper contrast is readable in every theme. Existing values, validation messages and accessible descriptions must remain visible and must not depend on placeholder text.
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

### Dialog action order and visual consistency (mandatory)

- Whenever a dialog provides both a secondary dismissal action and a primary task action, render **Cancel first and the primary action second**. This is the consistent left-to-right order on desktop and the consistent first-to-last order on narrow screens and in keyboard navigation.
- Cancel must always use the shared neutral/secondary dialog-button treatment: theme-matched surface, text and border tokens, consistent height, padding, radius, typography, hover and visible focus states. It must not look like a primary or destructive action.
- Save, Create, Update, Assign and other task-completion actions use the shared primary button treatment and remain visually more prominent than Cancel.
- A destructive confirmation action (for example Delete or Disable) remains second but uses the shared danger treatment. Its consequence must be explained before confirmation.
- Use the shared `.app-modal-actions` container and shared modal styles. Page-specific CSS must not redefine dialog Cancel button colors, dimensions or interaction states. Do not reverse action order to accommodate a page-specific layout.
- Keep action spacing, alignment, disabled/pending behavior and responsive sizing consistent across Platform Admin, School Admin and all future role-based workflows.

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

## 8. Button and icon alignment

- Primary action buttons use a shared inline-flex layout with centered cross-axis alignment and a consistent icon-to-label gap. Icons must not sit on the text baseline or shift the label vertically.
- Decorative/action icons inside buttons must not shrink; icon-only buttons use a fixed, centered hit area with an accessible name.
- Buttons that combine an icon and text must use a consistent inline-flex alignment, spacing, height, typography and focus treatment. Do not fix alignment on one page with arbitrary icon margins or transforms when a shared button rule is appropriate.
- Before shipping a page, compare its primary, secondary, refresh, row-action and destructive buttons with equivalent actions elsewhere. Keep visual differences intentional and documented.

## 9. Date picker accessibility and keyboard behavior

- A calendar exposed as an ARIA grid must use the complete grid structure: one weekday-header row, week rows, and grid cells containing the date controls. Do not apply `role="grid"` to a loose collection of buttons.
- Use roving tab focus so the calendar contributes one tab stop, not one tab stop per date. The active date control must remain visible and focusable; unavailable dates must be disabled and excluded from roving focus.
- Support Arrow keys for one-day/one-week movement, Home/End for the start/end of the displayed week, Page Up/Down for previous/next month, and Shift+Page Up/Down for previous/next year.
- Keep keyboard navigation within configured minimum/maximum dates. Month navigation controls must be disabled when the adjacent month contains no selectable date.
- Give each date control an accessible full-date name. Expose the selected date on its grid cell and distinguish today without relying on color alone.
- Escape closes the calendar and restores focus to its trigger. Selecting a date, clearing it or choosing Today also returns focus to the trigger. Clicking outside closes the calendar without stealing focus from the outside target.
- Verify keyboard-only operation, screen-reader announcements, focus visibility, date bounds, narrow screens and both themes before considering a date-picker workflow complete.

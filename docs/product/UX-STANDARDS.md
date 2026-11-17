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

## 10. Scroll-aware secondary navigation

- On long pages with a secondary tab/navigation row, keep the row available without permanently consuming viewport space: hide it after meaningful downward scrolling once it reaches the top navigation area, and reveal it when the user scrolls upward.
- Keep the primary application header above the secondary navigation; use theme-aware opaque surfaces and appropriate stacking so content never bleeds through the navigation.
- Do not change the user's scroll position when showing or hiding navigation. Reveal the navigation when keyboard focus enters it, and keep it visible at the top of the page.
- Ignore tiny scroll movements to prevent flicker, support narrow screens with horizontally scrollable tabs where needed, and respect reduced-motion preferences.
- Apply this pattern consistently to long tabbed workflows where it improves navigation; do not add it to short pages that do not need it.

## 11. Help tooltips, links and contextual notices

- Help icons start closed. Opening a dialog or automatically focusing a form control must never open a tooltip as a side effect.
- Help content opens only through intentional pointer hover, keyboard focus or touch/click interaction. Every trigger has an accessible name and expanded/description semantics.
- Tooltips must remain readable within the viewport and must not be clipped by modal, card, table or scrolling-container overflow. Reposition them above/below and horizontally as needed; users must not need to scroll the form merely to read help.
- Action links such as **Create location selection**, **Add missing State/UT** or similar secondary workflow actions must look interactive: theme-aware link color, clear hover/focus state and underline or equivalent non-color affordance. Do not style a navigation/action link as plain body text.
- Reference-data guidance must communicate its meaning explicitly. Use distinct information, warning and error treatments with a short heading or icon plus explanatory text and the next action. Never rely on color alone or render a missing-reference warning as an unstyled small-print sentence.

## 12. Save only actual changes

- Opening an edit dialog may perform the GET requests required to load the record and its supporting choices. It must not perform create, update, delete, archive, link or other write requests.
- Changing fields in a draft must not call a write endpoint. Writes occur only after the user explicitly submits the relevant action.
- Before sending an update, compare the normalized draft with the persisted values. If nothing changed, show a concise **No changes to save** message and do not send PUT/PATCH.
- Apply the same rule to basic details, address, contact, medical, physical, guardian relationship and shared guardian profile forms. Preserve intentional append-only actions (for example, creating a new dated measurement) only when the user actually changes the record/date and submits it.
- A GET caused by opening a modal or refreshing stale data is expected; a write caused solely by opening a modal is a defect. Tests must assert both behaviours separately.

## 13. Predefined relationship and classification choices

- Use a predefined select for common structured relationships instead of requiring free-text entry for standard values.
- Guardian relationship choices should include Father, Mother, Step-father, Step-mother, Grandfather, Grandmother, Brother, Sister, Uncle, Aunt, Legal Guardian, Foster Parent and Other.
- When Other is selected, reveal a clearly labelled custom relationship field and validate its length. Preserve existing custom values when editing historical records.
- Do not invent guardian identity or contact details from the relationship choice; guardian name, phone and email remain user-entered family information.

# Frontend UI/UX Standards

These standards apply to every frontend feature and page. A feature must reuse shared components and design tokens instead of introducing a page-specific visual pattern.

## Shared visual language

- Use the shared theme tokens for color, typography, spacing, borders, radii, shadows, and control heights. Do not hard-code a competing palette or sizing system in feature styles.
- Use the shared `Select` component for searchable/controlled application dropdowns where practical; do not mix browser-native selects with the shared dropdown design on the same workflow.
- Keep labels, helper text, required indicators, validation messages, and control spacing consistent across pages.
- Buttons must use the shared primary, secondary, or destructive visual hierarchy. Every action button needs consistent height, padding, focus-visible, hover, disabled, and loading states. Do not leave standalone buttons with browser-default styling.
- Reuse shared modal/dialog components and shared confirmation patterns. Do not use native browser `alert()` or `confirm()` for application workflows.

## Forms and validation

- Every form must use explicit, descriptive labels and indicate required fields.
- Validate all fields together on submit and show each validation message directly below its field. Do not rely only on a page-level error banner.
- Associate errors with controls using `aria-invalid` and `aria-describedby`; use an accessible form-level message only for non-field/server errors where useful.
- Preserve entered values when validation or server requests fail. Disable relevant controls while a mutation is in progress and show a clear loading label.

## Feedback and notifications

- Use the global `notifyToast` / `ToastViewport` pattern for successful create, update, and delete operations and for actionable mutation failures.
- Emit success notifications only after the server confirms the operation. Never show success optimistically before a request completes.
- Toasts supplement, rather than replace, inline field errors and important contextual information. Avoid duplicate persistent success banners for the same operation.
- Use clear, concise messages that identify what changed. Use error styling for failures and do not hide server validation details.

## Destructive and consequential actions

- Use an in-app confirmation modal for destructive or consequential actions. Explain the impact, identify affected records, and show exact counts when available before the user confirms.
- Make the destructive action explicit in its button label; provide a safe Cancel action. Never silently cascade changes that affect student records or selections.
- The backend remains responsible for authorization, validation, transactionality, and stale-impact checks; a confirmation dialog is not a security boundary.

## Post-mutation navigation and focus

- After creating a record, refresh the displayed data and bring the newly created record into view. For long pages, scroll the relevant result/list region into view rather than leaving the user at the form footer.
- After editing, keep the updated record visible and preserve the user's context.
- After closing a modal, restore focus to the control that opened it. After inline actions, maintain a predictable keyboard and screen-reader focus order.
- Respect reduced-motion preferences when using smooth scrolling or transitions.

## Content-first responsive layout (mandatory)

- Layout must adapt to its content; content must never be distorted merely to fit a breakpoint or a single row.
- Do not shrink cards, columns, form controls, text, action buttons or touch targets below readable/usable sizes to force a one-line layout.
- When a row no longer fits, wrap its items onto the next line with consistent spacing. Keep labels with their controls and keep related actions grouped.
- Cards and dashboard grids must reflow into fewer columns and then one column as space narrows. Preserve the full card content and readable hierarchy.
- Tables must preserve all columns and cell values. Use a clearly bounded, touch-friendly horizontal scroll region when columns cannot fit; do not hide columns, truncate important values, remove headers, or squeeze every column into unreadable widths. If a deliberate compact/mobile card representation is used, it must expose every field and its value.
- Sidebars/navigation may use the established mobile drawer pattern, but every navigation item must remain reachable; do not clip or silently hide navigation options.
- Forms must use min-width: 0 and content-aware grid/flex wrapping. Controls must remain fully visible, labels and helper text may wrap, and action buttons must move to another row rather than overlap or disappear.
- Avoid fixed heights for content containers. Let cards, rows, form fields, helper text, validation messages and buttons grow naturally. Use overflow only for a deliberately scrollable region, never to conceal meaningful content.
- Long names, emails, identifiers, status labels and translated text must wrap or have an accessible way to inspect the full value. Do not use ellipsis for essential student/staff/record data.
- Test at narrow phone widths, common tablet widths, desktop widths and landscape. Assert page-level overflow is absent while checking that intentionally scrollable tables/tab strips retain all their content and remain operable.

## Field-label composition

- Keep the field name, Required/Optional/conditional status and information/help icon visually grouped as one label unit.
- Use a consistent small gap between the status indicator and info icon; align their visual centers and baseline. Do not leave a large unexplained blank gap between them or place the info icon far away from its field.
- Let long field names wrap without pushing the status or help control off-screen. Keep the info icon's touch/focus target usable and its tooltip within the viewport.
- Required/Optional status must remain text-readable; do not rely on red/green color alone. Apply the same hierarchy and spacing in inline forms, modals, cards and narrow layouts.

## Mutation feedback coverage

- Audit every create, add, edit, update, remove, delete, archive, restore, assign, unassign, reset and bulk action on every in-scope page.
- Each completed mutation must show a concise global toast naming the action and affected record/context, only after the server confirms success.
- A failed mutation must show a useful error toast or persistent form-level error with a recovery hint; field-specific server errors remain directly below their fields.
- A no-op save must be disabled whenever the form can determine it is unchanged, and must be blocked again in the submit handler as a final guard. It must not send a write request or show a success toast; show a neutral informational toast such as **No change to save**.
- Destructive actions require a clear confirmation step and the success toast must describe what was actually removed/changed. Never show success when a request failed, was cancelled or only partially completed.
- Verify duplicate clicks, pending/loading state, network/server failures, validation failures and retry behavior. Avoid duplicate success banners and duplicate toasts for the same action.

## Acceptance checklist

- [ ] Controls visually match equivalent controls on other pages and in both light and dark themes.
- [ ] Primary, secondary, destructive, disabled, loading, and keyboard-focus states are present.
- [ ] Every invalid field displays its own accessible inline error.
- [ ] Successful mutations show a global toast only after server confirmation; failures show a useful error.
- [ ] Destructive operations explain the impact and require explicit confirmation.
- [ ] Created/updated records are visible after the operation without requiring the user to scroll manually.
- [ ] Responsive layout remains usable on narrow screens.

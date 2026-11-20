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

## Acceptance checklist

- [ ] Controls visually match equivalent controls on other pages and in both light and dark themes.
- [ ] Primary, secondary, destructive, disabled, loading, and keyboard-focus states are present.
- [ ] Every invalid field displays its own accessible inline error.
- [ ] Successful mutations show a global toast only after server confirmation; failures show a useful error.
- [ ] Destructive operations explain the impact and require explicit confirmation.
- [ ] Created/updated records are visible after the operation without requiring the user to scroll manually.
- [ ] Responsive layout remains usable on narrow screens.

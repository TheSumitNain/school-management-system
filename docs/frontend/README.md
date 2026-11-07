# Frontend Documentation

<badge>Implementation in progress</badge>

The web stack is React + TypeScript + Vite.

## Phase 1 users and delivery priority

The first frontend delivery must complete both administrator workspaces:

1. **Platform Admin workspace** — global platform operations, school onboarding, Platform Admin lifecycle, permission delegation and audit visibility where supported by backend APIs.
2. **School Admin workspace** — school operations, staff, roles and permissions, academic setup, students, attendance and guardians, subject to available backend APIs and authorization.

After both administrator workspaces are usable, continue with Teacher/staff workflows. Parent OTP login, OTP/SMS delivery, parent dashboards and Parent/Driver mobile interfaces are Phase 2 and remain deferred.

## Frontend UI and styling requirements

These requirements apply to every frontend page and shared component:

- **Responsive by default:** layouts must work on mobile phones, tablets, laptops and wide desktop screens. Navigation, tables, forms, dialogs and dashboards must adapt to smaller viewports without horizontal overflow or unusable controls.
- **Separate CSS files:** keep styling in dedicated CSS files (or CSS modules where the project convention supports them). Do not put large page/component styles inline or mix styling concerns into application logic. Reuse shared styles and design tokens.
- **One centralized design system:** login, MFA, onboarding, dashboards, settings, tables, dialogs and every other page must use the same shared design tokens and reusable components. No page may invent its own unrelated visual style.
- **Centralized visual tokens:** define and consistently use semantic tokens for primary and secondary colors, page and elevated backgrounds/surfaces, text and muted text colors, borders, status colors, focus rings, shadows, typography/font family, font sizes, font weights, line heights, spacing, sizing, radii, control heights, breakpoints and layering/z-index. Avoid scattered hard-coded color, font-size and spacing values.
- **Consistent sizing and typography:** establish a documented type scale and spacing scale. Buttons, inputs, labels, headings, cards, tables and alerts must have consistent sizing and alignment across routes.
- **Extensible theme system:** implement a theme architecture that supports at least Light and Dark themes initially and can add more themes later without rewriting individual pages. Use centralized semantic design tokens/CSS custom properties for all visual properties. Persist the user's selected theme appropriately and respect the operating-system preference when no explicit choice has been made. Every component and state—including hover, focus, disabled, loading, skeleton, validation, error, success, empty and modal states—must render correctly in every theme.
- **Consistent shared components:** use reusable buttons, inputs, selects, tables, cards, badges, dialogs, dropdowns, page headers, breadcrumbs, alerts, tooltips and navigation components. Prefer composition over duplicating page-specific implementations.
- **Consistent app shells:** use shared layout and components with permission-aware navigation and role-specific routes; do not create duplicated role-specific shells.
- **Polished alignment and interaction:** maintain consistent grids, spacing, alignment, responsive behavior, transitions and interaction feedback. Avoid layout shifts, clipped content, overlapping elements, flicker and inconsistent page-to-page behavior.
- **Accessibility:** provide keyboard navigation, visible focus indicators, sufficient contrast in every theme, semantic controls and labels, and reduced-motion support. Do not communicate status by color alone.

## Performance, loading and reliability requirements

The application must feel fast and remain lightweight. Performance is a product requirement, not a later optimization:

- **Keep the initial bundle small:** avoid unnecessary dependencies and heavy UI libraries. Prefer native platform capabilities and small, focused packages. Review the bundle impact before adding a dependency.
- **Route-level code splitting:** lazy-load substantial route/page modules where it reduces initial JavaScript. Keep the login and first authenticated shell responsive; do not introduce lazy-loading waterfalls or delay critical interactions.
- **Intentional prefetching:** prefetch likely next routes or data when it is useful and safe (for example, on navigation intent or idle time), without downloading every admin module at startup.
- **Efficient API state:** use TanStack Query for server/API state with deliberate query keys, caching, stale times, cancellation, invalidation and retry policies. Do not refetch needlessly or retry non-retryable mutations blindly. Keep client-only UI state separate (for example, in Zustand or React state).
- **Fast rendering:** paginate or virtualize genuinely large lists when appropriate, avoid unnecessary rerenders, keep components appropriately scoped, and debounce expensive search/filter requests. Do not add virtualization where normal lists are small.
- **Loading feedback:** every asynchronous page and action must provide an intentional loading state. Use theme-aware skeletons for content layouts, lightweight spinners for short actions, and disabled/progress states for submitted buttons. Avoid blank screens and avoid showing skeletons for longer than necessary.
- **Complete UI states:** design and implement loading, skeleton, empty, error, success, validation, offline/network and permission-denied states consistently. Provide retry/recovery actions where appropriate and do not expose raw stack traces or sensitive server details.
- **Prevent layout shift:** skeletons should approximate the final content dimensions. Reserve space for images, charts and asynchronously loaded sections.
- **Resilient failure handling:** use an application-level error boundary and appropriate route/component boundaries so one rendering failure does not unnecessarily take down the entire application. Handle failed API requests, expired sessions and network interruptions gracefully. Log actionable diagnostics without secrets or personal data.
- **Safe request lifecycle:** cancel obsolete requests where possible, prevent stale responses from overwriting newer state, and avoid duplicate submissions. Handle refresh/session expiry centrally and consistently.
- **Motion discipline:** keep transitions subtle and short, respect reduced-motion settings, and avoid animations that delay interaction or make the UI feel heavy.
- **Measure rather than guess:** check production build output and bundle sizes, test on realistic mobile/network conditions, and profile slow routes and interactions. Set and maintain sensible performance budgets as implementation matures.

Do not claim that crashes or delays can be made impossible. Instead, engineer for graceful degradation, clear recovery paths, automated checks and reliable behavior.

## Integration and security contract

- The backend is the source of truth for identity, permissions and resource access.
- `GET /auth/me` returns principal identity and, for Staff and Platform Admin, the current sorted effective permission keys.
- Permission keys may guide navigation visibility only. Every API operation must still be authorized server-side.
- The client must handle authentication expiry, logout, MFA challenges/enrollment and temporary-password restrictions.
- Do not trust school IDs, role labels, permission lists or selected student IDs supplied by the browser as authorization proof.
- Never put credentials, MFA secrets, recovery codes or access tokens in source control, logs or user-visible diagnostics.
- Clear user-specific cached server data on logout and account/session changes; do not persist sensitive API responses unnecessarily.

## Work required before and during frontend implementation

1. Confirm the existing frontend source structure and preserve the current application rather than creating a parallel frontend.
2. Establish typed API client, environment configuration and secure session handling.
3. Establish shared design tokens, typography/spacing scales, theme provider, reusable components and app shell before building page-specific styling.
4. Establish server-state, client-state and form-state conventions; avoid overlapping state-management solutions.
5. Complete and verify the Platform Admin workspace.
6. Complete and verify the School Admin workspace.
7. Implement teacher/staff views with the backend's assignment and field-level limits.
8. Implement each workflow against documented API contracts with loading, skeleton, empty, validation, success and error states.
9. Verify responsive behavior at mobile, tablet and desktop widths, in every supported theme.
10. Verify production build, bundle size, route loading, browser workflows and graceful error recovery; ensure backend authorization remains effective when UI controls are bypassed.

## Explicitly deferred

- Parent OTP login and OTP/SMS delivery.
- Parent dashboard and parent-facing web/mobile workflows.
- Driver mobile experience.

Do not create frontend routes, forms, mock authentication or placeholder flows for these deferred features as part of the current Phase 1 scope.

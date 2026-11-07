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
- **Extensible theme system:** implement a theme architecture that supports at least Light and Dark themes initially and can add more themes later without rewriting individual pages. Use centralized semantic design tokens/CSS custom properties for colors, surfaces, borders, text, focus states and shadows. Persist the user's selected theme appropriately and respect the operating-system preference when no explicit choice has been made.
- **Consistent design system:** shared typography, spacing, breakpoints, buttons, inputs, tables, cards, navigation and feedback states should be reusable across both admin workspaces.
- **Accessible interaction:** provide keyboard navigation, visible focus indicators, sufficient contrast in every theme, semantic controls and labels. Do not communicate status by color alone.
- **No role-specific duplicated app shells:** use shared layout and components with permission-aware navigation and role-specific routes/content.

## Integration and security contract

- The backend is the source of truth for identity, permissions and resource access.
- `GET /auth/me` returns principal identity and, for Staff and Platform Admin, the current sorted effective permission keys.
- Permission keys may guide navigation visibility only. Every API operation must still be authorized server-side.
- The client must handle authentication expiry, logout, MFA challenges/enrollment and temporary-password restrictions.
- Do not trust school IDs, role labels, permission lists or selected student IDs supplied by the browser as authorization proof.
- Never put credentials, MFA secrets, recovery codes or access tokens in source control, logs or user-visible diagnostics.

## Work required before and during frontend implementation

1. Confirm the existing frontend source structure and preserve the current application rather than creating a parallel frontend.
2. Establish typed API client, environment configuration and secure session handling.
3. Complete and verify the Platform Admin workspace.
4. Complete and verify the School Admin workspace.
5. Implement teacher/staff views with the backend's assignment and field-level limits.
6. Implement each workflow against documented API contracts with loading, empty, validation and error states.
7. Verify responsive behavior at mobile, tablet and desktop widths, in every supported theme.
8. Verify browser workflows and ensure backend authorization remains effective when UI controls are bypassed.

## Explicitly deferred

- Parent OTP login and OTP/SMS delivery.
- Parent dashboard and parent-facing web/mobile workflows.
- Driver mobile experience.

Do not create frontend routes, forms, mock authentication or placeholder flows for these deferred features as part of the current Phase 1 scope.

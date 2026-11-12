# Frontend Documentation

<badge>Implementation in progress</badge>

The web stack is React + TypeScript + Vite.

The mandatory cross-project usability contract is documented in [Product UX and Usability Standards](../product/UX-STANDARDS.md). Every frontend change must comply with it.

## Phase 1 users and delivery priority

The first frontend delivery must complete both administrator workspaces:

1. **Platform Admin workspace** — global platform operations, school onboarding, Platform Admin lifecycle, permission delegation and audit visibility where supported by backend APIs.
2. **School Admin workspace** — school operations, staff, roles and permissions, academic setup (including optional section medium), students, attendance, timetable and guardians, subject to available backend APIs and authorization.

The current timetable page also supports the Teacher read-only workflow using the current academic year and the teacher's own scheduled entries. Build and verify school operations in the dependency order documented in [Phase 1 School Operations — Task-by-Task Build Order](../product/PHASE1-WORKFLOW-ORDER.md): academic foundations → subjects → teacher/staff accounts → teacher assignments → student admission/enrollment → timetable → attendance → examinations/results → notices/dashboard integration. Reuse existing pages and APIs; do not duplicate Subject Management or rebuild the existing timetable. Parent OTP login, OTP/SMS delivery, parent dashboards and Parent/Driver mobile interfaces are Phase 2 and remain deferred.

## Staff Management

The School Admin Staff Management page maintains employee details separately from login credentials. Employee full name, employee code, designation and one mobile number are required; email and joining date are optional. Employee codes are unique within a school. Employment status is Active/Inactive and is distinct from login-account status. Creating a staff member creates the employee profile and login account in one backend transaction. Inactive employment disables login access; reactivating employment does not automatically enable a disabled account. Existing staff accounts without profile data remain visible as incomplete and must be completed by an administrator.

## Frontend UI and styling requirements

These requirements apply to every frontend page and shared component:

- **Responsive by default:** layouts must work on mobile phones, tablets, laptops and wide desktop screens. Navigation, tables, forms, dialogs and dashboards must adapt to smaller viewports without horizontal overflow or unusable controls.
- **Separate CSS files:** keep styling in dedicated CSS files (or CSS modules where the project convention supports them). Do not put large page/component styles inline or mix styling concerns into application logic. Reuse shared styles and design tokens.
- **One centralized design system:** login, MFA, onboarding, dashboards, settings, tables, dialogs and every other page must use the same shared design tokens and reusable components. No page may invent its own unrelated visual style.
- **Centralized visual tokens:** `src/styles/tokens.css` is the single source of truth for semantic colors and shared design primitives. Component styles must consume CSS variables; do not add literal hex, RGB/HSL or named colors outside this file. Run `npm run check:styles` to prevent hard-coded colors from returning. Also define and consistently use tokens for typography/font family, font sizes, font weights, line heights, spacing, sizing, radii, control heights, breakpoints and layering/z-index in every new or substantially updated component. Migrate existing layout dimensions to shared scales incrementally and verify each change.
- **Consistent sizing and typography:** establish a documented type scale and spacing scale. Buttons, inputs, labels, headings, cards, tables and alerts must have consistent sizing and alignment across routes.
- **Extensible theme system:** support Light, Dark and System appearance modes independently from the selected brand palette (Classic Navy, Forest Green, Royal Purple or Ocean Teal). Add future palettes by defining token values, not rewriting pages. Persist the user's preferences and respect operating-system preference when System is selected. Every component and state—including hover, focus, disabled, loading, skeleton, validation, error, success, empty and modal states—must render correctly in every combination.
- **Consistent shared components:** use reusable buttons, inputs, selects, tables, cards, badges, dialogs, dropdowns, page headers, breadcrumbs, alerts, tooltips and navigation components. Prefer composition over duplicating page-specific implementations.
- **Consistent app shells:** use shared layout and components with permission-aware navigation and role-specific routes; do not create duplicated role-specific shells.
- **Polished alignment and interaction:** maintain consistent grids, spacing, alignment, responsive behavior, transitions and interaction feedback. Avoid layout shifts, clipped content, overlapping elements, flicker and inconsistent page-to-page behavior.
- **Accessibility:** provide keyboard navigation, visible focus indicators, sufficient contrast in every theme, semantic controls and labels, and reduced-motion support. Do not communicate status by color alone.

## Cross-browser and cross-device UI consistency (mandatory)

The application must present one coherent, theme-matched interface across supported browsers and devices, including Chrome, Edge, Firefox, Safari and Apple platforms. Treat browser/OS defaults as implementation differences to manage, not as the product design.

- **Normalize native controls centrally:** define shared CSS for select, option/optgroup, date/time inputs, checkbox, radio, range, buttons, text inputs, textarea, placeholder, focus, autofill, text selection and scrollbars. Use semantic theme tokens for surfaces, text, borders, accent/checked states and focus indicators. Do not rely on browser-default blue selection, focus or accent colors.
- **Avoid per-page browser fixes:** shared control normalization belongs in the global theme layer. Page CSS may set layout and dimensions, but must not reintroduce platform-default control colors or inconsistent control chrome.
- **Select menus:** use the shared `src/shared/components/Select.tsx` component when app-owned popup styling is required. It owns option hover/selected colors and keyboard navigation; do not build one-off dropdowns inside feature pages. Apply it progressively to every user-facing native select, preserving each field's labels, disabled state and value semantics.
- **Date pickers:** use the shared `src/shared/components/DatePicker.tsx` for every user-editable date field; do not introduce page-specific calendars or native `input type="date"` controls. After selection, the field must immediately display the selected date in the consistent `DD MMM YYYY` format; edit forms must display the saved date, and clearing a date must visibly restore its placeholder. Preserve labels, inline validation, keyboard/touch access, theme-matched month navigation, selected/today states and min/max constraints. Keep date-only values in local `YYYY-MM-DD` form and avoid UTC conversion that can shift the selected day. Audit every new or modified page for date fields before considering it complete.
- **Native popup limitations:** style the input field and supported calendar indicator, but do not assume CSS can theme native select or calendar popups consistently. App-owned controls are the route to consistent popup appearance; do not add a one-off custom popup per page.
- **Custom controls must preserve behavior:** do not remove native semantics, keyboard support, labels, validation, form submission, autofill or accessibility merely to make controls look alike. Prefer native controls when their visual differences are acceptable; use custom components only where cross-platform visual parity is necessary.
- **Browser rendering limits:** identical pixel rendering cannot be guaranteed across different OS font rasterizers, system fonts, device pixel ratios and browser engines. Use bundled/consistent typography where licensing permits, explicit sizing/line-height, CSS normalization and responsive acceptance tests to keep the experience materially consistent.
- **Test real engines and devices:** verify Chromium, Firefox and WebKit/Safari, including iOS/iPadOS Safari and macOS Safari where supported. Check light/dark themes, keyboard and touch interaction, zoom, reduced motion, forced-colors/high-contrast mode, autofill, native popup behavior and narrow viewports. Record unavoidable native UI differences and replace them with app-owned components only when needed.

Acceptance criterion: no unintentional browser-default colors or control styling may leak into the product UI. Any remaining native popup difference must be documented and explicitly accepted, or replaced by a shared accessible custom component.

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

## Product-wide UX and session rules (mandatory)

These rules apply to every role and every screen. Treat them as acceptance criteria, not optional polish.

### Authentication, routing and session continuity
- A successfully authenticated user with a server-valid session must not be shown the login screen again just because they open or manually type the login URL. Redirect them to their permitted authenticated landing page.
- Apply the same behavior to Platform Admin, School Admin, Teacher and Parent when their respective login flows are in scope. Role differences must not create inconsistent session behavior.
- On initial load or browser refresh, show a lightweight branded loading/skeleton state while validating the existing session with the backend. Never flash the login page before session validation finishes.
- A token stored in the browser is not proof of authentication. Validate identity/session with the backend before rendering protected content.
- Handle access-token expiry centrally. If the supported refresh-session flow can renew the session, renew it once safely and replay eligible requests without asking the user to log in. Prevent concurrent requests and browser tabs from racing refresh-token rotation; coordinate rotation with the browser's cross-tab lock where supported.
- Use a same-origin persistent HttpOnly, SameSite=Lax refresh cookie (Secure in production). Keep short-lived access tokens in memory only; never persist refresh tokens in localStorage or sessionStorage. The refresh cookie is scoped to the browser-facing `/api/auth` path, uses a 400-day Max-Age and is renewed after each successful refresh. The server-side session has no absolute expiry; actual persistence remains subject to browser cookie-retention policies.
- On app startup, reload and validate the session through the backend before showing login. A new tab in the same browser profile must reuse a valid session without asking the user to sign in again.
- Broadcast logout, account changes and server-confirmed session expiry across tabs. A logout in any tab must revoke the server-side session family, expire the shared refresh cookie, clear user-scoped caches in every open tab and return every tab to login.
- Enforce one active account per browser profile, not one account per physical device. All tabs in the same browser profile share the same authenticated principal; a successful sign-in as another account replaces the active account in every tab. Each tab must clear the previous user's access token and cached data, then revalidate the shared HttpOnly refresh cookie with the backend before rendering the new account.
- Account-change notifications must contain no credentials or tokens. Serialize login, MFA completion, refresh-token rotation and logout with the same cross-tab lock where supported, so an older refresh response cannot overwrite a newly established session. If cross-tab locking is unavailable, retain server-side refresh replay protections and fail safely.
- Different browser profiles and different devices maintain independent sessions; users may sign in on multiple devices.
- If the refresh session is expired, revoked, replay-invalidated or otherwise rejected, clear user-scoped state and send the user to login with a clear, non-technical message. Platform Admin and School Admin sessions do not expire due to inactivity; they remain valid until manual logout or a security/account event revokes them.
- Logout must revoke the server session, clear the shared cookie, in-memory access token and user-scoped caches, and return to login. Account switching must never show the previous user's cached data.
- Protected deep links must return the user to the originally requested page after successful login, only if that page is authorized. Otherwise route to a safe permitted page and explain access denial.
- Browser back/forward, reload and opening a protected URL in a new tab must preserve correct authentication and authorization behavior.
- Platform Admin and School Admin must complete MFA on every fresh login. Background access-token refresh and restoration of an existing valid session must not repeat MFA. MFA enrollment and temporary-password change remain authentication gates; do not bypass them through direct URLs or client-side state manipulation.
- Never log, display in diagnostics, or ask users to share passwords, access/refresh tokens, MFA secrets, OTPs or recovery codes.

### Low-friction interaction rules
- Do not make users repeat information already provided in the current workflow. Preserve safe, non-secret form values after validation or recoverable server errors; clear password fields when appropriate.
- Prevent accidental duplicate submissions. Disable the relevant submit action while a request is in progress and show a concise progress label.
- Every action must provide clear success or failure feedback in plain language. Keep feedback near the affected form or item; do not rely only on color or transient toast messages.
- Ask for confirmation before destructive or high-impact actions (deactivation, deletion, suspension, bulk changes). Explain the consequence before confirmation.
- Use sensible defaults and inline validation. Validate on blur/submit without interrupting typing; show exactly what needs correction.
- Password inputs must provide accessible show/hide controls. Use password managers and browser autofill correctly; never block paste or password managers.
- Make primary actions obvious, keep secondary actions visually quieter, and avoid unnecessary steps, repeated confirmations or modal chains.
- Provide search, filters, pagination and bulk actions only where they materially reduce work. Preserve filter/search state when navigating back where practical.
- Use plain, concise language. Avoid technical error codes and backend implementation details in normal user-facing messages. Provide a recovery action when possible.
- Destructive or irreversible actions must never be triggered by a single ambiguous click. Do not silently discard unsaved changes.
- All controls must have labels, keyboard operation, visible focus, accessible names and touch-friendly targets. Respect reduced-motion preferences.
- Keep navigation and common actions consistent across roles; show only actions the current user can access, while the backend remains authoritative.

### Self-explanatory workflows, dialogs and field-level validation (mandatory)

The product must guide a first-time school user through every task without requiring repeated help from the support team. These are shared acceptance criteria for every existing and future page; do not wait for a page-specific request to apply them.

- **Choose the right interaction:** use the shared `src/shared/components/Modal.tsx` for focused create, edit, update, reset and other multi-field tasks that would otherwise insert a form into a list or shift the page layout. The underlying page/list must remain in place. Use confirmation dialogs for destructive or security-sensitive actions and explain the consequence. Do not open a modal for simple search, filtering, navigation or a quick reversible toggle when inline interaction is clearer.
- **One shared dialog system:** do not build page-specific modal overlays. The shared modal must provide an accessible title/description, `role="dialog"`, `aria-modal`, keyboard focus management, Escape handling, focus restoration, background scroll locking, responsive sizing, theme support and a visible close/cancel path. Do not allow dismissal while a submitted action is still pending if that could create ambiguous outcomes.
- **Errors belong to their fields:** when multiple fields are invalid, show every relevant error together directly below its corresponding field. Never rely on one generic error box at the top or bottom as the only validation feedback. Associate each message with its input using `aria-describedby`, set `aria-invalid`, and use clear corrective wording (for example, “Enter a valid 10-digit phone number” rather than “Invalid value”).
- **Validate the complete form:** validate all applicable fields on submit and present all detected field errors in the same response. Use blur validation where it helps, but do not interrupt typing. Clear a field's error once that field becomes valid. Preserve non-secret values after client or recoverable server validation errors; clear password fields after security-sensitive failures/success where appropriate.
- **Server errors:** map structured server validation errors to the matching fields. If the server returns a general failure that cannot be tied to a field, show a concise form-level alert with a recovery step; do not mislabel it as a field error. A form-level summary may supplement field errors for accessibility, but must never replace them.
- **Labels and guidance:** every field needs a visible, specific label. Mark required fields consistently, provide examples or helper text for non-obvious formats, explain constraints before submission, and use sensible defaults. Do not make users guess what a field expects.
- **Task orientation:** dialog title, description and submit action must name the task and the selected record where relevant (for example, “Reset password for Rahul”). Explain consequential effects such as session revocation or a required password change before confirmation.
- **Consistent outcomes:** show progress on the action button, prevent duplicate submissions, keep errors near the affected field/form, confirm successful completion and explain the next logical step. Empty states must explain what is missing and how to get started; permission states must explain what access is needed and whom to contact where known.
- **No silent dead ends:** every recoverable failure should provide a next step such as correcting a field, retrying, changing a filter or contacting the appropriate administrator. Never expose stack traces, internal SQL, tokens or raw implementation details.
- **No unnecessary modal chains:** keep related steps together when practical, avoid asking for the same information twice, and never use a modal merely to display static information that belongs on the page.

A new or modified page is not complete until the developer has checked this standard against every action and form on that page. During review, record any intentional exception and its usability reason; “the page already worked” is not an exception.

### Universal page-state and quality checklist
Every page and asynchronous operation must deliberately handle: initial loading, background refresh, empty data, validation errors, API/server errors, network/offline failure, permission denied, success, pending mutation and session expiry. Never leave users at a blank screen or an unexplained spinner.

Before marking a workflow complete, verify:
- Desktop, tablet and mobile layouts; no clipped controls or accidental horizontal scrolling.
- Light, Dark and System theme behavior, including focus, hover, disabled, validation, error, success and skeleton states.
- Keyboard-only use, screen-reader labels and adequate contrast.
- Browser refresh, direct URL entry, back/forward navigation, account switching from another tab and session expiry/recovery.
- Slow network, failed request, retry and duplicate-click behavior.
- Production build, relevant automated tests and real backend API contract.
- No mock success states, fake data presented as real, or frontend-only authorization.

Do not ask the user for repeated screenshots or uploads when command output, logs, existing repository code or a concise description can establish the same fact. Request an image only when visual evidence is genuinely necessary.

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


### Appearance controls

- Keep the authenticated header appearance control compact: it only switches between Light and Dark mode. Do not place palette selection or a multi-option appearance menu in the header.
- Put detailed appearance preferences in the authenticated Settings page: Light/Dark/System mode and the available brand palettes.
- Reuse the existing persisted Zustand theme store (`schooldesk.theme`). Moving controls must not reset saved mode or palette preferences.
- Keep the compact Light/Dark toggle accessible by keyboard and provide a clear accessible name describing the mode it will switch to.

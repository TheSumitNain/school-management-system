# Browser acceptance tests

The Student Management browser smoke test uses Playwright with a mocked school API. It verifies the frontend workflow and modal navigation without writing to a real school database or requiring credentials.

## Run locally

From `frontend`:

```cmd
npm ci
npm install --no-save --package-lock=false @playwright/test@1.56.1
npx playwright install chromium
npm run test:e2e
```

The Playwright package is installed as an ephemeral test tool in CI and is intentionally not added to the production dependency tree. The regular `package-lock.json` remains unchanged.

## Coverage

- Staff sign-in using a test-only mocked response
- Student admission form, required academic year/section/date and optional automatic roll-number path
- Student directory refresh after admission
- Student profile and academic history rendering
- Basic Details open/close navigation
- Linked parent details rendering, relationship edit/save, and return to the same profile
- Shared guardian profile email validation, explicit all-linked-records confirmation, and preservation of student-specific relationship flags
- Guardian creation validates email, saves a shared contact, and links it to the student with relationship, primary-contact and emergency-contact flags
- Guardian revocation requires a reason, removes the link from the active list, and retains it in revoked history
- Student archive confirmation can be cancelled; server conflicts remain visible and a successful retry refreshes the archived status
- View-only student and guardian access hides edit/archive/link/revoke controls and emits no write requests
- Promotion required-field validation, destination year/section selection, optional automatic roll-number path, and retention of the promoted enrollment alongside the new enrollment
- Student name editing from Basic Details, save feedback, and updated name preserved when returning to the student profile
- Basic Details field editing, save, and confirmation
- Current Address hierarchy selection (State/UT → District → Sub-District/Tehsil → Village/City), save, and resolved hierarchy IDs
- Permanent Address linked to Current Address and saved with the same_as_current flag
- Contact phone save and confirmation
- Medical profile Blood Group / Allergies / Known Conditions save and version increment
- Physical measurement validation (range + conditional date) and valid save
- Browser uncaught exceptions

All API calls are intercepted by the test. This is frontend browser acceptance coverage, not a replacement for backend PostgreSQL integration tests or live-browser acceptance against the user's configured school database.

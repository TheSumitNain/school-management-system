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
- Browser uncaught exceptions

All API calls are intercepted by the test. This is frontend browser acceptance coverage, not a replacement for backend PostgreSQL integration tests or live-browser acceptance against the user's configured school database.

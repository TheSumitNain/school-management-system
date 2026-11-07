# Frontend Architecture

The frontend is a feature-oriented React + TypeScript application. Extend the existing Vite app; do not create a parallel app or move files in a way that breaks the current login/session flow.

## Target source layout

```text
frontend/src/
├── app/
│   ├── App.tsx                 # Application composition (migrate from src/App.tsx)
│   ├── providers.tsx           # Query and theme providers
│   ├── ErrorBoundary.tsx       # Application-level rendering recovery
│   └── router/                 # Route definitions, guards and lazy boundaries
├── features/
│   ├── auth/                   # Login, MFA, session bootstrap and password flows
│   ├── theme/                  # Theme preference and theme behavior
│   ├── platform-admin/
│   │   ├── api/                # Platform Admin API functions
│   │   ├── queries/            # TanStack Query hooks and keys
│   │   ├── components/         # Feature-only components
│   │   └── pages/              # Platform Admin routes/pages
│   ├── school-admin/
│   │   ├── api.ts              # School Admin API functions
│   │   ├── queries.ts          # TanStack Query hooks and query keys
│   │   ├── components/
│   │   └── pages/
│   └── dashboard/              # Shared role-aware overview widgets
├── shared/
│   ├── api/                    # Shared API utilities and typed contracts
│   ├── components/             # Design-system primitives and feedback states
│   ├── hooks/                  # Reusable UI hooks
│   ├── layouts/                # Shared authenticated and public shells
│   ├── types/
│   └── utils/
├── lib/
│   └── api.ts                  # Existing authentication-aware HTTP client (migrate carefully)
├── styles/
│   ├── tokens.css              # Semantic design tokens
│   └── theme-overrides.css     # Theme-specific presentation
├── App.tsx                     # Transitional entry; migrate after routes are verified
├── main.tsx
└── styles.css                  # Existing styles; migrate incrementally to shared tokens
```

## State ownership

- **TanStack Query:** API/server state, caching, request cancellation, invalidation and retry policy.
- **Zustand:** small cross-page client preferences such as theme and selected UI preferences. Do not mirror API data into Zustand.
- **React state:** local transient UI state (open dialog, active tab, input text).
- **React Hook Form + Zod:** introduce when building substantial validated forms; do not add them to simple read-only screens.

## Implementation rules

- Keep feature API functions separate from React components.
- Keep query keys centralized within their feature and invalidate the narrowest affected keys after mutations.
- Keep authorization on the API. Client permissions only control presentation/navigation.
- Lazy-load route pages, not tiny shared primitives. Keep login, session bootstrap and the authenticated shell responsive.
- Use shared design tokens and shared loading/error/empty components. Do not duplicate one-off page styling.
- Every feature should define loading, empty, error, success and permission-denied behavior.
- Clear user-scoped query cache on logout/account changes.
- Do not add a dependency without a concrete need and bundle-size consideration.

## Current migration status

The current frontend already contains working sign-in/MFA/session handling and an initial dashboard. The foundation is being introduced incrementally: QueryClient and Zustand theme preference are wired at the root; shared theme tokens, theme control, skeleton and error boundary have been added. Existing pages and legacy CSS must be migrated in small verified steps rather than rewritten all at once.

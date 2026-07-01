# mrtutor — frontend

React SPA that talks to the Go backend at `/api/v0`. In production it is compiled
and embedded directly into the Go binary (`api/static/dist/`). In development it
runs on its own Vite server with a proxy.

## Stack

- React 19 + TypeScript
- [TanStack Router](https://tanstack.com/router) — file-based routing
- [TanStack Query](https://tanstack.com/query) — server state
- [Mantine](https://mantine.dev) — component library
- Bun + Vite

## Commands

Run from the repo root via Taskfile (preferred):

```bash
task dev          # start API + Vite dev server together
task web:dev      # Vite dev server only (port 3000, proxies /api → :8080)
task web:build    # production build → api/static/dist/
task web:test     # Vitest unit tests
```

Or directly from `web/`:

```bash
bun install
bun run dev
bun run build
bun run test
bun run generate-routes   # regenerate routeTree.gen.ts after adding route files
```

## Structure

```
src/
  features/          one directory per domain area
    auth/
      api.ts         typed fetch wrappers + TanStack Query option factories
      useAuth.ts     useAuth() hook — reads current-user from Query cache
      mutations.ts   useLogin / useSignout / useRegister
      pages/         LoginPage, RegisterPage
      components/    shared layout components (TitleLayout)
  routes/            file-based TanStack Router tree (auto-generates routeTree.gen.ts)
    __root.tsx       root layout (QueryClientProvider + MantineProvider)
    _authenticated.tsx   pathless layout guard — redirects to /auth/login when unauthenticated
    _authenticated/  protected pages; add new authenticated pages here
    auth/
      login.tsx
      register.tsx
  components/        global shared components (CustomLink)
  integrations/      library wiring (TanStack Query context + QueryClient)
  router.tsx         router factory — injects queryClient into router context
  main.tsx           entry point
```

## Adding a route

Place a `.tsx` file under `src/routes/`. The router plugin picks it up automatically
and regenerates `routeTree.gen.ts` on the next dev-server restart (or run
`bun run generate-routes` manually).

- **Protected page** → put it under `src/routes/_authenticated/`. The `_authenticated`
  layout guard runs `GET /api/v0/auth/me` in `beforeLoad` and redirects to login if
  the user is not authenticated.
- **Public page** → put it at the root of `src/routes/` (like `auth/login.tsx`).

## Auth state

The current user lives in the TanStack Query cache under key `["auth", "me"]`:

```ts
import { useAuth } from "#/features/auth/useAuth";

const { user, isAuthenticated, isLoading } = useAuth();
```

Mutations:

```ts
import { useLogin, useSignout } from "#/features/auth/mutations";

const login = useLogin();
login.mutate({ Token: "email@example.com", Password: "…" }, {
  onSuccess: () => navigate({ to: redirect ?? "/" }),
});

const signout = useSignout();
signout.mutate(); // clears cache and navigates to /auth/login
```

## Talking to the backend

All fetch calls live in `src/features/<name>/api.ts`. Conventions:

- Always pass `credentials: "include"` so the session cookie travels with requests.
- Go request structs use PascalCase field names (no `json` tags), so request bodies
  must match: `{ Token, Password }` not `{ token, password }`.
- A `401` from `GET /auth/me` returns `null` (not an error). All other non-ok
  responses throw.
- In dev, Vite proxies `/api/*` to `http://localhost:8080`. No base-URL config needed.

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

All fetch calls live in `src/features/<name>/api.ts` and go through the shared client
in `src/lib/api.ts`:

```ts
import { api, ApiError, NetworkError } from "#/lib/api";

const user  = await api.get<User>("/auth/me");
await api.post("/auth/login", { Token: "alice", Password: "…" });
await api.put("/items/42", { title: "updated" });
await api.delete("/items/42");
```

The client handles automatically:
- **Base path** — `/api/v0` prepended to every path (overridable via `VITE_API_BASE_PATH`).
- **Credentials** — `credentials: "include"` on every request so the session cookie travels.
- **Request serialization** — plain objects/arrays → `JSON.stringify` + `Content-Type: application/json`; `FormData`, `Blob`, `string`, etc. are passed through untouched.
- **Response parsing** — `Content-Type: application/json` → `res.json()`; `204`/empty body → `undefined`; otherwise `res.text()`.
- **HTTP errors** — non-ok responses throw `ApiError` (with `.status`, `.statusText`, `.body`).
- **Network errors** — when the server is never reached (offline, DNS failure, connection refused), throws `NetworkError` instead. It has no `.status`; the underlying `TypeError` is available as `.cause`.

Branch on error type:

```ts
try {
  return await api.get<Principal>("/auth/me");
} catch (e) {
  if (e instanceof ApiError && e.status === 401) return null;  // clean 401 → unauthenticated
  if (e instanceof NetworkError) { /* server unreachable */ }
  throw e;
}
```

Go request structs have no `json` tags — field names are PascalCase in JSON. Match them
in request bodies: `{ Token, Password }` not `{ token, password }`.

In dev, Vite proxies `/api/*` → `http://localhost:8080`. No base-URL config needed.

### Offline / server-down page

When any route's `beforeLoad` or loader throws a `NetworkError`, the router's
`defaultErrorComponent` (`src/components/RootErrorComponent.tsx`) catches it and renders
the `BackendUnreachable` page (`src/components/BackendUnreachable.tsx`): a full-screen
Mantine-styled message with a "Try again" button and support contact details.

Support contact info is stored in `src/config.ts` (`SUPPORT_EMAIL`, `SUPPORT_HOURS`).
Edit that file to set the real values before going live.

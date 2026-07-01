# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`mrtutor` is a full-stack web application: a Go HTTP API backed by SQLite with a
React SPA embedded directly into the binary. It is a Go workspace (`go.work`) monorepo
with three modules:

- `api/` — the Go application (module `mrtutor/api`)
- `gotools/` — pinned codegen tools (`sqlc`, `moq`) via Go's `tool` directive, so
  they run as `go tool <name>` without polluting app dependencies.
- `web/` — the React frontend (not a Go module; managed by Bun + Vite)

Tasks are driven by [Taskfile](https://taskfile.dev). The root `Taskfile.yml` includes
`api/Taskfile.yml` under `api:` (working dir `api/`) and `web/Taskfile.yml` under
`web:` (working dir `web/`).

## Common commands

Run from the repo root:

```bash
task dev              # run API + Vite dev server together (Ctrl-C stops both)
task build            # production build: frontend → api/static/dist, then Go binary

task api:run          # Go server only (port 8080, auto-migrates in dev)
task api:test         # backend unit tests; runs `generate` first
task api:integration  # integration tests (go test ./test/integration/ -it)
task api:e2e          # end-to-end tests (go test ./test/e2e/ -e2e)
task api:verify       # test → integration → e2e, failfast
task api:sqlcgen      # regenerate sqlc query code from SQL
task api:generate     # go generate ./api/... (regenerates moq mocks)

task web:dev          # Vite dev server (port 3000, proxies /api → :8080)
task web:build        # build frontend into api/static/dist/
task web:test         # Vitest unit tests
task web:generate-routes  # regenerate routeTree.gen.ts (auto-runs on dev server start)
```

Backend integration and e2e tests are gated behind `-it` / `-e2e` flags — `go test ./...`
skips them. All backend test tasks set `APP_ENV=test`.

Run a single backend test directly (from `api/`):

```bash
cd api && go test ./features/auth/ -run TestServiceLogin
cd api && go test ./test/integration/ -it -run TestAuth   # remember the gate flag
```

First-time setup: `scripts/setup-dev.sh` installs Go modules, `golang-migrate`, and `task`.

---

## Backend

### Code generation

Two generators; **regenerate after changing their inputs** (`task api:test` does this for you):

- **sqlc** (`sqlc.yml`): reads schema from `db/migrations/` and queries from `db/queries/*.sql`,
  emitting typed Go in `db/queries/*.sql.go`. Add a query in `db/queries/`, then `task api:sqlcgen`.
- **moq** (`go:generate` directives in `*_test.go`): generates interface mocks via `go tool moq`.
  Mocks live in `*_moq_test.go` / `*_mock_test.go` beside the feature they test.

SQL files are formatted with `sql-formatter` (config `.sql-formatter.json`);
`scripts/format-sql.sh` runs it over `db/**/*.sql`.

### Feature module pattern

Each feature lives in `api/features/<name>/` and follows a layered structure wired
by `InitModule`:

- **`init.go`** — domain types, interfaces (`Service`, repository, store interfaces,
  and a `module` combining `Service` + `RegisterRoutes`). `InitModule` constructs
  concrete implementations and returns an anonymous struct embedding them. This is the
  only exported constructor; it also registers background jobs.
- **`controller.go`** — HTTP layer. Builds handlers via `httpbind` and registers routes.
- **`service.go`** — business logic (`serviceImpl`), depends only on package interfaces.
- **`repository.go`** — data access; wraps generated `db/queries` and maps
  `sql.ErrNoRows` to domain errors.
- **`mapper.go`** — converts between sqlc row models and domain types.

To add a feature, mirror this layout and call `InitModule(...).RegisterRoutes(mux)`
from `api/routes.go::addRoutes`.

### Request pipeline (`transport/httpbind`)

`httpbind.NewHandler[In, Out]` composes a `decode → service fn → encode` pipeline
into an `http.Handler`. `NewNoOutputHandler` is the variant for handlers with no
response body. If the decoded input implements `Validate() error`, it is validated
automatically before the service call.

`writeError` is the single place that maps domain errors to HTTP status codes
(`validation.Error` → 400, `NotFoundError` → 404, `ErrUnauthorized` → 401, else 500).
Add new error mappings there, not in handlers.

### Errors & validation

- `api/errors/` — domain error types (`NotFoundError`, sentinel `ErrUnauthorized`).
- `api/validation/` — reusable validators; request structs implement `Validate()` and
  accumulate problems into a `*validation.Error`.

### Database & migrations

- SQLite via `mattn/go-sqlite3`. `db/db.go`: `New()` (file, DSN from `config.DSN`) or
  `NewInMemory()` (used by tests).
- Migrations in `db/migrations/`, **embedded** via `//go:embed`, run with `golang-migrate`.
  Auto-runs on startup in DEV/TEST only. `cmd/migrate/` is the standalone runner for prod.
  Convention: `<timestamp>_<name>.{up,down}.sql`.

### Server lifecycle

`main.go` wires dependencies and handles graceful shutdown: SIGINT/SIGTERM stops
background jobs **first**, then drains HTTP requests within `config.ShutdownTimeout`.
`/health` returns 503 once shutdown begins.

### Background jobs

`api/scheduler/` is an in-house scheduler for periodic and one-shot jobs.
Features register jobs inside `InitModule`. See
[docs/systems/scheduler.md](docs/systems/scheduler.md) for the full design.

### Backend configuration

All config from env vars (with defaults):

| Variable | Default | Notes |
|----------|---------|-------|
| `APP_ENV` | `dev` | `dev` \| `prod` \| `test`. Controls auto-migration, in-memory DB, cookie `Secure`. |
| `LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error`. |
| `PORT` | `8080` | |
| `BASEPATH` | `/api/v0` | API base path. If `/`, only the health mux is served. |
| `SHUTDOWN_TIMEOUT` | `5s` | Go duration. |

### Backend testing layout

- Unit tests beside the code (`*_test.go`), using moq-generated mocks.
- `test/integration/` — in-memory DB + migrations + `httptest.Server` (gate: `-it`).
- `test/e2e/` — builds the real binary and runs it on a free port (gate: `-e2e`).
- `test/helpers.go` — shared helpers.

---

## Frontend

### Tech and conventions

- **React 19 + TypeScript** — strict mode, no class components.
- **TanStack Router** — file-based routing (`src/routes/`). Route tree is
  auto-generated into `src/routeTree.gen.ts`; regenerate with `task web:generate-routes`
  or let the dev server do it on restart.
- **TanStack Query** — server state only. The `queryClient` lives in router context
  (created in `src/integrations/tanstack-query/root-provider.tsx`, injected via
  `getRouter()` in `src/router.tsx`). Components access it via `useQueryClient()`.
- **Mantine** — UI component library. Use Mantine components first; avoid raw HTML
  elements when a Mantine equivalent exists.
- **No Tailwind** — removed. Style with Mantine props or CSS Modules (`.module.css`).
- Path alias `#/*` → `./src/*`. Use `#/` imports everywhere, not relative paths across
  feature boundaries.

### Frontend feature pattern

Each feature lives in `src/features/<name>/` with this layout:

- **`api.ts`** — typed `fetch` wrappers and TanStack Query option factories
  (`queryOptions`, `mutationFn`). This is the only place that makes HTTP calls.
- **`useXxx.ts`** — read hooks (thin wrappers over `useQuery`).
- **`mutations.ts`** — write hooks (`useMutation` wrappers).
- **`pages/`** — page-level components, one per route.
- **`components/`** — feature-local UI components.

### Routing conventions

Route files go in `src/routes/` following TanStack Router's file-based conventions:

- **Protected pages** → `src/routes/_authenticated/<page>.tsx`. The `_authenticated`
  pathless layout guard (in `src/routes/_authenticated.tsx`) runs `GET /auth/me` in
  `beforeLoad` and redirects to `/auth/login?redirect=<href>` when unauthenticated.
- **Public pages** → anywhere outside `_authenticated/` (e.g. `src/routes/auth/`).
- Typed links: use `CustomLink` from `src/components/CustomLink.tsx` (a Mantine `Anchor`
  wrapped with `createLink`). Do not use bare `<a>` tags for internal navigation.

After adding or renaming a route file, run `task web:generate-routes`.

### Auth state

The current user is server state in the Query cache under key `["auth", "me"]`
(exported as `authKeys.me`). There is no separate client store.

```ts
import { useAuth } from "#/features/auth/useAuth";
const { user, isAuthenticated, isLoading } = useAuth();

import { useLogin, useSignout, useRegister } from "#/features/auth/mutations";
```

Key invariant: **use `removeQueries` (not `invalidateQueries`) after login** so the
pre-login `null` is evicted and `ensureQueryData` in `_authenticated beforeLoad` is
forced to do a fresh fetch. See [docs/features/auth.md](docs/features/auth.md) for the
full flow.

### API client

All HTTP calls go through `src/lib/api.ts`. Import it in every `features/<name>/api.ts`:

```ts
import { api, ApiError, NetworkError } from "#/lib/api";

await api.get<User>("/auth/me");
await api.post("/auth/login", { Token: "alice", Password: "…" });
await api.put("/items/42", payload);
await api.delete("/items/42");
```

What the client provides automatically:
- **Base path** prepended (`/api/v0` by default, overridable via `VITE_API_BASE_PATH`).
- **`credentials: "include"`** on every request.
- **Body serialization**: plain objects/arrays → JSON + `Content-Type: application/json`; `FormData`/`Blob`/`string` passed through (browser sets content type).
- **Response parsing**: JSON content-type → `res.json()`; `204`/empty → `undefined`; else `res.text()`.
- **HTTP errors**: non-ok response → `throw new ApiError(status, statusText, body)`.
- **Network errors**: server never reached (offline, DNS, refused) → `throw new NetworkError(cause)`. No `.status`; the raw `TypeError` is in `.cause`.

Branch on error type:

```ts
try {
  return await api.get<Principal>("/auth/me");
} catch (e) {
  if (e instanceof ApiError && e.status === 401) return null;  // clean 401
  if (e instanceof NetworkError) { /* server unreachable */ }
  throw e;
}
```

Go request structs have no `json` tags — field names are PascalCase. Match them in
request bodies: `{ Token, Password }`, not `{ token, password }`.

In dev, Vite proxies `/api/*` → `:8080`. Paths passed to `api.*` are relative to the
base path, e.g. `"/auth/me"` not `"/api/v0/auth/me"`.

### Error handling

`src/router.tsx` registers `defaultErrorComponent: RootErrorComponent`
(`src/components/RootErrorComponent.tsx`). It catches errors thrown from any route's
`beforeLoad` or loader and branches:

- **`NetworkError`** → renders `BackendUnreachable` (`src/components/BackendUnreachable.tsx`):
  full-screen Mantine page with a "Try again" button and support contact details
  (`SUPPORT_EMAIL`, `SUPPORT_HOURS` from `src/config.ts` — edit that file before going live).
- **Anything else** → generic "Something went wrong" fallback (also Mantine-styled).

### Frontend testing layout

- Unit tests alongside source files (`*.test.ts`, `*.test.tsx`), run with Vitest.
- Testing utilities: `@testing-library/react` + `jsdom` (configured in Vitest).
- No integration or e2e tests yet for the frontend.

# Auth feature

Covers registration, login, logout, and session management. The backend lives in
`api/features/auth/`; the frontend in `web/src/features/auth/`.

## HTTP API

All routes are mounted under the configured base path (default `/api/v0`).

| Method | Path | Auth required | Description |
|--------|------|:---:|-------------|
| `POST` | `/auth/register` | No | Create account + start session |
| `POST` | `/auth/login` | No | Start session |
| `POST` | `/auth/logout` | No | End session |
| `GET` | `/auth/me` | No* | Return current principal |

*`/auth/me` returns `401` when no valid session cookie is present rather than
requiring auth as a hard precondition — the frontend uses this to determine
authentication status.

### Register — `POST /auth/register`

Request body (JSON, PascalCase — Go structs have no `json` tags):

```json
{ "Username": "alice", "Email": "alice@example.com", "Password": "…" }
```

Validation: username required and not an email address; email must be valid; password
must meet strength requirements (see `api/validation/`). Returns `400` with a plain-text
list of problems on failure.

Response on success — `201 Created`:

```json
{ "ID": 1, "Username": "alice", "Email": "alice@example.com", "CreateAt": "…", "ModifiedAt": "…" }
```

Also sets the `session` cookie (see below). `HashedPassword` is always excluded from
JSON output (`json:"-"`).

### Login — `POST /auth/login`

```json
{ "Token": "alice", "Password": "…" }
```

`Token` is username **or** email. Returns `401` on bad credentials, `200` on
success with no body — the session is delivered via cookie only.

### Logout — `POST /auth/logout`

No body. Reads the `session` cookie, invalidates the session in the database, and
clears the cookie (`MaxAge: -1`). Succeeds (`200`) even when no cookie is present.

### Me — `GET /auth/me`

Returns the current principal for a valid session:

```json
{ "id": 1, "username": "alice", "email": "alice@example.com" }
```

Note: `/auth/me` uses lowercase JSON field names (a local response struct with `json`
tags), unlike the register response which returns the raw `Principal` struct
(PascalCase). Returns `401` when the session cookie is absent or expired.

## Session mechanism

Sessions are stored in SQLite. A session has two expiries:

| Kind | Duration | Behaviour |
|------|----------|-----------|
| Absolute | 24 hours | Hard ceiling from creation; never extended |
| Idle | 30 minutes | Refreshed on every `VerifySession` call |

The session token is a random string (`crypto/rand`) stored in an **HttpOnly** cookie
named `session` (`SameSite=Lax`, `Secure` only in `APP_ENV=prod`). The cookie carries
no expiry itself (session cookie), so the browser clears it on close; server-side
expiry is the authoritative source.

The idle refresh runs in a goroutine decoupled from the request context
(`context.WithoutCancel`) so it outlives the HTTP response.

An hourly background job (`session-cleanup`) deletes expired sessions from the
database. It is registered in `InitModule` via the scheduler.

## Domain types

```go
// api/features/auth/init.go
type Principal struct {
    ID             int64
    Username       string
    Email          string
    HashedPassword string `json:"-"`
    CreateAt       time.Time
    ModifiedAt     time.Time
}
```

There is no role field — the identity is `Username` + `Email`. Authorization is a
future concern.

## Backend layout

```
api/features/auth/
  init.go          domain types, interfaces (Service, principalRepository,
                   sessionStore), InitModule (wires everything, registers cleanup job)
  controller.go    HTTP handlers via httpbind; RegisterRoutes
  service.go       business logic (Login, Register, Logout, VerifySession)
  repository.go    SQL access; maps sql.ErrNoRows → domain errors
  mapper.go        sqlc row ↔ Principal conversions
  controller_test.go   unit tests for handlers (moq-generated ServiceMock)
  service_test.go      unit tests for service logic
```

## Frontend layout

```
web/src/lib/
  api.ts           shared HTTP client (api.get/post/put/patch/delete, ApiError)

web/src/features/auth/
  api.ts           auth-specific wrappers + TanStack Query option factories
  useAuth.ts       useAuth() hook
  mutations.ts     useLogin / useSignout / useRegister
  pages/
    LoginPage.tsx
    RegisterPage.tsx
    AuthPage.module.css
  components/
    TitleLayout.tsx   centred card layout shared by auth pages
```

## Frontend auth state

The current user is server state, owned by TanStack Query under key `["auth", "me"]`.
There is no client-side store — Query is the single source of truth.

`useAuth()` is a thin wrapper around `useQuery(meQueryOptions())`:

```ts
const { user, isAuthenticated, isLoading } = useAuth();
```

### Route guards

`web/src/routes/_authenticated.tsx` is a pathless TanStack Router layout route. Its
`beforeLoad` calls `queryClient.ensureQueryData(meQueryOptions())` and throws a
redirect to `/auth/login?redirect=<current href>` when the result is `null`. Any
route file placed under `src/routes/_authenticated/` inherits this guard automatically.

The resolved `user` is returned from `beforeLoad` and injected into router context, so
child routes can read it via `Route.useRouteContext().user` without an extra fetch.

### Login flow

```
POST /auth/login  →  server sets session cookie
useLogin onSuccess  →  removeQueries(["auth","me"])   // evict pre-login null
navigate({ to: redirect ?? "/" })
_authenticated beforeLoad  →  ensureQueryData  →  GET /auth/me (fresh)  →  200
route renders
```

`removeQueries` (not `invalidateQueries`) is critical: `invalidateQueries` leaves
stale data in the cache; `ensureQueryData` would short-circuit on it, still seeing
`null`, and redirect back to login.

### Register flow

`POST /auth/register` returns the new `Principal` in the response body (along with
the session cookie). `useRegister.onSuccess` writes it directly into the Query cache
via `setQueryData(authKeys.me, principal)` — no extra round-trip to `/auth/me` needed.

### Signout flow

```
POST /auth/logout  →  server clears cookie
useSignout onSuccess  →  setQueryData(["auth","me"], null)
                     →  queryClient.clear()            // drop all cached data
                     →  navigate({ to: "/auth/login" })
```

## Adding protected pages

1. Create a route file under `web/src/routes/_authenticated/`, e.g.
   `web/src/routes/_authenticated/dashboard.tsx`.
2. That's it — the guard runs automatically. Access the user with
   `Route.useRouteContext().user` or `useAuth()`.

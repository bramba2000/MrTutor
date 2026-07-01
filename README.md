# mrtutor

A full-stack web application: a Go HTTP API backed by SQLite, with a React SPA
embedded directly into the binary. The frontend is built at compile time and served
as static files by the Go server — there's no separate frontend deployment.

## Stack

**Backend**
- Go 1.26 (standard `net/http`, no web framework)
- SQLite via `mattn/go-sqlite3`, `golang-migrate` for migrations, `sqlc` for typed queries
- [Taskfile](https://taskfile.dev) as the task runner
- `moq` for test mocks

**Frontend**
- React 19, TypeScript
- [TanStack Router](https://tanstack.com/router) — file-based routing with type-safe params and search
- [TanStack Query](https://tanstack.com/query) — server state and async data
- [Mantine](https://mantine.dev) — UI components
- [Bun](https://bun.sh) as package manager and runtime; Vite as bundler

**Monorepo structure** — Go workspace (`go.work`) with three modules:
- `api/` — the application (`mrtutor/api`)
- `gotools/` — pinned codegen tools (`sqlc`, `moq`), so they run as `go tool …` without polluting app deps
- `web/` — the React frontend (not a Go module; managed by Bun)

## Getting started

You'll need Go and [Bun](https://bun.sh) installed. The setup script handles the rest:

```bash
./scripts/setup-dev.sh
```

Then start everything together:

```bash
task dev
```

This runs the Go API on `:8080` and the Vite dev server on `:3000` in parallel. The
Vite dev server proxies `/api/*` to the Go server, so the frontend talks to the real
API with hot-reload. Press `Ctrl-C` to stop both.

In dev mode the API migrates the database on startup — no separate step needed.

## Running individually

```bash
task api:run    # Go server only (port 8080)
task web:dev    # Vite dev server only (port 3000)
```

## Tests

```bash
task api:test         # backend unit tests
task api:integration  # integration tests (in-memory DB + httptest server)
task api:e2e          # end-to-end (builds the real binary, runs it)
task api:verify       # all three, stops at the first failure

task web:test         # frontend unit tests (Vitest)
```

The backend integration and e2e suites are gated behind `-it` / `-e2e` flags, so
`go test ./...` skips them. Always use the task targets for those.

To run a single backend test:

```bash
cd api
go test ./features/auth/ -run TestService
go test ./test/integration/ -it -run TestAuth   # don't forget the flag
```

## Building for production

```bash
task build
```

This builds the frontend first (outputs to `api/static/dist/`), then compiles the
Go binary with the frontend embedded via `//go:embed`. The result is a single
self-contained binary at `api/bin/app`.

## Code generation (backend)

```bash
task api:sqlcgen   # regenerate query code from db/queries/*.sql + db/migrations
task api:generate  # regenerate moq mocks (go generate)
```

`task api:test` runs `generate` for you first, so day to day you rarely call them
by hand. After adding or changing SQL, run `sqlcgen`.

## Layout

```
api/
  main.go, server.go, routes.go   server wiring and graceful shutdown
  config/                         env-driven configuration
  db/                             connection, migrations, generated queries
  features/auth/                  registration, login, sessions
  scheduler/                      background job scheduler
  static/                         embedded frontend assets (dist/ populated by build)
  transport/httpbind/             decode → service → encode handler glue
  test/                           integration and e2e suites
gotools/                          pinned codegen tools
web/
  src/
    features/auth/                auth API client, hooks, mutations, pages
    routes/                       file-based route tree
      _authenticated.tsx          pathless layout guard for protected routes
      _authenticated/             authenticated pages (guarded automatically)
      auth/                       login and register pages (public)
    components/                   shared UI components
    integrations/                 library setup (TanStack Query context)
docs/
  systems/                        architectural design notes
  features/                       per-feature documentation
```

## Configuration

Everything comes from environment variables with sensible defaults:

| Variable | Default | Notes |
|----------|---------|-------|
| `APP_ENV` | `dev` | `dev`, `prod`, or `test`. Controls auto-migration, in-memory DB, and the cookie `Secure` flag. |
| `PORT` | `8080` | |
| `BASEPATH` | `/api/v0` | API base path. |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`. |
| `SHUTDOWN_TIMEOUT` | `5s` | How long to drain in-flight work on shutdown. |

## Background jobs

There's a small in-house scheduler in `api/scheduler/` for periodic and one-shot
work (the session cleanup job runs on it). It handles cancellation, job errors and
panics, fatal errors escalating into graceful shutdown, and clean teardown on
SIGINT/SIGTERM. The design is written up in
[docs/systems/scheduler.md](docs/systems/scheduler.md).

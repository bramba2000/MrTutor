# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`mrtutor` is a Go HTTP API backed by SQLite. It is a Go workspace (`go.work`) monorepo
with two modules:

- `api/` — the application (module `mrtutor/api`)
- `gotools/` — a tools-only module pinning code-generation tools (`sqlc`, `moq`) via Go's
  `tool` directive, so they run as `go tool <name>` without polluting the app's dependencies.

Tasks are driven by [Taskfile](https://taskfile.dev). The root `Taskfile.yml` includes
`api/Taskfile.yml` under the `api:` namespace, with its working dir set to `api/`.

## Common commands

Run from the repo root:

```bash
task api:run          # go run ./ in api/  (dev mode, auto-migrates on startup)
task api:test         # unit tests only (excludes /test/e2e and /test/integration); runs `generate` first
task api:integration  # integration tests (go test ./test/integration/ -it)
task api:e2e          # end-to-end tests (go test ./test/e2e/ -e2e)
task api:verify       # test → integration → e2e, failfast
task api:sqlcgen      # regenerate sqlc query code from SQL
task api:generate     # go generate ./api/... (regenerates moq mocks)
```

Integration and e2e tests are gated behind the `-it` / `-e2e` flags (see `TestMain` in each
suite) — running `go test ./...` directly will skip them. All test tasks set `APP_ENV=test`.

Run a single test directly (the API module runs from `api/`):

```bash
cd api && go test ./features/auth/ -run TestServiceLogin
cd api && go test ./test/integration/ -it -run TestAuth   # remember the gate flag
```

First-time setup: `scripts/setup-dev.sh` installs Go modules, `golang-migrate`, and `task`.

## Code generation

Two generators; **regenerate after changing their inputs** (the `test` task does this for you):

- **sqlc** (`sqlc.yml`): reads schema from `db/migrations/` and queries from `db/queries/*.sql`,
  emitting typed Go in `db/queries/*.sql.go`. Add a query by writing SQL in `db/queries/`, then
  `task api:sqlcgen`.
- **moq** (`go:generate` directives in `*_test.go`): generates interface mocks via `go tool moq`.
  Mocks are generated from the interfaces declared in feature packages (e.g. `Service`,
  `principalRepository`, `sessionStore`) into `*_moq_test.go` / `*_mock_test.go`.

SQL files are formatted with `sql-formatter` (config `.sql-formatter.json`); `scripts/format-sql.sh`
runs it over `db/**/*.sql`.

## Architecture

### Feature module pattern

Each feature lives in `api/features/<name>/` and follows a layered structure wired together by
`InitModule`:

- **`init.go`** — declares the domain types and the package's interfaces (`Service`, repository,
  store interfaces, and a `module` interface combining `Service` + `RegisterRoutes`). `InitModule`
  constructs the concrete implementations and returns an anonymous struct embedding the service and
  controller. This is the only exported constructor; it also registers any background jobs.
- **`controller.go`** — HTTP layer. Builds handlers via `httpbind` and registers routes on a mux.
- **`service.go`** — business logic (`serviceImpl`), depends only on the package's interfaces.
- **`repository.go`** — data access; wraps generated `db/queries` and maps `sql.ErrNoRows` to
  domain errors (e.g. `apierrors.NotFoundError`).
- **`mapper.go`** — converts between sqlc row models and domain types.

To add a feature, mirror this layout and call its `InitModule(...).RegisterRoutes(mux)` from
`api/routes.go::addRoutes`.

### Request pipeline (`transport/httpbind`)

`httpbind.NewHandler[In, Out]` composes a `decode → service fn → encode` pipeline into an
`http.Handler`. `NewNoOutputHandler` is the variant for handlers with no response body. Decoders/
encoders are generic helpers (`NewJSONDecoder`, `NewJSONEncoder`). If the decoded input implements
`Validate() error`, it is validated automatically before the service call.

`writeError` is the single place that maps domain errors to HTTP status codes
(`validation.Error` → 400, `apierrors.NotFoundError` → 404, `apierrors.ErrUnauthorized` → 401,
else 500). Add new error mappings there rather than in handlers.

### Errors & validation

- `api/errors/` holds domain error types (`NotFoundError`, sentinel `ErrUnauthorized`, …).
- `api/validation/` holds reusable validators; request structs implement `Validate()` and
  accumulate human-readable problems into a `*validation.Error`.

### Database & migrations

- SQLite via `mattn/go-sqlite3`. Connection is opened in `db/db.go`: `New()` (file, DSN from
  `config.DSN`) or `NewInMemory()` (used by tests, preserving DSN query options).
- Migrations are SQL files in `db/migrations/`, **embedded** via `//go:embed` and run with
  `golang-migrate` (`migrations.NewWithDb`). On startup `main.go` auto-runs migrations **only in
  DEV/TEST**; PROD does not auto-migrate. `cmd/migrate/` is the standalone migration runner for
  prod/manual use. Migration files follow the `<timestamp>_<name>.{up,down}.sql` convention.

### Server lifecycle

`main.go` wires dependencies (DB, scheduler, server) and handles graceful shutdown: on
SIGINT/SIGTERM it stops background jobs **first**, then drains in-flight HTTP requests within
`config.ShutdownTimeout`. The `/health` endpoint returns 503 once shutdown begins
(`isShuttingDownServer`). Routes are mounted under `config.ApiBasePath` (default `/api/v0`) with
logging middleware applied via `applyMiddleware`.

### Background jobs

The app uses `go-co-op/gocron` (constructed in `main.go`, started with `StartAsync`). Features
register jobs inside their `InitModule` — e.g. auth schedules hourly cleanup of expired sessions
via `sessionStore.DeleteExpiredSessions`. (Note: `api/scheduler/` contains an in-progress custom
scheduler abstraction that is not yet wired into `main.go`.)

### Auth specifics

Passwords are hashed with bcrypt. Sessions use a random token stored in an HttpOnly cookie
(`Secure` only in PROD) and carry both an **absolute** expiry (24h) and a sliding **idle** expiry
(30m, refreshed asynchronously on each `VerifySession`).

## Configuration

All config is read from env vars in `api/config/` (with defaults):

- `APP_ENV` — `dev` | `prod` | `test` (default `dev`); controls auto-migration, in-memory DB,
  and cookie `Secure` flag.
- `LOG_LEVEL` — `debug` | `info` | `warn` | `error` (default `info`).
- `PORT` — default `8080`.
- `BASEPATH` — API base path (default `/api/v0`). If `/`, only the internal mux (health) is served.
- `SHUTDOWN_TIMEOUT` — Go duration (default `5s`).

## Testing layout

- Unit tests live beside the code (`*_test.go`), using moq-generated mocks.
- `test/integration/` — spins up an in-memory DB + migrations + `httptest.Server` (gated by `-it`).
- `test/e2e/` — builds the real binary and runs it as a subprocess on a free port (gated by `-e2e`).
- `test/helpers.go` provides shared helpers (e.g. writing logs to the test artifact dir).

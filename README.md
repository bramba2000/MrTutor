# mrtutor

The backend API for mrtutor. It's a plain Go HTTP service on top of SQLite, with
the standard library doing most of the work and a small number of libraries for
the things that aren't worth hand-rolling (migrations, SQLite, password hashing).

## Stack

- Go 1.26 (standard `net/http`, no web framework)
- SQLite, with `golang-migrate` for migrations and `sqlc` for typed queries
- [Taskfile](https://taskfile.dev) as the task runner
- `moq` for test mocks

It's a Go workspace (`go.work`) with two modules: `api/` is the application, and
`gotools/` exists only to pin the code-generation tools (`sqlc`, `moq`) so they
run as `go tool …` without ending up in the app's dependencies.

## Getting started

You'll need Go installed. The setup script takes care of the rest (Go modules,
`golang-migrate`, Taskfile):

```bash
./scripts/setup-dev.sh
```

Then run the server:

```bash
task api:run
```

It listens on `:8080` by default and serves everything under `/api/v0`. In dev
mode it migrates the database on startup, so there's no separate step.

## Tests

```bash
task api:test         # unit tests
task api:integration  # integration tests (in-memory DB + httptest server)
task api:e2e          # end-to-end (builds the binary and runs it for real)
task api:verify       # all three, stops at the first failure
```

Heads up: the integration and e2e suites are behind `-it` / `-e2e` flags, so a
plain `go test ./...` skips them. Use the task targets.

To run a single test, drop into `api/` and use `go test` as usual:

```bash
cd api
go test ./features/auth/ -run TestService
go test ./test/integration/ -it -run TestAuth   # don't forget the flag
```

## Code generation

Two generators, both wired into the tasks:

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
  scheduler/                      background job scheduler (see docs)
  transport/httpbind/             decode → service → encode glue for handlers
  test/                           integration and e2e suites
gotools/                          pinned codegen tools
docs/systems/                     design notes
```

Each feature under `features/` follows the same shape: `init.go` wires it up and
exposes the routes, with the controller/service/repository split underneath. Auth
is the one to copy from.

## Configuration

Everything comes from environment variables, with sensible defaults:

| Variable | Default | Notes |
|----------|---------|-------|
| `APP_ENV` | `dev` | `dev`, `prod`, or `test`. Controls auto-migration, in-memory DB, and the cookie `Secure` flag. |
| `PORT` | `8080` | |
| `BASEPATH` | `/api/v0` | Base path for the API. |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`. |
| `SHUTDOWN_TIMEOUT` | `5s` | How long to wait for in-flight work to drain on shutdown. |

## Background jobs

There's a small in-house scheduler in `api/scheduler/` for periodic and one-shot
work (the session cleanup job runs on it). It handles cancellation, surviving job
errors and panics, escalating a fatal error into a graceful shutdown, and tearing
down cleanly on SIGINT/SIGTERM. The design is written up in
[docs/systems/scheduler.md](docs/systems/scheduler.md).

# Scheduler

Package `mrtutor/api/scheduler` runs background jobs on configurable schedules.
It provides the same capabilities as Spring's `@Scheduled` — periodic and
one-shot jobs, each able to start after a delay or at a precise wall-clock time —
without copying its style. The design is deliberately small (two core types) and
is built so that **V2 cron support drops in as just another schedule, with no
change to the scheduler, the runner, or any caller**.

## Goals

- **Periodic and one-shot ("singleton") jobs**, each startable after a delay or at
  a precise wall-clock instant.
- **Context cancellation** — jobs receive a context that is cancelled on shutdown.
- **Error resilience** — a job that errors or panics is logged and keeps running.
- **Fatal escalation** — a job may declare an error fatal, gracefully stopping the
  whole application.
- **Graceful, leak-free shutdown** on OS signals (SIGINT/SIGTERM).
- **Timezone awareness** — the location is configurable.
- **A clean seam for V2 cron** — parsing a cron expression yields a `Schedule`.

## Core abstractions

The whole package pivots on one question: *given a reference time, when does this
job next fire?* That is the `Schedule` interface.

```go
type Schedule interface {
    // Next returns the next fire strictly after `after`, resolved in `loc`.
    // ok=false means the schedule is exhausted (e.g. a one-shot already fired).
    Next(after time.Time, loc *time.Location) (next time.Time, ok bool)
}

type Job func(ctx context.Context) error
```

A `Job` is the unit of work. A `Schedule` decides when it runs. Everything
else — periodic, one-shot, delayed, start-at, and (V2) cron — is an implementation
of `Schedule`. The runner is schedule-agnostic: it only asks `Next`, waits, runs,
and repeats.

### Why this shape is the cron seam

A cron expression is, by definition, a function "given a time, what is the next
matching instant?" — which is exactly `Next(after, loc)`. V2 adds only:

```go
func Cron(expr string) (Schedule, error) // walks civil time in loc, DST-correct
```

No other file changes. A relative `(delay, repeat bool)` shape (used by the
earlier broken stub) was rejected: it cannot express "02:30 daily in Europe/Rome
across DST" nor a finite-but-multi-fire schedule. The absolute-time `Next` is
strictly more expressive and matches the shape used by `robfig/cron` and
`gocron` v2.

### Built-in schedules (V1)

| Constructor | Meaning |
|---|---|
| `Periodic(interval)` | fixed-rate; first fire one interval after Start |
| `PeriodicAt(first, interval)` | aligned to a wall-clock anchor, then every interval |
| `Once(delay)` | singleton, `delay` after Start |
| `OnceAt(at)` | singleton at a precise wall-clock instant |
| `Delayed(delay, inner)` | first fire `delay` after Start, then defer to `inner` |

The four required combinations:

```go
scheduler.Periodic(time.Hour)                              // periodic
scheduler.PeriodicAt(at, time.Hour)                        // periodic, at a precise time
scheduler.Once(5 * time.Minute)                            // singleton, delayed
scheduler.OnceAt(at)                                       // singleton, at a precise time
scheduler.Delayed(30*time.Second, scheduler.Periodic(time.Hour)) // initial delay, then periodic
```

**Fixed-rate semantics.** `Periodic` anchors fire times to Start
(`anchor`, `anchor+interval`, `anchor+2·interval`, …). The runner computes the
next fire from the *current* time after each run, so a run that overruns its slot
simply **skips the missed slots** and fires at the next aligned slot rather than
bursting to catch up — identical to cron. Fixed-delay (next fire = completion +
interval) can be added later as a separate constructor without an interface
change.

> A `Schedule` instance is **bound to a single job** and may hold state across
> calls (e.g. `Periodic` records its anchor lazily; one-shots remember they
> fired). Do not share a `Schedule` value between jobs.

## Scheduler lifecycle

```go
sched := scheduler.New(logger,
    scheduler.WithLocation(time.UTC),               // default UTC
    scheduler.OnFatal(func(err error) { ... }),     // fatal-error hook
)
sched.Add("cleanup", scheduler.Periodic(time.Hour), cleanupFn) // before Start
sched.Start(appCtx)                                  // non-blocking; one goroutine per job
...
sched.Shutdown(shutdownCtx)                          // cancel + drain in-flight jobs
```

- **`Add`** registers a uniquely-named job; it must be called before `Start`
  (duplicate / empty name / nil schedule / adding after Start all return an error).
- **`Start(ctx)`** derives an internal `context.WithCancelCause(ctx)` and spawns
  one runner goroutine per job. `ctx` is the base of every job's context.
- **`Shutdown(ctx)`** cancels the jobs and waits for in-flight runs to finish,
  bounded by `ctx`; it returns `ctx.Err()` if the deadline elapses first.

### The runner loop

```go
next, ok := schedule.Next(now, loc)
for ok {
    timer := clock.NewTimer(next.Sub(now))
    select {
    case <-runCtx.Done():   // shutdown or fatal: stop the timer and exit
        return
    case <-timer.C():
        invoke(job)         // recover panics, log errors, detect Fatal
        next, ok = schedule.Next(now, loc) // skip slots missed during the run
    }
}
```

Guarantees:

- **No self-overlap** — the next fire is computed only after the current run
  returns, so a job never runs concurrently with itself. Distinct jobs run
  concurrently in their own goroutines.
- **No goroutine leak** — each runner does `defer wg.Done()`; `Shutdown` cancels
  and `wg.Wait()`s against the caller's deadline.
- **No busy-loop** — `Next` must return a time strictly after its argument.

## Error handling & resilience

`invoke` runs the job body inside a `recover()`. Outcomes:

| Job returns | Effect |
|---|---|
| `nil` | success; stays on schedule |
| a non-nil error | logged at error level; stays on schedule (retries next fire) |
| **panic** | recovered, logged as an error; stays on schedule |
| `scheduler.Fatal(err)` | logged, then triggers a graceful **application** shutdown |

A panic is treated as an ordinary (non-fatal) error so a single bad run cannot
crash the process. Wrap with `Fatal` only for conditions the whole app cannot
survive.

## Fatal errors and unified shutdown

A fatal job error and an OS signal converge on **one** shutdown path via
`context.WithCancelCause`:

```
                         ┌─ OS signal (SIGINT/SIGTERM) ─┐
                         │                              ▼
signalCtx ──► appCtx (WithCancelCause) ──► <-appCtx.Done() ──► drain scheduler ──► drain HTTP
                         ▲                              │
   job returns Fatal(err)│                              └─► context.Cause(appCtx) picks exit code
   → OnFatal(err) ───► cancelApp(err)
```

In `main`:

```go
signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
appCtx, cancelApp := context.WithCancelCause(signalCtx)

sched := scheduler.New(logger,
    scheduler.WithLocation(time.UTC),
    scheduler.OnFatal(func(err error) { cancelApp(err) }),
)
sched.Start(appCtx)
...
<-appCtx.Done()                                  // signal OR fatal job
var fatal *scheduler.FatalError
isFatal := errors.As(context.Cause(appCtx), &fatal)

sched.Shutdown(shutdownCtx)                       // jobs first
shutdownServer(...)                               // then drain HTTP
if isFatal { os.Exit(FatalTaskErrorExitCode) }    // exit code 5
```

Shutdown order is preserved from the previous gocron-based design: **background
jobs drain first, then in-flight HTTP requests**, bounded by
`config.ShutdownTimeout`.

## Timezone & DST

`WithLocation` sets the location (default `time.UTC`) passed into every
`Schedule.Next`. The V1 built-ins use **absolute** time arithmetic and therefore
ignore the location for computation; `loc` exists for wall-clock schedules — cron
in V2 — which must resolve civil time and DST in a location.

Consequence: `PeriodicAt(t, 24*time.Hour)` adds an absolute 24 hours, so across a
DST boundary the local wall-clock time shifts by the gained/lost hour (see
`TestPeriodicAtAcrossDST`). For true civil-time recurrence such as "every day at
02:30 local", use cron (V2), which walks civil time and is DST-correct.

## Testing

Time is abstracted behind a `Clock` (`Now` + `NewTimer`) and `Timer`
(`C` + `Stop`) so tests run deterministically without sleeping on the wall clock.
Production uses the real clock; tests inject a fake via `WithClock`.

- `schedule_test.go` — table-driven `Next` math for every constructor, including
  overrun slot-skipping, one-shot exhaustion, and the DST documentation test.
- `scheduler_test.go` — a deterministic fake clock drives the runner in lockstep
  (advance time, wait for the runner to register its next timer). Covers
  registration validation, firing, live job context, panic recovery, non-fatal
  retry, fatal → `OnFatal` + shutdown, no self-overlap, drain-on-shutdown, and the
  shutdown deadline.

Run: `task api:test` (unit) and `go test -race ./scheduler/`.

## V2: cron

V2 adds `func Cron(expr string) (Schedule, error)`. Its `Next` walks civil time in
the scheduler's location to the next matching instant (DST-correct). Nothing else
changes — the runner, lifecycle, error handling, and shutdown are all unaffected.
The application would register a cron job exactly like any other:

```go
spec, err := scheduler.Cron("0 2 * * *") // 02:00 daily, in the scheduler location
sched.Add("nightly", spec, nightlyFn)
```

A per-schedule location override (`InLocation(loc, inner)`) is also a trivial
decorator the `Next(after, loc)` signature already accommodates.

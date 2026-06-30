# Scheduler

`mrtutor/api/scheduler` runs background jobs on a schedule. I wanted what
Spring's `@Scheduled` gives you (periodic jobs, one-shot jobs, a delay or a fixed
start time) but written the way Go wants it, not as a port of the Java version.

The whole thing is built around one idea so that adding cron later (V2) doesn't
mean rewriting anything: a schedule is just "given a time, when do I fire next?".

## The two types

```go
type Schedule interface {
    // Next returns the next fire strictly after `after`, resolved in `loc`.
    // ok == false means there's nothing left to run (a one-shot that already
    // fired, for example).
    Next(after time.Time, loc *time.Location) (next time.Time, ok bool)
}

type Job func(ctx context.Context) error
```

A `Job` is the work. A `Schedule` says when. That's it. Periodic, one-shot,
delayed, start-at, and eventually cron are all just different `Schedule`s; the
runner never knows or cares which one it's driving. It asks `Next`, waits, runs
the job, asks again.

### Why `Next` looks like this

A cron expression is literally a "next matching time after t" function, so when
V2 lands it's only:

```go
func Cron(expr string) (Schedule, error)
```

and nothing else moves. The old (deleted) stub used `(delay, repeat bool)`
instead, which I didn't keep: a relative delay can't say "02:30 every day in
Europe/Rome" correctly once DST is involved, and a bool can't describe a schedule
that fires a few times and then stops. Returning an absolute time covers both,
and it's the same shape `robfig/cron` and gocron v2 use.

## What's built in

```go
scheduler.Periodic(time.Hour)            // every hour, first run an hour after Start
scheduler.PeriodicAt(at, time.Hour)      // anchored to a wall-clock time, then hourly
scheduler.Once(5 * time.Minute)          // run once, 5 minutes after Start
scheduler.OnceAt(at)                     // run once at a specific time
scheduler.Delayed(30*time.Second, scheduler.Periodic(time.Hour)) // wait 30s, then hourly
```

`Periodic` is fixed-rate: fire times are anchored to Start (`anchor`,
`anchor+interval`, `anchor+2·interval`, …). After each run the runner asks for the
next fire using the *current* time, so if a run takes longer than its interval the
missed slots are skipped and it picks up at the next one instead of firing a burst
to catch up. That's the same thing cron does. If we ever need fixed-delay (next
run measured from when the last one finished) it's a new constructor, not a change
to the interface.

One catch worth knowing: a `Schedule` value carries state (Periodic remembers its
anchor, the one-shots remember they've fired), so it belongs to a single job.
Don't reuse the same `Schedule` for two jobs.

## Lifecycle

```go
sched := scheduler.New(logger,
    scheduler.WithLocation(time.UTC),            // UTC if you don't set it
    scheduler.OnFatal(func(err error) { ... }),  // see "Fatal errors" below
)
sched.Add("cleanup", scheduler.Periodic(time.Hour), cleanupFn) // register before Start
sched.Start(appCtx)                              // returns immediately
...
sched.Shutdown(shutdownCtx)                      // stop and wait for running jobs
```

`Add` wants a unique, non-empty name and has to be called before `Start` (it
returns an error otherwise, along with duplicate names and nil arguments).
`Start` takes the context that becomes the parent of every job's context, then
spawns one goroutine per job. `Shutdown` cancels everything and waits for any
in-flight run to return, but only up to whatever deadline you put on its context.

### The runner

```go
next, ok := schedule.Next(now, loc)
for ok {
    timer := clock.NewTimer(next.Sub(now))
    select {
    case <-runCtx.Done():   // shutting down: stop the timer and leave
        return
    case <-timer.C():
        invoke(job)         // recovers panics, logs errors, spots Fatal
        next, ok = schedule.Next(now, loc) // skips slots a long run missed
    }
}
```

A couple of properties fall out of writing it this way. A job never overlaps
itself, because the next fire isn't computed until the current run returns
(different jobs still run in parallel). Nothing leaks, because every runner does
`defer wg.Done()` and `Shutdown` waits on that WaitGroup. And it can't spin,
because `Next` is required to return a time strictly later than the one you pass
in.

## When a job fails

The job runs inside a `recover()`, and what comes back decides what happens:

- `nil` — fine, stays on schedule.
- an error — logged, stays on schedule, runs again next time.
- a panic — recovered and logged like a normal error, stays on schedule.
- `scheduler.Fatal(err)` — logged, then the whole app shuts down (next section).

The point is that a broken job logs and keeps going instead of taking the process
down with it. A panic is just a noisier error. Only reach for `Fatal` when the
condition is something the rest of the app can't run without either.

## Fatal errors and shutdown

I didn't want two different shutdown paths, one for signals and one for fatal
jobs. They both end up cancelling the same context, so `main` only has to wait in
one place and then check *why* it woke up:

```
                         ┌─ OS signal (SIGINT/SIGTERM) ─┐
                         │                              ▼
signalCtx ──► appCtx (WithCancelCause) ──► <-appCtx.Done() ──► drain scheduler ──► drain HTTP
                         ▲                              │
   job returns Fatal(err)│                              └─► context.Cause(appCtx) tells us which
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
<-appCtx.Done()                                  // a signal, or a fatal job
var fatal *scheduler.FatalError
isFatal := errors.As(context.Cause(appCtx), &fatal)

sched.Shutdown(shutdownCtx)                       // jobs first
shutdownServer(...)                               // then HTTP
if isFatal { os.Exit(FatalTaskErrorExitCode) }    // exit 5
```

The order is the same as it was under gocron: stop the background jobs first,
then let in-flight HTTP requests finish, all inside `config.ShutdownTimeout`.

## Timezones and DST

`WithLocation` sets the location (UTC by default) that gets passed to every
`Next` call. The V1 schedules do absolute time arithmetic and don't actually look
at it; the argument is there for the wall-clock schedules that will need it, which
really means cron.

So be aware: `PeriodicAt(t, 24*time.Hour)` adds a literal 24 hours each time, and
across a DST switch the local clock time drifts by the hour you gained or lost
(there's a test for this, `TestPeriodicAtAcrossDST`). If you actually want "02:30
local every day" regardless of DST, that's a job for cron in V2, which walks civil
time properly.

## Tests

Time goes through a `Clock` (`Now` + `NewTimer`) so the tests don't sleep on the
real clock. Production gets the real one; tests pass a fake through `WithClock`.

`schedule_test.go` checks the `Next` arithmetic for each constructor — the slot
skipping on overrun, one-shots running exactly once, and the DST behaviour above.
`scheduler_test.go` uses a fake clock that lets the test step the runner forward
one timer at a time, and covers the registration errors, firing, the job getting a
live context, panic recovery, retry after a normal error, a fatal error reaching
`OnFatal`, jobs not overlapping themselves, draining on shutdown, and the shutdown
deadline.

```
task api:test
go test -race ./scheduler/
```

## V2: cron

V2 is just `Cron(expr string) (Schedule, error)` whose `Next` walks civil time in
the scheduler's location to the next match. The runner, the lifecycle, the error
handling and shutdown all stay as they are. Registering one looks like everything
else:

```go
spec, err := scheduler.Cron("0 2 * * *") // 02:00 daily, in the scheduler's location
sched.Add("nightly", spec, nightlyFn)
```

A per-job location override (`InLocation(loc, inner)`) drops in the same way,
since `Next` already takes the location.

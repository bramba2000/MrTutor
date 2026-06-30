// Package scheduler runs background jobs on configurable schedules.
//
// It offers the same capabilities as Spring's @Scheduled — periodic and one-shot
// jobs, each able to start after a delay or at a precise wall-clock time — in an
// idiomatic Go shape built around two small types: a [Job] (func(ctx) error) and
// a [Schedule] (next-fire calculator). The Schedule interface is the single
// extension point; a cron schedule (planned V2) is just another implementation.
//
// # Lifecycle
//
//	sched := scheduler.New(logger, scheduler.WithLocation(time.UTC))
//	sched.Add("cleanup", scheduler.Periodic(time.Hour), cleanupFn) // before Start
//	sched.Start(ctx)                                               // non-blocking
//	...
//	sched.Shutdown(shutdownCtx)                                    // drain in-flight jobs
//
// # Error handling
//
// A job that returns an error (or panics) is logged and stays on its schedule, so
// one bad run cannot take the process down. A job that returns [Fatal](err) tells
// the scheduler to begin a graceful shutdown of the whole application via the
// [OnFatal] hook — the same path an OS signal would take.
//
// # Cancellation
//
// The context passed to each job is cancelled when the scheduler shuts down, so
// long-running jobs should honor it. [Shutdown] waits for in-flight runs to
// finish, bounded by the context the caller provides.
package scheduler

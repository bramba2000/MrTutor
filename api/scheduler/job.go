package scheduler

import (
	"context"
	"errors"
)

// Job is the unit of work executed by the scheduler.
//
// The provided context is derived from the context passed to [Scheduler.Start]
// and is cancelled when the scheduler shuts down (on an OS signal, a fatal error,
// or an explicit [Scheduler.Shutdown]). Long-running jobs should honor it so they
// can abort promptly and let the application drain cleanly.
//
// Error handling is intentionally forgiving so a single misbehaving job cannot
// take down the process:
//
//   - returning nil       — success; the job stays on its schedule.
//   - returning an error  — logged at error level; the job stays on its schedule
//     and runs again at its next fire (resilient).
//   - returning Fatal(err) — logged, then the scheduler triggers a graceful
//     shutdown of the whole application (see [Fatal]).
//
// A panic inside a Job is recovered and treated as an ordinary (non-fatal) error:
// it is logged and the job keeps its schedule.
type Job func(ctx context.Context) error

// FatalError marks an error returned by a [Job] as fatal: when the scheduler
// observes one, it triggers a graceful shutdown of the entire application,
// following the same path as an OS signal (SIGINT/SIGTERM).
//
// Detect it with errors.As. It unwraps to the underlying error so errors.Is and
// errors.As against the cause keep working.
type FatalError struct {
	Err error
}

func (e *FatalError) Error() string {
	if e.Err == nil {
		return "fatal scheduler error"
	}
	return "fatal: " + e.Err.Error()
}

func (e *FatalError) Unwrap() error { return e.Err }

// Fatal wraps err so that returning it from a [Job] gracefully stops the whole
// application. Use it for conditions a job cannot recover from and that should
// not let the process keep running — e.g. a corrupt invariant or a lost
// dependency that the rest of the app also depends on.
//
//	return scheduler.Fatal(fmt.Errorf("license expired: %w", err))
func Fatal(err error) error { return &FatalError{Err: err} }

// ErrSchedulerClosed is the cancellation cause used when [Scheduler.Shutdown]
// stops the running jobs. It lets jobs distinguish a normal shutdown from other
// cancellations via context.Cause.
var ErrSchedulerClosed = errors.New("scheduler closed")

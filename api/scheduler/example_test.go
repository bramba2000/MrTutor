package scheduler_test

import (
	"context"
	"fmt"
	"log/slog"
	"mrtutor/api/scheduler"
	"time"
)

// Example shows wiring a periodic job, a delayed one-shot, and a fatal-error
// hook. It is compiled (not run) as documentation.
func Example() {
	logger := slog.Default()

	// appCtx is cancelled by an OS signal OR by a fatal job (via OnFatal).
	appCtx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	sched := scheduler.New(logger,
		scheduler.WithLocation(time.UTC),
		scheduler.OnFatal(func(err error) { cancel(err) }),
	)

	// Periodic: every hour, first run one hour after Start.
	_ = sched.Add("cleanup", scheduler.Periodic(time.Hour), func(ctx context.Context) error {
		return nil
	})

	// Singleton, delayed: run once, five minutes after Start.
	_ = sched.Add("warm-cache", scheduler.Once(5*time.Minute), func(ctx context.Context) error {
		return nil
	})

	// A job can escalate an unrecoverable condition to a graceful app shutdown.
	_ = sched.Add("guard", scheduler.Periodic(time.Minute), func(ctx context.Context) error {
		if err := checkInvariant(); err != nil {
			return scheduler.Fatal(fmt.Errorf("invariant violated: %w", err))
		}
		return nil
	})

	sched.Start(appCtx)
	<-appCtx.Done()

	shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	_ = sched.Shutdown(shutdownCtx)
}

func checkInvariant() error { return nil }

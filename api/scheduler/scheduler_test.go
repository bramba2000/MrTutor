package scheduler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mrtutor/api/scheduler"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- deterministic fake clock -------------------------------------------------

// fakeClock implements scheduler.Clock. Tests advance time explicitly and wait
// for the runner to register a timer (waitTimer) so steps happen in lockstep,
// without sleeping on the wall clock.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	timers  []*fakeTimer
	created chan time.Duration
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start, created: make(chan time.Duration, 64)}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) NewTimer(d time.Duration) scheduler.Timer {
	f.mu.Lock()
	t := &fakeTimer{c: make(chan time.Time, 1), deadline: f.now.Add(d)}
	f.timers = append(f.timers, t)
	f.mu.Unlock()

	select {
	case f.created <- d: // signal the test that a timer was registered
	default:
	}
	return t
}

// advance moves time forward and fires every timer whose deadline has passed.
func (f *fakeClock) advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	now := f.now
	var fire []*fakeTimer
	kept := f.timers[:0]
	for _, t := range f.timers {
		if t.deadline.After(now) {
			kept = append(kept, t)
		} else {
			fire = append(fire, t)
		}
	}
	f.timers = kept
	f.mu.Unlock()

	for _, t := range fire {
		t.c <- now
	}
}

// waitTimer blocks until the runner registers its next timer.
func (f *fakeClock) waitTimer(t *testing.T) {
	t.Helper()
	select {
	case <-f.created:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the runner to register a timer")
	}
}

type fakeTimer struct {
	c        chan time.Time
	deadline time.Time
}

func (t *fakeTimer) C() <-chan time.Time { return t.c }
func (t *fakeTimer) Stop() bool          { return true }

// --- registration -------------------------------------------------------------

func TestAddValidation(t *testing.T) {
	t.Parallel()

	noop := func(context.Context) error { return nil }

	t.Run("duplicate name", func(t *testing.T) {
		t.Parallel()
		s := scheduler.New(testLogger())
		if err := s.Add("dup", scheduler.Periodic(time.Hour), noop); err != nil {
			t.Fatalf("first add: %v", err)
		}
		if err := s.Add("dup", scheduler.Periodic(time.Hour), noop); err == nil {
			t.Error("expected duplicate-name error")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		t.Parallel()
		s := scheduler.New(testLogger())
		if err := s.Add("", scheduler.Periodic(time.Hour), noop); err == nil {
			t.Error("expected empty-name error")
		}
	})

	t.Run("after start", func(t *testing.T) {
		t.Parallel()
		s := scheduler.New(testLogger(), scheduler.WithClock(newFakeClock(base)))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s.Start(ctx)
		if err := s.Add("late", scheduler.Periodic(time.Hour), noop); err == nil {
			t.Error("expected error adding after Start")
		}
	})
}

// --- execution ----------------------------------------------------------------

func TestFiresOnSchedule(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(base)
	ran := make(chan struct{}, 16)
	s := scheduler.New(testLogger(), scheduler.WithClock(clock))
	if err := s.Add("tick", scheduler.Periodic(time.Hour), func(context.Context) error {
		ran <- struct{}{}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	for i := 0; i < 3; i++ {
		clock.waitTimer(t)
		clock.advance(time.Hour)
		select {
		case <-ran:
		case <-time.After(2 * time.Second):
			t.Fatalf("job did not run on fire %d", i+1)
		}
	}
}

func TestJobContextCancelledOnShutdown(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(base)
	gotCtx := make(chan context.Context, 1)
	s := scheduler.New(testLogger(), scheduler.WithClock(clock))
	_ = s.Add("ctx", scheduler.Periodic(time.Hour), func(ctx context.Context) error {
		gotCtx <- ctx
		return nil
	})

	s.Start(context.Background())
	clock.waitTimer(t)
	clock.advance(time.Hour)

	ctx := <-gotCtx
	if ctx.Err() != nil {
		t.Fatalf("job ctx already cancelled during run: %v", ctx.Err())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if ctx.Err() == nil {
		t.Error("job ctx should be cancelled after shutdown")
	}
}

func TestPanicIsRecoveredAndJobSurvives(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(base)
	var calls int32
	ran := make(chan struct{}, 16)
	s := scheduler.New(testLogger(), scheduler.WithClock(clock))
	_ = s.Add("panicky", scheduler.Periodic(time.Hour), func(context.Context) error {
		n := atomic.AddInt32(&calls, 1)
		ran <- struct{}{}
		if n == 1 {
			panic("boom")
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// First fire panics; second fire must still happen (scheduler stayed up).
	for i := 0; i < 2; i++ {
		clock.waitTimer(t)
		clock.advance(time.Hour)
		select {
		case <-ran:
		case <-time.After(2 * time.Second):
			t.Fatalf("job did not run on fire %d after panic", i+1)
		}
	}
}

func TestNonFatalErrorKeepsSchedule(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(base)
	ran := make(chan struct{}, 16)
	s := scheduler.New(testLogger(), scheduler.WithClock(clock))
	_ = s.Add("erroring", scheduler.Periodic(time.Hour), func(context.Context) error {
		ran <- struct{}{}
		return errors.New("transient failure")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	for i := 0; i < 2; i++ {
		clock.waitTimer(t)
		clock.advance(time.Hour)
		select {
		case <-ran:
		case <-time.After(2 * time.Second):
			t.Fatalf("job did not re-run after a non-fatal error (fire %d)", i+1)
		}
	}
}

func TestFatalErrorTriggersShutdown(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(base)
	fatalCh := make(chan error, 1)
	s := scheduler.New(testLogger(),
		scheduler.WithClock(clock),
		scheduler.OnFatal(func(err error) { fatalCh <- err }),
	)
	sentinel := errors.New("unrecoverable")
	_ = s.Add("guard", scheduler.Periodic(time.Hour), func(context.Context) error {
		return scheduler.Fatal(sentinel)
	})

	s.Start(context.Background())
	clock.waitTimer(t)
	clock.advance(time.Hour)

	select {
	case err := <-fatalCh:
		var fe *scheduler.FatalError
		if !errors.As(err, &fe) {
			t.Errorf("OnFatal err = %T, want *scheduler.FatalError", err)
		}
		if !errors.Is(err, sentinel) {
			t.Errorf("OnFatal err does not wrap the sentinel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnFatal was not called for a fatal job error")
	}

	// Jobs were cancelled by the fatal trigger, so shutdown returns promptly.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown after fatal: %v", err)
	}
}

func TestNoSelfOverlap(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(base)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var concurrent, maxSeen int32

	s := scheduler.New(testLogger(), scheduler.WithClock(clock))
	_ = s.Add("slow", scheduler.Periodic(time.Hour), func(context.Context) error {
		c := atomic.AddInt32(&concurrent, 1)
		for {
			m := atomic.LoadInt32(&maxSeen)
			if c <= m || atomic.CompareAndSwapInt32(&maxSeen, m, c) {
				break
			}
		}
		entered <- struct{}{}
		<-release
		atomic.AddInt32(&concurrent, -1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	clock.waitTimer(t)
	clock.advance(time.Hour)
	<-entered // job is running and blocked

	// While the job is busy, time marching forward must not start a second run:
	// the runner only computes the next fire after the current run returns.
	clock.advance(10 * time.Hour)
	select {
	case <-entered:
		t.Fatal("job overlapped itself")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	if got := atomic.LoadInt32(&maxSeen); got != 1 {
		t.Errorf("max concurrent runs = %d, want 1", got)
	}
}

func TestShutdownDrainsInFlight(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(base)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	s := scheduler.New(testLogger(), scheduler.WithClock(clock))
	_ = s.Add("inflight", scheduler.Periodic(time.Hour), func(context.Context) error {
		entered <- struct{}{}
		<-release
		return nil
	})

	s.Start(context.Background())
	clock.waitTimer(t)
	clock.advance(time.Hour)
	<-entered

	done := make(chan error, 1)
	go func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done <- s.Shutdown(shutdownCtx)
	}()

	// Shutdown must block while the job is still running.
	select {
	case <-done:
		t.Fatal("shutdown returned before the in-flight job finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("shutdown drained with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not return after the job finished")
	}
}

func TestShutdownDeadlineExceeded(t *testing.T) {
	t.Parallel()

	clock := newFakeClock(base)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release) // let the stuck job exit when the test ends

	s := scheduler.New(testLogger(), scheduler.WithClock(clock))
	_ = s.Add("stuck", scheduler.Periodic(time.Hour), func(context.Context) error {
		entered <- struct{}{}
		<-release // deliberately ignores ctx
		return nil
	})

	s.Start(context.Background())
	clock.waitTimer(t)
	clock.advance(time.Hour)
	<-entered

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("shutdown err = %v, want context.DeadlineExceeded", err)
	}
}

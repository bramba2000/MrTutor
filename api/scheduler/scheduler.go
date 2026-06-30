package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Scheduler runs registered jobs on their schedules. Construct it with [New],
// register jobs with [Add] before [Start], launch the background runners with
// [Start], and drain them with [Shutdown].
//
// Each job runs in its own goroutine and is serialized with respect to itself:
// a job never overlaps a previous run of the same job (the next fire is computed
// only after the current run returns). Different jobs run concurrently.
type Scheduler struct {
	logger  *slog.Logger
	loc     *time.Location
	clock   Clock
	onFatal func(error)

	mu      sync.Mutex
	started bool
	jobs    []*job

	runCtx      context.Context
	cancelCause context.CancelCauseFunc
	wg          sync.WaitGroup
	fatalOnce   sync.Once
}

type job struct {
	name     string
	schedule Schedule
	fn       Job
}

// Option configures a [Scheduler] in [New].
type Option func(*Scheduler)

// WithLocation sets the location used to resolve wall-clock schedules. The
// default is [time.UTC]. A nil location is ignored.
func WithLocation(loc *time.Location) Option {
	return func(s *Scheduler) {
		if loc != nil {
			s.loc = loc
		}
	}
}

// WithClock replaces the time source, primarily so tests can drive the scheduler
// deterministically. The default is the real system clock. A nil clock is ignored.
func WithClock(c Clock) Option {
	return func(s *Scheduler) {
		if c != nil {
			s.clock = c
		}
	}
}

// OnFatal registers a callback invoked exactly once, when a job returns a
// [FatalError]. It runs before the scheduler cancels its own jobs and is the hook
// the application uses to begin a graceful shutdown (e.g. cancel the root context).
func OnFatal(fn func(error)) Option {
	return func(s *Scheduler) { s.onFatal = fn }
}

// New creates a scheduler. The logger is decorated with component="scheduler".
func New(logger *slog.Logger, opts ...Option) *Scheduler {
	s := &Scheduler{
		logger: logger.With("component", "scheduler"),
		loc:    time.UTC,
		clock:  realClock{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Add registers a job under a unique, non-empty name. It must be called before
// [Start]; adding after Start, a duplicate name, an empty name, or a nil schedule
// or function returns an error (the caller decides whether that is fatal).
func (s *Scheduler) Add(name string, sched Schedule, fn Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("scheduler: cannot add job %q after Start", name)
	}
	if name == "" {
		return errors.New("scheduler: job name must not be empty")
	}
	if sched == nil || fn == nil {
		return fmt.Errorf("scheduler: job %q requires both a schedule and a function", name)
	}
	for _, j := range s.jobs {
		if j.name == name {
			return fmt.Errorf("scheduler: duplicate job name %q", name)
		}
	}
	s.jobs = append(s.jobs, &job{name: name, schedule: sched, fn: fn})
	return nil
}

// Start launches one goroutine per registered job. It is non-blocking and is a
// no-op if already started.
//
// ctx is the parent for the scheduler's lifecycle and the base of every job's
// context: cancelling ctx — or a job returning a [FatalError] — stops every job.
// Call [Shutdown] to wait for in-flight runs to finish.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return
	}
	s.started = true
	s.runCtx, s.cancelCause = context.WithCancelCause(ctx)

	s.logger.Info("starting", "jobs", len(s.jobs))
	for _, j := range s.jobs {
		s.wg.Add(1)
		go s.run(j)
	}
}

// Shutdown stops scheduling and waits for in-flight runs to finish, bounded by
// ctx. It returns ctx.Err() if the deadline elapses before all jobs drain, and
// nil otherwise (including when the scheduler was never started).
func (s *Scheduler) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancelCause
	s.mu.Unlock()

	cancel(ErrSchedulerClosed)

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("stopped")
		return nil
	case <-ctx.Done():
		s.logger.Warn("shutdown timed out; jobs still running", "error", ctx.Err())
		return ctx.Err()
	}
}

// run is the per-job loop: compute the next fire, wait for it (or for
// cancellation), execute, repeat until the schedule is exhausted or the
// scheduler is stopped.
func (s *Scheduler) run(j *job) {
	defer s.wg.Done()

	log := s.logger.With("job", j.name)

	next, ok := j.schedule.Next(s.clock.Now().In(s.loc), s.loc)
	if !ok {
		log.Debug("schedule exhausted before first run")
		return
	}

	for {
		timer := s.clock.NewTimer(next.Sub(s.clock.Now()))
		select {
		case <-s.runCtx.Done():
			timer.Stop()
			return
		case <-timer.C():
			s.invoke(j, log)
			// Compute the next fire from "now": an overrunning run skips the
			// slots it missed (fixed-rate) rather than bursting to catch up.
			next, ok = j.schedule.Next(s.clock.Now().In(s.loc), s.loc)
			if !ok {
				log.Debug("schedule exhausted; job complete")
				return
			}
		}
	}
}

// invoke runs the job body once, converting a panic into an ordinary error and
// routing a [FatalError] to a graceful application shutdown. A non-fatal error is
// logged; the job keeps its schedule.
func (s *Scheduler) invoke(j *job, log *slog.Logger) {
	err := s.call(j)
	if err == nil {
		return
	}

	var fatal *FatalError
	if errors.As(err, &fatal) {
		log.Error("job returned a fatal error; shutting down", "error", err)
		s.triggerFatal(err)
		return
	}
	log.Error("job failed; will retry on next schedule", "error", err)
}

// call executes the job function, recovering panics so a misbehaving job cannot
// crash the process.
func (s *Scheduler) call(j *job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("job panicked: %v\n%s", r, debug.Stack())
		}
	}()
	return j.fn(s.runCtx)
}

// triggerFatal notifies the OnFatal hook and cancels the scheduler's jobs. It
// runs at most once even if several jobs fail fatally at the same time.
func (s *Scheduler) triggerFatal(err error) {
	s.fatalOnce.Do(func() {
		if s.onFatal != nil {
			s.onFatal(err)
		}
		s.cancelCause(err)
	})
}

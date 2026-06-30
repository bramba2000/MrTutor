package scheduler

import "time"

// Schedule decides when a job runs. It is the single extension point of the
// scheduler: periodic, one-shot, delayed and start-at schedules are all just
// implementations of Next, and a future cron schedule (V2) is nothing more than
// another implementation — no change to the Scheduler, the runner, or any caller.
//
// The contract:
//
//   - Next returns the next instant the job should fire, strictly after `after`.
//     Returning a time that is not strictly greater than `after` is a bug: the
//     runner would fire back-to-back.
//   - ok reports whether there is a next fire at all. ok=false means the schedule
//     is exhausted (e.g. a one-shot that already fired); the runner then stops
//     the job cleanly, without error.
//   - `loc` is the scheduler's location (see [WithLocation]). The V1 built-ins use
//     absolute time arithmetic and ignore it; it exists for wall-clock schedules
//     such as cron, which must resolve civil time (and DST) in a location.
//
// A Schedule instance is BOUND TO A SINGLE JOB. Implementations may keep state
// across calls (for example, [Periodic] records its anchor on the first call, and
// the one-shots remember that they have fired). Do not share one Schedule value
// between two jobs.
type Schedule interface {
	Next(after time.Time, loc *time.Location) (next time.Time, ok bool)
}

// Periodic fires every interval using fixed-rate semantics: fire times are
// anchored to the moment of the first call (Start), at anchor+interval,
// anchor+2·interval, and so on. The first fire is one interval after Start.
//
// If a run overruns its slot, the missed slots are skipped and the job fires at
// the next aligned slot rather than bursting to catch up — the same behaviour a
// cron schedule exhibits. interval must be > 0.
//
//	scheduler.Periodic(time.Hour) // every hour, first run one hour after Start
func Periodic(interval time.Duration) Schedule {
	return &everySchedule{interval: interval}
}

// PeriodicAt fires at the wall-clock instant `first`, then every interval
// thereafter (first, first+interval, first+2·interval, …). If `first` is already
// in the past at Start, the first fire is the next aligned slot after now.
//
// Arithmetic is absolute (interval nanoseconds added each time), so PeriodicAt is
// not DST-aware; for civil-time recurrence such as "every day at 02:30 local",
// use a cron schedule (V2). interval must be > 0.
func PeriodicAt(first time.Time, interval time.Duration) Schedule {
	return everyAtSchedule{first: first, interval: interval}
}

// Once fires exactly one time, `delay` after Start, then the schedule is
// exhausted. This is the "singleton, delayed" task.
//
//	scheduler.Once(5 * time.Minute) // run once, five minutes after Start
func Once(delay time.Duration) Schedule {
	return &onceSchedule{delay: delay}
}

// OnceAt fires exactly one time at the wall-clock instant `at`, then the schedule
// is exhausted. This is the "singleton, at a precise time" task. If `at` is
// already in the past at Start, the job never runs.
func OnceAt(at time.Time) Schedule {
	return &onceAtSchedule{at: at}
}

// Delayed makes the first fire happen `delay` after Start and then defers to
// inner for every subsequent fire (inner anchors itself from that first fire).
// It is the idiomatic way to express "initial delay, then recurring":
//
//	// fire 30s after Start, then every hour
//	scheduler.Delayed(30*time.Second, scheduler.Periodic(time.Hour))
//
// Delayed is intended to wrap a recurring inner schedule; for a delayed one-shot
// use [Once] instead.
func Delayed(delay time.Duration, inner Schedule) Schedule {
	return &delayedSchedule{delay: delay, inner: inner}
}

// everySchedule implements fixed-rate [Periodic]. The anchor is recorded lazily
// on the first Next call so that fire times align to Start.
type everySchedule struct {
	interval time.Duration
	anchor   time.Time
	anchored bool
}

func (s *everySchedule) Next(after time.Time, _ *time.Location) (time.Time, bool) {
	if !s.anchored {
		s.anchor = after
		s.anchored = true
	}
	// Smallest anchor+k·interval strictly greater than `after` (k≥1 on the first
	// call). Passing the post-completion time as `after` naturally skips slots an
	// overrunning run missed.
	n := int64(after.Sub(s.anchor) / s.interval)
	return s.anchor.Add(time.Duration(n+1) * s.interval), true
}

// everyAtSchedule implements [PeriodicAt]. It is stateless: every fire is derived
// from the fixed anchor `first`.
type everyAtSchedule struct {
	first    time.Time
	interval time.Duration
}

func (s everyAtSchedule) Next(after time.Time, _ *time.Location) (time.Time, bool) {
	if after.Before(s.first) {
		return s.first, true
	}
	n := int64(after.Sub(s.first) / s.interval)
	return s.first.Add(time.Duration(n+1) * s.interval), true
}

// onceSchedule implements [Once].
type onceSchedule struct {
	delay     time.Duration
	scheduled bool
}

func (s *onceSchedule) Next(after time.Time, _ *time.Location) (time.Time, bool) {
	if s.scheduled {
		return time.Time{}, false
	}
	s.scheduled = true
	return after.Add(s.delay), true
}

// onceAtSchedule implements [OnceAt].
type onceAtSchedule struct {
	at        time.Time
	scheduled bool
}

func (s *onceAtSchedule) Next(after time.Time, _ *time.Location) (time.Time, bool) {
	if s.scheduled {
		return time.Time{}, false
	}
	s.scheduled = true
	if s.at.After(after) {
		return s.at, true
	}
	return time.Time{}, false
}

// delayedSchedule implements [Delayed].
type delayedSchedule struct {
	delay   time.Duration
	inner   Schedule
	started bool
}

func (s *delayedSchedule) Next(after time.Time, loc *time.Location) (time.Time, bool) {
	if !s.started {
		s.started = true
		return after.Add(s.delay), true
	}
	return s.inner.Next(after, loc)
}

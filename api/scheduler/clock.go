package scheduler

import "time"

// Clock abstracts the passage of time so the scheduler can be driven
// deterministically in tests instead of sleeping on the wall clock.
//
// Production uses [realClock] (the default). Tests inject a fake via
// [WithClock] to control Now and to fire timers on demand.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// NewTimer creates a Timer that fires once after at least duration d.
	// A non-positive d fires as soon as possible.
	NewTimer(d time.Duration) Timer
}

// Timer mirrors the small slice of *time.Timer the runner relies on, so a fake
// clock can supply its own timers.
type Timer interface {
	// C is the channel on which the time is delivered when the timer fires.
	C() <-chan time.Time
	// Stop prevents the timer from firing. It reports whether it stopped the
	// timer before it fired.
	Stop() bool
}

// realClock is the production [Clock], backed by the standard library.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) NewTimer(d time.Duration) Timer { return &realTimer{t: time.NewTimer(d)} }

// realTimer adapts *time.Timer to the [Timer] interface.
type realTimer struct{ t *time.Timer }

func (r *realTimer) C() <-chan time.Time { return r.t.C }

func (r *realTimer) Stop() bool { return r.t.Stop() }

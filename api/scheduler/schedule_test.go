package scheduler_test

import (
	"mrtutor/api/scheduler"
	"testing"
	"time"
)

var base = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestPeriodic(t *testing.T) {
	t.Parallel()

	s := scheduler.Periodic(time.Hour)

	// First fire is one interval after the anchor (set on the first call).
	if got, ok := s.Next(base, time.UTC); !ok || !got.Equal(base.Add(time.Hour)) {
		t.Fatalf("first fire = %v (ok=%v), want %v", got, ok, base.Add(time.Hour))
	}
	// Steady cadence.
	if got, _ := s.Next(base.Add(time.Hour), time.UTC); !got.Equal(base.Add(2 * time.Hour)) {
		t.Errorf("second fire = %v, want %v", got, base.Add(2*time.Hour))
	}
	// Overrun: a run that finishes at 2h30m skips the 2h slot and fires at 3h.
	if got, _ := s.Next(base.Add(2*time.Hour+30*time.Minute), time.UTC); !got.Equal(base.Add(3 * time.Hour)) {
		t.Errorf("after overrun = %v, want %v", got, base.Add(3*time.Hour))
	}
}

func TestPeriodicAt(t *testing.T) {
	t.Parallel()

	first := base.Add(2 * time.Hour)
	s := scheduler.PeriodicAt(first, time.Hour)

	tt := []struct {
		name  string
		after time.Time
		want  time.Time
	}{
		{"before anchor fires at anchor", base, first},
		{"at anchor advances one interval", first, first.Add(time.Hour)},
		{"skips missed slot", first.Add(90 * time.Minute), first.Add(2 * time.Hour)},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := s.Next(tc.after, time.UTC)
			if !ok || !got.Equal(tc.want) {
				t.Errorf("Next(%v) = %v (ok=%v), want %v", tc.after, got, ok, tc.want)
			}
		})
	}
}

func TestOnce(t *testing.T) {
	t.Parallel()

	s := scheduler.Once(5 * time.Minute)

	got, ok := s.Next(base, time.UTC)
	if !ok || !got.Equal(base.Add(5*time.Minute)) {
		t.Fatalf("first = %v (ok=%v), want %v", got, ok, base.Add(5*time.Minute))
	}
	if _, ok := s.Next(got, time.UTC); ok {
		t.Error("one-shot fired a second time")
	}
}

func TestOnceAt(t *testing.T) {
	t.Parallel()

	t.Run("future fires once then exhausts", func(t *testing.T) {
		t.Parallel()
		at := base.Add(time.Hour)
		s := scheduler.OnceAt(at)
		if got, ok := s.Next(base, time.UTC); !ok || !got.Equal(at) {
			t.Fatalf("first = %v (ok=%v), want %v", got, ok, at)
		}
		if _, ok := s.Next(at, time.UTC); ok {
			t.Error("one-shot fired a second time")
		}
	})

	t.Run("past never fires", func(t *testing.T) {
		t.Parallel()
		s := scheduler.OnceAt(base.Add(-time.Hour))
		if _, ok := s.Next(base, time.UTC); ok {
			t.Error("past one-shot should not fire")
		}
	})
}

func TestDelayed(t *testing.T) {
	t.Parallel()

	s := scheduler.Delayed(30*time.Second, scheduler.Periodic(time.Hour))

	// First fire is delay after start.
	first, ok := s.Next(base, time.UTC)
	if !ok || !first.Equal(base.Add(30*time.Second)) {
		t.Fatalf("first = %v (ok=%v), want %v", first, ok, base.Add(30*time.Second))
	}
	// Inner periodic anchors from the first fire: next is delay + one interval.
	second, _ := s.Next(first, time.UTC)
	if want := base.Add(30*time.Second + time.Hour); !second.Equal(want) {
		t.Errorf("second = %v, want %v", second, want)
	}
	third, _ := s.Next(second, time.UTC)
	if want := base.Add(30*time.Second + 2*time.Hour); !third.Equal(want) {
		t.Errorf("third = %v, want %v", third, want)
	}
}

// TestPeriodicAtAcrossDST documents that PeriodicAt uses absolute interval
// arithmetic and is therefore NOT civil-time/DST aware: adding 24h across the
// spring-forward boundary shifts the local wall-clock time by an hour. Civil-time
// recurrence is a job for cron (V2).
func TestPeriodicAtAcrossDST(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}

	// 2026-03-29 is the EU spring-forward day (02:00 -> 03:00).
	first := time.Date(2026, 3, 29, 1, 30, 0, 0, loc)
	s := scheduler.PeriodicAt(first, 24*time.Hour)

	got, _ := s.Next(first, loc)
	want := time.Date(2026, 3, 30, 2, 30, 0, 0, loc) // 01:30 + 24h absolute, now in CEST
	if !got.Equal(want) {
		t.Errorf("next = %v, want %v (absolute +24h crossing DST)", got.In(loc), want)
	}
	if h := got.In(loc).Hour(); h != 2 {
		t.Errorf("local hour after DST = %d, want 2 (wall-clock shifted by the lost hour)", h)
	}
}

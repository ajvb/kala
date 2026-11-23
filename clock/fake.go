package clock

import (
	"sync"
	"time"
)

// Fake is a clock for tests. Time stands still until Advance is called, and
// Advance runs the timers that become due, in time order, without sleeping.
// It is safe for concurrent use.
type Fake struct {
	mu     sync.Mutex
	now    time.Time
	seq    uint64
	timers []*fakeTimer // pending timers only
	fired  int
}

// NewFake returns a Fake whose current time is start.
func NewFake(start time.Time) *Fake {
	return &Fake{now: start}
}

// Now returns the fake time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// AfterFunc schedules fn to run when the clock reaches now+d. A zero or
// negative d makes the timer due at the current time, so it runs on the next
// call to Advance (including Advance(0)). fn runs inside Advance, on the
// caller's goroutine, with no lock held, so it may call AfterFunc, Stop and Now.
func (f *Fake) AfterFunc(d time.Duration, fn func()) Timer {
	f.mu.Lock()
	defer f.mu.Unlock()
	due := f.now
	if d > 0 {
		due = f.now.Add(d)
	}
	f.seq++
	t := &fakeTimer{clock: f, due: due, seq: f.seq, fn: fn}
	f.timers = append(f.timers, t)
	return t
}

// Advance moves the clock forward by d and runs every timer that is due on the
// way, earliest first (timers due at the same time run in the order they were
// created). While a timer's function runs, Now returns that timer's due time,
// and timers the function creates that are due before the end are run too.
// When Advance returns, Now is at least the start time plus d (if another
// goroutine advanced the clock further in the meantime, Now keeps that later
// time: the clock never moves backwards). Advance panics if d is negative. If a
// timer's function panics, the panic passes through Advance, Now stays at that
// timer's due time and the timers that have not run stay pending.
func (f *Fake) Advance(d time.Duration) {
	if d < 0 {
		panic("clock: Fake.Advance called with a negative duration")
	}
	f.mu.Lock()
	target := f.now.Add(d)
	for {
		next := f.earliestDueLocked(target)
		if next == nil {
			break
		}
		f.removeLocked(next)
		next.done = true
		f.fired++
		if next.due.After(f.now) {
			f.now = next.due
		}
		fn := next.fn
		f.mu.Unlock()
		fn()
		f.mu.Lock()
	}
	if target.After(f.now) {
		f.now = target
	}
	f.mu.Unlock()
}

// Fired returns how many timers have run so far. Because Advance runs due
// timers before it returns, a test can check right after Advance whether a
// timer ran, without waiting.
func (f *Fake) Fired() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fired
}

// earliestDueLocked returns the pending timer due first at or before limit, or
// nil. The caller holds f.mu.
func (f *Fake) earliestDueLocked(limit time.Time) *fakeTimer {
	var best *fakeTimer
	for _, t := range f.timers {
		if t.due.After(limit) {
			continue
		}
		if best == nil || t.due.Before(best.due) || (t.due.Equal(best.due) && t.seq < best.seq) {
			best = t
		}
	}
	return best
}

func (f *Fake) removeLocked(t *fakeTimer) {
	for i, other := range f.timers {
		if other == t {
			f.timers = append(f.timers[:i], f.timers[i+1:]...)
			return
		}
	}
}

type fakeTimer struct {
	clock *Fake
	due   time.Time
	seq   uint64
	fn    func()
	done  bool
}

// Stop prevents the function from running if it has not started.
func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	if t.done {
		return false
	}
	t.clock.removeLocked(t)
	t.done = true // stopped: a second Stop reports false
	return true
}

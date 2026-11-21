// Package clock is a wrapper around the standard library "time" to allow for
// easy use of fakes. Code that needs the time or a timer takes a Clock, so
// tests can run it with a fake that is advanced by hand instead of waiting for
// real time to pass.
//
// The interface is deliberately small: the current time and a stoppable timer
// that calls a function after a duration. Real is the clock backed by the
// standard library; Fake is the fake, and advancing time exists only on Fake.
package clock

import "time"

// Clock provides the current time and timers.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// AfterFunc waits for d to elapse and then calls f in its own goroutine
	// (Real) or from Fake.Advance (Fake). The returned Timer cancels the call.
	AfterFunc(d time.Duration, f func()) Timer
}

// Timer is a pending call started by Clock.AfterFunc.
type Timer interface {
	// Stop prevents the call if it has not started yet. It reports whether it
	// did so: true if the call was stopped, false if the call already started
	// or the timer was already stopped.
	Stop() bool
}

// Real is the clock of the operating system. The zero value is ready to use.
type Real struct{}

// Now returns time.Now().
func (Real) Now() time.Time { return time.Now() }

// AfterFunc calls time.AfterFunc. The timer is kept by pointer and is never
// copied.
func (Real) AfterFunc(d time.Duration, f func()) Timer {
	return realTimer{t: time.AfterFunc(d, f)}
}

type realTimer struct{ t *time.Timer }

func (r realTimer) Stop() bool { return r.t.Stop() }

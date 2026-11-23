package job

import (
	"sync/atomic"
	"time"

	"github.com/ajvb/kala/v2/clock"
)

// steppingClock is a fake clock whose reported time also moves forward by one
// nanosecond every time it is read. Events that happen one after another, such
// as a parent job and the jobs that depend on it, then get different, strictly
// increasing timestamps without any real waiting. Timers still follow the
// underlying fake: only Advance makes them fire.
type steppingClock struct {
	*clock.Fake
	reads atomic.Int64
}

func newSteppingClock(start time.Time) *steppingClock {
	return &steppingClock{Fake: clock.NewFake(start)}
}

func (s *steppingClock) Now() time.Time {
	return s.Fake.Now().Add(time.Duration(s.reads.Add(1)))
}

// newTestClock returns a fake clock that starts on a whole second. Schedules
// are written with whole seconds, so a job scheduled a whole number of seconds
// later becomes due exactly at the time an Advance moves the clock to.
func newTestClock() *clock.Fake {
	return clock.NewFake(time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC))
}

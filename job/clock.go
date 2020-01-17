package job

import (
	"sync"

	// This library abstracts the time functionality of the OS so that it can be controlled during unit tests.
	// It was selected over thejerf/abtime because abtime is geared towards precision timing rather than scheduling.
	// Other libraries were reviewed and rejected due to outstanding bugs.
	"github.com/mixer/clock"
)

type Clock struct {
	clock.Clock
	lock sync.RWMutex
}

func (clk *Clock) SetClock(in clock.Clock) {
	clk.lock.Lock()
	defer clk.lock.Unlock()
	clk.Clock = in
}

func (clk *Clock) Time() clock.Clock {
	clk.lock.RLock()
	defer clk.lock.RUnlock()

	if clk.Clock == nil {
		clk.lock.RUnlock()
		clk.lock.Lock()
		clk.Clock = clock.C
		clk.lock.Unlock()
		clk.lock.RLock()
	}

	return clk.Clock
}

func (clk *Clock) TimeSet() bool {
	clk.lock.RLock()
	defer clk.lock.RUnlock()
	return clk.Clock != nil
}

type Clocker interface {
	Time() clock.Clock
	TimeSet() bool
}

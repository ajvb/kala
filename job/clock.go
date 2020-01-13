package job

import (
	"sync"
	"time"

	// This library abstracts the time functionality of the OS so that it can be controlled during unit tests.
	// It was selected over thejerf/abtime because abtime is geared towards precision timing rather than scheduling.
	// Other libraries were reviewed and rejected due to outstanding bugs.
	"github.com/mixer/clock"
)

// From a truly ideal standpoint, a clock.Clock would be injected into the appropriate objects.
// This would allow unit tests to be run in parallel, etc.
// For now, for the sake of simplicity, we are using a package-level variable for this.
var pkgClock clock.Clock = clock.C

// Special hybrid clock that allows you to make time "play" in addition to moving it around manually.
type HybridClock struct {
	*clock.MockClock
	offset time.Duration
	m      sync.RWMutex
}

func NewHybridClock(start ...time.Time) *HybridClock {
	return &HybridClock{
		MockClock: clock.NewMockClock(start...),
	}
}

func (hc *HybridClock) Play() {
	hc.m.Lock()
	defer hc.m.Unlock()
	hc.offset = hc.MockClock.Now().Sub(time.Now())
}

func (hc *HybridClock) Pause() {
	hc.m.Lock()
	defer hc.m.Unlock()
	hc.offset = 0
}

func (hc *HybridClock) IsPlaying() bool {
	hc.m.RLock()
	defer hc.m.RUnlock()
	return hc.offset != 0
}

func (hc *HybridClock) Now() time.Time {
	hc.m.RLock()
	defer hc.m.RUnlock()
	if hc.offset == 0 {
		return hc.MockClock.Now()
	} else {
		return time.Now().Add(hc.offset)
	}
}

func (hc *HybridClock) AddTime(d time.Duration) {
	hc.MockClock.AddTime(d)
	if hc.IsPlaying() {
		hc.Play()
	}
}

func (hc *HybridClock) SetTime(t time.Time) {
	hc.MockClock.SetTime(t)
	if hc.IsPlaying() {
		hc.Play()
	}
}

// Utility function & mutex to swap the clock in/out
func mockPkgClock(clk clock.Clock) func() {
	m.Lock()
	pkgClock = clk
	return func() {
		pkgClock = clock.C
		m.Unlock()
	}
}

var m sync.Mutex

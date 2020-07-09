package job

import (
	"sync"
	"time"

	"github.com/mixer/clock"
)

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
	hc.offset = time.Until(hc.MockClock.Now())
}

func (hc *HybridClock) Pause() {
	hc.m.Lock()
	defer hc.m.Unlock()
	hc.offset = 0
}

func (hc *HybridClock) IsPlaying() (result bool) {
	hc.m.RLock()
	result = hc.offset != 0
	hc.m.RUnlock()
	return
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

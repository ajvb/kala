package job

import (
	"testing"
	"time"

	"github.com/ajvb/kala/v2/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file contains tests for specific JobCaches.

func TestCacheStart(t *testing.T) {
	cache := NewMockCache()
	cache.Start(time.Hour, time.Hour)
}

func TestCacheRetainShouldRemoveOldJobStats(t *testing.T) {
	clk := newTestClock()
	cache := NewMockCache(WithClock(clk))
	mockDb := &MockDBGetAll{}
	cache.jobDB = mockDb

	pastDate := time.Date(2016, time.April, 12, 20, 0, 0, 0, time.UTC)
	j := GetMockRecurringJobWithSchedule(pastDate, "PT1S")
	j.Stats = GetMockJobStats(pastDate, 5)
	j.Id = "0"
	j.ranChan = make(chan struct{})

	jobs := []*Job{j}
	mockDb.response = jobs

	cache.Start(0, 1*time.Minute) // Retain 1 minute
	j.lock.RLock()
	assert.Equal(t, 5, len(j.Stats))
	j.lock.RUnlock()

	clk.Advance(time.Second)
	awaitJobRan(t, j, 5*time.Second)
	cache.Retain()

	j.lock.RLock()
	assert.Equal(t, 1, len(j.Stats)) // New job stats should not be cleaned up
	j.lock.RUnlock()
}

func TestCacheStartStartsARecurringJobWithStartDateInThePast(t *testing.T) {
	clk := newTestClock()
	cache := NewMockCache(WithClock(clk))
	mockDb := &MockDBGetAll{}
	cache.jobDB = mockDb

	pastDate := time.Date(2016, time.April, 12, 20, 0, 0, 0, time.UTC)
	j := GetMockRecurringJobWithSchedule(pastDate, "PT1S")
	j.Id = "0"
	j.ranChan = make(chan struct{})

	jobs := []*Job{j}
	mockDb.response = jobs

	cache.Start(0, -1)
	clk.Advance(time.Second)
	awaitJobRan(t, j, 5*time.Second)

	j.lock.RLock()
	assert.Equal(t, j.Metadata.SuccessCount, uint(1))
	j.lock.RUnlock()
}

// A job that resumes at its next scheduled point runs only at those points,
// not while it is disabled, and not right away when it is enabled again.
func TestCacheStartCanResumeJobAtNextScheduledPoint(t *testing.T) {
	// Each read moves the clock on by a nanosecond, so a job that has just run
	// at a scheduled point does not find itself due at that point again.
	clk := newSteppingClock(newTestClock().Now())
	cache := NewMockCache(WithClock(clk))
	mockDb := &MockDBGetAll{}
	cache.jobDB = mockDb

	// Scheduled points: start-1s, start+2s, start+5s, start+8s, ...
	j := GetMockRecurringJobWithSchedule(clk.Now().Add(-1*time.Second), "PT3S")
	j.Id = "0"
	j.ResumeAtNextScheduledTime = true
	j.InitDelayDuration(false)
	j.ranChan = make(chan struct{})

	jobs := []*Job{j}
	mockDb.response = jobs

	cache.Start(0, -1)
	successes := func() int {
		j.lock.RLock()
		defer j.lock.RUnlock()
		return int(j.Metadata.SuccessCount)
	}

	clk.Advance(time.Second)
	assert.Equal(t, 0, clk.Fired(), "not run before its next point")
	assert.Equal(t, 0, successes())

	clk.Advance(time.Second)
	awaitJobRan(t, j, 5*time.Second)
	assert.Equal(t, 1, successes(), "run at start+2s")

	assert.NoError(t, j.Disable(cache))
	clk.Advance(2 * time.Second)
	assert.Equal(t, 1, clk.Fired(), "not run while disabled")
	assert.Equal(t, 1, successes())

	// Enabled at start+4s: the next point is start+5s.
	assert.NoError(t, j.Enable(cache))
	clk.Advance(500 * time.Millisecond)
	assert.Equal(t, 1, clk.Fired(), "not run right away when enabled")
	assert.Equal(t, 1, successes())

	clk.Advance(500 * time.Millisecond)
	awaitJobRan(t, j, 5*time.Second)
	assert.Equal(t, 2, successes(), "run again at start+5s")
}

func TestNilClockOptionKeepsTheRealClock(t *testing.T) {
	cache := NewMockCache(WithClock(nil))

	require.NotNil(t, cache.Clock())
	assert.WithinDuration(t, time.Now(), cache.Clock().Now(), 5*time.Second)
	assert.NotPanics(t, func() { NewKalaStats(cache) })
}

func TestCacheClockIsTheOneItWasBuiltWith(t *testing.T) {
	fake := clock.NewFake(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	cache := NewMockCache(WithClock(fake))

	assert.Same(t, fake, cache.Clock())
}

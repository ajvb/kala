package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// This file contains tests for specific JobCaches.

func TestCacheStart(t *testing.T) {
	cache := NewMockCache()
	cache.Start(time.Duration(time.Hour), time.Duration(time.Hour))
}

func TestCacheRetainShouldRemoveOldJobStats(t *testing.T) {
	cache := NewMockCache()
	mockDb := &MockDBGetAll{}
	cache.jobDB = mockDb

	pastDate := time.Date(2016, time.April, 12, 20, 00, 00, 0, time.UTC)
	j := GetMockRecurringJobWithSchedule(pastDate, "PT1S")
	j.Stats = GetMockJobStats(pastDate, 5)
	j.Id = "0"

	jobs := make([]*Job, 0, 0)
	jobs = append(jobs, j)
	mockDb.response = jobs

	cache.Start(0, 1*time.Minute) // Retain 1 minute
	j.lock.RLock()
	assert.Equal(t, 5, len(j.Stats))
	j.lock.RUnlock()

	time.Sleep(time.Second * 2)
	cache.Retain()

	j.lock.RLock()
	assert.Equal(t, 1, len(j.Stats)) // New job stats should not be cleaned up
	j.lock.RUnlock()
}

func TestCacheStartStartsARecurringJobWithStartDateInThePast(t *testing.T) {

	cache := NewMockCache()
	mockDb := &MockDBGetAll{}
	cache.jobDB = mockDb

	pastDate := time.Date(2016, time.April, 12, 20, 00, 00, 0, time.UTC)
	j := GetMockRecurringJobWithSchedule(pastDate, "PT1S")
	j.Id = "0"

	jobs := make([]*Job, 0, 0)
	jobs = append(jobs, j)
	mockDb.response = jobs

	cache.Start(0, -1)
	time.Sleep(time.Second * 2)

	j.lock.RLock()
	assert.Equal(t, j.Metadata.SuccessCount, uint(1))
	j.lock.RUnlock()
}

func TestCacheStartCanResumeJobAtNextScheduledPoint(t *testing.T) {

	cache := NewMockCache()
	mockDb := &MockDBGetAll{}
	cache.jobDB = mockDb

	pastDate := time.Now().Add(-1 * time.Second)
	j := GetMockRecurringJobWithSchedule(pastDate, "PT3S")
	j.Id = "0"
	j.ResumeAtNextScheduledTime = true
	j.InitDelayDuration(false)

	jobs := make([]*Job, 0, 0)
	jobs = append(jobs, j)
	mockDb.response = jobs

	cache.Start(0, -1)

	// After 1 second, the job should not have run.
	time.Sleep(time.Second * 1)
	j.lock.RLock()
	assert.Equal(t, 0, int(j.Metadata.SuccessCount))
	j.lock.RUnlock()

	// After 2 more seconds, it should have run.
	time.Sleep(time.Second * 2)
	j.lock.RLock()
	assert.Equal(t, 1, int(j.Metadata.SuccessCount))
	j.lock.RUnlock()

	// Disable to prevent from running
	assert.NoError(t, j.Disable(cache))

	// It shouldn't run while it's disabled.
	time.Sleep(time.Second * 3)
	j.lock.RLock()
	assert.Equal(t, 1, int(j.Metadata.SuccessCount))
	j.lock.RUnlock()

	// Re-enable
	assert.NoError(t, j.Enable(cache))

	// It shouldn't re-run right away; should wait for its next run point.
	time.Sleep(time.Second * 1)
	j.lock.RLock()
	assert.Equal(t, 1, int(j.Metadata.SuccessCount))
	j.lock.RUnlock()

	// Now it should have run again.
	time.Sleep(time.Second * 2)
	j.lock.RLock()
	assert.Equal(t, 2, int(j.Metadata.SuccessCount))
	j.lock.RUnlock()

}

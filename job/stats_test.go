package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJobStats(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJobWithGenericSchedule(time.Now())
	j.Init(cache)

	j.Run(cache)
	now := time.Now()

	assert.NotNil(t, j.Stats[0])
	assert.Equal(t, j.Stats[0].JobId, j.Id)
	assert.WithinDuration(t, j.Stats[0].RanAt, now, time.Second)
	assert.Equal(t, j.Stats[0].NumberOfRetries, uint(0))
	assert.True(t, j.Stats[0].Success)
	assert.NotNil(t, j.Stats[0].ExecutionDuration)
}

func TestKalaStats(t *testing.T) {
	clk := newTestClock()
	cache := NewMockCache(WithClock(clk))

	for i := 0; i < 5; i++ {
		j := GetMockJobWithGenericSchedule(clk.Now())
		j.Init(cache)
		j.Run(cache)
	}
	for i := 0; i < 5; i++ {
		j := GetMockJobWithGenericSchedule(clk.Now())
		j.Init(cache)
		assert.NoError(t, j.Disable(cache))
	}

	assert.Equal(t, &KalaStats{
		Jobs:             10,
		ActiveJobs:       5,
		DisabledJobs:     5,
		SuccessCount:     5,
		ErrorCount:       0,
		NextRunAt:        clk.Now().Add(5 * time.Minute),
		LastAttemptedRun: clk.Now(),
		CreatedAt:        clk.Now(),
	}, NewKalaStats(cache))
}

func TestNextRunAt(t *testing.T) {
	cache := NewMockCache()

	sched := time.Now().Add(time.Second)
	j := GetMockJobWithSchedule(2, sched, "P1DT10M10S")
	j.Init(cache)
	assert.NoError(t, j.Disable(cache))

	sched2 := time.Now().Add(2 * time.Second)
	j2 := GetMockJobWithSchedule(2, sched2, "P1DT10M10S")
	j2.Init(cache)
	assert.NoError(t, j2.Disable(cache))

	kalaStat := NewKalaStats(cache)
	j.lock.RLock()
	assert.Equal(t, j.NextRunAt.UnixNano(), kalaStat.NextRunAt.UnixNano())
	assert.Equal(t, j.Metadata.LastAttemptedRun.UnixNano(), kalaStat.LastAttemptedRun.UnixNano())
	assert.NotEqual(t, j2.NextRunAt.UnixNano(), kalaStat.NextRunAt.UnixNano())
	j.lock.RUnlock()
}

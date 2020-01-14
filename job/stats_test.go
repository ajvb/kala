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
	cache := NewMockCache()

	for i := 0; i < 5; i++ {
		j := GetMockJobWithGenericSchedule(time.Now())
		j.Init(cache)
		j.Run(cache)
	}
	now := time.Now()
	for i := 0; i < 5; i++ {
		j := GetMockJobWithGenericSchedule(time.Now())
		j.Init(cache)
		j.Disable()
	}

	kalaStat := NewKalaStats(cache)
	createdAt := time.Now()
	nextRunAt := time.Now().Add(
		time.Duration(time.Minute * 5),
	)

	assert.Equal(t, kalaStat.Jobs, 10)
	assert.Equal(t, kalaStat.ActiveJobs, 5)
	assert.Equal(t, kalaStat.DisabledJobs, 5)
	assert.Equal(t, kalaStat.SuccessCount, uint(5))
	assert.Equal(t, kalaStat.ErrorCount, uint(0))
	assert.True(t, (nextRunAt.UnixNano()-kalaStat.NextRunAt.UnixNano()) > 0)
	assert.WithinDuration(t, kalaStat.NextRunAt, nextRunAt, time.Second)
	assert.WithinDuration(t, kalaStat.LastAttemptedRun, now, time.Millisecond*100)
	assert.WithinDuration(t, kalaStat.CreatedAt, createdAt, time.Millisecond*100)
}

func TestNextRunAt(t *testing.T) {
	cache := NewMockCache()

	sched := time.Now().Add(time.Second)
	j := GetMockJobWithSchedule(2, sched, "P1DT10M10S")
	j.Init(cache)
	j.Disable()

	sched2 := time.Now().Add(2 * time.Second)
	j2 := GetMockJobWithSchedule(2, sched2, "P1DT10M10S")
	j2.Init(cache)
	j2.Disable()

	kalaStat := NewKalaStats(cache)
	assert.Equal(t, j.NextRunAt.UnixNano(), kalaStat.NextRunAt.UnixNano())
	assert.Equal(t, j.Metadata.LastAttemptedRun.UnixNano(), kalaStat.LastAttemptedRun.UnixNano())
	assert.NotEqual(t, j2.NextRunAt.UnixNano(), kalaStat.NextRunAt.UnixNano())
}

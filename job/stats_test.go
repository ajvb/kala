package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJobStats(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJobWithGenericSchedule()
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
		j := GetMockJobWithGenericSchedule()
		j.Init(cache)
		j.Run(cache)
	}
	now := time.Now()
	for i := 0; i < 5; i++ {
		j := GetMockJobWithGenericSchedule()
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

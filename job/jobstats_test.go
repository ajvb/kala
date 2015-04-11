package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJobStats(t *testing.T) {
	j := getMockJobWithGenericSchedule()
	j.Init()

	j.Run()
	now := time.Now()

	assert.NotNil(t, j.Stats[0])
	assert.Equal(t, j.Stats[0].JobId, j.Id)
	assert.WithinDuration(t, j.Stats[0].RanAt, now, time.Second)
	assert.Equal(t, j.Stats[0].NumberOfRetries, uint(0))
	assert.True(t, j.Stats[0].Success)
	assert.NotNil(t, j.Stats[0].ExecutionDuration)
}

func TestKalaStats(t *testing.T) {
	AllJobs.Jobs = make(map[string]*Job, 0)

	for i := 0; i < 5; i++ {
		j := getMockJobWithGenericSchedule()
		j.Init()
		j.Run()
		AllJobs.Set(j)
	}
	now := time.Now()
	for i := 0; i < 5; i++ {
		j := getMockJobWithGenericSchedule()
		j.Init()
		j.Disable()
		AllJobs.Set(j)
	}

	kalaStat := NewKalaStats()
	createdAt := time.Now()
	nextRunAt := time.Now().Add(
		time.Duration(time.Minute * 5),
	)

	assert.Equal(t, kalaStat.Jobs, 10)
	assert.Equal(t, kalaStat.ActiveJobs, 5)
	assert.Equal(t, kalaStat.DisabledJobs, 5)
	assert.Equal(t, kalaStat.SuccessCount, uint(5))
	assert.Equal(t, kalaStat.ErrorCount, uint(0))
	assert.WithinDuration(t, kalaStat.NextRunAt, nextRunAt, time.Millisecond*500)
	assert.WithinDuration(t, kalaStat.LastAttemptedRun, now, time.Millisecond*50)
	assert.WithinDuration(t, kalaStat.CreatedAt, createdAt, time.Millisecond*50)
}

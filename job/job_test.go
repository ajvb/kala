package job

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"
	"testing"
)

var testDbPath = ""

func getMockJob() *Job {
	return &Job{
		Name:    "mock_job",
		Command: "bash -c 'date'",
		Owner:   "aj@ajvb.me",
		Retries: 2,
	}
}

func getMockJobWithSchedule(repeat int, scheduleTime time.Time, delay string) *Job {
	genericMockJob := getMockJob()

	parsedTime := scheduleTime.Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R%d/%s/%s", repeat, parsedTime, delay)
	genericMockJob.Schedule = scheduleStr

	return genericMockJob
}

func getMockJobWithGenericSchedule() *Job {
	fiveMinutesFromNow := time.Now().Add(time.Minute * 5)
	return getMockJobWithSchedule(2, fiveMinutesFromNow, "P1DT10M10S")
}

func TestScheduleParsing(t *testing.T) {
	cache := NewMockCache()

	fiveMinutesFromNow := time.Now().Add(
		time.Duration(time.Minute * 5),
	)

	genericMockJob := getMockJobWithSchedule(2, fiveMinutesFromNow, "P1DT10M10S")

	genericMockJob.Init(cache)

	assert.WithinDuration(
		t, genericMockJob.scheduleTime, fiveMinutesFromNow,
		time.Second, "The difference between parsed time and created time is to great.",
	)
}

func TestBrokenSchedule(t *testing.T) {
	cache := NewMockCache()

	mockJob := getMockJobWithGenericSchedule()
	mockJob.Schedule = "hfhgasyuweu123"

	err := mockJob.Init(cache)

	assert.Error(t, err)
	assert.Nil(t, mockJob.jobTimer)
}

var delayParsingTests = []struct {
	expected    time.Duration
	intervalStr string
}{
	{time.Hour*24 + time.Second*10 + time.Minute*10, "P1DT10M10S"},
	{time.Second*10 + time.Minute*10, "PT10M10S"},
	{time.Hour*24 + time.Second*10, "P1DT10S"},
	{time.Hour*24*365 + time.Hour*24, "P1Y1DT"},
}

func TestDelayParsing(t *testing.T) {
	testTime := time.Now().Add(time.Minute * 1)

	for _, delayTest := range delayParsingTests {
		cache := NewMockCache()
		genericMockJob := getMockJobWithSchedule(1, testTime, delayTest.intervalStr)
		genericMockJob.Init(cache)
		assert.Equal(t, delayTest.expected, genericMockJob.delayDuration.ToDuration(), "Parsed duration was incorrect")
	}
}

func TestBrokenDelayHandling(t *testing.T) {
	testTime := time.Now().Add(time.Minute * 1)
	brokenIntervals := []string{
		"DTT",
		"000D",
		"ASDASD",
	}

	for _, intervalTest := range brokenIntervals {
		cache := NewMockCache()

		genericMockJob := getMockJobWithSchedule(1, testTime, intervalTest)
		err := genericMockJob.Init(cache)

		assert.Error(t, err)
		assert.Nil(t, genericMockJob.jobTimer)
	}
}

func TestJobInit(t *testing.T) {
	cache := NewMockCache()

	genericMockJob := getMockJobWithGenericSchedule()

	err := genericMockJob.Init(cache)
	assert.Nil(t, err, "err should be nil")

	assert.NotEmpty(t, genericMockJob.Id, "Job.Id should not be empty")
	assert.NotEmpty(t, genericMockJob.jobTimer, "Job.jobTimer should not be empty")
}

func TestJobDisable(t *testing.T) {
	cache := NewMockCache()

	genericMockJob := getMockJobWithGenericSchedule()
	genericMockJob.Init(cache)

	assert.False(t, genericMockJob.Disabled, "Job should start with disabled == false")

	genericMockJob.Disable()
	assert.True(t, genericMockJob.Disabled, "Job.Disable() should set Job.Disabled to true")
	assert.False(t, genericMockJob.jobTimer.Stop())
}

func TestJobRun(t *testing.T) {
	cache := NewMockCache()

	j := getMockJobWithGenericSchedule()
	j.Init(cache)
	j.Run(cache)

	now := time.Now()

	assert.Equal(t, j.SuccessCount, uint(1))
	assert.WithinDuration(t, j.LastSuccess, now, 2*time.Second)
	assert.WithinDuration(t, j.LastAttemptedRun, now, 2*time.Second)
}

func TestOneOffJobs(t *testing.T) {
	cache := NewMockCache()

	j := getMockJob()

	assert.Equal(t, j.SuccessCount, uint(0))
	assert.Equal(t, j.ErrorCount, uint(0))
	assert.Equal(t, j.LastSuccess, time.Time{})
	assert.Equal(t, j.LastError, time.Time{})
	assert.Equal(t, j.LastAttemptedRun, time.Time{})

	j.Init(cache)
	// Find a better way to test a goroutine
	time.Sleep(time.Second)
	now := time.Now()

	assert.Equal(t, j.SuccessCount, uint(1))
	assert.WithinDuration(t, j.LastSuccess, now, 2*time.Second)
	assert.WithinDuration(t, j.LastAttemptedRun, now, 2*time.Second)
	assert.Equal(t, j.scheduleTime, time.Time{})
	assert.Nil(t, j.jobTimer)
}

func TestDependentJobs(t *testing.T) {
	db := GetDB(testDbPath)
	cache := NewMemoryJobCache(db, time.Second*5)

	mockJob := getMockJobWithGenericSchedule()
	mockJob.Init(cache)
	cache.Set(mockJob)

	mockChildJob := getMockJob()
	mockChildJob.ParentJobs = []string{
		mockJob.Id,
	}
	mockChildJob.Init(cache)
	cache.Set(mockChildJob)

	assert.Equal(t, mockJob.DependentJobs[0], mockChildJob.Id)
	assert.True(t, len(mockJob.DependentJobs) == 1)

	mockJob.Save(db)

	j, _ := db.Get(mockJob.Id)

	assert.Equal(t, j.DependentJobs[0], mockChildJob.Id)

	j.Run(cache)
	time.Sleep(time.Second * 2)
	n := time.Now()

	assert.WithinDuration(t, mockChildJob.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJob.LastSuccess, n, 4*time.Second)
	db.Close()
}

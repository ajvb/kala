package job

import (
	"time"

	"github.com/stretchr/testify/assert"
	"testing"
)

var testDbPath = ""

func TestScheduleParsing(t *testing.T) {
	cache := NewMockCache()

	fiveMinutesFromNow := time.Now().Add(5 * time.Minute)

	genericMockJob := GetMockJobWithSchedule(2, fiveMinutesFromNow, "P1DT10M10S")

	genericMockJob.Init(cache)

	assert.WithinDuration(
		t, genericMockJob.scheduleTime, fiveMinutesFromNow,
		time.Second, "The difference between parsed time and created time is to great.",
	)
}

func TestBrokenSchedule(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
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
		genericMockJob := GetMockJobWithSchedule(1, testTime, delayTest.intervalStr)
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

		genericMockJob := GetMockJobWithSchedule(1, testTime, intervalTest)
		err := genericMockJob.Init(cache)

		assert.Error(t, err)
		assert.Nil(t, genericMockJob.jobTimer)
	}
}

func TestJobInit(t *testing.T) {
	cache := NewMockCache()

	genericMockJob := GetMockJobWithGenericSchedule()

	err := genericMockJob.Init(cache)
	assert.Nil(t, err, "err should be nil")

	assert.NotEmpty(t, genericMockJob.Id, "Job.Id should not be empty")
	assert.NotEmpty(t, genericMockJob.jobTimer, "Job.jobTimer should not be empty")
}

func TestJobDisable(t *testing.T) {
	cache := NewMockCache()

	genericMockJob := GetMockJobWithGenericSchedule()
	genericMockJob.Init(cache)

	assert.False(t, genericMockJob.Disabled, "Job should start with disabled == false")

	genericMockJob.Disable()
	assert.True(t, genericMockJob.Disabled, "Job.Disable() should set Job.Disabled to true")
	assert.False(t, genericMockJob.jobTimer.Stop())
}

func TestJobRun(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJobWithGenericSchedule()
	j.Init(cache)
	j.Run(cache)

	now := time.Now()

	assert.Equal(t, j.SuccessCount, uint(1))
	assert.WithinDuration(t, j.LastSuccess, now, 2*time.Second)
	assert.WithinDuration(t, j.LastAttemptedRun, now, 2*time.Second)
}

func TestOneOffJobs(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJob()

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

func TestDependentJobsSimple(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockChildJob := GetMockJob()
	mockChildJob.ParentJobs = []string{
		mockJob.Id,
	}
	mockChildJob.Init(cache)

	assert.Equal(t, mockJob.DependentJobs[0], mockChildJob.Id)
	assert.True(t, len(mockJob.DependentJobs) == 1)

	j, err := cache.Get(mockJob.Id)
	assert.NoError(t, err)

	assert.Equal(t, j.DependentJobs[0], mockChildJob.Id)

	j.Run(cache)
	time.Sleep(time.Second * 2)
	n := time.Now()

	assert.WithinDuration(t, mockChildJob.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJob.LastSuccess, n, 4*time.Second)
}

// TODO:
// Parent with two childs which each have a child.
// Parent with a chain of length 5 with the first being slow and the rest being fast.

// Parent with two childs
func TestDependentJobsTwoChilds(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockChildJobOne := GetMockJob()
	mockChildJobOne.Name = "mock_child_one"
	mockChildJobOne.ParentJobs = []string{
		mockJob.Id,
	}
	mockChildJobOne.Init(cache)

	mockChildJobTwo := GetMockJob()
	mockChildJobTwo.Name = "mock_child_two"
	mockChildJobTwo.ParentJobs = []string{
		mockJob.Id,
	}
	mockChildJobTwo.Init(cache)

	// Check that it gets placed in the array.
	assert.Equal(t, mockJob.DependentJobs[0], mockChildJobOne.Id)
	assert.Equal(t, mockJob.DependentJobs[1], mockChildJobTwo.Id)
	assert.True(t, len(mockJob.DependentJobs) == 2)

	j, err := cache.Get(mockJob.Id)
	assert.NoError(t, err)

	// Check that we can still get it from the cache.
	assert.Equal(t, j.DependentJobs[0], mockChildJobOne.Id)
	assert.Equal(t, j.DependentJobs[1], mockChildJobTwo.Id)
	assert.True(t, len(j.DependentJobs) == 2)

	j.Run(cache)
	time.Sleep(time.Second * 2)
	n := time.Now()

	// TODO use abtime
	assert.WithinDuration(t, mockChildJobOne.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobOne.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.LastSuccess, n, 4*time.Second)

	// Test the fact that the dependent jobs follow a rule of FIFO
	assert.True(t, mockChildJobOne.LastAttemptedRun.UnixNano() < mockChildJobTwo.LastAttemptedRun.UnixNano())
}

// Parent with child with two childs.
func TestDependentJobsChildWithTwoChilds(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockChildJob := GetMockJob()
	mockChildJob.Name = "mock_child_job"
	mockChildJob.ParentJobs = []string{
		mockJob.Id,
	}
	mockChildJob.Init(cache)

	mockChildJobOne := GetMockJob()
	mockChildJobOne.Name = "mock_child_one"
	mockChildJobOne.ParentJobs = []string{
		mockChildJob.Id,
	}
	mockChildJobOne.Init(cache)

	mockChildJobTwo := GetMockJob()
	mockChildJobTwo.Name = "mock_child_two"
	mockChildJobTwo.ParentJobs = []string{
		mockChildJob.Id,
	}
	mockChildJobTwo.Init(cache)

	// Check that it gets placed in the array.
	assert.Equal(t, mockJob.DependentJobs[0], mockChildJob.Id)
	assert.Equal(t, mockChildJob.DependentJobs[0], mockChildJobOne.Id)
	assert.Equal(t, mockChildJob.DependentJobs[1], mockChildJobTwo.Id)
	assert.True(t, len(mockJob.DependentJobs) == 1)
	assert.True(t, len(mockChildJob.DependentJobs) == 2)

	j, err := cache.Get(mockJob.Id)
	assert.NoError(t, err)

	c, err := cache.Get(mockChildJob.Id)
	assert.NoError(t, err)

	// Check that we can still get it from the cache.
	assert.Equal(t, j.DependentJobs[0], mockChildJob.Id)
	assert.Equal(t, c.DependentJobs[0], mockChildJobOne.Id)
	assert.Equal(t, c.DependentJobs[1], mockChildJobTwo.Id)
	assert.True(t, len(j.DependentJobs) == 1)
	assert.True(t, len(c.DependentJobs) == 2)

	j.Run(cache)
	time.Sleep(time.Second * 2)
	n := time.Now()

	// TODO use abtime
	assert.WithinDuration(t, mockChildJob.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJob.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobOne.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobOne.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.LastSuccess, n, 4*time.Second)

	// Test the fact that the first child is ran before its childern
	assert.True(t, mockChildJob.LastAttemptedRun.UnixNano() < mockChildJobOne.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJob.LastAttemptedRun.UnixNano() < mockChildJobTwo.LastAttemptedRun.UnixNano())

	// Test the fact that the dependent jobs follow a rule of FIFO
	assert.True(t, mockChildJobOne.LastAttemptedRun.UnixNano() < mockChildJobTwo.LastAttemptedRun.UnixNano())
}

// Parent with a chain of length 5
func TestDependentJobsFiveChain(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockChildJobOne := GetMockJob()
	mockChildJobOne.Name = "mock_child_one"
	mockChildJobOne.ParentJobs = []string{
		mockJob.Id,
	}
	mockChildJobOne.Init(cache)

	mockChildJobTwo := GetMockJob()
	mockChildJobTwo.Name = "mock_child_two"
	mockChildJobTwo.ParentJobs = []string{
		mockChildJobOne.Id,
	}
	mockChildJobTwo.Init(cache)

	mockChildJobThree := GetMockJob()
	mockChildJobThree.Name = "mock_child_three"
	mockChildJobThree.ParentJobs = []string{
		mockChildJobTwo.Id,
	}
	mockChildJobThree.Init(cache)

	mockChildJobFour := GetMockJob()
	mockChildJobFour.Name = "mock_child_four"
	mockChildJobFour.ParentJobs = []string{
		mockChildJobThree.Id,
	}
	mockChildJobFour.Init(cache)

	mockChildJobFive := GetMockJob()
	mockChildJobFive.Name = "mock_child_five"
	mockChildJobFive.ParentJobs = []string{
		mockChildJobFour.Id,
	}
	mockChildJobFive.Init(cache)

	// Check that it gets placed in the array.
	assert.Equal(t, mockJob.DependentJobs[0], mockChildJobOne.Id)
	assert.Equal(t, mockChildJobOne.DependentJobs[0], mockChildJobTwo.Id)
	assert.Equal(t, mockChildJobTwo.DependentJobs[0], mockChildJobThree.Id)
	assert.Equal(t, mockChildJobThree.DependentJobs[0], mockChildJobFour.Id)
	assert.Equal(t, mockChildJobFour.DependentJobs[0], mockChildJobFive.Id)
	assert.True(t, len(mockJob.DependentJobs) == 1)
	assert.True(t, len(mockChildJobOne.DependentJobs) == 1)
	assert.True(t, len(mockChildJobTwo.DependentJobs) == 1)
	assert.True(t, len(mockChildJobThree.DependentJobs) == 1)
	assert.True(t, len(mockChildJobFour.DependentJobs) == 1)

	j, err := cache.Get(mockJob.Id)
	assert.NoError(t, err)
	cOne, err := cache.Get(mockChildJobOne.Id)
	assert.NoError(t, err)
	cTwo, err := cache.Get(mockChildJobTwo.Id)
	assert.NoError(t, err)
	cThree, err := cache.Get(mockChildJobThree.Id)
	assert.NoError(t, err)
	cFour, err := cache.Get(mockChildJobFour.Id)
	assert.NoError(t, err)

	// Check that we can still get it from the cache.
	assert.Equal(t, j.DependentJobs[0], mockChildJobOne.Id)
	assert.Equal(t, cOne.DependentJobs[0], mockChildJobTwo.Id)
	assert.Equal(t, cTwo.DependentJobs[0], mockChildJobThree.Id)
	assert.Equal(t, cThree.DependentJobs[0], mockChildJobFour.Id)
	assert.Equal(t, cFour.DependentJobs[0], mockChildJobFive.Id)
	assert.True(t, len(j.DependentJobs) == 1)
	assert.True(t, len(cOne.DependentJobs) == 1)
	assert.True(t, len(cTwo.DependentJobs) == 1)
	assert.True(t, len(cThree.DependentJobs) == 1)
	assert.True(t, len(cFour.DependentJobs) == 1)

	j.Run(cache)
	time.Sleep(time.Second * 2)
	n := time.Now()

	// TODO use abtime
	assert.WithinDuration(t, mockChildJobOne.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobOne.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobThree.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobThree.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobFour.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobFour.LastSuccess, n, 4*time.Second)

	// Test the fact that the first child is ran before its childern
	assert.True(t, mockChildJobOne.LastAttemptedRun.UnixNano() < mockChildJobTwo.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobTwo.LastAttemptedRun.UnixNano() < mockChildJobThree.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobThree.LastAttemptedRun.UnixNano() < mockChildJobFour.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobFour.LastAttemptedRun.UnixNano() < mockChildJobFive.LastAttemptedRun.UnixNano())
}

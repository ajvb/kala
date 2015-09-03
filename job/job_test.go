package job

import (
	"strings"
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

func TestBrokenEpsilon(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Epsilon = "asdasd"

	err := mockJob.Init(cache)

	assert.Error(t, err)
	assert.Nil(t, mockJob.jobTimer)
}

func TestBrokenScheduleTimeHasAlreadyPassed(t *testing.T) {
	cache := NewMockCache()

	fiveMinutesFromNow := time.Now()
	time.Sleep(time.Millisecond * 30)
	mockJob := GetMockJobWithSchedule(2, fiveMinutesFromNow, "P1DT10M10S")

	err := mockJob.Init(cache)

	assert.Error(t, err)
	assert.Nil(t, mockJob.jobTimer)
}

func TestBrokenScheduleBrokenRepeat(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Schedule = strings.Replace(mockJob.Schedule, "R2", "RRR", 1)

	err := mockJob.Init(cache)

	assert.Error(t, err)
	assert.Nil(t, mockJob.jobTimer)
}

func TestBrokenScheduleBrokenInitialScheduleTime(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Schedule = "R0/hfhgasyuweu123/PT10S"

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

func TestJobRunAndRepeat(t *testing.T) {
	cache := NewMockCache()

	oneSecondFromNow := time.Now().Add(time.Second)
	j := GetMockJobWithSchedule(2, oneSecondFromNow, "PT1S")
	j.Init(cache)

	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)
		now := time.Now()
		assert.Equal(t, j.SuccessCount, uint(i+1))
		assert.WithinDuration(t, j.LastSuccess, now, 3*time.Second)
		assert.WithinDuration(t, j.LastAttemptedRun, now, 3*time.Second)
	}

}

func TestJobEpsilon(t *testing.T) {
	cache := NewMockCache()

	oneSecondFromNow := time.Now().Add(time.Second)
	j := GetMockJobWithSchedule(0, oneSecondFromNow, "P1DT1S")
	j.Epsilon = "PT2S"
	j.Command = "bash -c 'sleep 1 && cd /etc && touch l'"
	j.Retries = 200
	j.Init(cache)

	time.Sleep(time.Second * 4)

	now := time.Now()

	assert.Equal(t, j.SuccessCount, uint(0))
	assert.Equal(t, j.ErrorCount, uint(1))
	assert.Equal(t, j.numberOfAttempts, uint(2))
	assert.WithinDuration(t, j.LastError, now, 4*time.Second)
	assert.WithinDuration(t, j.LastAttemptedRun, now, 4*time.Second)
	assert.True(t, j.LastSuccess.IsZero())
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

//
// Dependent Job Tests
//

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

// Parent doesn't exist
func TestDependentJobsParentDoesNotExist(t *testing.T) {
	cache := NewMockCache()

	mockChildJob := GetMockJob()
	mockChildJob.ParentJobs = []string{
		"not-a-real-id",
	}
	err := mockChildJob.Init(cache)
	assert.Error(t, err)
	assert.Equal(t, err, JobDoesntExistErr)
}

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

// Link in the chain fails
func TestDependentJobsChainWithFailingJob(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockChildJobOne := GetMockJob()
	mockChildJobOne.Name = "mock_child_one"
	// ***Where we make it fail***
	mockChildJobOne.Command = "bash -c 'cd /etc && touch l'"
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

	// Check that it gets placed in the array.
	assert.Equal(t, mockJob.DependentJobs[0], mockChildJobOne.Id)
	assert.Equal(t, mockChildJobOne.DependentJobs[0], mockChildJobTwo.Id)
	assert.Equal(t, mockChildJobTwo.DependentJobs[0], mockChildJobThree.Id)
	assert.True(t, len(mockJob.DependentJobs) == 1)
	assert.True(t, len(mockChildJobOne.DependentJobs) == 1)
	assert.True(t, len(mockChildJobTwo.DependentJobs) == 1)

	j, err := cache.Get(mockJob.Id)
	assert.NoError(t, err)
	cOne, err := cache.Get(mockChildJobOne.Id)
	assert.NoError(t, err)
	cTwo, err := cache.Get(mockChildJobTwo.Id)
	assert.NoError(t, err)

	// Check that we can still get it from the cache.
	assert.Equal(t, j.DependentJobs[0], mockChildJobOne.Id)
	assert.Equal(t, cOne.DependentJobs[0], mockChildJobTwo.Id)
	assert.Equal(t, cTwo.DependentJobs[0], mockChildJobThree.Id)
	assert.True(t, len(j.DependentJobs) == 1)
	assert.True(t, len(cOne.DependentJobs) == 1)
	assert.True(t, len(cTwo.DependentJobs) == 1)

	j.Run(cache)
	time.Sleep(time.Second)
	n := time.Now()

	// TODO use abtime
	assert.WithinDuration(t, mockChildJobOne.LastAttemptedRun, n, 2*time.Second)
	assert.True(t, mockChildJobOne.LastSuccess.IsZero())
	assert.True(t, mockChildJobTwo.LastAttemptedRun.IsZero())
	assert.True(t, mockChildJobTwo.LastSuccess.IsZero())
	assert.True(t, mockChildJobThree.LastAttemptedRun.IsZero())
	assert.True(t, mockChildJobThree.LastSuccess.IsZero())
}

// TODO Use something like abtime - this test takes 5 seconds just in waiting....
// Parent with a chain of length 5 with the first being slow and the rest being fast.
func TestDependentJobsFiveChainWithSlowJob(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockChildJobOne := GetMockJob()
	mockChildJobOne.Name = "mock_child_one"
	// ***Where we make it slow***
	mockChildJobOne.Command = "bash -c 'sleep 2 && date'"
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
	time.Sleep(time.Second * 3)
	n := time.Now()

	// TODO use abtime
	assert.WithinDuration(t, mockChildJobOne.LastAttemptedRun, n, 6*time.Second)
	assert.WithinDuration(t, mockChildJobOne.LastSuccess, n, 6*time.Second)
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

// Two Parents with the same child
func TestDependentJobsTwoParentsSameChild(t *testing.T) {
	cache := NewMockCache()

	mockJobOne := GetMockJobWithGenericSchedule()
	mockJobOne.Name = "mock_parent_one"
	mockJobOne.Init(cache)

	mockJobTwo := GetMockJobWithGenericSchedule()
	mockJobTwo.Name = "mock_parent_two"
	mockJobTwo.Init(cache)

	mockChildJob := GetMockJob()
	mockChildJob.Name = "mock_child"
	mockChildJob.ParentJobs = []string{
		mockJobOne.Id,
		mockJobTwo.Id,
	}
	mockChildJob.Init(cache)

	// Check that it gets placed in the array.
	assert.Equal(t, mockJobOne.DependentJobs[0], mockChildJob.Id)
	assert.Equal(t, mockJobTwo.DependentJobs[0], mockChildJob.Id)
	assert.True(t, len(mockJobOne.DependentJobs) == 1)
	assert.True(t, len(mockJobTwo.DependentJobs) == 1)

	parentOne, err := cache.Get(mockJobOne.Id)
	assert.NoError(t, err)
	parentTwo, err := cache.Get(mockJobTwo.Id)
	assert.NoError(t, err)

	// Check that we can still get it from the cache.
	assert.Equal(t, parentOne.DependentJobs[0], mockChildJob.Id)
	assert.Equal(t, parentTwo.DependentJobs[0], mockChildJob.Id)
	assert.True(t, len(parentOne.DependentJobs) == 1)
	assert.True(t, len(parentTwo.DependentJobs) == 1)

	parentOne.Run(cache)
	time.Sleep(time.Second)
	n := time.Now()

	// TODO use abtime
	assert.WithinDuration(t, parentOne.LastAttemptedRun, n, 3*time.Second)
	assert.WithinDuration(t, parentOne.LastSuccess, n, 3*time.Second)
	assert.WithinDuration(t, mockChildJob.LastAttemptedRun, n, 3*time.Second)
	assert.WithinDuration(t, mockChildJob.LastSuccess, n, 3*time.Second)
	assert.True(t, parentTwo.LastAttemptedRun.IsZero())
	assert.True(t, parentTwo.LastSuccess.IsZero())

	// TODO use abtime
	time.Sleep(time.Second * 3)
	parentTwo.Run(cache)
	time.Sleep(time.Second)
	n = time.Now()

	// TODO use abtime
	assert.WithinDuration(t, parentTwo.LastAttemptedRun, n, 3*time.Second)
	assert.WithinDuration(t, parentTwo.LastSuccess, n, 3*time.Second)
	assert.WithinDuration(t, mockChildJob.LastAttemptedRun, n, 3*time.Second)
	assert.WithinDuration(t, mockChildJob.LastSuccess, n, 3*time.Second)
}

// Child gets deleted -- Make sure it is removed from its parent jobs.
func TestDependentJobsChildGetsDeleted(t *testing.T) {
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

	cache.Delete(mockChildJob.Id)
	time.Sleep(time.Second)

	// Check to make sure its deleted
	_, err := cache.Get(mockChildJob.Id)
	assert.Error(t, err)
	assert.Equal(t, err, JobDoesntExistErr)

	// Check to make sure its gone from the parent job.
	assert.True(t, len(mockJob.DependentJobs) == 0)
}

// Child gets disabled
func TestDependentJobsChildGetsDisabled(t *testing.T) {
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

	// Disables it from running.
	mockChildJob.Disable()

	j, err := cache.Get(mockJob.Id)
	assert.NoError(t, err)
	assert.Equal(t, j.DependentJobs[0], mockChildJob.Id)

	j.Run(cache)
	time.Sleep(time.Second * 2)

	assert.True(t, mockChildJob.LastAttemptedRun.IsZero())
	assert.True(t, mockChildJob.LastSuccess.IsZero())
}

// Parent gets deleted -- If a parent job is deleted, unless its child jobs have another parent, they will be deleted as well.
func TestDependentJobsParentJobGetsDeleted(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule()
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockJobBackup := GetMockJobWithGenericSchedule()
	mockJobBackup.Name = "mock_parent_job_backup"
	mockJobBackup.Init(cache)

	mockChildJobWithNoBackup := GetMockJob()
	mockChildJobWithNoBackup.ParentJobs = []string{
		mockJob.Id,
	}
	mockChildJobWithNoBackup.Init(cache)

	mockChildJobWithBackup := GetMockJob()
	mockChildJobWithBackup.ParentJobs = []string{
		mockJob.Id,
		mockJobBackup.Id,
	}
	mockChildJobWithBackup.Init(cache)

	assert.Equal(t, mockJob.DependentJobs[0], mockChildJobWithNoBackup.Id)
	assert.Equal(t, mockJob.DependentJobs[1], mockChildJobWithBackup.Id)
	assert.True(t, len(mockJob.DependentJobs) == 2)
	assert.Equal(t, mockJobBackup.DependentJobs[0], mockChildJobWithBackup.Id)
	assert.True(t, len(mockJobBackup.DependentJobs) == 1)

	cache.Delete(mockJob.Id)
	time.Sleep(time.Second * 2)

	// Make sure it is deleted
	_, err := cache.Get(mockJob.Id)
	assert.Error(t, err)
	assert.Equal(t, JobDoesntExistErr, err)

	// Check if mockChildJobWithNoBackup is deleted
	_, err = cache.Get(mockChildJobWithNoBackup.Id)
	assert.Error(t, err)
	assert.Equal(t, JobDoesntExistErr, err)

	// Check to make sure mockChildJboWithBackup is not deleted
	j, err := cache.Get(mockChildJobWithBackup.Id)
	assert.NoError(t, err)
	assert.Equal(t, j.ParentJobs[0], mockJobBackup.Id)
	assert.True(t, len(j.ParentJobs) == 1)

}

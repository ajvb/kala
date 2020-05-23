package job

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mixer/clock"
	"github.com/stretchr/testify/assert"
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

	mockJob := GetMockJobWithGenericSchedule(time.Now())
	mockJob.Schedule = "hfhgasyuweu123"

	err := mockJob.Init(cache)

	assert.Error(t, err)
	assert.Nil(t, mockJob.jobTimer)
}

func TestBrokenEpsilon(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule(time.Now())
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

	mockJob := GetMockJobWithGenericSchedule(time.Now())
	mockJob.Schedule = strings.Replace(mockJob.Schedule, "R2", "RRR", 1)

	err := mockJob.Init(cache)

	assert.Error(t, err)
	assert.Nil(t, mockJob.jobTimer)
}

func TestBrokenScheduleBrokenInitialScheduleTime(t *testing.T) {
	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule(time.Now())
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
	{time.Hour*24*365 + time.Hour*24, "P1Y1D"},
}

func TestDelayParsing(t *testing.T) {

	// We set the time because, now that durations are relative to a date,
	// a P1Y duration will equal 366 days when crossing a leap year boundary.
	now := parseTime(t, "2019-Jan-02 15:04")

	clk := clock.NewMockClock(now)

	testTime := clk.Now().Add(time.Minute * 1)

	for _, delayTest := range delayParsingTests {
		cache := NewMockCache()
		cache.Clock.SetClock(clk)
		genericMockJob := GetMockJobWithSchedule(1, testTime, delayTest.intervalStr)
		genericMockJob.Init(cache)
		assert.Equal(t, delayTest.expected, genericMockJob.delayDuration.RelativeTo(clk.Now()), "Parsed duration was incorrect")
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

	genericMockJob := GetMockJobWithGenericSchedule(time.Now())

	err := genericMockJob.Init(cache)
	assert.Nil(t, err, "err should be nil")

	assert.NotEmpty(t, genericMockJob.Id, "Job.Id should not be empty")
	assert.NotEmpty(t, genericMockJob.jobTimer, "Job.jobTimer should not be empty")
}

func TestJobDisable(t *testing.T) {
	cache := NewMockCache()

	genericMockJob := GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)

	assert.False(t, genericMockJob.Disabled, "Job should start with disabled == false")

	genericMockJob.Disable()
	assert.True(t, genericMockJob.Disabled, "Job.Disable() should set Job.Disabled to true")
	assert.False(t, genericMockJob.jobTimer.Stop())
}

func TestJobRun(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJobWithGenericSchedule(time.Now())
	j.Init(cache)
	j.Run(cache)

	now := time.Now()

	assert.Equal(t, j.Metadata.SuccessCount, uint(1))
	assert.WithinDuration(t, j.Metadata.LastSuccess, now, 2*time.Second)
	assert.WithinDuration(t, j.Metadata.LastAttemptedRun, now, 2*time.Second)
}

func TestJobWithRepeatOfZeroAndNoInterval(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJob()
	parsedTime := time.Now().Add(time.Minute * 5).Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R%d/%s/", 0, parsedTime)
	j.Schedule = scheduleStr
	j.Init(cache)
	j.Run(cache)

	now := time.Now()

	assert.Equal(t, j.Metadata.SuccessCount, uint(1))
	assert.WithinDuration(t, j.Metadata.LastSuccess, now, 2*time.Second)
	assert.WithinDuration(t, j.Metadata.LastAttemptedRun, now, 2*time.Second)
}

func TestJobWithRepeatOfZeroThatIsDoneAfterRun(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJob()
	parsedTime := time.Now().Add(time.Minute * 5).Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R%d/%s/", 0, parsedTime)
	j.Schedule = scheduleStr
	j.Init(cache)
	j.Run(cache)

	assert.Equal(t, j.Metadata.SuccessCount, uint(1))
	assert.Equal(t, j.IsDone, true)
}

func TestJobRunAndRepeat(t *testing.T) {
	cache := NewMockCache()

	oneSecondFromNow := time.Now().Add(time.Second)
	j := GetMockJobWithSchedule(2, oneSecondFromNow, "PT1S")
	j.Init(cache)

	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)
		now := time.Now()
		j.lock.RLock()
		assert.WithinDuration(t, j.Metadata.LastSuccess, now, 2*time.Second)
		assert.WithinDuration(t, j.Metadata.LastAttemptedRun, now, 2*time.Second)
		j.lock.RUnlock()
	}
}

func TestRecurringJobIsRepeating(t *testing.T) {
	clk := clock.NewMockClock(time.Now())

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	oneSecondFromNow := clk.Now().Add(time.Millisecond * 900)
	j := GetMockRecurringJobWithSchedule(oneSecondFromNow, "PT5S")
	j.Init(cache)
	j.ranChan = make(chan struct{})

	for i := 0; i < 2; i++ {
		clk.AddTime(time.Millisecond * 6000)
		awaitJobRan(t, j, time.Second*10)
		now := clk.Now()
		j.lock.RLock()
		assert.WithinDuration(t, j.Metadata.LastSuccess, now, 2*time.Second)
		assert.WithinDuration(t, j.Metadata.LastAttemptedRun, now, 2*time.Second)
		j.lock.RUnlock()
	}

	j.lock.RLock()
	assert.Equal(t, j.IsDone, false)
	assert.Equal(t, j.Metadata.SuccessCount, uint(2))
	j.lock.RUnlock()
}

func TestTwoTimesRecurringJobIsDoneAfterThirdRun(t *testing.T) {
	cache := NewMockCache()

	oneSecondFromNow := time.Now().Add(time.Second)
	j := GetMockJobWithSchedule(2, oneSecondFromNow, "PT1S")
	j.Init(cache)

	for i := 0; i < 2; i++ {
		time.Sleep(time.Second)
		j.lock.RLock()
		assert.Equal(t, j.IsDone, false)
		j.lock.RUnlock()
	}

	time.Sleep(time.Second)

	j.lock.RLock()
	assert.Equal(t, j.IsDone, true)
	j.lock.RUnlock()
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

	j.lock.RLock()
	assert.Equal(t, j.Metadata.SuccessCount, uint(0))
	assert.Equal(t, j.Metadata.ErrorCount, uint(2))
	assert.WithinDuration(t, j.Metadata.LastError, now, 4*time.Second)
	assert.WithinDuration(t, j.Metadata.LastAttemptedRun, now, 4*time.Second)
	assert.True(t, j.Metadata.LastSuccess.IsZero())
	j.lock.RUnlock()
}

func TestOneOffJobs(t *testing.T) {
	cache := NewMockCache()

	j := GetMockJob()

	j.lock.RLock()
	assert.Equal(t, j.Metadata.SuccessCount, uint(0))
	assert.Equal(t, j.Metadata.ErrorCount, uint(0))
	assert.Equal(t, j.Metadata.LastSuccess, time.Time{})
	assert.Equal(t, j.Metadata.LastError, time.Time{})
	assert.Equal(t, j.Metadata.LastAttemptedRun, time.Time{})
	j.lock.RUnlock()

	j.Init(cache)
	// Find a better way to test a goroutine
	time.Sleep(time.Second)
	now := time.Now()

	j.lock.RLock()
	assert.Equal(t, j.Metadata.SuccessCount, uint(1))
	assert.WithinDuration(t, j.Metadata.LastSuccess, now, 2*time.Second)
	assert.WithinDuration(t, j.Metadata.LastAttemptedRun, now, 2*time.Second)
	assert.Equal(t, j.scheduleTime, time.Time{})
	assert.Nil(t, j.jobTimer)
	j.lock.RUnlock()
}

//
// Dependent Job Tests
//
func TestDependentJobsSimple(t *testing.T) {

	clk := NewHybridClock()
	clk.Play()

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	mockJob := GetMockJobWithGenericSchedule(clk.Now())
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
	clk.AddTime(time.Second * 2)
	briefPause()
	n := clk.Now()

	mcj, err := cache.Get(mockChildJob.Id)
	assert.NoError(t, err)

	assert.WithinDuration(t, mcj.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mcj.Metadata.LastSuccess, n, 4*time.Second)
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
	assert.Equal(t, err, ErrJobDoesntExist)
}

// Parent with two childs
func TestDependentJobsTwoChilds(t *testing.T) {

	clk := NewHybridClock()
	clk.Play()

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	mockJob := GetMockJobWithGenericSchedule(clk.Now())
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
	clk.AddTime(time.Second * 2)
	briefPause()
	n := clk.Now()

	assert.WithinDuration(t, mockChildJobOne.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobOne.Metadata.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.Metadata.LastSuccess, n, 4*time.Second)

	// Test the fact that the dependent jobs follow a rule of FIFO
	assert.True(t, mockChildJobOne.Metadata.LastAttemptedRun.UnixNano() < mockChildJobTwo.Metadata.LastAttemptedRun.UnixNano())
}

// Parent with child with two childs.
func TestDependentJobsChildWithTwoChilds(t *testing.T) {

	clk := NewHybridClock()
	clk.Play()

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	mockJob := GetMockJobWithGenericSchedule(clk.Now())
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
	clk.AddTime(time.Second * 2)
	briefPause()
	n := clk.Now()

	assert.WithinDuration(t, mockChildJob.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJob.Metadata.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobOne.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobOne.Metadata.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.Metadata.LastSuccess, n, 4*time.Second)

	// Test the fact that the first child is ran before its childern
	assert.True(t, mockChildJob.Metadata.LastAttemptedRun.UnixNano() < mockChildJobOne.Metadata.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJob.Metadata.LastAttemptedRun.UnixNano() < mockChildJobTwo.Metadata.LastAttemptedRun.UnixNano())

	// Test the fact that the dependent jobs follow a rule of FIFO
	assert.True(t, mockChildJobOne.Metadata.LastAttemptedRun.UnixNano() < mockChildJobTwo.Metadata.LastAttemptedRun.UnixNano())
}

// Parent with a chain of length 5
func TestDependentJobsFiveChain(t *testing.T) {

	clk := NewHybridClock()
	clk.Play()

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	mockJob := GetMockJobWithGenericSchedule(clk.Now())
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
	clk.AddTime(time.Second * 2)
	briefPause()
	n := clk.Now()

	assert.WithinDuration(t, mockChildJobOne.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobOne.Metadata.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.Metadata.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobThree.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobThree.Metadata.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobFour.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobFour.Metadata.LastSuccess, n, 4*time.Second)

	// Test the fact that the first child is ran before its childern
	assert.True(t, mockChildJobOne.Metadata.LastAttemptedRun.UnixNano() < mockChildJobTwo.Metadata.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobTwo.Metadata.LastAttemptedRun.UnixNano() < mockChildJobThree.Metadata.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobThree.Metadata.LastAttemptedRun.UnixNano() < mockChildJobFour.Metadata.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobFour.Metadata.LastAttemptedRun.UnixNano() < mockChildJobFive.Metadata.LastAttemptedRun.UnixNano())
}

// Link in the chain fails
func TestDependentJobsChainWithFailingJob(t *testing.T) {

	clk := NewHybridClock()
	clk.Play()

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	mockJob := GetMockJobWithGenericSchedule(clk.Now())
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
	clk.AddTime(time.Second)
	briefPause()
	n := clk.Now()

	assert.WithinDuration(t, mockChildJobOne.Metadata.LastAttemptedRun, n, 2*time.Second)
	assert.True(t, mockChildJobOne.Metadata.LastSuccess.IsZero())
	assert.True(t, mockChildJobTwo.Metadata.LastAttemptedRun.IsZero())
	assert.True(t, mockChildJobTwo.Metadata.LastSuccess.IsZero())
	assert.True(t, mockChildJobThree.Metadata.LastAttemptedRun.IsZero())
	assert.True(t, mockChildJobThree.Metadata.LastSuccess.IsZero())
}

// Parent with a chain of length 5 with the first being slow and the rest being fast.
func TestDependentJobsFiveChainWithSlowJob(t *testing.T) {

	clk := NewHybridClock()
	clk.Play()

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	mockJob := GetMockJobWithGenericSchedule(clk.Now())
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
	clk.AddTime(time.Second * 3)
	briefPause()
	n := clk.Now()

	// TODO use abtime
	assert.WithinDuration(t, mockChildJobOne.Metadata.LastAttemptedRun, n, 6*time.Second)
	assert.WithinDuration(t, mockChildJobOne.Metadata.LastSuccess, n, 6*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobTwo.Metadata.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobThree.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobThree.Metadata.LastSuccess, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobFour.Metadata.LastAttemptedRun, n, 4*time.Second)
	assert.WithinDuration(t, mockChildJobFour.Metadata.LastSuccess, n, 4*time.Second)

	// Test the fact that the first child is ran before its childern
	assert.True(t, mockChildJobOne.Metadata.LastAttemptedRun.UnixNano() < mockChildJobTwo.Metadata.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobTwo.Metadata.LastAttemptedRun.UnixNano() < mockChildJobThree.Metadata.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobThree.Metadata.LastAttemptedRun.UnixNano() < mockChildJobFour.Metadata.LastAttemptedRun.UnixNano())
	assert.True(t, mockChildJobFour.Metadata.LastAttemptedRun.UnixNano() < mockChildJobFive.Metadata.LastAttemptedRun.UnixNano())
}

// Two Parents with the same child
func TestDependentJobsTwoParentsSameChild(t *testing.T) {

	clk := NewHybridClock()
	clk.Play()

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	mockJobOne := GetMockJobWithGenericSchedule(clk.Now())
	mockJobOne.Name = "mock_parent_one"
	mockJobOne.Init(cache)

	mockJobTwo := GetMockJobWithGenericSchedule(clk.Now())
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
	clk.AddTime(time.Second)
	briefPause()
	n := clk.Now()

	assert.WithinDuration(t, parentOne.Metadata.LastAttemptedRun, n, 3*time.Second)
	assert.WithinDuration(t, parentOne.Metadata.LastSuccess, n, 3*time.Second)
	assert.WithinDuration(t, mockChildJob.Metadata.LastAttemptedRun, n, 3*time.Second)
	assert.WithinDuration(t, mockChildJob.Metadata.LastSuccess, n, 3*time.Second)
	assert.True(t, parentTwo.Metadata.LastAttemptedRun.IsZero())
	assert.True(t, parentTwo.Metadata.LastSuccess.IsZero())

	clk.AddTime(time.Second * 3)
	briefPause()
	parentTwo.Run(cache)
	clk.AddTime(time.Second)
	briefPause()
	n = clk.Now()

	assert.WithinDuration(t, parentTwo.Metadata.LastAttemptedRun, n, 3*time.Second)
	assert.WithinDuration(t, parentTwo.Metadata.LastSuccess, n, 3*time.Second)
	assert.WithinDuration(t, mockChildJob.Metadata.LastAttemptedRun, n, 3*time.Second)
	assert.WithinDuration(t, mockChildJob.Metadata.LastSuccess, n, 3*time.Second)
}

// Child gets deleted -- Make sure it is removed from its parent jobs.
func TestDependentJobsChildGetsDeleted(t *testing.T) {

	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule(time.Now())
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
	assert.Equal(t, err, ErrJobDoesntExist)

	// Check to make sure its gone from the parent job.
	mockJob.lock.RLock()
	assert.True(t, len(mockJob.DependentJobs) == 0)
	mockJob.lock.RUnlock()
}

// Child gets disabled
func TestDependentJobsChildGetsDisabled(t *testing.T) {

	clk := NewHybridClock()
	clk.Play()

	cache := NewMockCache()
	cache.Clock.SetClock(clk)

	// Creates parent job with a schedule
	// Sets up timer to run job
	mockJob := GetMockJobWithGenericSchedule(clk.Now())
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	// Creates child job
	// J.Init() then searches the cache for parent job and appends child to dep jobs
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

	n := clk.Now()
	j.Run(cache)
	clk.AddTime(time.Second * 2)
	briefPause()

	// Child job should not have been run
	assert.True(t, mockChildJob.Metadata.LastSuccess.IsZero())
	assert.False(t, mockChildJob.Metadata.LastAttemptedRun.IsZero())

	// Within a second this job should have attempted to be run
	assert.WithinDuration(t, mockChildJob.Metadata.LastAttemptedRun, n, time.Duration(time.Second))
}

// Parent gets deleted -- If a parent job is deleted, unless its child jobs have another parent, they will be deleted as well.
func TestDependentJobsParentJobGetsDeleted(t *testing.T) {

	cache := NewMockCache()

	mockJob := GetMockJobWithGenericSchedule(time.Now())
	mockJob.Name = "mock_parent_job"
	mockJob.Init(cache)

	mockJobBackup := GetMockJobWithGenericSchedule(time.Now())
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
	time.Sleep(time.Second)

	// Make sure it is deleted
	_, err := cache.Get(mockJob.Id)
	assert.Error(t, err)
	assert.Equal(t, ErrJobDoesntExist, err)

	// Check if mockChildJobWithNoBackup is deleted
	_, err = cache.Get(mockChildJobWithNoBackup.Id)
	assert.Error(t, err)
	assert.Equal(t, ErrJobDoesntExist, err)

	// Check to make sure mockChildJboWithBackup is not deleted
	j, err := cache.Get(mockChildJobWithBackup.Id)
	assert.NoError(t, err)

	j.lock.RLock()
	assert.Equal(t, j.ParentJobs[0], mockJobBackup.Id)
	assert.True(t, len(j.ParentJobs) == 1)
	j.lock.RUnlock()
}

func TestRemoteJobRunner(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer testServer.Close()

	mockRemoteJob := GetMockRemoteJob(RemoteProperties{
		Url: testServer.URL,
	})

	cache := NewMockCache()
	mockRemoteJob.Run(cache)
	assert.True(t, mockRemoteJob.Metadata.SuccessCount == 1)
}

func TestRemoteJobBadStatus(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "something failed", http.StatusInternalServerError)
	}))
	defer testServer.Close()

	mockRemoteJob := GetMockRemoteJob(RemoteProperties{
		Url: testServer.URL,
	})

	cache := NewMockCache()
	mockRemoteJob.Run(cache)
	assert.True(t, mockRemoteJob.Metadata.SuccessCount == 0)
}

func TestRemoteJobBadStatusSuccess(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "something failed", http.StatusInternalServerError)
	}))
	defer testServer.Close()

	mockRemoteJob := GetMockRemoteJob(RemoteProperties{
		Url:                   testServer.URL,
		ExpectedResponseCodes: []int{500},
	})

	cache := NewMockCache()
	mockRemoteJob.Run(cache)
	assert.True(t, mockRemoteJob.Metadata.SuccessCount == 1)
}

func waitForJob(j *Job) {
	for {
		j.lock.RLock()
		if j.IsDone {
			j.lock.RUnlock()
			break
		}
		j.lock.RUnlock()
		time.Sleep(1)
	}
}

func TestOnFailureJobTriggersOnFailure(t *testing.T) {
	cache := NewMockCache()

	onFailureJob := GetMockJob()
	onFailureJob.Init(cache)
	waitForJob(onFailureJob)

	j := GetMockFailingJob()
	j.OnFailureJob = onFailureJob.Id

	j.Init(cache)

	for {
		onFailureJob.lock.RLock()
		if onFailureJob.Metadata.NumberOfFinishedRuns >= 2 {
			onFailureJob.lock.RUnlock()
			break
		}
		onFailureJob.lock.RUnlock()
		time.Sleep(1)
	}

	j.lock.RLock()
	onFailureJob.lock.RLock()
	assert.Equal(t, j.Metadata.SuccessCount, uint(0))
	// onFailureJob ran once from init, and once from getting triggered on failure
	assert.Equal(t, onFailureJob.Metadata.SuccessCount, uint(2))
	assert.True(t, onFailureJob.Metadata.LastAttemptedRun.UnixNano() >= j.Metadata.LastAttemptedRun.UnixNano())
	onFailureJob.lock.RUnlock()
	j.lock.RUnlock()
}

func TestOnFailureJobDoesntTriggerOnSuccess(t *testing.T) {
	cache := NewMockCache()

	onFailureJob := GetMockJob()
	onFailureJob.Init(cache)
	waitForJob(onFailureJob)

	j := GetMockJob()
	j.OnFailureJob = onFailureJob.Id

	j.Init(cache)

	waitForJob(j)

	j.lock.RLock()
	onFailureJob.lock.RLock()
	assert.Equal(t, j.Metadata.SuccessCount, uint(1))
	// onFailureJob already ran once from init
	assert.Equal(t, onFailureJob.Metadata.SuccessCount, uint(1))
	assert.True(t, onFailureJob.Metadata.LastAttemptedRun.UnixNano() <= j.Metadata.LastAttemptedRun.UnixNano())
	onFailureJob.lock.RUnlock()
	j.lock.RUnlock()
}

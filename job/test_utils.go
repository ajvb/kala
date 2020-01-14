package job

import (
	"fmt"
	"testing"
	"time"

	"github.com/ajvb/kala/utils/iso8601"
)

type MockDB struct{}

func (m *MockDB) GetAll() ([]*Job, error) {
	return nil, nil
}
func (m *MockDB) Get(id string) (*Job, error) {
	return nil, nil
}
func (m *MockDB) Delete(id string) error {
	return nil
}
func (m *MockDB) Save(job *Job) error {
	return nil
}
func (m *MockDB) Close() error {
	return nil
}

func NewMockCache() *LockFreeJobCache {
	return NewLockFreeJobCache(&MockDB{})
}

func GetMockJob() *Job {
	return &Job{
		Name:    "mock_job",
		Command: "bash -c 'date'",
		Owner:   "example@example.com",
		Retries: 2,
	}
}

func GetMockFailingJob() *Job {
	return &Job{
		Name:    "mock_failing_job",
		Command: "asdf",
		Owner:   "example@example.com",
		Retries: 2,
	}
}

func GetMockRemoteJob(props RemoteProperties) *Job {
	return &Job{
		Name:             "mock_remote_job",
		Command:          "",
		JobType:          RemoteJob,
		RemoteProperties: props,
	}
}

func GetMockJobWithSchedule(repeat int, scheduleTime time.Time, delay string) *Job {
	genericMockJob := GetMockJob()

	parsedTime := scheduleTime.Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R%d/%s/%s", repeat, parsedTime, delay)
	genericMockJob.Schedule = scheduleStr

	return genericMockJob
}

func GetMockRecurringJobWithSchedule(scheduleTime time.Time, delay string) *Job {
	genericMockJob := GetMockJob()

	parsedTime := scheduleTime.Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R/%s/%s", parsedTime, delay)
	genericMockJob.Schedule = scheduleStr
	genericMockJob.timesToRepeat = -1

	parsedDuration, _ := iso8601.FromString(delay)

	genericMockJob.delayDuration = parsedDuration

	return genericMockJob
}

func GetMockJobStats(oldDate time.Time, count int) []*JobStat {
	stats := make([]*JobStat, 0)
	for i := 1; i <= count; i++ {
		el := &JobStat{
			JobId:             "stats-id-" + string(i),
			NumberOfRetries:   0,
			ExecutionDuration: 10000,
			Success:           true,
			RanAt:             oldDate,
		}
		stats = append(stats, el)
	}
	return stats
}

func GetMockJobWithGenericSchedule(now time.Time) *Job {
	fiveMinutesFromNow := now.Add(time.Minute * 5)
	return GetMockJobWithSchedule(2, fiveMinutesFromNow, "P1DT10M10S")
}

func parseTime(t *testing.T, value string) time.Time {
	now, err := time.Parse("2006-Jan-02 15:04 MST", value)
	if err != nil {
		now, err = time.Parse("2006-Jan-02 15:04", value)
		if err != nil {
			t.Fatal(err)
		}
	}
	return now
}

// Used to hand off execution briefly so that jobs can run and so on.
func briefPause() {
	time.Sleep(time.Millisecond * 100)
}

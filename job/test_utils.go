package job

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ajvb/kala/v2/internal/fakecommands"
	"github.com/ajvb/kala/v2/utils/iso8601"
)

type MockDBGetAll struct {
	MockDB
	response []*Job
}

func (d *MockDBGetAll) GetAll() ([]*Job, error) {
	return d.response, nil
}

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

// NewMockCache returns a cache with no storage whose jobs run their commands
// with a fakecommands.Runner, so no process is started, for tests in this and
// other packages. Options given here override that, for example
// WithCommandRunner with a runner the test configured.
func NewMockCache(opts ...CacheOption) *LockFreeJobCache {
	opts = append([]CacheOption{WithCommandRunner(fakecommands.New())}, opts...)
	return NewLockFreeJobCache(&MockDB{}, opts...)
}

func GetMockJob() *Job {
	return &Job{
		Name:    "mock_job",
		Command: "echo mock_job ran",
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
			JobId:             "stats-id-" + fmt.Sprint(i),
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

var _ JobDB = (*MemoryDB)(nil)

type MemoryDB struct {
	m    map[string]*Job
	lock sync.RWMutex
}

func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		m: map[string]*Job{},
	}
}

func (m *MemoryDB) GetAll() (ret []*Job, _ error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, v := range m.m {
		ret = append(ret, v)
	}
	return
}

func (m *MemoryDB) Get(id string) (*Job, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	j, exist := m.m[id]
	if !exist {
		return nil, ErrJobNotFound(id)
	}
	return j, nil
}

func (m *MemoryDB) Delete(id string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, exists := m.m[id]; !exists {
		return errors.New("doesn't exist") // Used for testing
	}
	delete(m.m, id)
	// log.Printf("After delete: %+v", m)
	return nil
}

func (m *MemoryDB) Save(j *Job) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.m[j.Id] = j
	return nil
}

func (m *MemoryDB) Close() error {
	return nil
}

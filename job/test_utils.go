package job

import (
	"fmt"
	"time"
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

func NewMockCache() *MemoryJobCache {
	return NewMemoryJobCache(&MockDB{}, time.Hour*5)
}

func GetMockJob() *Job {
	return &Job{
		Name:    "mock_job",
		Command: "bash -c 'date'",
		Owner:   "aj@ajvb.me",
		Retries: 2,
	}
}

func GetMockJobWithSchedule(repeat int, scheduleTime time.Time, delay string) *Job {
	genericMockJob := GetMockJob()

	parsedTime := scheduleTime.Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R%d/%s/%s", repeat, parsedTime, delay)
	genericMockJob.Schedule = scheduleStr

	return genericMockJob
}

func GetMockJobWithGenericSchedule() *Job {
	fiveMinutesFromNow := time.Now().Add(time.Minute * 5)
	return GetMockJobWithSchedule(2, fiveMinutesFromNow, "P1DT10M10S")
}

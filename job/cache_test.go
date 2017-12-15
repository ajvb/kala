package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheStart(t *testing.T) {
	cache := NewMockCache()
	cache.Start(time.Duration(time.Hour), time.Duration(time.Hour))
}

func TestCacheDeleteJobNotFound(t *testing.T) {
	cache := NewMockCache()
	err := cache.Delete("not-a-real-id")
	assert.Equal(t, ErrJobDoesntExist, err)
}

func TestCachePersist(t *testing.T) {
	cache := NewMockCache()
	err := cache.Persist()
	assert.NoError(t, err)
}

func TestCacheRetainShouldRemoveOldJobStats(t *testing.T) {
	cache := NewMockCache()
	mockDb := &MockDBGetAll{}
	cache.jobDB = mockDb

	pastDate := time.Date(2016, time.April, 12, 20, 00, 00, 0, time.UTC)
	j := GetMockRecurringJobWithSchedule(pastDate, "PT1S")
	j.Stats = GetMockJobStats(pastDate, 5)
	j.Id = "0"

	jobs := make([]*Job, 0, 0)
	jobs = append(jobs, j)
	mockDb.response = jobs

	cache.Start(0, 1*time.Minute) // Retain 1 minute
	assert.Equal(t, 5, len(j.Stats))

	time.Sleep(time.Second * 2)
	cache.Retain()

	j.lock.RLock()
	assert.Equal(t, 1, len(j.Stats)) // New job stats should not be cleaned up
	j.lock.RUnlock()
}

type MockDBGetAll struct {
	MockDB
	response []*Job
}

func (d *MockDBGetAll) GetAll() ([]*Job, error) {
	return d.response, nil
}

func TestCacheStartStartsARecurringJobWithStartDateInThePast(t *testing.T) {

	cache := NewMockCache()
	mockDb := &MockDBGetAll{}
	cache.jobDB = mockDb

	pastDate := time.Date(2016, time.April, 12, 20, 00, 00, 0, time.UTC)
	j := GetMockRecurringJobWithSchedule(pastDate, "PT1S")
	j.Id = "0"

	jobs := make([]*Job, 0, 0)
	jobs = append(jobs, j)
	mockDb.response = jobs

	cache.Start(0, -1)
	time.Sleep(time.Second * 2)

	j.lock.RLock()
	assert.Equal(t, j.Metadata.SuccessCount, uint(1))
	j.lock.RUnlock()
}

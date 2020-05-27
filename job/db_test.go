package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDelete(t *testing.T) {
	cache := NewMockCache()
	job := GetMockJobWithGenericSchedule(time.Now())
	job.Init(cache)

	err := job.Delete(cache)
	assert.NoError(t, err)

	val, err := cache.Get(job.Id)
	assert.Error(t, err)
	assert.Nil(t, val)
}

func TestDeleteDoesNotExists(t *testing.T) {
	cache := NewMockCache()
	jobOne := GetMockJobWithGenericSchedule(time.Now())
	jobOne.Init(cache)
	jobTwo := GetMockJobWithGenericSchedule(time.Now())

	err := jobTwo.Delete(cache)
	assert.Error(t, err)

	val, err := cache.Get(jobOne.Id)
	assert.NoError(t, err)
	assert.Equal(t, jobOne, val)
}

func TestDeleteAll(t *testing.T) {
	cache := NewMockCache()
	for i := 0; i < 10; i++ {
		job := GetMockJobWithGenericSchedule(time.Now())
		job.Init(cache)
	}

	err := DeleteAll(cache)
	assert.NoError(t, err)

	allJobs := cache.GetAll()
	assert.Empty(t, allJobs.Jobs)
}

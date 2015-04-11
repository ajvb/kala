package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSaveAndGetJob(t *testing.T) {
	genericMockJob := getMockJobWithGenericSchedule()
	genericMockJob.Init()
	genericMockJob.Save()

	j, err := GetJob(genericMockJob.Id)
	assert.Nil(t, err)

	// TODO - Should be no difference....
	assert.WithinDuration(t, j.NextRunAt, genericMockJob.NextRunAt, 30*time.Microsecond)
	assert.Equal(t, j.Name, genericMockJob.Name)
	assert.Equal(t, j.Id, genericMockJob.Id)
	assert.Equal(t, j.Command, genericMockJob.Command)
	assert.Equal(t, j.Schedule, genericMockJob.Schedule)
	assert.Equal(t, j.Owner, genericMockJob.Owner)
	assert.Equal(t, j.SuccessCount, genericMockJob.SuccessCount)
}

func TestDeleteJob(t *testing.T) {
	genericMockJob := getMockJobWithGenericSchedule()
	genericMockJob.Init()
	genericMockJob.Save()
	AllJobs.Set(genericMockJob)

	// Make sure its there
	j, err := GetJob(genericMockJob.Id)
	assert.Nil(t, err)
	assert.Equal(t, j.Name, genericMockJob.Name)
	assert.NotNil(t, AllJobs.Get(genericMockJob.Id))

	// Delete it
	genericMockJob.Delete()

	k, err := GetJob(genericMockJob.Id)
	assert.Error(t, err)
	assert.Nil(t, k)
	assert.Nil(t, AllJobs.Get(genericMockJob.Id))

	genericMockJob.Delete()
}

//TODO
func TestSaveAllJobs(t *testing.T) {
}

//TODO
func TestSaveAllJobsEvery(t *testing.T) {
}

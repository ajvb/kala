package boltdb

import (
	"testing"
	"time"

	"github.com/ajvb/kala/v2/job"

	"github.com/stretchr/testify/assert"
)

var testDbPath = ""

func setupTest(t *testing.T) {
	db := GetBoltDB(testDbPath)
	defer db.Close()

	jobs, err := db.GetAll()
	assert.NoError(t, err)

	for _, j := range jobs {
		err = db.Delete(j.Id)
		assert.NoError(t, err)
	}

}

func TestSaveAndGetJob(t *testing.T) {
	setupTest(t)

	db := GetBoltDB(testDbPath)
	cache := job.NewLockFreeJobCache(db)
	defer db.Close()

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)
	db.Save(genericMockJob)

	j, err := db.Get(genericMockJob.Id)
	assert.Nil(t, err)

	ok, err := genericMockJob.Compare(j)
	assert.True(t, ok, err)
}

func TestDeleteJob(t *testing.T) {
	setupTest(t)

	db := GetBoltDB(testDbPath)
	cache := job.NewLockFreeJobCache(db)
	defer db.Close()

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)
	db.Save(genericMockJob)

	// Make sure its there
	j, err := db.Get(genericMockJob.Id)
	assert.Nil(t, err)
	assert.Equal(t, j.Name, genericMockJob.Name)
	retrievedJob, err := cache.Get(genericMockJob.Id)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedJob)

	// Delete it
	genericMockJob.Delete(cache)

	k, err := db.Get(genericMockJob.Id)
	assert.Error(t, err)
	assert.Nil(t, k)
	retrievedJobTwo, err := cache.Get(genericMockJob.Id)
	assert.Error(t, err)
	assert.Nil(t, retrievedJobTwo)

	genericMockJob.Delete(cache)
}

func TestSaveAndGetAllJobs(t *testing.T) {
	setupTest(t)

	db := GetBoltDB(testDbPath)
	cache := job.NewLockFreeJobCache(db)
	defer db.Close()

	genericMockJobOne := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJobOne.Init(cache)
	err := db.Save(genericMockJobOne)
	assert.NoError(t, err)

	genericMockJobTwo := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJobTwo.Init(cache)
	err = db.Save(genericMockJobTwo)
	assert.NoError(t, err)

	jobs, err := db.GetAll()
	assert.Nil(t, err)
	assert.Equal(t, len(jobs), 2)
}

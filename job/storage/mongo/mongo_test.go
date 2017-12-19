package mongo

import (
	"testing"
	"time"

	"github.com/ajvb/kala/job"
	mgo "gopkg.in/mgo.v2"

	"github.com/stretchr/testify/assert"
)

func NewTestDb(t *testing.T) *DB {
	collection = "test"
	var db = New("", &mgo.Credential{})

	jobs, err := db.GetAll()
	assert.NoError(t, err)
	for _, j := range jobs {
		err = db.Delete(j.Id)
		assert.NoError(t, err)
	}
	return db
}

func TestSaveAndGetJob(t *testing.T) {
	db := NewTestDb(t)

	cache := job.NewLockFreeJobCache(db)
	defer db.Close()

	genericMockJob := job.GetMockJobWithGenericSchedule()
	genericMockJob.Init(cache)
	err := db.Save(genericMockJob)
	if assert.NoError(t, err) {
		j, err := db.Get(genericMockJob.Id)
		if assert.Nil(t, err) {
			assert.WithinDuration(t, j.NextRunAt, genericMockJob.NextRunAt, 400*time.Microsecond)
			assert.Equal(t, j.Name, genericMockJob.Name)
			assert.Equal(t, j.Id, genericMockJob.Id)
			assert.Equal(t, j.Command, genericMockJob.Command)
			assert.Equal(t, j.Schedule, genericMockJob.Schedule)
			assert.Equal(t, j.Owner, genericMockJob.Owner)
			assert.Equal(t, j.Metadata.SuccessCount, genericMockJob.Metadata.SuccessCount)
		}
	}
}

func TestDeleteJob(t *testing.T) {
	db := NewTestDb(t)

	cache := job.NewLockFreeJobCache(db)

	genericMockJob := job.GetMockJobWithGenericSchedule()
	genericMockJob.Init(cache)
	err := db.Save(genericMockJob)
	if assert.NoError(t, err) {

		// Make sure its there
		j, err := db.Get(genericMockJob.Id)
		assert.Nil(t, err)
		assert.Equal(t, j.Name, genericMockJob.Name)
		retrievedJob, err := cache.Get(genericMockJob.Id)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedJob)

		// Delete it
		err = genericMockJob.Delete(cache, db)
		assert.Nil(t, err)

		k, err := db.Get(genericMockJob.Id)
		assert.Error(t, err)
		assert.Nil(t, k)
		retrievedJobTwo, err := cache.Get(genericMockJob.Id)
		assert.Error(t, err)
		assert.Nil(t, retrievedJobTwo)

		genericMockJob.Delete(cache, db)
	}
}

func TestSaveAndGetAllJobs(t *testing.T) {
	db := NewTestDb(t)

	cache := job.NewLockFreeJobCache(db)

	genericMockJobOne := job.GetMockJobWithGenericSchedule()
	genericMockJobOne.Init(cache)
	err := db.Save(genericMockJobOne)
	assert.NoError(t, err)

	genericMockJobTwo := job.GetMockJobWithGenericSchedule()
	genericMockJobTwo.Init(cache)
	err = db.Save(genericMockJobTwo)
	assert.NoError(t, err)

	jobs, err := db.GetAll()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(jobs))
}

func TestEnd(t *testing.T) {
	db := NewTestDb(t)
	db.Close()
}

package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSaveAndGetJob(t *testing.T) {
	db := GetBoltDB(testDbPath)
	cache := NewMemoryJobCache(db, time.Second*60)
	defer db.Close()

	genericMockJob := GetMockJobWithGenericSchedule()
	genericMockJob.Init(cache)
	db.Save(genericMockJob)

	j, err := db.Get(genericMockJob.Id)
	assert.Nil(t, err)

	assert.WithinDuration(t, j.NextRunAt, genericMockJob.NextRunAt, 100*time.Microsecond)
	assert.Equal(t, j.Name, genericMockJob.Name)
	assert.Equal(t, j.Id, genericMockJob.Id)
	assert.Equal(t, j.Command, genericMockJob.Command)
	assert.Equal(t, j.Schedule, genericMockJob.Schedule)
	assert.Equal(t, j.Owner, genericMockJob.Owner)
	assert.Equal(t, j.SuccessCount, genericMockJob.SuccessCount)
}

func TestDeleteJob(t *testing.T) {
	db := GetBoltDB(testDbPath)
	cache := NewMemoryJobCache(db, time.Second*60)
	defer db.Close()

	genericMockJob := GetMockJobWithGenericSchedule()
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
	genericMockJob.Delete(cache, db)

	k, err := db.Get(genericMockJob.Id)
	assert.Error(t, err)
	assert.Nil(t, k)
	retrievedJobTwo, err := cache.Get(genericMockJob.Id)
	assert.Error(t, err)
	assert.Nil(t, retrievedJobTwo)

	genericMockJob.Delete(cache, db)
}

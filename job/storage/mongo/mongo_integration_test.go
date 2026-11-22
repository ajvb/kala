//go:build integration

package mongo

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ajvb/kala/v2/internal/containers"
	"github.com/ajvb/kala/v2/job"
	mgo "gopkg.in/mgo.v2"

	"github.com/stretchr/testify/assert"
)

// One MongoDB container for all the tests of this package.
var mongoContainer = containers.NewShared(containers.Spec{
	Repository: "mongo",
	Tag:        "3.4",
	Port:       "27017/tcp",
	Ready: func(hostPort string) error {
		session, err := mgo.DialWithTimeout(hostPort, 2*time.Second)
		if err != nil {
			return err
		}
		defer session.Close()
		return session.Ping()
	},
})

func TestMain(m *testing.M) {
	code := m.Run()
	if err := mongoContainer.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not remove the MongoDB container:", err)
	}
	os.Exit(code)
}

// NewTestDb connects to the shared container (failing the test if there is no
// Docker daemon) and deletes every job, so each test starts empty.
func NewTestDb(t *testing.T) *DB {
	t.Helper()
	collection = "test"
	var db = New(mongoContainer.HostPort(t), &mgo.Credential{})

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

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)
	err := db.Save(genericMockJob)
	if assert.NoError(t, err) {
		j, err := db.Get(genericMockJob.Id)
		if assert.Nil(t, err) {
			// MongoDB stores times with millisecond precision.
			genericMockJob.NextRunAt = genericMockJob.NextRunAt.Truncate(time.Millisecond)
			ok, err := genericMockJob.Compare(j)
			assert.True(t, ok, err)
		}
	}
}

func TestDeleteJob(t *testing.T) {
	db := NewTestDb(t)

	cache := job.NewLockFreeJobCache(db)

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
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
		err = genericMockJob.Delete(cache)
		assert.Nil(t, err)

		k, err := db.Get(genericMockJob.Id)
		assert.Error(t, err)
		assert.Nil(t, k)
		retrievedJobTwo, err := cache.Get(genericMockJob.Id)
		assert.Error(t, err)
		assert.Nil(t, retrievedJobTwo)

		genericMockJob.Delete(cache)
	}
}

func TestSaveAndGetAllJobs(t *testing.T) {
	db := NewTestDb(t)

	cache := job.NewLockFreeJobCache(db)

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
	assert.Equal(t, 2, len(jobs))
}

func TestEnd(t *testing.T) {
	db := NewTestDb(t)
	db.Close()
}

package redis

import (
	"errors"
	"testing"
	"time"

	"github.com/ajvb/kala/job"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"
)

type testJob struct {
	Job   *job.Job
	Bytes []byte
}

var (
	conn     = redigomock.NewConn()
	db       = mockRedisDB()
	cache    = job.NewMemoryJobCache(db)
	testJobs = initTestJobs(3)
)

// mockRedisDB returns a new DB with a mock Redis connection.
func mockRedisDB() DB {
	return DB{
		conn: conn,
	}
}

// testJobs initializes n testJobs
func initTestJobs(n int) []testJob {
	testJobs := []testJob{}

	for i := 0; i < n; i++ {
		j := job.GetMockJobWithGenericSchedule()
		j.Init(cache)

		bytes, err := j.Bytes()
		if err != nil {
			panic(err)
		}
		t := testJob{
			Job:   j,
			Bytes: bytes,
		}

		testJobs = append(testJobs, t)
	}

	return testJobs
}

func TestSaveJob(t *testing.T) {
	testJob := testJobs[0]

	// Expect a HSET operation to be performed the job hash key, job ID and encoded job
	conn.Command("HSET", HashKey, testJob.Job.Id, testJob.Bytes).
		Expect("ok")

	err := db.Save(testJob.Job)
	assert.Nil(t, err)

	// Test error handling
	conn.Command("HSET", HashKey, testJob.Job.Id, testJob.Bytes).
		ExpectError(errors.New("Redis error"))

	err = db.Save(testJob.Job)
	assert.NotNil(t, err)
}

func TestGetJob(t *testing.T) {
	testJob := testJobs[0]

	// Expect a HGET operation to be preformed with the job hash key and job ID
	conn.Command("HGET", HashKey, testJob.Job.Id).
		Expect(testJob.Bytes).
		ExpectError(nil)

	storedJob, err := db.Get(testJob.Job.Id)
	assert.Nil(t, err)

	assert.WithinDuration(t, storedJob.NextRunAt, testJob.Job.NextRunAt, 100*time.Microsecond)
	assert.Equal(t, testJob.Job.Name, storedJob.Name)
	assert.Equal(t, testJob.Job.Id, storedJob.Id)
	assert.Equal(t, testJob.Job.Command, storedJob.Command)
	assert.Equal(t, testJob.Job.Schedule, storedJob.Schedule)
	assert.Equal(t, testJob.Job.Owner, storedJob.Owner)
	assert.Equal(t, testJob.Job.Metadata.SuccessCount, storedJob.Metadata.SuccessCount)

	// Test error handling
	conn.Command("HGET", HashKey, testJob.Job.Id).
		ExpectError(errors.New("Redis error"))

	storedJob, err = db.Get(testJob.Job.Id)
	assert.NotNil(t, err)
}

func TestDeleteJob(t *testing.T) {
	testJob := testJobs[0]

	// Expect a HDEL operation to be preformed with the job ID
	conn.Command("HDEL", HashKey, testJob.Job.Id).
		Expect("ok").
		ExpectError(nil)

	err := db.Delete(testJob.Job.Id)
	assert.Nil(t, err)

	// Test error handling
	conn.Command("HDEL", HashKey, testJob.Job.Id).
		ExpectError(errors.New("Redis error"))

	err = db.Delete(testJob.Job.Id)
	assert.NotNil(t, err)
}

func TestGetAllJobs(t *testing.T) {
	// Expect a HVALS operation to be preformed with the job hash key
	conn.Command("HVALS", HashKey).
		Expect([]interface{}{
		testJobs[0].Bytes,
		testJobs[1].Bytes,
		testJobs[2].Bytes,
	}).
		ExpectError(nil)

	jobs, err := db.GetAll()
	assert.Nil(t, err)

	for i, j := range jobs {
		assert.WithinDuration(t, testJobs[i].Job.NextRunAt, j.NextRunAt, 100*time.Microsecond)
		assert.Equal(t, testJobs[i].Job.Name, j.Name)
		assert.Equal(t, testJobs[i].Job.Id, j.Id)
		assert.Equal(t, testJobs[i].Job.Command, j.Command)
		assert.Equal(t, testJobs[i].Job.Schedule, j.Schedule)
		assert.Equal(t, testJobs[i].Job.Owner, j.Owner)
		assert.Equal(t, testJobs[i].Job.Metadata.SuccessCount, j.Metadata.SuccessCount)
	}

	// Test erorr handling
	conn.Command("HVALS", HashKey).
		ExpectError(errors.New("Redis error"))

	jobs, err = db.GetAll()
	assert.NotNil(t, err)
}

func TestNew(t *testing.T) {

}

func TestClose(t *testing.T) {
	closedConn := false

	conn.CloseMock = func() error {
		closedConn = true
		return nil
	}

	err := db.Close()

	assert.Nil(t, err)
	assert.True(t, closedConn)

	// Reset closedConn
	closedConn = false

	conn.CloseMock = func() error {
		closedConn = false
		return errors.New("Error closing connection")
	}

	err = db.Close()

	assert.NotNil(t, err)
	assert.False(t, closedConn)
}

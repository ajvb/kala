package redis

import (
	"errors"
	"testing"
	"time"

	"github.com/ajvb/kala/v2/job"
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
	cache    = job.NewLockFreeJobCache(db)
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
	tjs := []testJob{}

	for i := 0; i < n; i++ {
		j := job.GetMockJobWithGenericSchedule(time.Now())
		j.Init(cache)

		bytes, err := j.Bytes()
		if err != nil {
			panic(err)
		}
		t := testJob{
			Job:   j,
			Bytes: bytes,
		}

		tjs = append(tjs, t)
	}

	return tjs
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

	ok, err := testJob.Job.Compare(storedJob)
	assert.True(t, ok, err)

	// Test error handling
	conn.Command("HGET", HashKey, testJob.Job.Id).
		ExpectError(errors.New("Redis error"))

	_, err = db.Get(testJob.Job.Id)
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
		ok, err := testJobs[i].Job.Compare(j)
		assert.True(t, ok, err)
	}

	// Test erorr handling
	conn.Command("HVALS", HashKey).
		ExpectError(errors.New("Redis error"))

	_, err = db.GetAll()
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

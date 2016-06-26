package client

import (
	"fmt"
	"net/http/httptest"
	"os"
	"time"

	"github.com/cescoferraro/kala/api"
	"github.com/cescoferraro/kala/job"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"testing"
)

func NewTestServer() *httptest.Server {
	r := mux.NewRouter()
	db := &job.MockDB{}
	cache := job.NewMemoryJobCache(db)
	api.SetupApiRoutes(r, cache, db, "")
	return httptest.NewServer(r)
}

func cleanUp() {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)
	jobs, err := kc.GetAllJobs()
	if len(jobs) == 0 {
		return
	}
	if err != nil {
		fmt.Printf("Problem running clean up (can't get all jobs from the server)")
		os.Exit(1)
	}
	for _, job := range jobs {
		kc.DeleteJob(job.Id)
	}
}

func NewJobMap() *job.Job {
	scheduleTime := time.Now().Add(time.Minute * 5)
	repeat := 1
	delay := "P1DT10M10S"
	parsedTime := scheduleTime.Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R%d/%s/%s", repeat, parsedTime, delay)

	return &job.Job{
		Schedule: scheduleStr,
		Name:     "mock_job",
		Command:  "bash -c 'date'",
		Owner:    "aj@cescoferraro.me",
	}
}

func TestCreateGetDeleteJob(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)
	j := NewJobMap()

	id, err := kc.CreateJob(j)
	assert.NoError(t, err)
	assert.NotEqual(t, id, "")

	respJob, err := kc.GetJob(id)
	assert.NoError(t, err)
	assert.Equal(t, j.Schedule, respJob.Schedule)
	assert.Equal(t, j.Name, respJob.Name)
	assert.Equal(t, j.Command, respJob.Command)
	assert.Equal(t, j.Owner, respJob.Owner)

	ok, err := kc.DeleteJob(id)
	assert.NoError(t, err)
	assert.True(t, ok)

	cleanUp()
}

func TestCreateJobError(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)
	j := NewJobMap()

	j.Schedule = "bbbbbbbbbbbbbbb"

	id, err := kc.CreateJob(j)
	assert.Error(t, err)
	assert.Equal(t, id, "")

	cleanUp()
}

func TestGetJobError(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)

	respJob, err := kc.GetJob("id-that-doesnt-exist")
	assert.Error(t, err)
	assert.Nil(t, respJob)

	cleanUp()
}

func TestDeleteJobError(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)

	ok, err := kc.DeleteJob("id-that-doesnt-exist")
	assert.Error(t, err)
	assert.False(t, ok)

	cleanUp()
}

func TestGetAllJobs(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)
	j := NewJobMap()

	id, err := kc.CreateJob(j)
	assert.NoError(t, err)

	jobs, err := kc.GetAllJobs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, j.Schedule, jobs[id].Schedule)
	assert.Equal(t, j.Name, jobs[id].Name)
	assert.Equal(t, j.Command, jobs[id].Command)
	assert.Equal(t, j.Owner, jobs[id].Owner)

	cleanUp()
}

func TestGetAllJobsNoJobsExist(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)

	jobs, err := kc.GetAllJobs()
	fmt.Printf("JOBS: %#v", jobs)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(jobs))

	cleanUp()
}

func TestDeleteJob(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)
	j := NewJobMap()

	id, err := kc.CreateJob(j)
	assert.NoError(t, err)

	ok, err := kc.DeleteJob(id)
	assert.NoError(t, err)
	assert.True(t, ok)

	respJob, err := kc.GetJob(id)
	assert.Nil(t, respJob)

	cleanUp()
}

func TestGetJobStats(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)
	j := NewJobMap()

	// Create the job
	id, err := kc.CreateJob(j)
	assert.NoError(t, err)
	// Start the job
	ok, err := kc.StartJob(id)
	now := time.Now()
	assert.NoError(t, err)
	assert.True(t, ok)
	// Wait let the job run
	time.Sleep(time.Second * 1)

	stats, err := kc.GetJobStats(id)
	assert.NoError(t, err)
	assert.Equal(t, id, stats[0].JobId)
	assert.Equal(t, uint(0), stats[0].NumberOfRetries)
	assert.True(t, stats[0].Success)
	assert.True(t, stats[0].ExecutionDuration != time.Duration(0))
	assert.WithinDuration(t, now, stats[0].RanAt, time.Second)

	cleanUp()
}

func TestGetJobStatsError(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)

	stats, err := kc.GetJobStats("not-an-actual-id")
	assert.Error(t, err)
	assert.Nil(t, stats)
}

func TestStartJob(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)
	j := NewJobMap()

	id, err := kc.CreateJob(j)
	assert.NoError(t, err)
	assert.NotEqual(t, id, "")

	now := time.Now()
	ok, err := kc.StartJob(id)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Wait let the job run
	time.Sleep(time.Second * 1)

	respJob, err := kc.GetJob(id)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), respJob.Metadata.SuccessCount)
	assert.WithinDuration(t, now, respJob.Metadata.LastSuccess, time.Second*2)
	assert.WithinDuration(t, now, respJob.Metadata.LastAttemptedRun, time.Second*2)

	cleanUp()
}

func TestStartJobError(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)

	ok, err := kc.StartJob("not-an-actual-id")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGetKalaStats(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()
	kc := New(ts.URL)

	for i := 0; i < 5; i++ {
		// Generate new job
		j := NewJobMap()

		id, err := kc.CreateJob(j)
		assert.NoError(t, err)
		assert.NotEqual(t, id, "")

		ok, err := kc.StartJob(id)
		assert.NoError(t, err)
		assert.True(t, ok)
	}
	time.Sleep(time.Second * 3)

	stats, err := kc.GetKalaStats()
	assert.NoError(t, err)

	assert.Equal(t, 5, stats.ActiveJobs)
	assert.Equal(t, 0, stats.DisabledJobs)
	assert.Equal(t, 5, stats.Jobs)

	assert.Equal(t, uint(0), stats.ErrorCount)
	assert.Equal(t, uint(5), stats.SuccessCount)

	cleanUp()
}

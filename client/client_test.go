package client

import (
	"fmt"
	"net/http/httptest"
	"time"

	"../api"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"testing"
)

func getNewTestServer() *httptest.Server {
	r := mux.NewRouter()
	api.SetupApiRoutes(r)
	return httptest.NewServer(r)
}

func generateNewJobMap() map[string]string {
	scheduleTime := time.Now().Add(time.Minute * 5)
	repeat := 1
	delay := "P1DT10M10S"
	parsedTime := scheduleTime.Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R%d/%s/%s", repeat, parsedTime, delay)

	return map[string]string{
		"schedule": scheduleStr,
		"name":     "mock_job",
		"command":  "bash -c 'date'",
		"owner":    "aj@ajvb.me",
	}
}

func TestCreateGetDeleteJob(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)
	j := generateNewJobMap()

	id, err := kc.CreateJob(j)
	assert.NoError(t, err)
	assert.NotEqual(t, id, "")

	respJob, err := kc.GetJob(id)
	assert.NoError(t, err)
	assert.Equal(t, j["schedule"], respJob.Schedule)
	assert.Equal(t, j["name"], respJob.Name)
	assert.Equal(t, j["command"], respJob.Command)
	assert.Equal(t, j["owner"], respJob.Owner)

	ok, err := kc.DeleteJob(id)
	assert.NoError(t, err)
	assert.True(t, ok)
}

// TODO
func TestCreateJobError(t *testing.T) {
}

func TestGetJobError(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)

	respJob, err := kc.GetJob("id-that-doesnt-exist")
	assert.Error(t, err)
	assert.Nil(t, respJob)
}

func TestDeleteJobError(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)

	ok, err := kc.DeleteJob("id-that-doesnt-exist")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestGetAllJobs(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)
	j := generateNewJobMap()

	id, err := kc.CreateJob(j)
	assert.NoError(t, err)

	jobs, err := kc.GetAllJobs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, j["schedule"], jobs[id].Schedule)
	assert.Equal(t, j["name"], jobs[id].Name)
	assert.Equal(t, j["command"], jobs[id].Command)
	assert.Equal(t, j["owner"], jobs[id].Owner)
}

func TestDeleteJob(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)
	j := generateNewJobMap()

	id, err := kc.CreateJob(j)
	assert.NoError(t, err)

	ok, err := kc.DeleteJob(id)
	assert.NoError(t, err)
	assert.True(t, ok)

	respJob, err := kc.GetJob(id)
	assert.Nil(t, respJob)
}

func TestGetJobStats(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)
	j := generateNewJobMap()

	// Create the job
	id, err := kc.CreateJob(j)
	assert.NoError(t, err)
	// Start the job
	ok, err := kc.StartJob(id)
	assert.NoError(t, err)
	assert.True(t, ok)
	// Wait let the job run
	time.Sleep(time.Second * 1)

	stats, err := kc.GetJobStats(id)
	assert.Equal(t, id, stats[0].JobId)
	assert.Equal(t, uint(0), stats[0].NumberOfRetries)
	assert.True(t, stats[0].Success)
}

func TestStartJob(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)
	j := generateNewJobMap()

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
	assert.Equal(t, uint(1), respJob.SuccessCount)
	assert.WithinDuration(t, now, respJob.LastSuccess, time.Second*2)
	assert.WithinDuration(t, now, respJob.LastAttemptedRun, time.Second*2)
}

func TestGetKalaStats(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)

	for i := 0; i < 5; i++ {
		// Generate new job
		j := generateNewJobMap()

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
}

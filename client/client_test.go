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

var (
	kc = NewKalaClient("http://127.0.0.1:8080")
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

func TestGetAllJobs(t *testing.T) {
	ts := getNewTestServer()
	defer ts.Close()
	kc := NewKalaClient(ts.URL)
	j := generateNewJobMap()

	id, err := kc.CreateJob(j)
	assert.NoError(t, err)

	jobs, err := kc.GetAllJobs()
	assert.NoError(t, err)
	fmt.Println("%#v", jobs)
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
}

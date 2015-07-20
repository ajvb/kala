package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/ajvb/kala/job"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

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
func generateJobAndCache() (*job.MemoryJobCache, *job.Job) {
	cache := job.NewMockCache()
	job := job.GetMockJobWithGenericSchedule()
	job.Init(cache)
	cache.Set(job)
	return cache, job
}

type ApiTestSuite struct {
	suite.Suite
}

func TestApiTestSuite(t *testing.T) {
	suite.Run(t, new(ApiTestSuite))
}

func (a *ApiTestSuite) TestHandleAddJob() {
	t := a.T()
	cache := job.NewMockCache()
	jobMap := generateNewJobMap()
	handler := HandleAddJob(cache)

	jsonJobMap, err := json.Marshal(jobMap)
	a.NoError(err)
	w, req := setupTestReq(t, "POST", ApiJobPath, jsonJobMap)
	handler(w, req)

	var addJobResp AddJobResponse
	err = json.Unmarshal(w.Body.Bytes(), &addJobResp)
	a.NoError(err)
	retrievedJob := cache.Get(addJobResp.Id)
	a.Equal(jobMap["name"], retrievedJob.Name)
	a.Equal(jobMap["owner"], retrievedJob.Owner)
	a.Equal(w.Code, http.StatusCreated)
}
func (a *ApiTestSuite) TestHandleAddJobFailureBadJson() {
	t := a.T()
	cache := job.NewMockCache()
	handler := HandleAddJob(cache)

	w, req := setupTestReq(t, "POST", ApiJobPath, []byte("asd"))
	handler(w, req)
	a.Equal(w.Code, http.StatusBadRequest)
}
func (a *ApiTestSuite) TestHandleAddJobFailureBadSchedule() {
	t := a.T()
	cache := job.NewMockCache()
	jobMap := generateNewJobMap()
	handler := HandleAddJob(cache)

	// Mess up schedule
	jobMap["schedule"] = "asdf"

	jsonJobMap, err := json.Marshal(jobMap)
	a.NoError(err)
	w, req := setupTestReq(t, "POST", ApiJobPath, jsonJobMap)
	handler(w, req)
	a.Equal(w.Code, http.StatusBadRequest)
	a.True(strings.Contains(bytes.NewBuffer(w.Body.Bytes()).String(), "when initializing"))
}

// TODO - needs mux
func (a *ApiTestSuite) TestDeleteJobSuccess() {
}

func (a *ApiTestSuite) TestGetJobSuccess() {
	t := a.T()
	db := &job.MockDB{}
	cache, job := generateJobAndCache()

	r := mux.NewRouter()
	r.HandleFunc(ApiJobPath+"{id}", HandleJobRequest(cache, db)).Methods("DELETE", "GET")
	ts := httptest.NewServer(r)

	_, req := setupTestReq(t, "GET", ts.URL+ApiJobPath+job.Id, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	a.NoError(err)

	var jobResp JobResponse
	body, err := ioutil.ReadAll(resp.Body)
	a.NoError(err)
	resp.Body.Close()
	err = json.Unmarshal(body, &jobResp)
	a.NoError(err)
	a.Equal(job.Id, jobResp.Job.Id)
	a.Equal(job.Owner, jobResp.Job.Owner)
	a.Equal(job.Name, jobResp.Job.Name)
	a.Equal(resp.StatusCode, http.StatusOK)
}

func (a *ApiTestSuite) TestHandleListJobStatsRequest() {
}
func (a *ApiTestSuite) TestHandleListJobStatsRequestNotFound() {
}

func (a *ApiTestSuite) TestHandleListJobsRequest() {
}

func (a *ApiTestSuite) TestHandleStartJobRequest() {
	t := a.T()
	cache, job := generateJobAndCache()
	r := mux.NewRouter()
	r.HandleFunc(ApiJobPath+"start/{id}", HandleStartJobRequest(cache)).Methods("POST")
	ts := httptest.NewServer(r)

	_, req := setupTestReq(t, "POST", ts.URL+ApiJobPath+"start/"+job.Id, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	a.NoError(err)

	now := time.Now()

	a.Equal(resp.StatusCode, http.StatusNoContent)

	a.Equal(job.SuccessCount, uint(1))
	a.WithinDuration(job.LastSuccess, now, 2*time.Second)
	a.WithinDuration(job.LastAttemptedRun, now, 2*time.Second)
}
func (a *ApiTestSuite) TestHandleStartJobRequestNotFound() {
	t := a.T()
	cache := job.NewMockCache()
	handler := HandleStartJobRequest(cache)
	w, req := setupTestReq(t, "POST", ApiJobPath+"start/asdasd", nil)
	handler(w, req)
	a.Equal(w.Code, http.StatusNotFound)
}

func (a *ApiTestSuite) TestHandleKalaStatsRequest() {
}

// setupTestReq constructs the writer recorder and request obj for use in tests
func setupTestReq(t assert.TestingT, method, path string, data []byte) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, path, bytes.NewReader(data))
	assert.NoError(t, err)
	return w, req
}

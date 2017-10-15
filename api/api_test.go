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

	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
		"owner":    "example@example.com",
	}
}

func generateNewRemoteJobMap() map[string]interface{} {
	return map[string]interface{}{
		"name":  "mock_remote_job",
		"owner": "example@example.com",
		"type":  1,
		"remote_properties": map[string]string{
			"url": "http://example.com",
		},
	}
}

func generateJobAndCache() (*job.LockFreeJobCache, *job.Job) {
	cache := job.NewMockCache()
	j := job.GetMockJobWithGenericSchedule()
	j.Init(cache)
	return cache, j
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
	jobMap["owner"] = ""
	defaultOwner := "aj+tester@ajvb.me"
	handler := HandleAddJob(cache, defaultOwner)

	jsonJobMap, err := json.Marshal(jobMap)
	a.NoError(err)
	w, req := setupTestReq(t, "POST", ApiJobPath, jsonJobMap)
	handler(w, req)

	var addJobResp AddJobResponse
	err = json.Unmarshal(w.Body.Bytes(), &addJobResp)
	a.NoError(err)
	retrievedJob, err := cache.Get(addJobResp.Id)
	a.NoError(err)
	a.Equal(jobMap["name"], retrievedJob.Name)
	a.NotEqual(jobMap["owner"], retrievedJob.Owner)
	a.Equal(defaultOwner, retrievedJob.Owner)
	a.Equal(w.Code, http.StatusCreated)
}

func (a *ApiTestSuite) TestHandleAddRemoteJob() {
	t := a.T()
	cache := job.NewMockCache()
	jobMap := generateNewRemoteJobMap()
	jobMap["owner"] = ""
	defaultOwner := "aj+tester@ajvb.me"
	handler := HandleAddJob(cache, defaultOwner)

	jsonJobMap, err := json.Marshal(jobMap)
	a.NoError(err)
	w, req := setupTestReq(t, "POST", ApiJobPath, jsonJobMap)
	handler(w, req)

	var addJobResp AddJobResponse
	err = json.Unmarshal(w.Body.Bytes(), &addJobResp)
	a.NoError(err)
	retrievedJob, err := cache.Get(addJobResp.Id)
	a.NoError(err)
	a.Equal(jobMap["name"], retrievedJob.Name)
	a.NotEqual(jobMap["owner"], retrievedJob.Owner)
	a.Equal(defaultOwner, retrievedJob.Owner)
	a.Equal(w.Code, http.StatusCreated)
}

func (a *ApiTestSuite) TestHandleAddJobFailureBadJson() {
	t := a.T()
	cache := job.NewMockCache()
	handler := HandleAddJob(cache, "")

	w, req := setupTestReq(t, "POST", ApiJobPath, []byte("asd"))
	handler(w, req)
	a.Equal(w.Code, http.StatusBadRequest)

}
func (a *ApiTestSuite) TestHandleAddJobFailureBadSchedule() {
	t := a.T()
	cache := job.NewMockCache()
	jobMap := generateNewJobMap()
	handler := HandleAddJob(cache, "")

	// Mess up schedule
	jobMap["schedule"] = "asdf"

	jsonJobMap, err := json.Marshal(jobMap)
	a.NoError(err)
	w, req := setupTestReq(t, "POST", ApiJobPath, jsonJobMap)
	handler(w, req)
	a.Equal(w.Code, http.StatusBadRequest)
	var respErr apiError
	err = json.Unmarshal(w.Body.Bytes(), &respErr)
	a.NoError(err)
	a.True(strings.Contains(respErr.Error, "when initializing"))
}

func (a *ApiTestSuite) TestDeleteJobSuccess() {
	t := a.T()
	db := &job.MockDB{}
	cache, job := generateJobAndCache()

	r := mux.NewRouter()
	r.HandleFunc(ApiJobPath+"{id}", HandleJobRequest(cache, db)).Methods("DELETE", "GET")
	ts := httptest.NewServer(r)

	_, req := setupTestReq(t, "DELETE", ts.URL+ApiJobPath+job.Id, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	a.NoError(err)
	a.Equal(resp.StatusCode, http.StatusNoContent)

	a.Nil(cache.Get(job.Id))
}

func (a *ApiTestSuite) TestHandleJobRequestJobDoesNotExist() {
	t := a.T()
	db := &job.MockDB{}
	cache := job.NewMockCache()

	r := mux.NewRouter()
	r.HandleFunc(ApiJobPath+"{id}", HandleJobRequest(cache, db)).Methods("DELETE", "GET")
	ts := httptest.NewServer(r)

	_, req := setupTestReq(t, "DELETE", ts.URL+ApiJobPath+"not-a-real-id", nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	a.NoError(err)

	a.Equal(resp.StatusCode, http.StatusNotFound)
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
	cache, job := generateJobAndCache()
	job.Run(cache)

	r := mux.NewRouter()
	r.HandleFunc(ApiJobPath+"stats/{id}", HandleListJobStatsRequest(cache)).Methods("GET")
	ts := httptest.NewServer(r)

	_, req := setupTestReq(a.T(), "GET", ts.URL+ApiJobPath+"stats/"+job.Id, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	a.NoError(err)

	var jobStatsResp ListJobStatsResponse
	body, err := ioutil.ReadAll(resp.Body)
	a.NoError(err)
	resp.Body.Close()
	err = json.Unmarshal(body, &jobStatsResp)
	a.NoError(err)

	a.Equal(len(jobStatsResp.JobStats), 1)
	a.Equal(jobStatsResp.JobStats[0].JobId, job.Id)
	a.Equal(jobStatsResp.JobStats[0].NumberOfRetries, uint(0))
	a.True(jobStatsResp.JobStats[0].Success)
}
func (a *ApiTestSuite) TestHandleListJobStatsRequestNotFound() {
	cache, _ := generateJobAndCache()
	r := mux.NewRouter()

	r.HandleFunc(ApiJobPath+"stats/{id}", HandleListJobStatsRequest(cache)).Methods("GET")
	ts := httptest.NewServer(r)

	_, req := setupTestReq(a.T(), "GET", ts.URL+ApiJobPath+"stats/not-a-real-id", nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	a.NoError(err)

	a.Equal(resp.StatusCode, http.StatusNotFound)
}

func (a *ApiTestSuite) TestHandleListJobsRequest() {
	cache, jobOne := generateJobAndCache()
	jobTwo := job.GetMockJobWithGenericSchedule()
	jobTwo.Init(cache)

	r := mux.NewRouter()
	r.HandleFunc(ApiJobPath, HandleListJobsRequest(cache)).Methods("GET")
	ts := httptest.NewServer(r)

	_, req := setupTestReq(a.T(), "GET", ts.URL+ApiJobPath, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	a.NoError(err)

	var jobsResp ListJobsResponse
	unmarshallRequestBody(a.T(), resp, &jobsResp)

	a.Equal(len(jobsResp.Jobs), 2)
	a.Equal(jobsResp.Jobs[jobOne.Id].Schedule, jobOne.Schedule)
	a.Equal(jobsResp.Jobs[jobOne.Id].Name, jobOne.Name)
	a.Equal(jobsResp.Jobs[jobOne.Id].Owner, jobOne.Owner)
	a.Equal(jobsResp.Jobs[jobOne.Id].Command, jobOne.Command)

	a.Equal(jobsResp.Jobs[jobTwo.Id].Schedule, jobTwo.Schedule)
	a.Equal(jobsResp.Jobs[jobTwo.Id].Name, jobTwo.Name)
	a.Equal(jobsResp.Jobs[jobTwo.Id].Owner, jobTwo.Owner)
	a.Equal(jobsResp.Jobs[jobTwo.Id].Command, jobTwo.Command)
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

	a.Equal(job.Metadata.SuccessCount, uint(1))
	a.WithinDuration(job.Metadata.LastSuccess, now, 2*time.Second)
	a.WithinDuration(job.Metadata.LastAttemptedRun, now, 2*time.Second)
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
	cache, _ := generateJobAndCache()
	jobTwo := job.GetMockJobWithGenericSchedule()
	jobTwo.Init(cache)
	jobTwo.Run(cache)

	r := mux.NewRouter()
	r.HandleFunc(ApiUrlPrefix+"stats", HandleKalaStatsRequest(cache)).Methods("GET")
	ts := httptest.NewServer(r)

	_, req := setupTestReq(a.T(), "GET", ts.URL+ApiUrlPrefix+"stats", nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	a.NoError(err)

	now := time.Now()

	var statsResp KalaStatsResponse
	unmarshallRequestBody(a.T(), resp, &statsResp)

	a.Equal(statsResp.Stats.Jobs, 2)
	a.Equal(statsResp.Stats.ActiveJobs, 2)
	a.Equal(statsResp.Stats.DisabledJobs, 0)

	a.Equal(statsResp.Stats.ErrorCount, uint(0))
	a.Equal(statsResp.Stats.SuccessCount, uint(1))

	a.WithinDuration(statsResp.Stats.LastAttemptedRun, now, 2*time.Second)
	a.WithinDuration(statsResp.Stats.CreatedAt, now, 2*time.Second)
}

func (a *ApiTestSuite) TestSetupApiRoutes() {
	db := &job.MockDB{}
	cache := job.NewMockCache()
	r := mux.NewRouter()

	SetupApiRoutes(r, cache, db, "")

	a.NotNil(r)
	a.IsType(r, mux.NewRouter())
}

// setupTestReq constructs the writer recorder and request obj for use in tests
func setupTestReq(t assert.TestingT, method, path string, data []byte) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, path, bytes.NewReader(data))
	assert.NoError(t, err)
	return w, req
}

func unmarshallRequestBody(t assert.TestingT, resp *http.Response, obj interface{}) {
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	resp.Body.Close()
	err = json.Unmarshal(body, obj)
	assert.NoError(t, err)
}

package client

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/job"

	"github.com/dghubble/sling"
)

var (
	JobNotFound      = errors.New("Job not found")
	JobCreationError = errors.New("Error creating job")

	GenericError = errors.New("An error occured performing your request")
)

// KalaClient is the base struct for this package.
type KalaClient struct {
	apiEndpoint string
	requester   *sling.Sling
}

// New is used to create a new KalaClient based off of the apiEndpoint
// Example:
// 		c := New("http://127.0.0.1:8000")
func New(apiEndpoint string) *KalaClient {
	if strings.HasSuffix(apiEndpoint, "/") {
		apiEndpoint = apiEndpoint[:len(apiEndpoint)-1]
	}

	return &KalaClient{
		apiEndpoint: apiEndpoint + api.ApiUrlPrefix,
		requester:   sling.New().Base(apiEndpoint + api.ApiUrlPrefix),
	}
}

// CreateJob is used for creating a new job within Kala. Note that the
// Name and Command fields are the only ones that are required.
// Example:
// 		c := New("http://127.0.0.1:8000")
// 		body := &job.Job{
//			Schedule: "R2/2015-06-04T19:25:16.828696-07:00/PT10S",
//			Name:	  "test_job",
//			Command:  "bash -c 'date'",
//		}
//		id, err := c.CreateJob(body)
func (kc *KalaClient) CreateJob(body *job.Job) (string, error) {
	id := &api.AddJobResponse{}
	resp, err := kc.requester.New().Post(api.JobPath).BodyJSON(body).ReceiveSuccess(id)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusCreated {
		return "", JobCreationError
	}
	return id.Id, nil
}

// GetJob is used to retrieve a Job from Kala by its ID.
// Example:
// 		c := New("http://127.0.0.1:8000")
//		id := "93b65499-b211-49ce-57e0-19e735cc5abd"
//		job, err := c.GetJob(id)
func (kc *KalaClient) GetJob(id string) (*job.Job, error) {
	j := &api.JobResponse{}
	resp, err := kc.requester.New().Get(api.JobPath + id + "/").ReceiveSuccess(j)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, JobNotFound
	}
	return j.Job, nil
}

// GetAllJobs returns a map of string (ID's) to job.Job's which contains
// all Jobs currently within Kala.
// Example:
// 		c := New("http://127.0.0.1:8000")
//		jobs, err := c.GetAllJobs()
func (kc *KalaClient) GetAllJobs() (map[string]*job.Job, error) {
	jobs := &api.ListJobsResponse{}
	resp, err := kc.requester.New().Get(api.JobPath).ReceiveSuccess(jobs)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, GenericError
	}
	return jobs.Jobs, nil
}

// DeleteJob is used to delete a Job from Kala by its ID.
// Example:
// 		c := New("http://127.0.0.1:8000")
//		id := "93b65499-b211-49ce-57e0-19e735cc5abd"
//		ok, err := c.DeleteJob(id)
func (kc *KalaClient) DeleteJob(id string) (bool, error) {
	// nil is completely safe to use, as it is simply ignored in the sling library.
	resp, err := kc.requester.New().Delete(api.JobPath + id + "/").ReceiveSuccess(nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusNoContent {
		return false, fmt.Errorf("Delete failed with a status code of %d", resp.StatusCode)
	}
	return true, nil
}

// GetJobStats is used to retrieve stats about a Job from Kala by its ID.
// Example:
// 		c := New("http://127.0.0.1:8000")
//		id := "93b65499-b211-49ce-57e0-19e735cc5abd"
//		stats, err := c.GetJobStats(id)
func (kc *KalaClient) GetJobStats(id string) ([]*job.JobStat, error) {
	js := &api.ListJobStatsResponse{}
	resp, err := kc.requester.New().Get(api.JobPath+"stats/"+id+"/").Receive(js, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, GenericError
	}
	return js.JobStats, nil
}

// StartJob is used to manually start a Job by its ID.
// Example:
// 		c := New("http://127.0.0.1:8000")
//		id := "93b65499-b211-49ce-57e0-19e735cc5abd"
//		ok, err := c.StartJob(id)
func (kc *KalaClient) StartJob(id string) (bool, error) {
	resp, err := kc.requester.New().Post(api.JobPath+"start/"+id+"/").Receive(nil, nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusNoContent {
		return false, nil
	}
	return true, nil
}

// GetKalaStats retrieves system-level metrics about Kala
// Example:
// 		c := New("http://127.0.0.1:8000")
//		stats, err := c.GetKalaStats()
func (kc *KalaClient) GetKalaStats() (*job.KalaStats, error) {
	ks := &api.KalaStatsResponse{}
	resp, err := kc.requester.New().Get("stats/").Receive(ks, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, GenericError
	}
	return ks.Stats, nil
}

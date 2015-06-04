package client

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/job"

	"github.com/dghubble/sling"
)

// KalaClient is the base struct for this package.
type KalaClient struct {
	apiEndpoint string
	requester   *sling.Sling
}

// New is used to create a new KalaClient based off of the apiEndpoint
func New(apiEndpoint string) *KalaClient {
	if strings.HasPrefix(apiEndpoint, "/") {
		apiEndpoint = apiEndpoint[:len(apiEndpoint)-1]
	}

	return &KalaClient{
		apiEndpoint: apiEndpoint + api.ApiUrlPrefix,
		requester:   sling.New().Base(apiEndpoint + api.ApiUrlPrefix),
	}
}

func (kc *KalaClient) CreateJob(body map[string]string) (string, error) {
	id := &api.AddJobResponse{}
	_, err := kc.requester.New().Post(api.JobPath).JsonBody(body).Receive(id)
	if err != nil {
		return "", err
	}
	return id.Id, nil
}

func (kc *KalaClient) GetJob(id string) (*job.Job, error) {
	j := &api.JobResponse{}
	_, err := kc.requester.New().Get(api.JobPath + id).Receive(j)
	if err != nil {
		return nil, err
	}
	return j.Job, nil
}

func (kc *KalaClient) GetAllJobs() (map[string]*job.Job, error) {
	jobs := &api.ListJobsResponse{}
	_, err := kc.requester.New().Get(api.JobPath).Receive(jobs)
	if err != nil {
		return nil, err
	}
	return jobs.Jobs, nil
}

func (kc *KalaClient) DeleteJob(id string) (bool, error) {
	// nil is completely safe to use, as it is simply ignored in the sling library.
	resp, err := kc.requester.New().Delete(api.JobPath + id).Receive(nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusNoContent {
		return false, fmt.Errorf("Delete failed with a status code of %d", resp.StatusCode)
	}
	return true, nil
}

func (kc *KalaClient) GetJobStats(id string) ([]*job.JobStat, error) {
	js := &api.ListJobStatsResponse{}
	_, err := kc.requester.New().Get(api.JobPath + "stats/" + id).Receive(js)
	if err != nil {
		return nil, err
	}
	return js.JobStats, nil
}

func (kc *KalaClient) StartJob(id string) (bool, error) {
	resp, err := kc.requester.New().Post(api.JobPath + "start/" + id).Receive(nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusNoContent {
		return false, nil
	}
	return true, nil
}

func (kc *KalaClient) GetKalaStats() (*job.KalaStats, error) {
	ks := &api.KalaStatsResponse{}
	_, err := kc.requester.New().Get("stats").Receive(ks)
	if err != nil {
		return nil, err
	}
	return ks.Stats, nil
}

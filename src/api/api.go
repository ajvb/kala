package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"../job"

	"github.com/222Labs/common/go/logging"
	"github.com/gorilla/mux"
)

var (
	log = logging.GetLogger("api")
)

type KalaStatsResponse struct {
	Stats *job.KalaStats
}

func HandleKalaStats(w http.ResponseWriter, r *http.Request) {
	resp := &KalaStatsResponse{
		Stats: job.NewKalaStats(),
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}

}

type ListJobsResponse struct {
	Jobs map[string]*job.Job `json:"jobs"`
}

// HandleListJobs responds with an array of all Jobs within the server,
// active or disabled.
func HandleListJobs(w http.ResponseWriter, r *http.Request) {
	resp := &ListJobsResponse{
		Jobs: job.AllJobs.GetAll(),
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}

}

type AddJobResponse struct {
	Id string `json:"id"`
}

func unmarshalNewJob(r *http.Request) (*job.Job, error) {
	newJob := &job.Job{}

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Error("Error occured when reading r.Body: %s", err)
		return nil, err
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, newJob); err != nil {
		log.Error("Error occured when unmarshalling data: %s", err)
		return nil, err
	}

	return newJob, nil
}

// HandleAddJob takes a job object and unmarshals it to a Job type,
// and then throws the job in the schedulers.
func HandleAddJob(w http.ResponseWriter, r *http.Request) {
	newJob, err := unmarshalNewJob(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// TODO
	// 2. Verify that "protected" fields were not touched.

	err = newJob.Init()
	if err != nil {
		errStr := "Error occured when initializing the job"
		log.Error(errStr+": %s", err)
		http.Error(w, errStr, 400)
		return
	}
	job.AllJobs.Set(newJob)

	resp := &AddJobResponse{
		Id: newJob.Id,
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}
}

func HandleJobRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if r.Method == "DELETE" {
		handleDeleteJob(w, r, id)
	} else if r.Method == "GET" {
		handleGetJob(w, r, id)
	}
}

func handleDeleteJob(w http.ResponseWriter, r *http.Request, id string) {
	log.Info("Deleting job: %s", id)

	j := job.AllJobs.Get(id)
	j.Delete()

	w.WriteHeader(http.StatusNoContent)
}

type JobResponse struct {
	Job *job.Job `json:"job"`
}

func handleGetJob(w http.ResponseWriter, r *http.Request, id string) {
	j := job.AllJobs.Get(id)

	resp := &JobResponse{
		Job: j,
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}
}

func HandleStartJobRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	j := job.AllJobs.Get(id)

	j.Run()

	resp := &JobResponse{
		Job: j,
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}
}

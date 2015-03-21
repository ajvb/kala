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
	log = logging.GetLogger("kala")
)

type ListJobsResponse struct {
	Jobs map[string]*job.Job `json:"jobs"`
}

// HandleListJobs responds with an array of all Jobs within the server,
// active or disabled.
func HandleListJobs(w http.ResponseWriter, r *http.Request) {
	resp := &ListJobsResponse{
		Jobs: job.AllJobs,
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

// HandleAddJob takes a job object and unmarshals it to a Job type,
// and then throws the job in the schedulers.
func HandleAddJob(w http.ResponseWriter, r *http.Request) {
	newJob := &job.Job{}

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Error("Error occured when reading r.Body: %s", err)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, newJob); err != nil {
		errStr := "Error occured when unmarshalling data"
		log.Error(errStr+": %s", err)
		http.Error(w, errStr, 400)
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
	log.Info("New Job: %#v", newJob)
	job.AllJobs[newJob.Id] = newJob

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
	delete(job.AllJobs, id)

	w.WriteHeader(http.StatusNoContent)
}

type GetJobResponse struct {
	Job *job.Job `json:"job"`
}

func handleGetJob(w http.ResponseWriter, r *http.Request, id string) {
	j := job.AllJobs[id]

	resp := &GetJobResponse{
		Job: j,
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}
}

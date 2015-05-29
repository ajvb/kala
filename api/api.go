package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"../job"
	"../utils/logging"

	"github.com/gorilla/mux"
)

const (
	ApiUrlPrefix = "/api/v1/"
	JobPath      = "job/"
	ApiJobPath   = ApiUrlPrefix + JobPath

	contentType     = "Content-Type"
	jsonContentType = "application/json;charset=UTF-8"
)

var (
	log = logging.GetLogger("api")
)

type KalaStatsResponse struct {
	Stats *job.KalaStats
}

func HandleKalaStatsRequest(w http.ResponseWriter, r *http.Request) {
	resp := &KalaStatsResponse{
		Stats: job.NewKalaStats(),
	}

	w.Header().Set(contentType, jsonContentType)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}

}

type ListJobStatsResponse struct {
	JobStats []*job.JobStat `json:"job_stats"`
}

func HandleListJobStatsRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	j := job.AllJobs.Get(id)

	resp := &ListJobStatsResponse{
		JobStats: j.Stats,
	}

	w.Header().Set(contentType, jsonContentType)
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
func HandleListJobsRequest(w http.ResponseWriter, r *http.Request) {
	resp := &ListJobsResponse{
		Jobs: job.AllJobs.GetAll(),
	}

	w.Header().Set(contentType, jsonContentType)
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

	w.Header().Set(contentType, jsonContentType)
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
	log.Debug("Deleting job: %s", id)

	j := job.AllJobs.Get(id)
	if j == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	j.Delete()

	w.WriteHeader(http.StatusNoContent)
}

type JobResponse struct {
	Job *job.Job `json:"job"`
}

func handleGetJob(w http.ResponseWriter, r *http.Request, id string) {
	j := job.AllJobs.Get(id)
	if j == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp := &JobResponse{
		Job: j,
	}

	w.Header().Set(contentType, jsonContentType)
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

	w.WriteHeader(http.StatusNoContent)
}

func SetupApiRoutes(r *mux.Router) {
	// Route for creating a job
	r.HandleFunc(ApiJobPath, HandleAddJob).Methods("POST")
	// Route for deleting and getting a job
	r.HandleFunc(ApiJobPath+"{id}", HandleJobRequest).Methods("DELETE", "GET")
	// Route for getting job stats
	r.HandleFunc(ApiJobPath+"stats/{id}", HandleListJobStatsRequest).Methods("GET")
	// Route for listing all jops
	r.HandleFunc(ApiJobPath, HandleListJobsRequest).Methods("GET")
	// Route for manually start a job
	r.HandleFunc(ApiJobPath+"start/{id}", HandleStartJobRequest).Methods("POST")
	// Route for getting app-level metrics
	r.HandleFunc(ApiUrlPrefix+"stats", HandleKalaStatsRequest).Methods("GET")
}

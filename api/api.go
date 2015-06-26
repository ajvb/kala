package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ajvb/kala/job"
	"github.com/ajvb/kala/utils/logging"

	"github.com/gorilla/mux"
)

const (
	// Base API v1 Path
	ApiUrlPrefix = "/api/v1/"

	JobPath    = "job/"
	ApiJobPath = ApiUrlPrefix + JobPath

	contentType     = "Content-Type"
	jsonContentType = "application/json;charset=UTF-8"
)

var (
	log = logging.GetLogger("api")
)

type KalaStatsResponse struct {
	Stats *job.KalaStats
}

// HandleKalaStatsRequest is the hanlder for getting system-level metrics
// /api/v1/stats
func HandleKalaStatsRequest(w http.ResponseWriter, r *http.Request) {
	resp := &KalaStatsResponse{
		Stats: job.NewKalaStats(),
	}

	w.Header().Set(contentType, jsonContentType)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}

}

type ListJobStatsResponse struct {
	JobStats []*job.JobStat `json:"job_stats"`
}

// HandleListJobStatsRequest is the handler for getting job-specific stats
// /api/v1/job/stats/{id}
func HandleListJobStatsRequest(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	j := job.AllJobs.Get(id)

	resp := &ListJobStatsResponse{
		JobStats: j.Stats,
	}

	w.Header().Set(contentType, jsonContentType)
	w.WriteHeader(http.StatusOK)
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
	w.WriteHeader(http.StatusOK)
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

// HandleJobRequest routes requests to /api/v1/job/{id} to either
// handleDeleteJob if its a DELETE or handleGetJob if its a GET request.
func HandleJobRequest(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	j := job.AllJobs.Get(id)
	if j == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method == "DELETE" {
		handleDeleteJob(w, r, j)
	} else if r.Method == "GET" {
		handleGetJob(w, r, j)
	}
}

func handleDeleteJob(w http.ResponseWriter, r *http.Request, j *job.Job) {
	j.Delete()

	w.WriteHeader(http.StatusNoContent)
}

type JobResponse struct {
	Job *job.Job `json:"job"`
}

func handleGetJob(w http.ResponseWriter, r *http.Request, j *job.Job) {
	resp := &JobResponse{
		Job: j,
	}

	w.Header().Set(contentType, jsonContentType)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}
}

// HandleStartJobRequest is the handler for manually starting jobs
// /api/v1/job/start/{id}
func HandleStartJobRequest(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	j := job.AllJobs.Get(id)

	j.Run()

	w.WriteHeader(http.StatusNoContent)
}

// SetupApiRoutes is used within main to initialize all of the routes
func SetupApiRoutes(r *mux.Router) {
	// Route for creating a job
	r.HandleFunc(ApiJobPath, HandleAddJob).Methods("POST")
	r.HandleFunc(ApiUrlPrefix+"job", HandleAddJob).Methods("POST")
	// Route for deleting and getting a job
	r.HandleFunc(ApiJobPath+"{id}", HandleJobRequest).Methods("DELETE", "GET")
	r.HandleFunc(ApiJobPath+"{id}/", HandleJobRequest).Methods("DELETE", "GET")
	// Route for getting job stats
	r.HandleFunc(ApiJobPath+"stats/{id}", HandleListJobStatsRequest).Methods("GET")
	r.HandleFunc(ApiJobPath+"stats/{id}/", HandleListJobStatsRequest).Methods("GET")
	// Route for listing all jops
	r.HandleFunc(ApiJobPath, HandleListJobsRequest).Methods("GET")
	r.HandleFunc(ApiUrlPrefix+"job", HandleListJobsRequest).Methods("GET")
	// Route for manually start a job
	r.HandleFunc(ApiJobPath+"start/{id}", HandleStartJobRequest).Methods("POST")
	r.HandleFunc(ApiJobPath+"start/{id}/", HandleStartJobRequest).Methods("POST")
	// Route for getting app-level metrics
	r.HandleFunc(ApiUrlPrefix+"stats", HandleKalaStatsRequest).Methods("GET")
	r.HandleFunc(ApiUrlPrefix+"stats/", HandleKalaStatsRequest).Methods("GET")
}

package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ajvb/kala/api/middleware"
	"github.com/ajvb/kala/job"
	"github.com/ajvb/kala/utils/logging"

	"github.com/codegangsta/negroni"
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
	log = logging.GetLogger("kala.api")
)

type KalaStatsResponse struct {
	Stats *job.KalaStats
}

// HandleKalaStatsRequest is the hanlder for getting system-level metrics
// /api/v1/stats
func HandleKalaStatsRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := &KalaStatsResponse{
			Stats: job.NewKalaStats(cache),
		}

		w.Header().Set(contentType, jsonContentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Error occured when marshalling response: %s", err)
			return
		}
	}
}

type ListJobStatsResponse struct {
	JobStats []*job.JobStat `json:"job_stats"`
}

// HandleListJobStatsRequest is the handler for getting job-specific stats
// /api/v1/job/stats/{id}
func HandleListJobStatsRequest(stats *job.StatsManager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		jobStats := stats.GetStats(id)
		if len(jobStats) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := &ListJobStatsResponse{
			JobStats: jobStats,
		}

		w.Header().Set(contentType, jsonContentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Error occured when marshalling response: %s", err)
			return
		}
	}
}

type ListJobsResponse struct {
	Jobs map[string]*job.Job `json:"jobs"`
}

// HandleListJobs responds with an array of all Jobs within the server,
// active or disabled.
func HandleListJobsRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		allJobs := cache.GetAll()
		allJobs.Lock.RLock()
		defer allJobs.Lock.RUnlock()

		resp := &ListJobsResponse{
			Jobs: allJobs.Jobs,
		}

		w.Header().Set(contentType, jsonContentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Error occured when marshalling response: %s", err)
			return
		}
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
func HandleAddJob(cache job.JobCache, stats *job.StatsManager) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		newJob, err := unmarshalNewJob(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = newJob.Init(cache, stats)
		if err != nil {
			errStr := "Error occured when initializing the job"
			log.Error(errStr+": %s", err)
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}

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
}

// HandleJobRequest routes requests to /api/v1/job/{id} to either
// handleDeleteJob if its a DELETE or handleGetJob if its a GET request.
func HandleJobRequest(cache job.JobCache, db job.JobDB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		j, err := cache.Get(id)
		if err != nil {
			log.Error("Error occured when trying to get the job you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if j == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method == "DELETE" {
			err = j.Delete(cache, db)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		} else if r.Method == "GET" {
			handleGetJob(w, r, j)
		}
	}
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
func HandleStartJobRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		j, err := cache.Get(id)
		if err != nil {
			log.Error("Error occured when trying to get the job you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if j == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// TODO: Stop job timer as well.
		j.Run(cache)

		w.WriteHeader(http.StatusNoContent)
	}
}

// SetupApiRoutes is used within main to initialize all of the routes
func SetupApiRoutes(r *mux.Router, cache job.JobCache, db job.JobDB, stats *job.StatsManager) {
	// Route for creating a job
	r.HandleFunc(ApiJobPath, HandleAddJob(cache, stats)).Methods("POST")
	r.HandleFunc(ApiUrlPrefix+"job", HandleAddJob(cache, stats)).Methods("POST")
	// Route for deleting and getting a job
	r.HandleFunc(ApiJobPath+"{id}", HandleJobRequest(cache, db)).Methods("DELETE", "GET")
	r.HandleFunc(ApiJobPath+"{id}/", HandleJobRequest(cache, db)).Methods("DELETE", "GET")
	// Route for getting job stats
	r.HandleFunc(ApiJobPath+"stats/{id}", HandleListJobStatsRequest(stats)).Methods("GET")
	r.HandleFunc(ApiJobPath+"stats/{id}/", HandleListJobStatsRequest(stats)).Methods("GET")
	// Route for listing all jops
	r.HandleFunc(ApiJobPath, HandleListJobsRequest(cache)).Methods("GET")
	r.HandleFunc(ApiUrlPrefix+"job", HandleListJobsRequest(cache)).Methods("GET")
	// Route for manually start a job
	r.HandleFunc(ApiJobPath+"start/{id}", HandleStartJobRequest(cache)).Methods("POST")
	r.HandleFunc(ApiJobPath+"start/{id}/", HandleStartJobRequest(cache)).Methods("POST")
	// Route for getting app-level metrics
	r.HandleFunc(ApiUrlPrefix+"stats", HandleKalaStatsRequest(cache)).Methods("GET")
	r.HandleFunc(ApiUrlPrefix+"stats/", HandleKalaStatsRequest(cache)).Methods("GET")
}

func StartServer(listenAddr string, cache job.JobCache, db job.JobDB, stats *job.StatsManager) error {
	r := mux.NewRouter()
	SetupApiRoutes(r, cache, db, stats)
	n := negroni.New(negroni.NewRecovery(), &middleware.Logger{log})
	n.UseHandler(r)
	return http.ListenAndServe(listenAddr, n)
}

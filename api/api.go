package api

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ajvb/kala/api/middleware"
	"github.com/ajvb/kala/job"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	// Base API v1 Path
	ApiUrlPrefix = "/api/v1/"

	JobPath    = "job/"
	ApiJobPath = ApiUrlPrefix + JobPath

	contentType     = "Content-Type"
	jsonContentType = "application/json;charset=UTF-8"
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
			log.Errorf("Error occured when marshalling response: %s", err)
			return
		}
	}
}

type ListJobStatsResponse struct {
	JobStats []*job.JobStat `json:"job_stats"`
}

// HandleListJobStatsRequest is the handler for getting job-specific stats
// /api/v1/job/stats/{id}
func HandleListJobStatsRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		j, err := cache.Get(id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := &ListJobStatsResponse{
			JobStats: j.Stats,
		}

		w.Header().Set(contentType, jsonContentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Errorf("Error occured when marshalling response: %s", err)
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
			log.Errorf("Error occured when marshalling response: %s", err)
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
		log.Errorf("Error occured when reading r.Body: %s", err)
		return nil, err
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, newJob); err != nil {
		log.Errorf("Error occured when unmarshalling data: %s", err)
		return nil, err
	}

	return newJob, nil
}

// HandleAddJob takes a job object and unmarshals it to a Job type,
// and then throws the job in the schedulers.
func HandleAddJob(cache job.JobCache, defaultOwner string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		newJob, err := unmarshalNewJob(r)
		if err != nil {
			errorEncodeJSON(err, http.StatusBadRequest, w)
			return
		}

		if defaultOwner != "" && newJob.Owner == "" {
			newJob.Owner = defaultOwner
		}

		err = newJob.Init(cache)
		if err != nil {
			errStr := "Error occured when initializing the job"
			log.Errorf(errStr+": %s", err)
			errorEncodeJSON(errors.New(errStr), http.StatusBadRequest, w)
			return
		}

		resp := &AddJobResponse{
			Id: newJob.Id,
		}

		w.Header().Set(contentType, jsonContentType)
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Errorf("Error occured when marshalling response: %s", err)
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
			log.Errorf("Error occured when trying to get the job you requested.")
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
				errorEncodeJSON(err, http.StatusInternalServerError, w)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		} else if r.Method == "GET" {
			handleGetJob(w, r, j)
		}
	}
}

// HandleDeleteAllJobs is the handler for deleting all jobs
// DELETE /api/v1/job/all
func HandleDeleteAllJobs(cache job.JobCache, db job.JobDB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := job.DeleteAll(cache, db); err != nil {
			errorEncodeJSON(err, http.StatusInternalServerError, w)
		} else {
			w.WriteHeader(http.StatusNoContent)
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
		log.Errorf("Error occured when marshalling response: %s", err)
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
			log.Errorf("Error occured when trying to get the job you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if j == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		j.StopTimer()
		j.Run(cache)

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleDisableJobRequest is the handler for mdisabling jobs
// /api/v1/job/disable/{id}
func HandleDisableJobRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		j, err := cache.Get(id)
		if err != nil {
			log.Errorf("Error occured when trying to get the job you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if j == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		j.Disable()

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleEnableJobRequest is the handler for enable jobs
// /api/v1/job/enable/{id}
func HandleEnableJobRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		j, err := cache.Get(id)
		if err != nil {
			log.Errorf("Error occured when trying to get the job you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if j == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		j.Enable(cache)

		w.WriteHeader(http.StatusNoContent)
	}
}

type apiError struct {
	Error string `json:"error"`
}

func errorEncodeJSON(errToEncode error, status int, w http.ResponseWriter) {
	js, err := json.Marshal(apiError{Error: errToEncode.Error()})
	if err != nil {
		log.Errorf("could not encode error message: %v", err)
		return
	}
	w.Header().Set(contentType, jsonContentType)
	http.Error(w, string(js), status)
}

// SetupApiRoutes is used within main to initialize all of the routes
func SetupApiRoutes(r *mux.Router, cache job.JobCache, db job.JobDB, defaultOwner string) {
	// Route for creating a job
	r.HandleFunc(ApiJobPath, HandleAddJob(cache, defaultOwner)).Methods("POST")
	// Route for deleting all jobs
	r.HandleFunc(ApiJobPath+"all/", HandleDeleteAllJobs(cache, db)).Methods("DELETE")
	// Route for deleting and getting a job
	r.HandleFunc(ApiJobPath+"{id}/", HandleJobRequest(cache, db)).Methods("DELETE", "GET")
	// Route for getting job stats
	r.HandleFunc(ApiJobPath+"stats/{id}/", HandleListJobStatsRequest(cache)).Methods("GET")
	// Route for listing all jops
	r.HandleFunc(ApiJobPath, HandleListJobsRequest(cache)).Methods("GET")
	// Route for manually start a job
	r.HandleFunc(ApiJobPath+"start/{id}/", HandleStartJobRequest(cache)).Methods("POST")
	// Route for manually start a job
	r.HandleFunc(ApiJobPath+"enable/{id}/", HandleEnableJobRequest(cache)).Methods("POST")
	// Route for manually disable a job
	r.HandleFunc(ApiJobPath+"disable/{id}/", HandleDisableJobRequest(cache)).Methods("POST")
	// Route for getting app-level metrics
	r.HandleFunc(ApiUrlPrefix+"stats/", HandleKalaStatsRequest(cache)).Methods("GET")
}

func MakeServer(listenAddr string, cache job.JobCache, db job.JobDB, defaultOwner string) *http.Server {
	r := mux.NewRouter()
	// Allows for the use for /job as well as /job/
	r.StrictSlash(true)
	SetupApiRoutes(r, cache, db, defaultOwner)
	r.PathPrefix("/webui/").Handler(http.StripPrefix("/webui/", http.FileServer(http.Dir("./webui/"))))
	n := negroni.New(negroni.NewRecovery(), &middleware.Logger{log.Logger{}})
	n.UseHandler(r)
	return &http.Server{
		Addr:    listenAddr,
		Handler: n,
	}
}

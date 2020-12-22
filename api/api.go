package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"runtime"

	"bitbucket.org/nextiva/nextkala/api/middleware"
	"bitbucket.org/nextiva/nextkala/job"

	"github.com/gorilla/mux"
	"github.com/phyber/negroni-gzip/gzip"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
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

// HandleKalaStatsRequest is the handler for getting system-level metrics
// /api/v1/stats
func HandleKalaStatsRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := &KalaStatsResponse{
			Stats: job.NewKalaStats(cache),
		}

		w.Header().Set(contentType, jsonContentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Errorf("Error occurred when marshaling response: %s", err)
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
			log.Errorf("Error occurred when marshaling response: %s", err)
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
		log.Errorf("Error occurred when reading r.Body: %s", err)
		return nil, err
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, newJob); err != nil {
		log.Errorf("Error occurred when unmarshaling data: %s", err)
		return nil, err
	}

	return newJob, nil
}

// HandleAddJob takes a job object and unmarshals it to a Job type,
// and then throws the job in the schedulers.
func HandleAddJob(cache job.JobCache, defaultOwner string, disableLocalJobs bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		newJob, err := unmarshalNewJob(r)
		if err != nil {
			errorEncodeJSON(err, http.StatusBadRequest, w)
			return
		}

		if disableLocalJobs && (newJob.JobType == job.LocalJob) {
			errorEncodeJSON(errors.New("local jobs are disabled"), http.StatusForbidden, w)
			return
		}

		if defaultOwner != "" && newJob.Owner == "" {
			newJob.Owner = defaultOwner
		}

		err = newJob.Init(cache)
		if err != nil {
			errStr := fmt.Sprintf("Error occurred when initializing the job: %+v", newJob)
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
			log.Errorf("Error occurred when marshaling response: %s", err)
			return
		}
	}
}

// HandleJobRequest routes requests to /api/v1/job/{id} to either
// handleDeleteJob if its a DELETE or handleGetJob if its a GET request
// or updates the job if its a PUT request.
func HandleJobRequest(cache job.JobCache, disableLocalJobs bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		j, err := cache.Get(id)
		if err != nil {
			log.Errorf("Error occurred when trying to get the job you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if j == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method == "DELETE" {
			err = j.Delete(cache)
			if err != nil {
				errorEncodeJSON(err, http.StatusInternalServerError, w)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		} else if r.Method == "GET" {
			handleGetJob(w, r, j)
		} else if r.Method == "PUT" {
			updatedJob, err := unmarshalNewJob(r)
			if err != nil {
				errorEncodeJSON(err, http.StatusBadRequest, w)
				return
			}

			if disableLocalJobs && (updatedJob.JobType == job.LocalJob) {
				errorEncodeJSON(errors.New("local jobs are disabled"), http.StatusForbidden, w)
				return
			}

			updatedJob.Id = j.Id
			err = updatedJob.Init(cache)

			if err != nil {
				errStr := fmt.Sprintf("Error occurred when initializing the job: %+v", updatedJob)
				log.Errorf(errStr+": %s", err)
				errorEncodeJSON(errors.New(errStr), http.StatusBadRequest, w)
				return
			}

			handleGetJob(w, r, updatedJob)
		}
	}
}

// HandleDeleteAllJobs is the handler for deleting all jobs
// DELETE /api/v1/job/all
func HandleDeleteAllJobs(cache job.JobCache, disableDeleteAll bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if disableDeleteAll {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if err := job.DeleteAll(cache); err != nil {
			errorEncodeJSON(err, http.StatusInternalServerError, w)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

type JobResponse struct {
	Job *job.Job `json:"job"`
}

func handleGetJob(w http.ResponseWriter, _ *http.Request, j *job.Job) {
	resp := &JobResponse{
		Job: j,
	}

	w.Header().Set(contentType, jsonContentType)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("Error occurred when marshaling response: %s", err)
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
			log.Errorf("Error occurred when trying to get the job you requested.")
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
			log.Errorf("Error occurred when trying to get the job you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if j == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := j.Disable(cache); err != nil {
			errorEncodeJSON(err, http.StatusInternalServerError, w)
			return
		}

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
			log.Errorf("Error occurred when trying to get the job you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if j == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := j.Enable(cache); err != nil {
			errorEncodeJSON(err, http.StatusInternalServerError, w)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type ListJobStatsResponse struct {
	JobStats []*job.JobStat `json:"job_stats"`
}

// HandleListJobRunsRequest is the handler listing executions
// /api/v1/job/{id}/executions
func HandleListJobRunsRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		_, err := cache.Get(id)
		if err != nil {
			log.Errorf("Error occurred--the requested job is not in the job cache.")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		allJobs, err := cache.GetAllRuns(id)
		if err != nil {
			log.Errorf("Error occurred when trying to get the job runs you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := &ListJobStatsResponse{
			JobStats: allJobs,
		}

		w.Header().Set(contentType, jsonContentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Errorf("Error occurred when marshaling response: %s", err)
			return
		}
	}
}

// JobRunResponse is for returning a single job execution
type JobRunResponse struct {
	JobRun *job.JobStat `json:"job_run"`
}

// HandleJobRunRequest is the handler for doing things to a single job run
// /api/v1/job/{job_id}/executions/{run_id}/
func HandleJobRunRequest(cache job.JobCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		runID := mux.Vars(r)["id"]

		run, err := cache.GetRun(runID)
		if err != nil {
			log.Errorf("Error occurred when trying to get the job run you requested.")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := &JobRunResponse{
			JobRun: run,
		}

		w.Header().Set(contentType, jsonContentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Errorf("Error occurred when marshaling response: %s", err)
			return
		}
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
func SetupApiRoutes(r *mux.Router, cache job.JobCache, defaultOwner string, disableDeleteAll bool, disableLocalJobs bool) {
	// Route for creating a job
	r.HandleFunc(ApiJobPath, HandleAddJob(cache, defaultOwner, disableLocalJobs)).Methods("POST")
	// Route for deleting all jobs
	r.HandleFunc(ApiJobPath+"all/", HandleDeleteAllJobs(cache, disableDeleteAll)).Methods("DELETE")
	// Route for deleting, editing and getting a job
	r.HandleFunc(ApiJobPath+"{id}/", HandleJobRequest(cache, disableLocalJobs)).Methods("DELETE", "GET", "PUT")
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
	// Route for a single job execution actions
	r.HandleFunc(ApiJobPath+"{job_id}/executions/{id}/", HandleJobRunRequest(cache)).Methods("GET")
	// Route for a single job execution actions
	r.HandleFunc(ApiJobPath+"{id}/executions/", HandleListJobRunsRequest(cache)).Methods("GET")
}

func MakeServer(listenAddr string, cache job.JobCache, defaultOwner string, profile bool, disableDeleteAll bool, disableLocalJobs bool) *http.Server {
	r := mux.NewRouter()
	// Allows for the use for /job as well as /job/
	r.StrictSlash(true)
	SetupApiRoutes(r, cache, defaultOwner, disableDeleteAll, disableLocalJobs)
	r.PathPrefix("/webui/").Handler(http.StripPrefix("/webui/", http.FileServer(http.Dir("./webui/"))))

	if profile {
		runtime.SetMutexProfileFraction(5) //nolint:gomnd
		// Register pprof handlers
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
		r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		r.Handle("/debug/pprof/block", pprof.Handler("block"))
		r.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	}

	n := negroni.New(negroni.NewRecovery(), &middleware.Logger{Logger: log.Logger{}}, gzip.Gzip(gzip.DefaultCompression))
	n.UseHandler(r)

	return &http.Server{
		Addr:    listenAddr,
		Handler: n,
	}
}

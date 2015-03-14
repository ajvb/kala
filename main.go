package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"./utils/iso8601"

	"github.com/222Labs/common/go/logging"
	"github.com/gorilla/mux"
	"github.com/nu7hatch/gouuid"
)

var (
	AllJobs []*Job
	log     = logging.GetLogger("kala")
)

type Job struct {
	Name      string     `json:"name"`
	Id        *uuid.UUID `json:"id"`
	Command   string     `json:"command"`
	Owner     string     `json:"owner"`
	Disabled  bool       `json:"disabled"`
	ChildJobs []*Job     `json:"child_jobs"`

	Schedule     string `json:"schedule"`
	scheduleTime time.Time
	delayDuration time.Duration

	timesToRepeat int64

	Retries        uint `json:"retries"`
	currentRetries uint

	SuccessCount uint      `json:"success_count"`
	LastSuccess  time.Time `json:"last_success"`
	ErrorCount   uint      `json:"error_count"`
	LastError    time.Time `json:"last_error"`

	LastAttemptedRun time.Time `json:"last_attempted_run"`

	jobTimer *time.Timer

	// TODO
	// Epilson time.Duration `json:""`
	// RunAsUser string `json:""`
	// EnvironmentVariables map[string]string `json:""`
}

func (j *Job) Init() error {
	u4, err := uuid.NewV4()
	if err != nil {
		log.Error("Error occured when generating uuid: %s", err)
		return err
	}
	j.Id = u4

	splitTime := strings.Split(j.Schedule, "/")
	if len(splitTime) != 3 {
		return fmt.Errorf("Schedule not formatted correctly. Should look like: R/2014-03-08T20:00:00Z/PT2H")
	}

	// Handle Repeat Amount
	if splitTime[0] == "R" {
		// Repeat forever
		j.timesToRepeat = -1
	} else {
		j.timesToRepeat, err = strconv.ParseInt(strings.Split(splitTime[0], "R")[1], 10, 0)
		if err != nil {
			log.Error("Error converting timesToRepeat to an int: %s", err)
			return err
		}
	}
	log.Debug("timesToRepeat: %d", j.timesToRepeat)

	j.scheduleTime, err = time.Parse(time.RFC3339, splitTime[1])
	if err != nil {
		log.Error("Error converting scheduleTime to a time.Time: %s", err)
		return err
	}
	if (time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())) < 0 {
		return fmt.Errorf("Schedule time has passed.")
	}
	log.Debug("Schedule Time: %s", j.scheduleTime)

	delayDuration, err := iso8601.FromString(splitTime[2])
	if err != nil {
		log.Error("Error converting delayDuration to a time.Duration: %s", err)
		return err
	}
	j.delayDuration = delayDuration.ToDuration()
	log.Debug("Delay Duration: %s", j.delayDuration)

	j.StartWaiting()

	return nil
}

func (j *Job) StartWaiting() {
	waitDuration := time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())
	log.Debug("Wait Duration initial: %s", waitDuration)
	if waitDuration < 0 {
		waitDuration = j.delayDuration
	}
	log.Info("Job Scheduled to run in: %s", waitDuration)
	j.jobTimer = time.AfterFunc(waitDuration, j.Run)
}

func (j *Job) Disable() {
	//hasBeenStopped := j.jobTimer.Stop()
	_ = j.jobTimer.Stop()
	j.Disabled = true
}

func (j *Job) Run() {
	log.Info("Job %s running", j.Name)

	// Schedule next run
	if j.timesToRepeat != 0 {
		j.timesToRepeat -= 1
		go j.StartWaiting()
	}

	j.LastAttemptedRun = time.Now()

	// TODO - Make thread safe
	// Init retries
	if j.currentRetries == 0 && j.Retries != 0 {
		j.currentRetries = j.Retries
	}

	// Execute command
	args := strings.Split(j.Command, " ")
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()
	if err != nil {
		log.Error("Run Command got an Error: %s", err)
		j.ErrorCount += 1
		j.LastError = time.Now()
		// Handle retrying
		if j.currentRetries != 0 {
			j.currentRetries -= 0
			j.Run()
		}
		return
	}

	j.SuccessCount += 1
	j.LastSuccess = time.Now()

	// Run Child Jobs
	if len(j.ChildJobs) != 0 {
		for _, job := range j.ChildJobs {
			go job.Run()
		}
	}
}

func main() {
	r := mux.NewRouter()

	apiUrlPrefix := "/api/v1/job/"
	// CRUD
	r.HandleFunc(apiUrlPrefix, handleAddJob).Methods("POST")
	r.HandleFunc(apiUrlPrefix+"{id}", handleJobRequest)
	r.HandleFunc(apiUrlPrefix+"list", handleListJobs).Methods("GET")
	// TODO
	// Manually start a job
	// Adding a dependent job

	log.Info("Starting server...")

	log.Fatal(http.ListenAndServe(":8000", r))
}

type ListJobsResponse struct {
	Jobs []*Job `json:"jobs"`
}

func handleListJobs(w http.ResponseWriter, r *http.Request) {
	resp := &ListJobsResponse{
		Jobs: AllJobs,
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

func handleAddJob(w http.ResponseWriter, r *http.Request) {
	newJob := &Job{}

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Error("Error occured when reading r.Body: %s", err)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, newJob); err != nil {
		// TODO return 400
		//w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		log.Error("Error occured when unmarshalling data: %s", err)
		return
	}

	// TODO
	// 1. Verify there is a scheduled time and a command
	// 2. Verify that "protected" fields were not touched.

	err = newJob.Init()
	if err != nil {
		// TODO return 400
		//w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		log.Error("Error occured when initializing the job: %s", err)
		return
	}
	AllJobs = append(AllJobs, newJob)

	resp := &AddJobResponse{
		Id: newJob.Id.String(),
	}

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error("Error occured when marshalling response: %s", err)
		return
	}
}

func handleJobRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if r.Method == "DELETE" {
		handleDeleteJob(w, r, id)
	} else if r.Method == "PUT" {
		//handleUpdateJob(w, r, id)
	} else if r.Method == "GET" {
		//handleGetJob(w, r, id)
	}

	// TODO - Method not allow

}

func handleDeleteJob(w http.ResponseWriter, r *http.Request, id string) {
	// Find and delete job
	for i, job := range AllJobs {
		if job.Id.String() == id {
			// Stop Job Timer
			job.Disable()

			// Remove from AllJobs
			s := AllJobs
			log.Info("Deleting job: %s", job.Id)
			s = append(s[:i], s[i+1:]...)
			AllJobs = s
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

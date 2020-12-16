package job

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nextiva/nextkala/utils/iso8601"
	"github.com/mixer/clock"
	uuid "github.com/nu7hatch/gouuid"

	log "github.com/sirupsen/logrus"
)

var (
	RFC3339WithoutTimezone = "2006-01-02T15:04:05"

	ErrInvalidJob       = errors.New("Invalid Local Job. Job's must contain a Name and a Command field")
	ErrInvalidRemoteJob = errors.New("Invalid Remote Job. Job's must contain a Name and a url field")
	ErrInvalidJobType   = errors.New("Invalid Job type. Types supported: 0 for local and 1 for remote")
)

type Job struct {
	Name string `json:"name"`
	Id   string `json:"id"`

	// Command to run
	// e.g. "bash /path/to/my/script.sh"
	Command string `json:"command"`

	// Email of the owner of this job
	// e.g. "admin@example.com"
	Owner string `json:"owner"`

	// Is this job disabled?
	Disabled bool `json:"disabled"`

	// Jobs that are dependent upon this one will be run after this job runs.
	DependentJobs []string `json:"dependent_jobs"`

	// List of ids of jobs that this job is dependent upon.
	ParentJobs []string `json:"parent_jobs"`

	// Job that gets run after all retries have failed consecutively
	OnFailureJob string `json:"on_failure_job"`

	// ISO 8601 String
	// e.g. "R/2014-03-08T20:00:00.000Z/PT2H"
	Schedule     string `json:"schedule"`
	scheduleTime time.Time
	// ISO 8601 Duration struct, used for scheduling
	// job after each run.
	delayDuration *iso8601.Duration

	// Number of times to schedule this job after the
	// first run.
	timesToRepeat int64

	// Number of times to retry on failed attempt for each run.
	Retries uint `json:"retries"`

	// Duration in which it is safe to retry the Job.
	Epsilon         string `json:"epsilon"`
	epsilonDuration *iso8601.Duration

	jobTimer  clock.Timer
	NextRunAt time.Time `json:"next_run_at"`

	// Templating delimiters, the left & right separated by space,
	// for example `{{ }}` or `${ }`.
	//
	// If this field is non-empty, then each time this
	// job is executed, Kala will template its main
	// content as a Go Template with the job itself as data.
	//
	// The Command is templated for local jobs,
	// and Url and Body in RemoteProperties.
	TemplateDelimiters string

	// The clock for this job; used to mock time during tests.
	clk Clock

	// If the job is disabled (or the system inoperative) and we pass
	// the scheduled run point, when the job becomes active again,
	// normally the job will run immediately.
	// With this setting on, it will not run immediately, but will wait
	// until the next scheduled run time comes along.
	ResumeAtNextScheduledTime bool `json:"resume_at_next_scheduled_time"`

	// Meta data about successful and failed runs.
	Metadata Metadata `json:"metadata"`

	// Type of the job
	JobType jobType `json:"type"`

	// Custom properties for the remote job type
	RemoteProperties RemoteProperties `json:"remote_properties"`

	// Collection of Job Stats
	// Stats []*JobStat `json:"stats"`

	lock sync.RWMutex

	// Says if a job has been executed right numbers of time
	// and should not been executed again in the future
	IsDone bool `json:"is_done"`

	// The job will send on this channel when it's done running; used for tests.
	// Note that if the job should be rescheduled, it will send on this channel
	// when it's done rescheduling rather than when the job is done running.
	// That's most useful for testing the scheduling aspect of jobs.
	ranChan chan struct{}

	// Used for testing schedules.
	succeedInstantly bool
}

type jobType int

const (
	LocalJob jobType = iota
	RemoteJob
)

// RemoteProperties Custom properties for the remote job type
type RemoteProperties struct {
	Url    string `json:"url"`
	Method string `json:"method"`

	// A body to attach to the http request
	Body string `json:"body"`

	// A list of headers to add to http request (e.g. [{"key": "charset", "value": "UTF-8"}])
	Headers http.Header `json:"headers"`

	// A timeout property for the http request in seconds
	Timeout int `json:"timeout"`

	// A list of expected response codes (e.g. [200, 201])
	ExpectedResponseCodes []int `json:"expected_response_codes"`
}

type Metadata struct {
	SuccessCount         uint      `json:"success_count"`
	LastSuccess          time.Time `json:"last_success"`
	ErrorCount           uint      `json:"error_count"`
	LastError            time.Time `json:"last_error"`
	LastAttemptedRun     time.Time `json:"last_attempted_run"`
	NumberOfFinishedRuns uint      `json:"number_of_finished_runs"`
}

// Bytes returns the byte representation of the Job.
func (j Job) Bytes() ([]byte, error) { //nolint:govet // Copying the lock is okay here
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	err := enc.Encode(j) //nolint:govet // Copying the lock is okay here
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// NewFromBytes returns a Job instance from a byte representation.
func NewFromBytes(b []byte) (*Job, error) {
	j := &Job{}

	buf := bytes.NewBuffer(b)
	err := gob.NewDecoder(buf).Decode(j)
	if err != nil {
		return nil, err
	}

	return j, nil
}

// Init fills in the protected fields and parses the iso8601 notation.
// It also adds the job to the Cache
func (j *Job) Init(cache JobCache) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	// use the cache's clock if available (useful for tests)
	if clker, ok := cache.(Clocker); ok {
		if clker.TimeSet() {
			j.clk.SetClock(clker.Time())
		}
	}

	// validate job type and params
	err := j.validation()
	if err != nil {
		return err
	}

	/// set the id if not provided.
	j.setID()

	// Add Job to the cache.
	j.lock.Unlock()
	err = cache.Set(j)
	j.lock.Lock()
	if err != nil {
		return err
	}

	if len(j.ParentJobs) != 0 {
		// Add new job to parent jobs
		for _, p := range j.ParentJobs {
			parentJob, err := cache.Get(p)
			if err != nil {
				return err
			}
			parentJob.DependentJobs = append(parentJob.DependentJobs, j.Id)
		}

		return nil
	}

	// TODO: Delete from cache after running.
	if j.Schedule == "" {
		// If schedule is empty, its a one-off job.
		go j.Run(cache)
		return nil
	}

	j.lock.Unlock()
	err = j.InitDelayDuration(true)
	j.lock.Lock()
	if err != nil {
		j.lock.Unlock()
		_ = cache.Delete(j.Id)
		j.lock.Lock()
		return err
	}

	j.lock.Unlock()
	j.StartWaiting(cache, false)

	j.lock.Lock()

	return nil
}

// InitDelayDuration is used to parsed the iso8601 Schedule notation into its relevant fields in the Job struct.
// If checkTime is true, then it will return an error if the Scheduled time has passed.
func (j *Job) InitDelayDuration(checkTime bool) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.Schedule == "" {
		return nil
	}

	var err error
	splitTime := strings.Split(j.Schedule, "/")
	if len(splitTime) != 3 { //nolint:gomnd
		return fmt.Errorf(
			"Schedule not formatted correctly. Should look like: R/2014-03-08T20:00:00Z/PT2H",
		)
	}

	// Handle Repeat Amount
	if splitTime[0] == "R" {
		// Repeat forever
		j.timesToRepeat = -1
	} else {
		j.timesToRepeat, err = strconv.ParseInt(strings.Split(splitTime[0], "R")[1], 10, 0)
		if err != nil {
			log.Errorf("Error converting timesToRepeat to an int: %s", err)
			return err
		}
	}

	j.scheduleTime, err = time.Parse(time.RFC3339, splitTime[1])
	if err != nil {
		j.scheduleTime, err = time.Parse(RFC3339WithoutTimezone, splitTime[1])
		if err != nil {
			log.Errorf("Error converting scheduleTime to a time.Time: %s", err)
			return err
		}
	}
	if checkTime {
		diff := j.scheduleTime.Sub(j.clk.Time().Now())
		if diff < 0 {
			return fmt.Errorf("Job %s:%s cannot be scheduled %s ago", j.Name, j.Id, diff.String())
		}
	}
	log.Debugf("Job %s:%s scheduled", j.Name, j.Id)
	log.Debugf("Starting %s will repeat for %d", j.scheduleTime, j.timesToRepeat)

	if j.timesToRepeat != 0 {
		j.delayDuration, err = iso8601.FromString(splitTime[2])
		if err != nil {
			log.Errorf("Error converting delayDuration to a iso8601.Duration: %s", err)
			return err
		}
		log.Debugf("Delay duration is %s", j.delayDuration.RelativeTo(j.clk.Time().Now()))
	}

	if j.Epsilon != "" {
		j.epsilonDuration, err = iso8601.FromString(j.Epsilon)
		if err != nil {
			log.Errorf("Error converting j.Epsilon to iso8601.Duration: %s", err)
			return err
		}
	}
	return nil
}

// StartWaiting begins a timer for when it should execute the Jobs .Run() method.
func (j *Job) StartWaiting(cache JobCache, justRan bool) {
	waitDuration := j.GetWaitDuration()

	j.lock.Lock()
	defer j.lock.Unlock()

	log.Infof("Job %s:%s repeating in %s", j.Name, j.Id, waitDuration)

	j.NextRunAt = j.clk.Time().Now().Add(waitDuration)

	jobRun := func() { j.Run(cache) }
	j.jobTimer = j.clk.Time().AfterFunc(waitDuration, jobRun)

	if justRan && j.ranChan != nil {
		j.ranChan <- struct{}{}
	}
}

func (j *Job) GetWaitDuration() time.Duration {
	j.lock.RLock()
	defer j.lock.RUnlock()

	waitDuration := time.Duration(j.scheduleTime.UnixNano() - j.clk.Time().Now().UnixNano())

	if waitDuration >= 0 {
		return waitDuration
	}

	if j.timesToRepeat == 0 {
		return 0
	}

	if j.ResumeAtNextScheduledTime {

		// In cases where the scheduled point is very long ago,
		// and the delayDuration interval is very small,
		// it can take an extremely long time to compute the next scheduled run time.
		// One potential case (programmer error) is having the scheduleTime be zero.
		// (This would be due to not calling InitDelayDuration on a job.)
		// For now, we spot this and handle it as a special case.
		if j.scheduleTime.IsZero() {
			return 0
		}

		newRunPoint := j.scheduleTime
		for newRunPoint.Before(j.clk.Time().Now()) {
			newRunPoint = j.delayDuration.Add(newRunPoint)
		}

		return newRunPoint.Sub(j.clk.Time().Now())
	}

	if j.Metadata.LastAttemptedRun.IsZero() {
		waitDuration = j.delayDuration.RelativeTo(j.clk.Time().Now())
	} else {
		lastRun := j.Metadata.LastAttemptedRun
		// Needs to be recalculated each time because of Months.
		lastRun = j.delayDuration.Add(lastRun)
		waitDuration = lastRun.Sub(j.clk.Time().Now())
	}

	return waitDuration
}

// Disable stops the job from running by stopping its jobTimer. It also sets Job.Disabled to true,
// which is reflected in the UI.
func (j *Job) Disable(cache JobCache) error {
	return cache.Disable(j)
}

func (j *Job) Enable(cache JobCache) error {
	return cache.Enable(j)
}

// DeleteFromParentJobs goes through and deletes the current job from any parent jobs.
func (j *Job) DeleteFromParentJobs(cache JobCache) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	for _, p := range j.ParentJobs {
		parentJob, err := cache.Get(p)

		if err != nil {
			return err
		}

		parentJob.lock.Lock()

		ndx := 0
		for i, id := range parentJob.DependentJobs {
			if id == j.Id {
				ndx = i
				break
			}
		}
		parentJob.DependentJobs = append(
			parentJob.DependentJobs[:ndx], parentJob.DependentJobs[ndx+1:]...,
		)
		err = cache.Set(parentJob)
		if err != nil {
			return err
		}
		parentJob.lock.Unlock()
	}

	return nil
}

// DeleteFromDependentJobs
func (j *Job) DeleteFromDependentJobs(cache JobCache) error {
	j.lock.RLock()
	defer j.lock.RUnlock()

	for _, id := range j.DependentJobs {
		childJob, err := cache.Get(id)
		if err != nil {
			return err
		}

		// If there are no other parent jobs, delete this job.
		if len(childJob.ParentJobs) == 1 {
			log.Infof("Deleting child %s", id)
			_ = cache.Delete(childJob.Id)
			continue
		}

		childJob.lock.Lock()

		ndx := 0
		for i, id := range childJob.ParentJobs {
			if id == j.Id {
				ndx = i
				break
			}
		}
		childJob.ParentJobs = append(
			childJob.ParentJobs[:ndx], childJob.ParentJobs[ndx+1:]...,
		)

		childJob.lock.Unlock()

	}

	return nil
}

// Runs the on failure job, if it exists. Does not lock the parent job - it is up to you to do this
// however you want
func (j *Job) RunOnFailureJob(cache JobCache) {
	if j.OnFailureJob != "" {
		onFailureJob, cacheErr := cache.Get(j.OnFailureJob)
		if cacheErr == ErrJobDoesntExist {
			log.Errorf("Error retrieving dependent job with id of %s", j.OnFailureJob)
		} else {
			onFailureJob.Run(cache)
		}
	}
}

func (j *Job) Run(cache JobCache) {
	j.lock.RLock()
	jobRunner := &JobRunner{job: j, meta: j.Metadata}
	j.lock.RUnlock()

	newStat, newMeta, err := jobRunner.Run(cache)
	if err != nil {
		j.lock.RLock()
		j.RunOnFailureJob(cache)
		j.lock.RUnlock()
	}

	j.lock.Lock()
	j.Metadata = newMeta

	if newStat != nil {
		log.Infof("Saving stat run %+v", newStat)
		cache.SaveRun(newStat)
	}

	// Kinda annoying and inefficient that it needs to be done this way.
	// Some refactoring is probably in order.
	if got, err := cache.Get(j.Id); got == nil || err != nil {
		log.Warnf("Job %s with id %s ran, but seems to have been deleted from cache; results won't be persisted.", j.Name, j.Id)
	} else {
		j.lock.Unlock()
		j.lock.RLock()
		if err := cache.Set(j); err != nil {
			log.Errorf("Job %s with id %s ran, but the results couldn't be persisted: %v", j.Name, j.Id, err)
		}
		j.lock.RUnlock()
		j.lock.Lock()
	}

	if j.ShouldStartWaiting() {
		go j.StartWaiting(cache, true)
	} else {
		j.IsDone = true

		if j.ranChan != nil {
			j.ranChan <- struct{}{}
		}
	}

	j.lock.Unlock()
}

func (j *Job) StopTimer() {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.jobTimer != nil {
		j.jobTimer.Stop()
	}
}

func (j *Job) RunCmd() (string, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	jobRunner := &JobRunner{job: j}
	return jobRunner.runCmd()
}

func (j *Job) hasFixedRepetitions() bool {
	return j.timesToRepeat != -1
}

func (j *Job) ShouldStartWaiting() bool {
	if j.Disabled {
		return false
	}

	if j.hasFixedRepetitions() && int(j.timesToRepeat) < int(j.Metadata.NumberOfFinishedRuns) {
		return false
	}
	return true
}

func (j *Job) validation() error {
	var err error
	switch {
	case j.JobType == LocalJob && (j.Name == "" || j.Command == ""):
		err = ErrInvalidJob
	case j.JobType == RemoteJob && (j.Name == "" || j.RemoteProperties.Url == ""):
		err = ErrInvalidRemoteJob
	case j.JobType != LocalJob && j.JobType != RemoteJob:
		err = ErrInvalidJobType
	default:
		return nil
	}
	log.Errorf(err.Error())
	return err
}

func (j *Job) setID() error {
	log.Debugf("setting id for job with current id: %s", j.Id)

	if len(j.Id) > 0 {
		return nil
	}

	u4, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Error occurred when generating uuid: %s", err)
		return err
	}

	j.Id = u4.String()

	return nil
}

func (j *Job) SetClock(clk clock.Clock) {
	j.clk.SetClock(clk)
}

// Type alias for the recursive call
type RJob Job

// need this to fix race condition
func (j *Job) MarshalJSON() ([]byte, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()
	return json.Marshal(RJob(*j)) //nolint:govet // Copying the lock is okay here
}

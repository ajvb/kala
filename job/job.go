package job

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gwoo/kala/utils/iso8601"

	log "github.com/Sirupsen/logrus"
	"github.com/nu7hatch/gouuid"
)

var (
	// nano seconds allowed between current time and when job is supposed to start
	validOffset            = time.Minute
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

	jobTimer  *time.Timer
	NextRunAt time.Time `json:"next_run_at"`

	// Meta data about successful and failed runs.
	Metadata Metadata `json:"metadata"`

	// Type of the job
	JobType jobType `json:"type"`

	// Custom properties for the remote job type
	RemoteProperties RemoteProperties `json:"remote_properties"`

	// Collection of Job Stats
	Stats []*JobStat `json:"stats"`

	lock sync.RWMutex

	// Says if a job has been executed right numbers of time
	// and should not been executed again in the future
	IsDone bool `json:"is_done"`
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
	SuccessCount     uint      `json:"success_count"`
	LastSuccess      time.Time `json:"last_success"`
	ErrorCount       uint      `json:"error_count"`
	LastError        time.Time `json:"last_error"`
	LastAttemptedRun time.Time `json:"last_attempted_run"`
}

// Bytes returns the byte representation of the Job.
func (j Job) Bytes() ([]byte, error) {
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	err := enc.Encode(j)
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

	//validate job type and params
	err := j.validation()
	if err != nil {
		return err
	}

	u4, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Error occured when generating uuid: %s", err)
		return err
	}
	j.Id = u4.String()

	// Add Job to the cache.
	err = cache.Set(j)
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
		cache.Delete(j.Id)
		j.lock.Lock()
		return err
	}

	j.lock.Unlock()
	j.StartWaiting(cache)

	j.lock.Lock()

	return nil
}

// InitDelayDuration is used to parsed the iso8601 Schedule notation into its relevent fields in the Job struct.
// If checkTime is true, then it will return an error if the Scheduled time has passed.
func (j *Job) InitDelayDuration(checkTime bool) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.Schedule == "" {
		return nil
	}

	var err error
	splitTime := strings.Split(j.Schedule, "/")
	if len(splitTime) != 3 {
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
		duration := time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())
		if duration < 0 {
			return fmt.Errorf("Job %s:%s cannot be scheduled %s ago", j.Name, j.Id, duration.String())
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
		log.Debugf("Delay duration is %s", j.delayDuration.ToDuration())
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
func (j *Job) StartWaiting(cache JobCache) {
	waitDuration := j.GetWaitDuration()

	j.lock.Lock()
	defer j.lock.Unlock()

	log.Infof("Job %s:%s repeating in %s", j.Name, j.Id, waitDuration)

	j.NextRunAt = time.Now().Add(waitDuration)

	jobRun := func() { j.Run(cache) }
	j.jobTimer = time.AfterFunc(waitDuration, jobRun)
}

func (j *Job) GetWaitDuration() time.Duration {
	j.lock.RLock()
	defer j.lock.RUnlock()

	waitDuration := time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())

	if waitDuration < 0 {
		if j.timesToRepeat == 0 {
			return 0
		}

		if j.Metadata.LastAttemptedRun.IsZero() {
			waitDuration = j.delayDuration.ToDuration()
		} else {
			lastRun := j.Metadata.LastAttemptedRun
			// Needs to be recalculated each time because of Months.
			lastRun = lastRun.Add(j.delayDuration.ToDuration())
			waitDuration = lastRun.Sub(time.Now())
		}
	}

	return waitDuration
}

// Disable stops the job from running by stopping its jobTimer. It also sets Job.Disabled to true,
// which is reflected in the UI.
func (j *Job) Disable() {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.jobTimer != nil {
		j.jobTimer.Stop()
	}
	j.Disabled = true
}

func (j *Job) Enable(cache JobCache) {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.jobTimer != nil && j.Disabled {
		go j.StartWaiting(cache)
	}
	j.Disabled = false
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
		cache.Set(parentJob)
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
			cache.Delete(childJob.Id)
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

func (j *Job) Run(cache JobCache) {
	// Schedule next run
	j.lock.RLock()
	jobRunner := &JobRunner{job: j, meta: j.Metadata}
	j.lock.RUnlock()
	newStat, newMeta, err := jobRunner.Run(cache)
	if err != nil {
		log.Errorf("Error running job: %s", err)
	}

	j.lock.Lock()
	j.Metadata = newMeta
	if newStat != nil {
		j.Stats = append(j.Stats, newStat)
	}

	if j.ShouldStartWaiting() {
		go j.StartWaiting(cache)
	} else {
		j.IsDone = true
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

func (j *Job) RunCmd() error {
	j.lock.RLock()
	defer j.lock.RUnlock()

	jobRunner := &JobRunner{job: j}
	return jobRunner.runCmd()
}

func (j *Job) hasFixedRepetitions() bool {
	if j.timesToRepeat != -1 {
		return true
	}
	return false
}

func (j *Job) ShouldStartWaiting() bool {
	if j.Disabled {
		return false
	}

	if j.hasFixedRepetitions() && int(j.timesToRepeat) < len(j.Stats) {
		return false
	}
	return true
}

func (j *Job) validation() error {
	var err error
	if j.JobType == LocalJob && (j.Name == "" || j.Command == "") {
		err = ErrInvalidJob
	} else if j.JobType == RemoteJob && (j.Name == "" || j.RemoteProperties.Url == "") {
		err = ErrInvalidRemoteJob
	} else if j.JobType != LocalJob && j.JobType != RemoteJob {
		err = ErrInvalidJobType
	} else {
		return nil
	}
	log.Errorf(err.Error())
	return err
}

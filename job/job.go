package job

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ajvb/kala/utils/iso8601"
	"github.com/ajvb/kala/utils/logging"

	"github.com/mattn/go-shellwords"
	"github.com/nu7hatch/gouuid"
)

var (
	log = logging.GetLogger("kala.job")

	shParser = shellwords.NewParser()
)

func init() {
	shParser.ParseEnv = true
	shParser.ParseBacktick = true
}

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
	Retries        uint `json:"retries"`
	currentRetries uint

	// Duration in which it is safe to retry the Job.
	Epsilon         string `json:"epsilon"`
	epsilonDuration *iso8601.Duration

	// Meta data about successful and failed runs.
	SuccessCount     uint      `json:"success_count"`
	LastSuccess      time.Time `json:"last_success"`
	ErrorCount       uint      `json:"error_count"`
	LastError        time.Time `json:"last_error"`
	LastAttemptedRun time.Time `json:"last_attempted_run"`

	jobTimer  *time.Timer
	NextRunAt time.Time `json:"next_run_at"`

	currentStat *JobStat
	Stats       []*JobStat `json:"-"`

	lock sync.RWMutex

	numberOfAttempts uint
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

	u4, err := uuid.NewV4()
	if err != nil {
		log.Error("Error occured when generating uuid: %s", err)
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

	if j.Schedule == "" {
		// If schedule is empty, its a one-off job.
		go j.Run(cache)
		return nil
	}

	j.lock.Unlock()
	err = j.InitDelayDuration(true)
	j.lock.Lock()
	if err != nil {
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
	if checkTime {
		if (time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())) < 0 {
			return fmt.Errorf("Schedule time has passed on Job with id of %s", j.Id)
		}
	}
	log.Debug("Schedule Time: %s", j.scheduleTime)

	j.delayDuration, err = iso8601.FromString(splitTime[2])
	if err != nil {
		log.Error("Error converting delayDuration to a iso8601.Duration: %s", err)
		return err
	}
	log.Debug("Delay Duration: %s", j.delayDuration.ToDuration())

	if j.Epsilon != "" {
		j.epsilonDuration, err = iso8601.FromString(j.Epsilon)
		if err != nil {
			log.Error("Error converting j.Epsilon to iso8601.Duration: %s", err)
			return err
		}
	}

	return nil
}

// StartWaiting begins a timer for when it should execute the Jobs .Run() method.
func (j *Job) StartWaiting(cache JobCache) {
	j.lock.Lock()
	defer j.lock.Unlock()

	waitDuration := time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())
	log.Debug("Wait Duration initial: %s", waitDuration)
	if waitDuration < 0 {
		// Needs to be recalculated each time because of Months.
		if j.LastAttemptedRun.IsZero() {
			waitDuration = j.delayDuration.ToDuration()
		} else {
			lastRun := j.LastAttemptedRun
			lastRun = lastRun.Add(j.delayDuration.ToDuration())
			waitDuration = lastRun.Sub(time.Now())
		}
	}
	log.Info("Job Scheduled to run in: %s", waitDuration)
	j.NextRunAt = time.Now().Add(waitDuration)
	jobRun := func() { j.Run(cache) }
	j.jobTimer = time.AfterFunc(waitDuration, jobRun)
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

// DeleteFromParentJobs goes through and deletes the current job from any parent jobs.
func (j *Job) DeleteFromParentJobs(cache JobCache) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	if len(j.ParentJobs) != 0 {
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

			parentJob.lock.Unlock()
		}
	}
	return nil
}

// DeleteFromDependentJobs
func (j *Job) DeleteFromDependentJobs(cache JobCache) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	if len(j.DependentJobs) != 0 {
		for _, id := range j.DependentJobs {
			childJob, err := cache.Get(id)
			if err != nil {
				return err
			}

			// If there are no other parent jobs, delete this job.
			if len(childJob.ParentJobs) == 1 {
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
	}
	return nil
}

// Run executes the Job's command, collects metadata around the success
// or failure of the Job's execution, and schedules the next run.
func (j *Job) Run(cache JobCache) {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.Disabled {
		log.Info("Job %s tried to run, but exited early because its disabled.", j.Name)
		return
	}

	log.Info("Job %s running", j.Name)

	j.runSetup(cache)

	for {
		err := j.runCmd()
		if err != nil {
			// Log Error in Metadata
			// TODO - Error Reporting, email error
			log.Error("Run Command got an Error: %s", err)

			// Handle retrying
			if j.shouldRetry() {
				j.currentRetries--
				continue
			}

			j.ErrorCount++
			j.LastError = time.Now()

			// If it doesn't retry, cleanup and exit.
			j.runCleanup()
			return
		} else {
			break
		}
	}

	log.Info("%s was successful!", j.Name)
	j.SuccessCount++
	j.LastSuccess = time.Now()

	// Get Execution Duration
	j.currentStat.ExecutionDuration = time.Duration(
		j.LastSuccess.UnixNano() - j.currentStat.RanAt.UnixNano(),
	)
	j.currentStat.Success = true
	j.currentStat.NumberOfRetries = j.Retries - j.currentRetries

	// Run Dependent Jobs
	if len(j.DependentJobs) != 0 {
		for _, id := range j.DependentJobs {
			newJob, err := cache.Get(id)
			if err != nil {
				log.Error("Error retrieving dependent job with id of %s", id)
			} else {
				newJob.Run(cache)
			}
		}
	}

	j.runCleanup()
	return
}

func (j *Job) runCmd() error {
	j.numberOfAttempts++

	// Execute command
	args, err := shParser.Parse(j.Command)
	if err != nil {
		return err
	}
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Run()
}

func (j *Job) shouldRetry() bool {
	// Check number of retries left
	if j.currentRetries == 0 {
		return false
	}

	// Check Epsilon
	if j.Epsilon != "" {
		if j.epsilonDuration.ToDuration() != 0 {
			timeSinceStart := time.Now().Sub(j.NextRunAt)
			timeLeftToRetry := j.epsilonDuration.ToDuration() - timeSinceStart
			if timeLeftToRetry < 0 {
				return false
			}
		}
	}

	return true
}

func (j *Job) runSetup(cache JobCache) {
	// Setup Job Stat
	j.currentStat = NewJobStat(j.Id)

	// Schedule next run
	if j.timesToRepeat != 0 {
		j.timesToRepeat--
		go j.StartWaiting(cache)
	}

	j.LastAttemptedRun = time.Now()

	// Init retries
	j.currentRetries = j.Retries
}

func (j *Job) runCleanup() {
	j.Stats = append(j.Stats, j.currentStat)
	j.currentStat = nil
	j.currentRetries = 0
}

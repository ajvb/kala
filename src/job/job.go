package job

import (
	"fmt"
	"os/exec"
	//"os/user"
	"strconv"
	"strings"
	//"syscall"
	"time"

	"../utils/iso8601"

	"github.com/222Labs/common/go/logging"
	"github.com/nu7hatch/gouuid"
)

var (
	AllJobs = make(map[string]*Job)

	log = logging.GetLogger("kala")
)

func init() {
	// Prep cache
	allJobs, err := GetAllJobs()
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range allJobs {
		AllJobs[v.Id] = v
	}
	// Occasionally, save items in cache to db.
	go SaveAllJobsEvery(SaveAllJobsWaitTime)
}

}

type Job struct {
	Name string `json:"name"`
	Id   string `json:"id"`

	// Command to run
	// e.g. "bash /path/to/my/script.sh"
	Command string `json:"command"`
	// Run command as a different user?
	//TODO
	//RunAsUser string `json:""`
	//user *user.User

	// Email of the owner of this job
	// e.g. "admin@example.com"
	Owner string `json:"owner"`

	// Is this job disabled?
	Disabled bool `json:"disabled"`

	// Jobs that are dependent upon this one.
	// Will be run after this job runs.
	DependentJobs []string `json:"dependent_jobs"`
	ParentJobs    []string `json:"parent_jobs"`

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
	Retries         uint `json:"retries"`
	currentRetries  uint
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

	// TODO
	// EnvironmentVariables map[string]string `json:""`
}

// Init() fills in the protected fields and parses the iso8601 notation.
func (j *Job) Init() error {
	u4, err := uuid.NewV4()
	if err != nil {
		log.Error("Error occured when generating uuid: %s", err)
		return err
	}
	j.Id = u4.String()

	if len(j.ParentJobs) != 0 {
		// Add new job to parent jobs
		for _, p := range j.ParentJobs {
			AllJobs[p].DependentJobs = append(AllJobs[p].DependentJobs, j.Id)
		}
		return nil
	}

	if j.Schedule == "" {
		// If schedule is empty, its a one-off job.
		go j.Run()
		return nil
	}

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

	//TODO
	/*
		if j.RunAsUser != "" {
			j.user, err = user.Lookup(j.RunAsUser)
			if err != nil {
				log.Error("Error looking up user %s", j.RunAsUser)
				return err
			}
		}
	*/

	j.StartWaiting()

	return nil
}

// StartWaiting begins a timer for when it should execute the Jobs .Run() method.
func (j *Job) StartWaiting() {
	waitDuration := time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())
	log.Debug("Wait Duration initial: %s", waitDuration)
	if waitDuration < 0 {
		// Needs to be recalculated each time because of Months.
		waitDuration = j.delayDuration.ToDuration()
	}
	log.Info("Job Scheduled to run in: %s", waitDuration)
	j.NextRunAt = time.Now().Add(waitDuration)
	j.jobTimer = time.AfterFunc(waitDuration, j.Run)
}

func (j *Job) Disable() {
	// TODO - revisit error handling
	//hasBeenStopped := j.jobTimer.Stop()
	_ = j.jobTimer.Stop()
	j.Disabled = true
}

// Run() executes the Job's command, collects metadata around the success
// or failure of the Job's execution, and schedules the next run.
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

	//TODO
	/*
		if j.RunAsUser != "" {
			uid, err := strconv.Atoi(j.user.Uid)
			if err != nil {
				//TODO
				//return err
				return
			}
			gid, err := strconv.Atoi(j.user.Gid)
			if err != nil {
				//TODO
				//return err
				return
			}
			cmd.SysProcAttr = &syscall.SysProcAttr{}
			cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
		}
	*/
	err := cmd.Run()
	if err != nil {
		log.Error("Run Command got an Error: %s", err)
		j.ErrorCount += 1
		j.LastError = time.Now()
		// Handle retrying
		if j.currentRetries != 0 {
			if j.Epsilon != "" {
				if j.epsilonDuration.ToDuration() == 0 {
					timeLeftToRetry := time.Duration(j.epsilonDuration.ToDuration()) - time.Duration(time.Now().UnixNano()-j.NextRunAt.UnixNano())
					if timeLeftToRetry < 0 {
						// TODO - Make thread safe
						// Reset retries and exit.
						j.currentRetries = 0
						return
					}
				}
			}
			j.currentRetries -= 0
			j.Run()
		}
		return
	}

	log.Info("%s was successful!", j.Name)
	j.SuccessCount += 1
	j.LastSuccess = time.Now()

	// Run Dependent Jobs
	if len(j.DependentJobs) != 0 {
		for _, id := range j.DependentJobs {
			go AllJobs[id].Run()
		}
	}
}

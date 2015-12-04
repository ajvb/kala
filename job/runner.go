package job

import (
	"errors"
	"os/exec"
	"time"

	log "github.com/Sirupsen/logrus"
)

type JobRunner struct {
	job  *Job
	meta Metadata

	numberOfAttempts uint
	currentRetries   uint
	currentStat      *JobStat
}

var (
	ErrJobDisabled = errors.New("Job cannot run, as it is disabled")
	ErrCmdIsEmpty  = errors.New("Job Command is empity.")
)

// Run executes the Job's command, collects metadata around the success
// or failure of the Job's execution, and schedules the next run.
func (j *JobRunner) Run(cache JobCache) (*JobStat, Metadata, error) {
	j.job.lock.RLock()
	defer j.job.lock.RUnlock()

	j.meta.LastAttemptedRun = time.Now()

	if j.job.Disabled {
		log.Infof("Job %s tried to run, but exited early because its disabled.", j.job.Name)
		return nil, j.meta, ErrJobDisabled
	}

	log.Infof("Job %s running", j.job.Name)

	j.runSetup()

	for {
		err := j.runCmd()
		if err != nil {
			// Log Error in Metadata
			// TODO - Error Reporting, email error
			log.Errorf("Run Command got an Error: %s", err)

			j.meta.ErrorCount++
			j.meta.LastError = time.Now()

			// Handle retrying
			if j.shouldRetry() {
				j.currentRetries--
				continue
			}

			j.collectStats(false)

			// TODO: Wrap error into something better.
			return j.currentStat, j.meta, err
		} else {
			break
		}
	}

	log.Infof("%s was successful!", j.job.Name)
	j.meta.SuccessCount++
	j.meta.LastSuccess = time.Now()

	j.collectStats(true)

	// Run Dependent Jobs
	if len(j.job.DependentJobs) != 0 {
		for _, id := range j.job.DependentJobs {
			newJob, err := cache.Get(id)
			if err != nil {
				log.Errorf("Error retrieving dependent job with id of %s", id)
			} else {
				newJob.Run(cache)
			}
		}
	}

	return j.currentStat, j.meta, nil
}

func (j *JobRunner) runCmd() error {
	j.numberOfAttempts++

	// Execute command
	args, err := shParser.Parse(j.job.Command)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return ErrCmdIsEmpty
	}
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Run()
}

func (j *JobRunner) shouldRetry() bool {
	// Check number of retries left
	if j.currentRetries == 0 {
		return false
	}

	// Check Epsilon
	if j.job.Epsilon != "" {
		if j.job.epsilonDuration.ToDuration() != 0 {
			timeSinceStart := time.Now().Sub(j.job.NextRunAt)
			timeLeftToRetry := j.job.epsilonDuration.ToDuration() - timeSinceStart
			if timeLeftToRetry < 0 {
				return false
			}
		}
	}

	return true
}

func (j *JobRunner) runSetup() {
	// Setup Job Stat
	j.currentStat = NewJobStat(j.job.Id)

	// Init retries
	j.currentRetries = j.job.Retries
}

func (j *JobRunner) collectStats(success bool) {
	j.currentStat.ExecutionDuration = time.Now().Sub(j.currentStat.RanAt)
	j.currentStat.Success = success
	j.currentStat.NumberOfRetries = j.job.Retries - j.currentRetries
}

package job

import (
	"time"
)

// KalaStats is the struct for storing app-level metrics
type KalaStats struct {
	ActiveJobs   int `json:"active_jobs"`
	DisabledJobs int `json:"disabled_jobs"`
	Jobs         int `json:"jobs"`

	ErrorCount   uint `json:"error_count"`
	SuccessCount uint `json:"success_count"`

	NextRunAt        time.Time `json:"next_run_at"`
	LastAttemptedRun time.Time `json:"last_attempted_run"`

	CreatedAt time.Time `json:"created"`
}

// NewKalaStats is used to easily generate a current app-level metrics report.
func NewKalaStats(cache JobCache) *KalaStats {
	ks := &KalaStats{
		CreatedAt: pkgClock.Now(),
	}
	jobs := cache.GetAll()
	jobs.Lock.RLock()
	defer jobs.Lock.RUnlock()

	ks.Jobs = len(jobs.Jobs)
	if ks.Jobs == 0 {
		return ks
	}

	nextRun := time.Time{}
	lastRun := time.Time{}
	for _, job := range jobs.Jobs {
		job.lock.RLock()
		defer job.lock.RUnlock()

		if job.Disabled {
			ks.DisabledJobs += 1
		} else {
			ks.ActiveJobs += 1
		}

		if nextRun.IsZero() {
			nextRun = job.NextRunAt
		} else if (nextRun.UnixNano() - job.NextRunAt.UnixNano()) > 0 {
			nextRun = job.NextRunAt
		}

		if lastRun.IsZero() {
			if !job.Metadata.LastAttemptedRun.IsZero() {
				lastRun = job.Metadata.LastAttemptedRun
			}
		} else if (lastRun.UnixNano() - job.Metadata.LastAttemptedRun.UnixNano()) < 0 {
			lastRun = job.Metadata.LastAttemptedRun
		}

		ks.ErrorCount += job.Metadata.ErrorCount
		ks.SuccessCount += job.Metadata.SuccessCount
	}
	ks.NextRunAt = nextRun
	ks.LastAttemptedRun = lastRun

	return ks
}

// JobStat is used to store metrics about a specific Job .Run()
type JobStat struct {
	JobId             string        `json:"job_id"`
	RanAt             time.Time     `json:"ran_at"`
	NumberOfRetries   uint          `json:"number_of_retries"`
	Success           bool          `json:"success"`
	ExecutionDuration time.Duration `json:"execution_duration"`
}

func NewJobStat(id string) *JobStat {
	return &JobStat{
		JobId: id,
		RanAt: pkgClock.Now(),
	}
}

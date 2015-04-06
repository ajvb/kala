package job

import (
	"time"
)

type KalaStats struct {
	ActiveJobs   int
	DisabledJobs int
	Jobs         int

	ErrorCount   uint
	SuccessCount uint

	NextJobRunAt time.Time

	CreatedAt time.Time
}

func NewKalaStats() *KalaStats {
	ks := &KalaStats{
		CreatedAt: time.Now(),
	}
	jobs := AllJobs.GetAll()

	ks.Jobs = len(jobs)
	if len(jobs) == 0 {
		return ks
	}

	nextRun := time.Time{}
	for _, job := range jobs {
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

		ks.ErrorCount += job.ErrorCount
		ks.SuccessCount += job.SuccessCount
	}
	ks.NextJobRunAt = nextRun

	return ks
}

type JobStat struct {
	JobId             string
	RanAt             time.Time
	NumberOfRetries   uint
	Success           bool
	ExecutionDuration time.Duration
}

func NewJobStat(id string) *JobStat {
	return &JobStat{
		JobId: id,
		RanAt: time.Now(),
	}
}

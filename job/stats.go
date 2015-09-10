package job

import (
	"sync"
	"time"
)

// KalaStats is the struct for storing app-level metrics
type KalaStats struct {
	ActiveJobs   int
	DisabledJobs int
	Jobs         int

	ErrorCount   uint
	SuccessCount uint

	NextRunAt        time.Time
	LastAttemptedRun time.Time

	CreatedAt time.Time
}

// NewKalaStats is used to easily generate a current app-level metrics report.
func NewKalaStats(cache JobCache) *KalaStats {
	ks := &KalaStats{
		CreatedAt: time.Now(),
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
		} else if (nextRun.UnixNano() - job.NextRunAt.UnixNano()) < 0 {
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

type StatsMap struct {
	// Job.Id's to there job stats
	Stats map[string][]*JobStat
	Lock  sync.RWMutex
}

type JobStatsManager struct {
	stats *StatsMap
}

func NewJobStatsManager() *JobStatsManager {
	return &JobStatsManager{
		stats: &StatsMap{
			Stats: map[string][]*JobStat{},
		},
	}
}

func (jsm *JobStatsManager) AddStat(stat *JobStat) {
	jsm.stats.Lock.Lock()
	defer jsm.stats.Lock.Unlock()

	if jsm.stats.Stats[stat.JobId] == nil {
		jsm.stats.Stats[stat.JobId] = []*JobStat{stat}
	} else {
		jsm.stats.Stats[stat.JobId] = append(jsm.stats.Stats[stat.JobId], stat)
	}
}

func (jsm *JobStatsManager) GetAllStats() *StatsMap {
	jsm.stats.Lock.RLock()
	defer jsm.stats.Lock.RUnlock()

	return jsm.stats
}

func (jsm *JobStatsManager) GetStats(id string) []*JobStat {
	jsm.stats.Lock.RLock()
	defer jsm.stats.Lock.RUnlock()

	return jsm.stats.Stats[id]
}

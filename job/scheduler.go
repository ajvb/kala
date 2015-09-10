package job

import (
	"time"
)

type Scheduler struct {
	ScheduledJobs map[string]*ScheduledJob
}

type ScheduledJob struct {
	jobTimer *time.Timer
}

func (s *Scheduler) GetWaitDuration(j *Job) time.Duration {
	j.lock.RLock()
	defer j.lock.RUnlock()

	waitDuration := time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())
	log.Debug("Wait Duration initial: %s", waitDuration)

	if waitDuration < 0 {
		// Needs to be recalculated each time because of Months.
		if j.Metadata.LastAttemptedRun.IsZero() {
			waitDuration = j.delayDuration.ToDuration()
		} else {
			lastRun := j.Metadata.LastAttemptedRun
			lastRun = lastRun.Add(j.delayDuration.ToDuration())
			waitDuration = lastRun.Sub(time.Now())
		}
	}

	return waitDuration
}

func (s *Scheduler) Schedule(job *Job) {
}

package job

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// ErrJobNotFound is raised when a Job is unable to be found within a database.
type ErrJobNotFound string

func (id ErrJobNotFound) Error() string {
	return fmt.Sprintf("Job with id of %s not found.", string(id))
}

type JobDB interface {
	GetAll() ([]*Job, error)
	Get(id string) (*Job, error)
	Delete(id string) error
	Save(job *Job) error
	Close() error
}

func (j *Job) Delete(cache JobCache, db JobDB) error {
	var err error
	errOne := cache.Delete(j.Id)
	if errOne != nil {
		log.Errorf("Error occured while trying to delete job from cache: %s", errOne)
		err = errOne
	}
	errTwo := db.Delete(j.Id)
	if errTwo != nil {
		log.Errorf("Error occured while trying to delete job from db: %s", errTwo)
		err = errTwo
	}
	return err
}

func DeleteAll(cache JobCache, db JobDB) error {
	allJobs := cache.GetAll()
	allJobs.Lock.RLock()
	// make a copy of all jobs to prevent deadlock on delete
	jobsCopy := make([]*Job, 0, len(allJobs.Jobs))
	for _, j := range allJobs.Jobs {
		jobsCopy = append(jobsCopy, j)
	}
	allJobs.Lock.RUnlock()

	for _, j := range jobsCopy {
		if err := j.Delete(cache, db); err != nil {
			return err
		}
	}
	return nil
}

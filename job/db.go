package job

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

// ErrJobNotFound is raised when a Job is able to be found within a database.
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
	j.Disable()
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

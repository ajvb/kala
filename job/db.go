package job

import "fmt"

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

func (j *Job) Delete(cache JobCache, db JobDB) {
	j.Disable()
	cache.Delete(j.Id)
	db.Delete(j.Id)
}

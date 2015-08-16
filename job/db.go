package job

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

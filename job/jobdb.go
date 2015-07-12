package job

type JobDB interface {
	GetAll() ([]*Job, error)
	Get(id string) (*Job, error)
	Delete(id string)
	Save(job *Job) error
	Close()
}

func (j *Job) Delete(cache JobCache, db JobDB) {
	j.Disable()
	cache.Delete(j.Id)
	db.Delete(j.Id)
}

func (j *Job) Save(db JobDB) error {
	return db.Save(j)
}

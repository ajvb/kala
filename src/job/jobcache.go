package job

import (
	"sync"
)

var (
	AllJobs = &JobCache{
		Jobs:   make(map[string]*Job),
		rwLock: sync.Mutex{},
	}
)

type JobCache struct {
	Jobs   map[string]*Job
	rwLock sync.Mutex
}

func (c *JobCache) Get(id string) *Job {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	return c.Jobs[id]
}

func (c *JobCache) Set(j *Job) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	if j == nil {
		return
	}

	c.Jobs[j.Id] = j
	return
}

func (c *JobCache) Delete(id string) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	delete(c.Jobs, id)
}

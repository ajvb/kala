package job

import (
	"errors"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	JobDoesntExistErr = errors.New("The job you requested does not exist")
)

type JobCache interface {
	Get(id string) (*Job, error)
	GetAll() map[string]*Job
	set(j *Job) error
	Delete(id string)
	Persist() error
}

type MemoryJobCache struct {
	// Jobs is a map from Job id's to pointers to the jobs.
	// Used as the main "data store" within this cache implementation.
	jobs            map[string]*Job
	rwLock          sync.Mutex
	jobDB           JobDB
	persistWaitTime time.Duration
}

func NewMemoryJobCache(jobDB JobDB, persistWaitTime time.Duration) *MemoryJobCache {
	if persistWaitTime == 0 {
		persistWaitTime = 5 * time.Second
	}
	return &MemoryJobCache{
		jobs:            map[string]*Job{},
		rwLock:          sync.Mutex{},
		jobDB:           jobDB,
		persistWaitTime: persistWaitTime,
	}
}

func (c *MemoryJobCache) Start() {
	// Prep cache
	allJobs, err := c.jobDB.GetAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, j := range allJobs {
		j.StartWaiting(c)
		c.set(j)
	}

	// Occasionally, save items in cache to db.
	go c.PersistEvery()

	// Process-level defer for shutting down the db.
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go func() {
		s := <-ch
		log.Info("Process got signal: %s", s)
		log.Info("Shutting down....")

		// Persist all jobs to database
		c.Persist()

		// Close the database
		c.jobDB.Close()

		os.Exit(1)
	}()
}

func (c *MemoryJobCache) Get(id string) (*Job, error) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	j := c.jobs[id]
	if j == nil {
		return nil, JobDoesntExistErr
	}

	return j, nil
}

func (c *MemoryJobCache) GetAll() map[string]*Job {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	return c.jobs
}

func (c *MemoryJobCache) set(j *Job) error {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	if j == nil {
		return nil
	}

	c.jobs[j.Id] = j
	return nil
}

func (c *MemoryJobCache) Delete(id string) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	delete(c.jobs, id)
}

func (c *MemoryJobCache) Persist() error {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	for _, j := range c.jobs {
		err := c.jobDB.Save(j)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *MemoryJobCache) PersistEvery() {
	wait := time.Tick(c.persistWaitTime)
	var err error
	for {
		<-wait
		err = c.Persist()
		if err != nil {
			log.Error("Error occured persisting the database. Err: %s", err)
		}
	}
}

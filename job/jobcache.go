package job

import (
	"os"
	"os/signal"
	"sync"
	"time"
)

type JobCache interface {
	Get(id string) *Job
	GetAll() map[string]*Job
	Set(j *Job)
	Delete(id string)
	Persist() error
}

type MemoryJobCache struct {
	Jobs            map[string]*Job
	rwLock          sync.Mutex
	jobDB           JobDB
	persistWaitTime time.Duration
}

func NewMemoryJobCache(jobDB JobDB, persistWaitTime time.Duration) *MemoryJobCache {
	if persistWaitTime == time.Duration(0) {
		persistWaitTime = 5 * time.Second
	}
	return &MemoryJobCache{
		Jobs:            map[string]*Job{},
		rwLock:          sync.Mutex{},
		jobDB:           jobDB,
		persistWaitTime: persistWaitTime,
	}
}

func (c *MemoryJobCache) Init() {
	// Prep cache
	allJobs, err := c.jobDB.GetAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range allJobs {
		v.StartWaiting(c)
		c.Set(v)
	}

	// Occasionally, save items in cache to db.
	go c.PersistEvery(c.persistWaitTime)

	// Process-level defer for shutting down the db.
	ch := make(chan os.Signal, 1)
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

func (c *MemoryJobCache) Get(id string) *Job {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	return c.Jobs[id]
}

func (c *MemoryJobCache) GetAll() map[string]*Job {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	return c.Jobs
}

func (c *MemoryJobCache) Set(j *Job) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	if j == nil {
		return
	}

	c.Jobs[j.Id] = j
	return
}

func (c *MemoryJobCache) Delete(id string) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	delete(c.Jobs, id)
}

func (c *MemoryJobCache) Persist() error {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	for _, v := range c.Jobs {
		err := v.Save(c.jobDB)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *MemoryJobCache) PersistEvery(waitTime time.Duration) {
	for {
		time.Sleep(waitTime)
		go c.Persist()
	}
}

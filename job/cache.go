package job

import (
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"unsafe"

	log "github.com/Sirupsen/logrus"
	"github.com/cornelk/hashmap"
)

var (
	ErrJobDoesntExist = errors.New("The job you requested does not exist")
)

type JobCache interface {
	Get(id string) (*Job, error)
	GetAll() *JobsMap
	Set(j *Job) error
	Delete(id string) error
	Persist() error
}

type JobsMap struct {
	Jobs map[string]*Job
	Lock sync.RWMutex
}

func NewJobsMap() *JobsMap {
	return &JobsMap{
		Jobs: map[string]*Job{},
		Lock: sync.RWMutex{},
	}
}

type MemoryJobCache struct {
	// Jobs is a map from Job id's to pointers to the jobs.
	// Used as the main "data store" within this cache implementation.
	jobs  *JobsMap
	jobDB JobDB
}

func NewMemoryJobCache(jobDB JobDB) *MemoryJobCache {
	return &MemoryJobCache{
		jobs:  NewJobsMap(),
		jobDB: jobDB,
	}
}

func (c *MemoryJobCache) Start(persistWaitTime time.Duration) {
	if persistWaitTime == 0 {
		persistWaitTime = 5 * time.Second
	}

	// Prep cache
	allJobs, err := c.jobDB.GetAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, j := range allJobs {
		if j.ShouldStartWaiting() {
			j.StartWaiting(c)
		}
		c.Set(j)
	}

	// Occasionally, save items in cache to db.
	go c.PersistEvery(persistWaitTime)

	// Process-level defer for shutting down the db.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		s := <-ch
		log.Infof("Process got signal: %s", s)
		log.Infof("Shutting down....")

		// Persist all jobs to database
		c.Persist()

		// Close the database
		c.jobDB.Close()

		os.Exit(0)
	}()
}

func (c *MemoryJobCache) Get(id string) (*Job, error) {
	c.jobs.Lock.RLock()
	defer c.jobs.Lock.RUnlock()

	j := c.jobs.Jobs[id]
	if j == nil {
		return nil, ErrJobDoesntExist
	}

	return j, nil
}

func (c *MemoryJobCache) GetAll() *JobsMap {
	return c.jobs
}

func (c *MemoryJobCache) Set(j *Job) error {
	c.jobs.Lock.Lock()
	defer c.jobs.Lock.Unlock()
	if j == nil {
		return nil
	}
	c.jobs.Jobs[j.Id] = j
	return nil
}

func (c *MemoryJobCache) Delete(id string) error {
	log.Infoln("Lock on delete")
	c.jobs.Lock.Lock()
	defer c.jobs.Lock.Unlock()

	j := c.jobs.Jobs[id]
	if j == nil {
		return ErrJobDoesntExist
	}

	j.Disable()

	go j.DeleteFromParentJobs(c)

	// Remove itself from dependent jobs as a parent job
	// and possibly delete child jobs if they don't have any other parents.
	go j.DeleteFromDependentJobs(c)

	delete(c.jobs.Jobs, id)

	return nil
}

func (c *MemoryJobCache) Persist() error {
	c.jobs.Lock.RLock()
	defer c.jobs.Lock.RUnlock()
	for _, j := range c.jobs.Jobs {
		err := c.jobDB.Save(j)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *MemoryJobCache) PersistEvery(persistWaitTime time.Duration) {
	wait := time.Tick(persistWaitTime)
	var err error
	for {
		<-wait
		err = c.Persist()
		if err != nil {
			log.Errorf("Error occured persisting the database. Err: %s", err)
		}
	}
}

type FreeJobCache struct {
	jobs  *hashmap.HashMap
	jobDB JobDB
}

func NewFreeJobCache(jobDB JobDB) *FreeJobCache {
	return &FreeJobCache{
		jobs:  hashmap.New(),
		jobDB: jobDB,
	}
}

func (c *FreeJobCache) Start(persistWaitTime time.Duration) {
	if persistWaitTime == 0 {
		persistWaitTime = 5 * time.Second
	}

	// Prep cache
	allJobs, err := c.jobDB.GetAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, j := range allJobs {
		if j.ShouldStartWaiting() {
			j.StartWaiting(c)
		}
		c.Set(j)
	}

	// Occasionally, save items in cache to db.
	go c.PersistEvery(persistWaitTime)

	// Process-level defer for shutting down the db.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		s := <-ch
		log.Infof("Process got signal: %s", s)
		log.Infof("Shutting down....")

		// Persist all jobs to database
		c.Persist()

		// Close the database
		c.jobDB.Close()

		os.Exit(0)
	}()
}

func (c *FreeJobCache) Get(id string) (*Job, error) {
	val, exists := c.jobs.GetStringKey(id)
	if val == nil || !exists {
		return nil, ErrJobDoesntExist
	}
	j := (*Job)(val)
	if j == nil {
		return nil, ErrJobDoesntExist
	}
	return j, nil
}

func (c *FreeJobCache) GetAll() *JobsMap {
	jm := NewJobsMap()
	for el := range c.jobs.Iter() {
		jm.Jobs[el.Key.(string)] = (*Job)(el.Value)
	}
	return jm
}

func (c *FreeJobCache) Set(j *Job) error {
	if j == nil {
		return nil
	}
	c.jobs.Set(j.Id, unsafe.Pointer(j))
	return nil
}

func (c *FreeJobCache) Delete(id string) error {
	j, err := c.Get(id)
	if j == nil {
		return ErrJobDoesntExist
	}
	if err != nil {
		return err
	}

	j.Disable()
	go j.DeleteFromParentJobs(c)
	// Remove itself from dependent jobs as a parent job
	// and possibly delete child jobs if they don't have any other parents.
	go j.DeleteFromDependentJobs(c)
	c.jobs.Del(id)
	return nil
}

func (c *FreeJobCache) Persist() error {
	jm := c.GetAll()
	for _, j := range jm.Jobs {
		err := c.jobDB.Save(j)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *FreeJobCache) PersistEvery(persistWaitTime time.Duration) {
	wait := time.Tick(persistWaitTime)
	var err error
	for {
		<-wait
		err = c.Persist()
		if err != nil {
			log.Errorf("Error occured persisting the database. Err: %s", err)
		}
	}
}

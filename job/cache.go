package job

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cornelk/hashmap"
	log "github.com/sirupsen/logrus"
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
	Enable(j *Job) error
	Disable(j *Job) error
	SaveRun(run *JobStat) error
	GetAllRuns(jobID string) ([]*JobStat, error)
	GetRun(runID string) (*JobStat, error)
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

func (c *MemoryJobCache) Start() {
	// Prep cache
	allJobs, err := c.jobDB.GetAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, j := range allJobs {
		if j.ShouldStartWaiting() {
			j.StartWaiting(c, false)
		}
		err = c.Set(j)
		if err != nil {
			log.Errorln(err)
		}
	}

	// Process-level defer for shutting down the db.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		s := <-ch
		log.Infof("Process got signal: %s", s)
		log.Infof("Shutting down....")

		// Persist all jobs to database
		log.Errorln(c.Persist())

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

	if err := c.jobDB.Save(j); err != nil {
		return err
	}

	c.jobs.Jobs[j.Id] = j
	return nil
}

func (c *MemoryJobCache) Delete(id string) error {
	c.jobs.Lock.Lock()
	defer c.jobs.Lock.Unlock()

	j := c.jobs.Jobs[id]
	if j == nil {
		return ErrJobDoesntExist
	}
	j.lock.Lock()
	defer j.lock.Unlock()

	err := c.jobDB.Delete(id)
	if err != nil {
		return fmt.Errorf("Error occurred while trying to delete job from db: %s", err)
	}

	j.lock.Unlock()
	j.StopTimer()
	j.lock.Lock()

	go func() {
		log.Errorln(j.DeleteFromParentJobs(c)) // todo: review
	}()

	// Remove itself from dependent jobs as a parent job
	// and possibly delete child jobs if they don't have any other parents.
	go func() {
		log.Errorln(j.DeleteFromDependentJobs(c)) // todo: review
	}()

	delete(c.jobs.Jobs, id)

	return err
}

func (c *MemoryJobCache) Enable(j *Job) error {
	return enable(j, c)
}

// Disable stops a job from running by stopping its jobTimer. It also sets Job.Disabled to true,
// which is reflected in the UI.
func (c *MemoryJobCache) Disable(j *Job) error {
	return disable(j, c)
}

func (c *MemoryJobCache) SaveRun(run *JobStat) error {
	err := c.jobDB.SaveRun(run)

	return err
}

func (c *MemoryJobCache) GetAllRuns(jobID string) ([]*JobStat, error) {
	return c.jobDB.GetAllRuns(jobID)
}

func (c *MemoryJobCache) GetRun(runID string) (*JobStat, error) {
	return c.jobDB.GetRun(runID)
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

type LockFreeJobCache struct {
	jobs            *hashmap.HashMap
	jobDB           JobDB
	retentionPeriod time.Duration
	Clock
}

func NewLockFreeJobCache(jobDB JobDB) *LockFreeJobCache {
	return &LockFreeJobCache{
		jobs:  hashmap.New(8), //nolint:gomnd
		jobDB: jobDB,
	}
}

func (c *LockFreeJobCache) Start() {
	// Prep cache
	allJobs, err := c.jobDB.GetAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, j := range allJobs {
		if j.Schedule == "" {
			log.Infof("Job %s:%s skipped.", j.Name, j.Id)
			continue
		}
		if j.ShouldStartWaiting() {
			j.StartWaiting(c, false)
		}
		log.Infof("Job %s:%s added to cache.", j.Name, j.Id)
		err := c.Set(j)
		if err != nil {
			log.Errorln(err)
		}
	}

	// Process-level defer for shutting down the db.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		s := <-ch
		log.Infof("Process got signal: %s", s)
		log.Infof("Shutting down....")

		// Persist all jobs to database
		log.Errorln(c.Persist())

		// Close the database
		c.jobDB.Close()

		os.Exit(0)
	}()
}

func (c *LockFreeJobCache) Get(id string) (*Job, error) {
	val, exists := c.jobs.GetStringKey(id)
	if val == nil || !exists {
		return nil, ErrJobDoesntExist
	}
	j := val.(*Job)
	if j == nil {
		return nil, ErrJobDoesntExist
	}
	return j, nil
}

func (c *LockFreeJobCache) GetAll() *JobsMap {
	jm := NewJobsMap()
	for el := range c.jobs.Iter() {
		jm.Jobs[el.Key.(string)] = el.Value.(*Job)
	}
	return jm
}

func (c *LockFreeJobCache) Set(j *Job) error {
	if j == nil {
		return nil
	}

	if err := c.jobDB.Save(j); err != nil {
		return err
	}

	c.jobs.Set(j.Id, j)
	return nil
}

func (c *LockFreeJobCache) Delete(id string) error {
	j, err := c.Get(id)
	if err != nil {
		return ErrJobDoesntExist
	}
	j.lock.Lock()
	defer j.lock.Unlock()

	err = c.jobDB.Delete(id)
	if err != nil {
		return fmt.Errorf("Error occurred while trying to delete job from db: %s", err)
	}

	j.lock.Unlock()
	j.StopTimer()
	j.lock.Lock()

	go func() {
		log.Errorln(j.DeleteFromParentJobs(c)) // todo: review
	}()
	// Remove itself from dependent jobs as a parent job
	// and possibly delete child jobs if they don't have any other parents.
	go func() {
		log.Errorln(j.DeleteFromDependentJobs(c)) // todo: review
	}()
	log.Infof("Deleting %s", id)
	c.jobs.Del(id)
	return err
}

func (c *LockFreeJobCache) Enable(j *Job) error {
	return enable(j, c)
}

// Disable stops a job from running by stopping its jobTimer. It also sets Job.Disabled to true,
// which is reflected in the UI.
func (c *LockFreeJobCache) Disable(j *Job) error {
	return disable(j, c)
}

func (c *LockFreeJobCache) SaveRun(run *JobStat) error {
	err := c.jobDB.SaveRun(run)

	return err
}

func (c *LockFreeJobCache) GetAllRuns(jobID string) ([]*JobStat, error) {
	return c.jobDB.GetAllRuns(jobID)
}

func (c *LockFreeJobCache) GetRun(runID string) (*JobStat, error) {
	return c.jobDB.GetRun(runID)
}

func (c *LockFreeJobCache) Persist() error {
	jm := c.GetAll()
	for _, j := range jm.Jobs {
		j.lock.RLock()
		err := c.jobDB.Save(j)
		if err != nil {
			j.lock.RUnlock()
			return err
		}
		j.lock.RUnlock()
	}
	return nil
}

package job

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
)

var (
	SaveAllJobsWaitTime = 5 * time.Second

	db = getDB()

	jobBucket = []byte("jobs")
)

func getDB() *bolt.DB {
	database, err := bolt.Open("jobdb.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	return database
}

func GetAllJobs() ([]*Job, error) {
	allJobs := []*Job{}

	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(jobBucket)
		if err != nil {
			return err
		}

		err = bucket.ForEach(func(k, v []byte) error {
			buffer := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buffer)
			j := new(Job)
			err := dec.Decode(j)

			if err != nil {
				return err
			}

			err = j.InitDelayDuration(false)

			if err != nil {
				return err
			}

			allJobs = append(allJobs, j)

			return nil
		})

		return err
	})

	return allJobs, err
}

func GetJob(id string) (*Job, error) {
	j := new(Job)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(jobBucket)

		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("Job with id of %s not found.", id)
		}

		buffer := bytes.NewBuffer(v)
		dec := gob.NewDecoder(buffer)
		err := dec.Decode(j)

		return err
	})
	if err != nil {
		return nil, err
	}

	j.Init()
	j.Id = id
	return j, nil
}

func (j *Job) Delete() {
	j.Disable()
	AllJobs.Delete(j.Id)
	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(jobBucket)
		bucket.Delete([]byte(j.Id))
		return nil
	})
}

func (j *Job) Save() error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(jobBucket)
		if err != nil {
			return err
		}

		buffer := new(bytes.Buffer)
		enc := gob.NewEncoder(buffer)
		err = enc.Encode(j)
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(j.Id), buffer.Bytes())
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func (c *JobCache) Persist() error {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	for _, v := range c.Jobs {
		err := v.Save()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *JobCache) PersistEvery(waitTime time.Duration) {
	for {
		time.Sleep(waitTime)
		go c.Persist()
	}
}

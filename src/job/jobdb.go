package job

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
)

var (
	SaveAllJobsWaitTime = time.Duration(5 * time.Second)

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

func StartWatchingAllJobs() error {
	allJobs, err := GetAllJobs()
	if err != nil {
		return err
	}

	for _, v := range allJobs {
		go v.StartWaiting()
	}

	return nil
}

func GetAllJobs() ([]*Job, error) {
	allJobs := make([]*Job, 0)

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

			allJobs = append(allJobs, j)

			return err
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
	return j, err
}

func (j *Job) Delete() {
	j.Disable()
	delete(AllJobs, j.Id)
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

func SaveAllJobs() error {
	for _, v := range AllJobs {
		err := v.Save()
		if err != nil {
			return err
		}
	}
	return nil
}

func SaveAllJobsEvery(waitTime time.Duration) {
	for {
		time.Sleep(waitTime)
		go SaveAllJobs()
	}
}

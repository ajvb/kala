package redis

import (
	"github.com/ajvb/kala/job"

	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
)

var (
	// HashKey is the hash key where jobs are persisted.
	HashKey = "kala:jobs"
)

// DB is concrete implementation of the JobDB interface, that uses Redis for persistence.
type DB struct {
	conn      redis.Conn
	keyprefix string
}

// New instantiates a new DB.
func New(address string, password redis.DialOption, passinterface int) *DB {
	var conn redis.Conn
	var err error
	if passinterface == 1 {
		conn, err = redis.Dial("tcp", address, password)
	} else {
		conn, err = redis.Dial("tcp", address)
	}
	if err != nil {
		log.Fatal(err)
	}
	return &DB{
		conn: conn,
	}
}

// GetAll returns all persisted Jobs.
func (d DB) GetAll() ([]*job.Job, error) {
	jobs := []*job.Job{}

	vals, err := d.conn.Do("HVALS", HashKey)
	if err != nil {
		return jobs, err
	}

	for _, val := range vals.([]interface{}) {
		j, err := job.NewFromBytes(val.([]byte))
		if err != nil {
			return nil, err
		}

		err = j.InitDelayDuration(false)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, j)
	}

	return jobs, nil
}

// Get returns a persisted Job.
func (d DB) Get(id string) (*job.Job, error) {
	val, err := d.conn.Do("HGET", HashKey, id)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, job.ErrJobNotFound(id)
	}

	return job.NewFromBytes(val.([]byte))
}

// Delete deletes a persisted Job.
func (d DB) Delete(id string) error {
	_, err := d.conn.Do("HDEL", HashKey, id)
	if err != nil {
		return err
	}

	return nil
}

// Save persists a Job.
func (d DB) Save(j *job.Job) error {
	bytes, err := j.Bytes()
	if err != nil {
		return err
	}

	_, err = d.conn.Do("HSET", HashKey, j.Id, bytes)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the connection to Redis.
func (d DB) Close() error {
	err := d.conn.Close()
	if err != nil {
		return err
	}
	return nil
}

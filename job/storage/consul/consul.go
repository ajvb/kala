package consul

import (
	log "github.com/Sirupsen/logrus"
	"github.com/ajvb/kala/job"
	"github.com/hashicorp/consul/api"
	"path"
)

// prefixKey is the key where jobs are persisted.
var prefixKey = "kala/jobs"

// DB is concrete implementation of the JobDB interface, that uses Consul for persistence.
type DB struct {
	conn         *api.Client
	queryOptions *api.QueryOptions
	writeOptions *api.WriteOptions
	keyprefix    string
}

// New instantiates a new DB.
func New(address string) *DB {

	config := &api.Config{
		Address: address,
	}

	queryOptions := &api.QueryOptions{
		RequireConsistent: true,
	}

	writeOptions := &api.WriteOptions{}

	conn, err := api.NewClient(config)

	if err != nil {
		log.Fatal(err)
	}

	return &DB{
		conn:         conn,
		queryOptions: queryOptions,
		writeOptions: writeOptions,
		keyprefix:    prefixKey,
	}
}

// GetAll returns all persisted Jobs.
func (d DB) GetAll() ([]*job.Job, error) {
	jobs := []*job.Job{}

	keys, _, err := d.conn.KV().Keys(d.keyprefix, "", d.queryOptions)
	if err != nil {
		return jobs, err
	}

	for _, key := range keys {
		kv, _, err := d.conn.KV().Get(key, d.queryOptions)
		if err != nil {
			return nil, err
		}

		j, err := job.NewFromBytes(kv.Value)
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
	kv, _, err := d.conn.KV().Get(path.Join(d.keyprefix, id), d.queryOptions)
	if err != nil {
		return nil, err
	}
	if kv.Value == nil {
		return nil, job.ErrJobNotFound(id)
	}

	return job.NewFromBytes(kv.Value)
}

// Delete deletes a persisted Job.
func (d DB) Delete(id string) error {
	_, err := d.conn.KV().Delete(path.Join(d.keyprefix, id), d.writeOptions)
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

	jobKV := &api.KVPair{
		Key:   path.Join(d.keyprefix, j.Id),
		Value: bytes,
	}

	_, err = d.conn.KV().Put(jobKV, d.writeOptions)
	if err != nil {
		return err
	}

	return nil
}

// Consul API client does not support a close method
func (d DB) Close() error {
	return nil
}

package consul

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ajvb/kala/job"

	"github.com/hashicorp/consul/api"

	log "github.com/Sirupsen/logrus"
)

var (
	prefix = "kala/jobs/"
)

func New(address string) *ConsulJobDB {
	config := api.DefaultConfig()
	if address != "" {
		config.Address = address
	}
	client, err := api.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	return &ConsulJobDB{
		conn: client.KV(),
	}
}

type ConsulJobDB struct {
	conn *api.KV
}

func (db *ConsulJobDB) Close() error {
	return nil
}

func (db *ConsulJobDB) GetAll() ([]*job.Job, error) {
	allJobs := []*job.Job{}

	pairs, _, err := db.conn.List(prefix, &api.QueryOptions{RequireConsistent: true})
	if err != nil {
		return allJobs, err
	}
	for _, pair := range pairs {

		buffer := bytes.NewBuffer(pair.Value)
		dec := json.NewDecoder(buffer)
		j := new(job.Job)
		err := dec.Decode(j)
		if err != nil {
			continue
		}
		j.InitDelayDuration(false)
		allJobs = append(allJobs, j)
	}

	return allJobs, err
}

func (db *ConsulJobDB) Get(id string) (*job.Job, error) {
	j := new(job.Job)

	pair, _, err := db.conn.Get(prefix+id, &api.QueryOptions{RequireConsistent: true})
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, fmt.Errorf("ID %s not found", id)
	}
	buf := bytes.NewBuffer(pair.Value)
	err = json.NewDecoder(buf).Decode(j)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (db *ConsulJobDB) Delete(id string) error {
	_, err := db.conn.Delete(prefix+id, &api.WriteOptions{})
	return err
}

func (db *ConsulJobDB) Save(j *job.Job) error {
	buffer := new(bytes.Buffer)
	enc := json.NewEncoder(buffer)
	err := enc.Encode(j)
	if err != nil {
		return err
	}
	pair := &api.KVPair{Key: prefix + j.Id, Value: buffer.Bytes()}
	_, err = db.conn.Put(pair, &api.WriteOptions{})
	return err
}

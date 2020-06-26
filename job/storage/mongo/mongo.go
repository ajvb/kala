package mongo

import (
	"github.com/ajvb/kala/job"

	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	database   = "kala"
	collection = "jobs"
)

// DB is concrete implementation of the JobDB interface, that uses Redis for persistence.
type DB struct {
	collection *mgo.Collection
	database   *mgo.Database
	session    *mgo.Session
}

// New instantiates a new DB.
func New(addrs string, cred *mgo.Credential) *DB {
	session, err := mgo.Dial(addrs)
	if err != nil {
		log.Fatal(err)
	}
	if cred.Username != "" {
		err = session.Login(cred)
		if err != nil {
			log.Fatal(err)
		}
	}
	db := session.DB(database)
	c := db.C(collection)
	session.SetMode(mgo.Monotonic, true)
	if err := c.EnsureIndexKey("id"); err != nil {
		log.Fatal(err)
	}
	return &DB{
		collection: c,
		database:   db,
		session:    session,
	}
}

// GetAll returns all persisted Jobs.
func (d DB) GetAll() ([]*job.Job, error) {
	jobs := []*job.Job{}
	err := d.collection.Find(bson.M{}).All(&jobs)
	if err != nil {
		return jobs, err
	}
	return jobs, nil
}

// Get returns a persisted Job.
func (d DB) Get(id string) (*job.Job, error) {
	result := job.Job{}
	err := d.collection.Find(bson.M{"id": id}).One(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a persisted Job.
func (d DB) Delete(id string) error {
	err := d.collection.Remove(bson.M{"id": id})
	if err != nil {
		return err
	}

	return nil
}

// Save persists a Job.
func (d DB) Save(j *job.Job) error {
	err := d.collection.Insert(j)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the connection to Redis.
func (d DB) Close() error {
	d.session.Close()
	return nil
}

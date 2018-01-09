package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/ajvb/kala/job"

	log "github.com/Sirupsen/logrus"
)

var (
	table = "jobs"
)

type DB struct {
	conn *sql.DB
}

// New instantiates a new DB.
func New(dsn string) *DB {
	connection, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	// passive attempt to create table
	_, _ = connection.Query(fmt.Sprintf(`create table %s (job jsonb);`, table))
	return &DB{
		conn: connection,
	}
}

// GetAll returns all persisted Jobs.
func (d DB) GetAll() ([]*job.Job, error) {
	jobs := []*job.Job{}
	query := fmt.Sprintf(`select jsonb_agg(j.job) from (select * from %[1]s) as j;`, table)
	var r sql.NullString
	err := d.conn.QueryRow(query).Scan(&r)
	if err == nil {
		if r.Valid {
			err = json.Unmarshal([]byte(r.String), &jobs)
		}
	} else {
		if err == sql.ErrNoRows {
			err = nil
		}
	}
	return jobs, err
}

// Get returns a persisted Job.
func (d DB) Get(id string) (*job.Job, error) {
	result := &job.Job{}
	template := `select to_jsonb(j.job) from (select * from %[1]s where job -> 'id' = $1) as j;`
	query := fmt.Sprintf(template, table)
	var r sql.NullString
	err := d.conn.QueryRow(query, id).Scan(&r)
	if err == nil {
		if r.Valid {
			err = json.Unmarshal([]byte(r.String), &result)
			if err == nil {
				return result, nil
			}
		}
	}
	return nil, err
}

// Delete deletes a persisted Job.
func (d DB) Delete(id string) error {
	query := fmt.Sprintf(`delete from %v where job -> = 'id' = $1;`, table)
	_, err := d.conn.Exec(query, id)
	return err
}

// Save persists a Job.
func (d DB) Save(j *job.Job) error {
	template := `insert into %[1]s (job) values($1);`
	query := fmt.Sprintf(template, table)
	r, err := json.Marshal(j)
	if err != nil {
		return err
	}
	transaction, err := d.conn.Begin()
	if err != nil {
		return err
	}
	statement, err := transaction.Prepare(query)
	if err != nil {
		transaction.Rollback()
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(string(r))
	if err != nil {
		transaction.Rollback()
		return err
	}
	return transaction.Commit()
}

// Close closes the connection to Postgres.
func (d DB) Close() error {
	return d.conn.Close()
}

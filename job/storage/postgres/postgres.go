package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/ajvb/kala/job"

	log "github.com/sirupsen/logrus"
)

const TABLE_NAME = "jobs"

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
	_, _ = connection.Exec(fmt.Sprintf(`create table %s (job jsonb);`, TABLE_NAME))
	return &DB{
		conn: connection,
	}
}

// GetAll returns all persisted Jobs.
func (d DB) GetAll() ([]*job.Job, error) {
	query := fmt.Sprintf(`select coalesce(json_agg(j.job), '[]'::json) from (select * from %[1]s) as j;`, TABLE_NAME)
	var r sql.NullString
	err := d.conn.QueryRow(query).Scan(&r)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	err = nil
	jobs := []*job.Job{}
	if r.Valid {
		err = json.Unmarshal([]byte(r.String), &jobs)
	}
	return jobs, err
}

// Get returns a persisted Job.
func (d DB) Get(id string) (*job.Job, error) {
	template := `select to_jsonb(j.job) from (select * from %[1]s where job -> 'id' = $1) as j;`
	query := fmt.Sprintf(template, TABLE_NAME)
	var r sql.NullString
	err := d.conn.QueryRow(query, id).Scan(&r)
	if err != nil {
		return nil, err
	}
	result := &job.Job{}
	if r.Valid {
		err = json.Unmarshal([]byte(r.String), &result)
	}
	return result, err
}

// Delete deletes a persisted Job.
func (d DB) Delete(id string) error {
	query := fmt.Sprintf(`delete from %v where job -> = 'id' = $1;`, TABLE_NAME)
	_, err := d.conn.Exec(query, id)
	return err
}

// Save persists a Job.
func (d DB) Save(j *job.Job) error {
	template := `insert into %[1]s (job) values($1);`
	query := fmt.Sprintf(template, TABLE_NAME)
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
		transaction.Rollback() //nolint:errcheck // adding insult to injury
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(string(r))
	if err != nil {
		transaction.Rollback() //nolint:errcheck // adding insult to injury
		return err
	}
	return transaction.Commit()
}

// Close closes the connection to Postgres.
func (d DB) Close() error {
	return d.conn.Close()
}

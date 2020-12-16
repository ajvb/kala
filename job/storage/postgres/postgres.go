package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"

	"bitbucket.org/nextiva/nextkala/job"

	log "github.com/sirupsen/logrus"
)

const (
	JobTable    = "jobs"
	JobRunTable = "job_runs"
)

type DB struct {
	conn *sql.DB
}

// New instantiates a new DB.
func New(dsn string) *DB {
	log.Debugf("pg dsn: %s", dsn)
	connection, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	// passive attempt to create table
	_, _ = connection.Exec(fmt.Sprintf(`create table if not exists %s (id uuid primary key, job jsonb);`, JobTable))
	_, _ = connection.Exec(fmt.Sprintf(`create table if not exists %s (id uuid primary key, job_id uuid not null references %s (id) on delete cascade, run jsonb);`, JobRunTable, JobTable))

	return &DB{
		conn: connection,
	}
}

// GetAll returns all persisted Jobs.
func (d DB) GetAll() ([]*job.Job, error) {
	query := fmt.Sprintf(`select coalesce(json_agg(j.job), '[]'::json) from (select * from %[1]s) as j;`, JobTable)
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
	template := `select to_jsonb(j.job) from (select * from %[1]s where id = $1) as j;`
	query := fmt.Sprintf(template, JobTable)
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
	query := fmt.Sprintf(`delete from %v where id = $1;`, JobTable)
	_, err := d.conn.Exec(query, id)
	return err
}

// Save persists a Job.
func (d DB) Save(j *job.Job) error {
	template := `insert into %[1]s (id, job) values($1, $2) on conflict (id) do update set job = EXCLUDED.job;`
	query := fmt.Sprintf(template, JobTable)
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
	_, err = statement.Exec(j.Id, string(r))
	if err != nil {
		transaction.Rollback() //nolint:errcheck // adding insult to injury
		return err
	}
	return transaction.Commit()
}

// SaveRun persists a Job Run.
func (d DB) SaveRun(run *job.JobStat) error {
	template := `insert into %[1]s (id, job_id, run) values($1, $2, $3);`
	query := fmt.Sprintf(template, JobRunTable)
	r, err := json.Marshal(run)
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
	_, err = statement.Exec(run.Id, run.JobId, string(r))
	if err != nil {
		transaction.Rollback() //nolint:errcheck // adding insult to injury
		return err
	}
	return transaction.Commit()
}

// GetAllRuns returns all persisted runs for a job.
func (d DB) GetAllRuns(jobID string) ([]*job.JobStat, error) {
	query := fmt.Sprintf(`select coalesce(json_agg(run), '[]'::json) from (select run from %[1]s) as run;`, JobRunTable)
	var r sql.NullString
	err := d.conn.QueryRow(query).Scan(&r)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	err = nil
	jobRuns := []*job.JobStat{}
	if r.Valid {
		err = json.Unmarshal([]byte(r.String), &jobRuns)
	}
	return jobRuns, err
}

// GetRun returns a persisted job run.
func (d DB) GetRun(id string) (*job.JobStat, error) {
	template := `select to_jsonb(run) from (select run from %[1]s where id = $1) as run;`
	query := fmt.Sprintf(template, JobRunTable)
	var r sql.NullString
	err := d.conn.QueryRow(query, id).Scan(&r)
	if err != nil {
		return nil, err
	}
	result := &job.JobStat{}
	if r.Valid {
		err = json.Unmarshal([]byte(r.String), &result)
	}
	return result, err
}

// Close closes the connection to Postgres.
func (d DB) Close() error {
	return d.conn.Close()
}

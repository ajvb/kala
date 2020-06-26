package mysql

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"github.com/jmoiron/sqlx"

	"github.com/ajvb/kala/job"

	log "github.com/sirupsen/logrus"
)

var (
	TableName = "jobs"
)

type DB struct {
	conn *sqlx.DB
}

// New instantiates a new DB.
func New(dsn string, tlsConfig *tls.Config) *DB {
	if tlsConfig != nil {
		log.Infof("Register TLS config")
		err := mysql.RegisterTLSConfig("custom", tlsConfig)
		if err != nil {
			log.Fatal(err)
		}
	}
	connection, err := sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	// passive attempt to create table
	_, _ = connection.Exec(fmt.Sprintf(`create table %s (id varchar(36), job JSON, primary key (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, TableName))
	return &DB{
		conn: connection,
	}
}

// GetAll returns all persisted Jobs.
func (d DB) GetAll() ([]*job.Job, error) {
	query := fmt.Sprintf(`select job from %[1]s;`, TableName)
	var results []sql.NullString
	err := d.conn.Select(&results, query)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	err = nil
	jobs := make([]*job.Job, 0, len(results))
	for _, v := range results {
		if v.Valid {
			j := job.Job{}
			if err = json.Unmarshal([]byte(v.String), &j); err != nil {
				break
			}
			if err = j.InitDelayDuration(false); err != nil {
				break
			}
			jobs = append(jobs, &j)
		}
	}

	return jobs, err
}

// Get returns a persisted Job.
func (d DB) Get(id string) (*job.Job, error) {
	template := `select job from %[1]s where id = ?;`
	query := fmt.Sprintf(template, TableName)
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
	query := fmt.Sprintf(`delete from %[1]s where id = ?;`, TableName)
	_, err := d.conn.Exec(query, id)
	return err
}

// Save persists a Job.
func (d DB) Save(j *job.Job) error {
	template := `replace into %[1]s (id, job) values(?, ?);`
	query := fmt.Sprintf(template, TableName)
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

// Close closes the connection to Postgres.
func (d DB) Close() error {
	return d.conn.Close()
}

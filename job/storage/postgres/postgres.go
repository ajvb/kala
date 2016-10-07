package postgres

import (
  "fmt"
  "encoding/json"

  "database/sql"
  _ "github.com/lib/pq"
  "github.com/ajvb/kala/job"
)

var (
	CONN_STRING = "user=%s password=%s dbname=%s sslmode=disable"

	SELECT_QUERY = "SELECT * FROM kala_jobs WHERE id = $1"
	INSERT_QUERY = "INSERT INTO kala_jobs VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)"
	SELECT_QUERY_ALL = "SELECT * FROM kala_jobs"
	DELETE_QUERY = "DELETE FROM kala_jobs WHERE id = $1"
)

type PostgresJobDB struct {
  conn *sql.DB
}

func New(username string, password string, dbname string) *PostgresJobDB {
  dbinfo := fmt.Sprintf(CONN_STRING, username, password, dbname)
  db, err := sql.Open("postgres", dbinfo)
  checkErr(err)
  return &PostgresJobDB {
    conn: db,
  }
}

func (db *PostgresJobDB) Get(id string) (*job.Job, error) {
  rows, err := db.conn.Query(SELECT_QUERY, id)
  checkErr(err)
  defer rows.Close()

  j := new(job.Job)

  for rows.Next() {
    // name string, id string, command string, owner string, disabled bool, dependent_jobs string[], ParentJobs []string, Schedule string, Retries uint, Epsilon string, NextRunAt time.Time, cd $GOPATH/src/github.com/ajvb/kala
    var dependentJobsStr string
    var parentJobsStr string
    var meta job.Metadata
    err := rows.Scan(
      &j.Name, 
      &j.Id, 
      &j.Command, 
      &j.Owner,
      &j.Disabled,
      &dependentJobsStr, 
      &parentJobsStr, 
      &j.Schedule, 
      &j.Retries,
      &j.Epsilon, 
      &meta.SuccessCount, 
      &meta.LastSuccess,
      &meta.ErrorCount, 
      &meta.LastError, 
      &meta.LastAttemptedRun, 
      &j.NextRunAt)
    checkErr(err)

    dependentJobsByte := []byte(dependentJobsStr)
    json.Unmarshal(dependentJobsByte, &j.DependentJobs);

    parentJobsByte := []byte(parentJobsStr)
    json.Unmarshal(parentJobsByte, &j.ParentJobs);

    j.Metadata = meta
  }

  return j, nil
}

func (db *PostgresJobDB) Save(j *job.Job) error {
  dependentJobs, _ := json.Marshal(j.DependentJobs)
  dependentJobsStr := string(dependentJobs)

  parentJobs, _ := json.Marshal(j.ParentJobs)
  parentJobsStr := string(parentJobs)

  row, err := db.conn.Query(INSERT_QUERY, 
    j.Name, 
    j.Id, 
    j.Command, 
    j.Owner,
    j.Disabled,
    dependentJobsStr, 
    parentJobsStr, 
    j.Schedule, 
    j.Retries,
    j.Epsilon, 
    j.Metadata.SuccessCount, 
    j.Metadata.LastSuccess,
    j.Metadata.ErrorCount, 
    j.Metadata.LastError, 
    j.Metadata.LastAttemptedRun, 
    j.NextRunAt)
  if row == nil{
    panic(err)
  }
  checkErr(err)


  return err
}

func (db *PostgresJobDB) GetAll() ([]*job.Job, error) {
  jobs := []*job.Job{}

  rows, err := db.conn.Query(SELECT_QUERY_ALL)
  checkErr(err)
  defer rows.Close()

  for rows.Next() {
    // name string, id string, command string, owner string, disabled bool, dependent_jobs string[], ParentJobs []string, Schedule string, Retries uint, Epsilon string, NextRunAt time.Time, cd $GOPATH/src/github.com/ajvb/kala
    j := new(job.Job)
    
    var dependentJobsStr string
    var parentJobsStr string
    var meta job.Metadata

    err := rows.Scan(
      &j.Name, 
      &j.Id, 
      &j.Command, 
      &j.Owner,
      &j.Disabled,
      &dependentJobsStr, 
      &parentJobsStr, 
      &j.Schedule, 
      &j.Retries,
      &j.Epsilon, 
      &meta.SuccessCount, 
      &meta.LastSuccess,
      &meta.ErrorCount, 
      &meta.LastError, 
      &meta.LastAttemptedRun, 
      &j.NextRunAt)
    checkErr(err)

    dependentJobsByte := []byte(dependentJobsStr)
    json.Unmarshal(dependentJobsByte, &j.DependentJobs);

    parentJobsByte := []byte(parentJobsStr)
    json.Unmarshal(parentJobsByte, &j.ParentJobs);

    j.Metadata = meta

    jobs = append(jobs, j)
  }

  return jobs, nil
}

func (db *PostgresJobDB) Delete(id string) error {
  _, err := db.conn.Exec(DELETE_QUERY, id)
  checkErr(err)
  return err
}

func (db *PostgresJobDB) Close() error {
  return db.conn.Close()
}

func checkErr(err error) {
  if err != nil {
    panic(err)
  }
}
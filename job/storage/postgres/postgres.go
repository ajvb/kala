package postgres

import (
  "encoding/json"
  "fmt"

  "database/sql"
  "github.com/ajvb/kala/job"
  _ "github.com/lib/pq"

  log "github.com/Sirupsen/logrus"
)

var (
  CONN_STRING = "user=%s password=%s dbname=%s sslmode=disable"

  SELECT_QUERY     = "SELECT * FROM kala_jobs WHERE id = $1"
  INSERT_QUERY     = "INSERT INTO kala_jobs VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)"
  SELECT_QUERY_ALL = "SELECT * FROM kala_jobs"
  DELETE_QUERY     = "DELETE FROM kala_jobs WHERE id = $1"
)

type PostgresJobDB struct {
  conn *sql.DB
}

func New(username string, password string, dbname string) *PostgresJobDB {
  dbinfo := fmt.Sprintf(CONN_STRING, username, password, dbname)
  db, err := sql.Open("postgres", dbinfo)
  if err != nil {
    log.Error("Unable to connect postgresql database.")
    panic(err)
  }
  return &PostgresJobDB{
    conn: db,
  }
}

func (db *PostgresJobDB) Get(id string) (*job.Job, error) {
  rows, err := db.conn.Query(SELECT_QUERY, id)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  j := new(job.Job)

  for rows.Next() {
    // name string, id string, command string, owner string, disabled bool, dependent_jobs string[], ParentJobs []string, Schedule string, Retries uint, Epsilon string, NextRunAt time.Time
    var dependentJobsStr string
    var parentJobsStr string
    var meta job.Metadata
    err = rows.Scan(
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
    if err != nil {
      return nil, err
    }

    dependentJobsByte := []byte(dependentJobsStr)
    err = json.Unmarshal(dependentJobsByte, &j.DependentJobs)
    if err != nil {
      return nil, err
    }

    parentJobsByte := []byte(parentJobsStr)
    err = json.Unmarshal(parentJobsByte, &j.ParentJobs)
    if err != nil {
      return nil, err
    }

    j.Metadata = meta
  }

  return j, nil
}

func (db *PostgresJobDB) Save(j *job.Job) error {
  dependentJobs, err := json.Marshal(j.DependentJobs)
  if err != nil {
    return err
  }
  dependentJobsStr := string(dependentJobs)

  parentJobs, err := json.Marshal(j.ParentJobs)
  if err != nil {
    return err
  }
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
  if row == nil {
    log.Error(err)
  }

  return err
}

func (db *PostgresJobDB) GetAll() ([]*job.Job, error) {
  jobs := []*job.Job{}

  rows, err := db.conn.Query(SELECT_QUERY_ALL)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  for rows.Next() {
    // name string, id string, command string, owner string, disabled bool, dependent_jobs string[], ParentJobs []string, Schedule string, Retries uint, Epsilon string, NextRunAt time.Time
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
    if err != nil {
      return nil, err
    }

    dependentJobsByte := []byte(dependentJobsStr)
    err = json.Unmarshal(dependentJobsByte, &j.DependentJobs)
    if err != nil {
      return nil, err
    }

    parentJobsByte := []byte(parentJobsStr)
    err = json.Unmarshal(parentJobsByte, &j.ParentJobs)
    if err != nil {
      return nil, err
    }

    j.Metadata = meta

    jobs = append(jobs, j)
  }

  return jobs, nil
}

func (db *PostgresJobDB) Delete(id string) error {
  _, err := db.conn.Exec(DELETE_QUERY, id)
  if err != nil {
    return err
  }

  return nil
}

func (db *PostgresJobDB) Close() error {
  err := db.conn.Close()
  if err != nil {
    return err
  }

  return nil
}

package orm

import (
    "fmt"
    "time"

    "github.com/ajvb/kala/job"

    _"github.com/go-sql-driver/mysql"
    _"github.com/lib/pq"

    "github.com/jinzhu/gorm"
    "github.com/Sirupsen/logrus"

    "github.com/davecgh/go-spew/spew"
)

// orm for sql abstraction
type ORM struct {
    db  *gorm.DB
}

// table where jobs are persisted
type Job struct {
    Id              string      `gorm:"column:id;primary_key"   sql:"type:uuid"             json:"id"`
    Name            string      `gorm:"column:name"             sql:"unique_index;not null" json:"name"`
    Command         string      `gorm:"column:command"          sql:"type:text;not null"    json:"command"`
    Owner           string      `gorm:"column:owner"                                        json:"owner"`
    Disabled        bool        `gorm:"column:disabled"                                     json:"disabled"`
    DependentJobs   []string    `gorm:"column:dependent_jobs"   sql:"type:uuid[]"           json:"dependent_jobs"`
    ParentJobs      []string    `gorm:"column:parent_jobs"      sql:"type:uuid[]"           json:"parent_jobs"`
    Schedule        string      `gorm:"column:schedule"                                     json:"schedule"`
    Retries         uint        `gorm:"column:retries"                                      json:"retries"`
    Epsilon         string      `gorm:"column:epsilon"                                      json:"epsilon"`
    NextRunAt       time.Time   `gorm:"column:next_run_at"                                  json:"next_run_at"`
    Metadata        Metadata    `gorm:"column:metadata"         sql:"type:json"             json:"metadata"`
    Stats           []JobStat   `gorm:"column:stats"            sql:"type:json"             json:"stats"`
}

type Metadata struct {
    SuccessCount     uint      `json:"success_count"`
    LastSuccess      time.Time `json:"last_success"`
    ErrorCount       uint      `json:"error_count"`
    LastError        time.Time `json:"last_error"`
    LastAttemptedRun time.Time `json:"last_attempted_run"`
}

type JobStat struct {
    JobId             string        `json:"job_id"`
    RanAt             time.Time     `json:"ran_at"`
    NumberOfRetries   uint          `json:"number_of_retries"`
    Success           bool          `json:"success"`
    ExecutionDuration time.Duration `json:"execution_duration"`
}

func (* Job) TableName() string {
    return "kala"
}

// open database connection
func New(driver, address string) *ORM {

    var (
        db gorm.DB
        err error
        dsn string
    )

    switch driver {
        case "mysql": dsn = address
        case "postgres": dsn = fmt.Sprintf("%s://%s",driver,address)
    }

    if db, err = gorm.Open(driver,dsn); err == nil {
        err = db.DB().Ping()
    }

    if err != nil {
        logrus.Fatal(err)
    }

    return &ORM{db:&db}
}

// GetAll returns all persisted Jobs.
func (this ORM) GetAll() ([]*job.Job, error) {

    jobs := []*job.Job{}

    if !this.db.HasTable(&Job{}) {
        if err := this.db.Debug().CreateTable(&Job{}).Error; err != nil {
            logrus.Fatal(err)
        }
    }
    spew.Dump(jobs)

//    vals, err := d.conn.Do("HVALS", HashKey)
//    if err != nil {
//        return jobs, err
//    }
//
//    for _, val := range vals.([]interface{}) {
//        j, err := job.NewFromBytes(val.([]byte))
//        if err != nil {
//            return nil, err
//        }
//
//        err = j.InitDelayDuration(false)
//        if err != nil {
//            return nil, err
//        }
//
//        jobs = append(jobs, j)
//    }

    return jobs, nil
}

// Get returns a persisted Job.
func (this ORM) Get(id string) (j *job.Job, e error) {
//    val, err := d.conn.Do("HGET", HashKey, id)
//    if err != nil {
//        return nil, err
//    }
//    if val == nil {
//        return nil, job.ErrJobNotFound(id)
//    }
//
//    return job.NewFromBytes(val.([]byte))

    return
}

// Delete deletes a persisted Job.
func (this ORM) Delete(id string) error {
//    _, err := d.conn.Do("HDEL", id)
//    if err != nil {
//        return err
//    }

    return nil
}

// Save persists a Job.
func (this ORM) Save(j *job.Job) error {
//    bytes, err := j.Bytes()
//    if err != nil {
//        return err
//    }
//
//    _, err = d.conn.Do("HSET", HashKey, j.Id, bytes)
//    if err != nil {
//        return err
//    }

    return nil
}

// Close closes the connection to Redis.
func (this ORM) Close() error {
    err := this.db.Close()
    if err != nil {
        return err
    }
    return nil
}

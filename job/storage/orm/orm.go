package orm

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "time"

    "github.com/ajvb/kala/job"

   _"github.com/go-sql-driver/mysql"
   _"github.com/lib/pq"

    "github.com/jinzhu/gorm"
    "github.com/Sirupsen/logrus"
)

// orm for sql abstraction
type ORM struct {
    db *gorm.DB
}

// Model where jobs are stored
type Model struct {
    Id              string          `gorm:"column:id;primary_key"   sql:"type:varchar(36)"      json:"id"`
    Name            string          `gorm:"column:name"             sql:"unique_index;not null" json:"name"`
    Command         string          `gorm:"column:command"          sql:"type:text;not null"    json:"command"`
    Owner           string          `gorm:"column:owner"                                        json:"owner"`
    Disabled        bool            `gorm:"column:disabled"                                     json:"disabled"`
    Schedule        string          `gorm:"column:schedule"                                     json:"schedule"`
    Retries         uint            `gorm:"column:retries"          sql:"type:integer"          json:"retries"`
    Epsilon         string          `gorm:"column:epsilon"                                      json:"epsilon"`
    NextRunAt       time.Time       `gorm:"column:next_run_at"                                  json:"next_run_at"`

    JDependentJobs  sql.NullString  `gorm:"column:dependent_jobs"   sql:"type:text"             json:"-"`
    JParentJobs     sql.NullString  `gorm:"column:parent_jobs"      sql:"type:text"             json:"-"`
    JMetadata       sql.NullString  `gorm:"column:metadata"         sql:"type:text"             json:"-"`
    JStats          sql.NullString  `gorm:"column:stats"            sql:"type:text"             json:"-"`

    DependentJobs   []string        `sql:"-"                                                    json:"dependent_jobs"`
    ParentJobs      []string        `sql:"-"                                                    json:"parent_jobs"`
    Metadata        job.Metadata    `sql:"-"                                                    json:"metadata"`
    Stats           []*job.JobStat  `sql:"-"                                                    json:"stats"`
}

// table name
func (this *Model) TableName() string {
    return "kala"
}

func (this *Model) BeforeCreate() error {
    return parse(this)
}

func (this *Model) BeforeUpdate() error {
    return parse(this)
}

func (this *Model) AfterFind() (err error) {

    if this.JDependentJobs.Valid {
        value, _ := this.JDependentJobs.Value()
        err = json.Unmarshal([]byte(value.(string)),&this.DependentJobs)
    }

    if err == nil && this.JParentJobs.Valid {
        value, _ := this.JParentJobs.Value()
        err = json.Unmarshal([]byte(value.(string)),&this.ParentJobs)
    }

    if err == nil && this.JMetadata.Valid {
        value, _ := this.JMetadata.Value()
        err = json.Unmarshal([]byte(value.(string)),&this.Metadata)
    }

    if err == nil && this.JStats.Valid {
        value, _ := this.JStats.Value()
        err = json.Unmarshal([]byte(value.(string)),&this.Stats)
    }

    return
}

func cast(source interface{}, dest interface{}) (err error) {

    var data []byte

    if data, err = json.Marshal(source); err == nil {
        err = json.Unmarshal(data,dest)
    }

    return err
}

func parse(this *Model) (err error) {

    var data []byte

    if len(this.DependentJobs) > 0 {
        if data, err = json.Marshal(this.DependentJobs); err == nil {
            this.JDependentJobs.Scan(string(data))
        }
    }

    if err == nil && len(this.ParentJobs) > 0 {
        if data, err = json.Marshal(this.ParentJobs); err == nil {
            this.JParentJobs.Scan(string(data))
        }
    }

    if err == nil {
        if data, err = json.Marshal(this.Metadata); err == nil {
            this.JMetadata.Scan(string(data))
        }
    }

    if err == nil && len(this.Stats) > 0 {
        if data, err = json.Marshal(this.Stats); err == nil {
            this.JStats.Scan(string(data))
        }
    }

    return
}

// Open opens a database connection
func Open(driver, address string) *ORM {

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
        if err = db.DB().Ping(); err == nil {
            if !db.HasTable(&Model{}) {
                err = db.CreateTable(&Model{}).Error
            }
        }
    }

    if err != nil {
        logrus.Fatal(dsn,err)
    }

    return &ORM{db:&db}
}

// GetAll returns all Jobs stored in database
func (this ORM) GetAll() (jobs []*job.Job, err error) {

    models := []Model{}

    if err = this.db.Find(&models).Error; err == nil {

        jobs = make([]*job.Job,len(models))

        for i, model := range models {

            if err = cast(&model,&jobs[i]); err != nil {
                return
            }

        }

    }

    return
}

// Get selects a Job from database.
func (this ORM) Get(id string) (*job.Job, error) {

    jb := &job.Job{}
    model := &Model{Id:id}

    err := this.db.Find(model).Error; if err == nil {
        err = cast(model,jb)
    }

    if err == gorm.RecordNotFound {
        return jb, job.ErrJobNotFound(id)
    }

    return jb, err
}

// Delete deletes a Job from database.
func (this ORM) Delete(id string) error {
    return this.db.Unscoped().Delete(&Model{Id:id}).Error
}

// Save inserts or updates a Job in database
func (this ORM) Save(jb *job.Job) error {

    var (
        err error
        model Model
    )

    q := this.db.Where("id=?",jb.Id).Or("name=?",jb.Name).Find(&model)

    if q.Error != nil && q.Error != gorm.RecordNotFound {
        return q.Error
    }

    if err = cast(jb,&model); err == nil {
        if q.RecordNotFound() {
            err = this.db.Create(&model).Error
        } else {
            err = this.db.Save(&model).Error
        }
    }

    return err
}

// Close closes the connection to database
func (this ORM) Close() error {
    return this.db.Close()
}
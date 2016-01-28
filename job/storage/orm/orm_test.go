package orm

import (
    "fmt"
    "testing"
    "time"
    "github.com/ajvb/kala/job"
    "github.com/stretchr/testify/assert"
)

var (
    postgres, mysql *ORM
    mockPostgres, mockMySQL *job.Job
)

func init() {

    // tested with PostgreSQL 9.4.5 (hosted on heroku for testing purposes)
    user := "soufepkljhyieu"
    pass := "UWRxXE_8fp6ICzS6_ABUf60cJ9"
    host := "ec2-54-83-194-117.compute-1.amazonaws.com"
    port := "5432"
    database := "dd3dda1tlvvths"
    postgres = Open("postgres",fmt.Sprintf("%s:%s@%s:%s/%s",user,pass,host,port,database))
    postgres.db.LogMode(true)

    cachePostgres := job.NewMemoryJobCache(postgres)
    mockPostgres = job.GetMockJobWithGenericSchedule()
    mockPostgres.Init(cachePostgres)

    // tested with MySQL 5.6.23 (hosted on heroku for testing purposes)
    user = "ro7lqhta7ef33p13"
    pass = "ifn1cgf5spi5hvkf"
    host = "tviw6wn55xwxejwj.cbetxkdyhwsb.us-east-1.rds.amazonaws.com"
    port = "3306"
    database = "mhkb1rv7ionjkjrh"
    mysql = Open("mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",user,pass,host,port,database))
    mysql.db.LogMode(true)

    cacheMySQL := job.NewMemoryJobCache(mysql)
    mockMySQL = job.GetMockJobWithGenericSchedule()
    mockMySQL.Init(cacheMySQL)

}

func TestSaveJobPostgres(t *testing.T) {
    err := postgres.Save(mockPostgres)
    assert.NoError(t, err)
}

func TestSaveJobMySQL(t *testing.T) {
    err := mysql.Save(mockMySQL)
    assert.NoError(t, err)
}

func TestGetJobPostgres(t *testing.T) {

    dbjob, err := postgres.Get(mockPostgres.Id)
    assert.Nil(t, err)

    assert.WithinDuration(t, dbjob.NextRunAt, mockPostgres.NextRunAt, 100*time.Microsecond)
    assert.Equal(t, mockPostgres.Name, dbjob.Name)
    assert.Equal(t, mockPostgres.Id, dbjob.Id)
    assert.Equal(t, mockPostgres.Command, dbjob.Command)
    assert.Equal(t, mockPostgres.Schedule, dbjob.Schedule)
    assert.Equal(t, mockPostgres.Owner, dbjob.Owner)
    assert.Equal(t, mockPostgres.Metadata.SuccessCount, dbjob.Metadata.SuccessCount)

}

func TestGetJobMySQL(t *testing.T) {

    dbjob, err := mysql.Get(mockMySQL.Id)
    assert.Nil(t, err)

    assert.WithinDuration(t, dbjob.NextRunAt, mockMySQL.NextRunAt, 100*time.Microsecond)
    assert.Equal(t, mockMySQL.Name, dbjob.Name)
    assert.Equal(t, mockMySQL.Id, dbjob.Id)
    assert.Equal(t, mockMySQL.Command, dbjob.Command)
    assert.Equal(t, mockMySQL.Schedule, dbjob.Schedule)
    assert.Equal(t, mockMySQL.Owner, dbjob.Owner)
    assert.Equal(t, mockMySQL.Metadata.SuccessCount, dbjob.Metadata.SuccessCount)

}

func TestDeleteJobPostgres(t *testing.T) {

    err := postgres.Delete(mockPostgres.Id)
    assert.NoError(t, err)

    _, err = postgres.Get(mockPostgres.Id)
    assert.Error(t, err)

}

func TestDeleteJobMySQL(t *testing.T) {

    err := mysql.Delete(mockMySQL.Id)
    assert.NoError(t, err)

    _, err = mysql.Get(mockMySQL.Id)
    assert.Error(t, err)

}

func TestGetAllJobsPostgres(t *testing.T) {

    cache := job.NewMemoryJobCache(postgres)

    for i:=1; i<4; i++{
        theMockJob := job.GetMockJobWithGenericSchedule()
        theMockJob.Init(cache)
        theMockJob.Name = fmt.Sprintf("Mock Job %d",i)
        err := postgres.Save(theMockJob)
        assert.NoError(t, err)
    }

    jobs, err := postgres.GetAll()
    assert.Nil(t, err)
    assert.Equal(t, len(jobs), 3)

    for _, j := range jobs {
        postgres.Delete(j.Id)
    }

    jobs, err = postgres.GetAll()
    assert.Nil(t, err)
    assert.Equal(t, len(jobs), 0)

}

func TestGetAllJobsMySQL(t *testing.T) {

    cache := job.NewMemoryJobCache(mysql)

    for i:=1; i<4; i++{
        theMockJob := job.GetMockJobWithGenericSchedule()
        theMockJob.Init(cache)
        theMockJob.Name = fmt.Sprintf("Mock Job %d",i)
        err := mysql.Save(theMockJob)
        assert.NoError(t, err)
    }

    jobs, err := mysql.GetAll()
    assert.Nil(t, err)
    assert.Equal(t, len(jobs), 3)

    for _, j := range jobs {
        mysql.Delete(j.Id)
    }

    jobs, err = mysql.GetAll()
    assert.Nil(t, err)
    assert.Equal(t, len(jobs), 0)

}

func TestClosePostgres(t *testing.T) {
    err := postgres.Close()
    assert.Nil(t, err)
}

func TestCloseMySQL(t *testing.T) {
    err := mysql.Close()
    assert.Nil(t, err)
}
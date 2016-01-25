package orm

import (
    "fmt"
//    "errors"
    "testing"
//    "time"

    "github.com/ajvb/kala/job"
    "github.com/stretchr/testify/assert"
)

type testJob struct {
    Job   *job.Job
    Bytes []byte
}

var (
    mysql, postgres *ORM
    cache []testJob
)

func init() {

    // this db is hosted on heroku for testing (PostgreSQL 9.4.5)
    user := "whadsrtbjomjqg"
    pass := "pZqpD9-1VGO843dNDsSTw_GMEF"
    host := "ec2-54-83-204-228.compute-1.amazonaws.com"
    port := "5432"
    database := "d3d2vce93prhk7"
    postgres = Open("postgres",fmt.Sprintf("%s:%s@%s:%s/%s",user,pass,host,port,database))

    // this db is hosted on heroku for testing (MySQL 5.4.5)
    user = "ro7lqhta7ef33p13"
    pass = "ifn1cgf5spi5hvkf"
    host = "tviw6wn55xwxejwj.cbetxkdyhwsb.us-east-1.rds.amazonaws.com"
    port = "3306"
    database = "mhkb1rv7ionjkjrh"
    mysql = Open("mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",user,pass,host,port,database))

}

func TestSaveJob(t *testing.T) {

    cache := job.NewMemoryJobCache(postgres)
    genericMockJob := job.GetMockJobWithGenericSchedule()
    genericMockJob.Init(cache)

    err := postgres.Save(genericMockJob)
    assert.NotNil(t, err)

}

// testJobs initializes n testJobs
//func initTestJobs(n int) []testJob {
//    testJobs := []testJob{}
//
//    for i := 0; i < n; i++ {
//        j := job.GetMockJobWithGenericSchedule()
//        j.Init(cache)
//
//        bytes, err := j.Bytes()
//        if err != nil {
//            panic(err)
//        }
//        t := testJob{
//            Job:   j,
//            Bytes: bytes,
//        }
//
//        testJobs = append(testJobs, t)
//    }
//
//    return testJobs
//}

//
//func TestGetJob(t *testing.T) {
//    testJob := testJobs[0]
//
//    // Expect a HGET operation to be preformed with the job hash key and job ID
//    conn.Command("HGET", HashKey, testJob.Job.Id).
//    Expect(testJob.Bytes).
//    ExpectError(nil)
//
//    storedJob, err := db.Get(testJob.Job.Id)
//    assert.Nil(t, err)
//
//    assert.WithinDuration(t, storedJob.NextRunAt, testJob.Job.NextRunAt, 100*time.Microsecond)
//    assert.Equal(t, testJob.Job.Name, storedJob.Name)
//    assert.Equal(t, testJob.Job.Id, storedJob.Id)
//    assert.Equal(t, testJob.Job.Command, storedJob.Command)
//    assert.Equal(t, testJob.Job.Schedule, storedJob.Schedule)
//    assert.Equal(t, testJob.Job.Owner, storedJob.Owner)
//    assert.Equal(t, testJob.Job.Metadata.SuccessCount, storedJob.Metadata.SuccessCount)
//
//    // Test error handling
//    conn.Command("HGET", HashKey, testJob.Job.Id).
//    ExpectError(errors.New("Redis error"))
//
//    storedJob, err = db.Get(testJob.Job.Id)
//    assert.NotNil(t, err)
//}
//
//func TestDeleteJob(t *testing.T) {
//    testJob := testJobs[0]
//
//    // Expect a HDEL operation to be preformed with the job ID
//    conn.Command("HDEL", testJob.Job.Id).
//    Expect("ok").
//    ExpectError(nil)
//
//    err := db.Delete(testJob.Job.Id)
//    assert.Nil(t, err)
//
//    // Test error handling
//    conn.Command("HDEL", testJob.Job.Id).
//    ExpectError(errors.New("Redis error"))
//
//    err = db.Delete(testJob.Job.Id)
//    assert.NotNil(t, err)
//}
//
//func TestGetAllJobs(t *testing.T) {
//    // Expect a HVALS operation to be preformed with the job hash key
//    conn.Command("HVALS", HashKey).
//    Expect([]interface{}{
//        testJobs[0].Bytes,
//        testJobs[1].Bytes,
//        testJobs[2].Bytes,
//    }).
//    ExpectError(nil)
//
//    jobs, err := db.GetAll()
//    assert.Nil(t, err)
//
//    for i, j := range jobs {
//        assert.WithinDuration(t, testJobs[i].Job.NextRunAt, j.NextRunAt, 100*time.Microsecond)
//        assert.Equal(t, testJobs[i].Job.Name, j.Name)
//        assert.Equal(t, testJobs[i].Job.Id, j.Id)
//        assert.Equal(t, testJobs[i].Job.Command, j.Command)
//        assert.Equal(t, testJobs[i].Job.Schedule, j.Schedule)
//        assert.Equal(t, testJobs[i].Job.Owner, j.Owner)
//        assert.Equal(t, testJobs[i].Job.Metadata.SuccessCount, j.Metadata.SuccessCount)
//    }
//
//    // Test erorr handling
//    conn.Command("HVALS", HashKey).
//    ExpectError(errors.New("Redis error"))
//
//    jobs, err = db.GetAll()
//    assert.NotNil(t, err)
//}
//
//func TestNew(t *testing.T) {
//
//}
//
//func TestClose(t *testing.T) {
//    closedConn := false
//
//    conn.CloseMock = func() error {
//        closedConn = true
//        return nil
//    }
//
//    err := db.Close()
//
//    assert.Nil(t, err)
//    assert.True(t, closedConn)
//
//    // Reset closedConn
//    closedConn = false
//
//    conn.CloseMock = func() error {
//        closedConn = false
//        return errors.New("Error closing connection")
//    }
//
//    err = db.Close()
//
//    assert.NotNil(t, err)
//    assert.False(t, closedConn)
//}

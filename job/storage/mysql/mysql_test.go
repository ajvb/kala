// +build linux

package mysql

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	mysqltest "github.com/lestrrat-go/test-mysqld"
	"github.com/stretchr/testify/assert"

	"github.com/ajvb/kala/job"
)

func NewTestDb() (*DB, sqlmock.Sqlmock) {
	connection, m, _ := sqlmock.New()

	var db = &DB{
		conn: sqlx.NewDb(connection, "sqlmock"),
	}
	return db, m
}

func TestSaveAndGetJob(t *testing.T) {
	db, m := NewTestDb()

	cache := job.NewLockFreeJobCache(db)
	defer db.Close()

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)

	j, err := json.Marshal(genericMockJob)
	if assert.NoError(t, err) {
		m.ExpectBegin()
		m.ExpectPrepare("replace .*").
			ExpectExec().
			WithArgs(genericMockJob.Id, string(j)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		m.ExpectCommit()
		err := db.Save(genericMockJob)
		if assert.NoError(t, err) {
			m.ExpectQuery("select .*").
				WithArgs(genericMockJob.Id).
				WillReturnRows(sqlmock.NewRows([]string{"job"}).AddRow(j))
			j2, err := db.Get(genericMockJob.Id)
			if assert.Nil(t, err) {
				assert.WithinDuration(t, j2.NextRunAt, genericMockJob.NextRunAt, 400*time.Microsecond)
				assert.Equal(t, j2.Name, genericMockJob.Name)
				assert.Equal(t, j2.Id, genericMockJob.Id)
				assert.Equal(t, j2.Command, genericMockJob.Command)
				assert.Equal(t, j2.Schedule, genericMockJob.Schedule)
				assert.Equal(t, j2.Owner, genericMockJob.Owner)
				assert.Equal(t, j2.Metadata.SuccessCount, genericMockJob.Metadata.SuccessCount)
			}
		}
	}
}

func TestDeleteJob(t *testing.T) {
	db, m := NewTestDb()

	cache := job.NewLockFreeJobCache(db)

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)

	j, err := json.Marshal(genericMockJob)
	if assert.NoError(t, err) {

		m.ExpectBegin()
		m.ExpectPrepare("replace .*").
			ExpectExec().
			WithArgs(genericMockJob.Id, string(j)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		m.ExpectCommit()

		err := db.Save(genericMockJob)
		if assert.NoError(t, err) {

			// Delete it
			m.ExpectExec("delete .*").
				WithArgs(genericMockJob.Id).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err = db.Delete(genericMockJob.Id)
			assert.Nil(t, err)

			// Verify deletion
			m.ExpectQuery("select .*").
				WithArgs(genericMockJob.Id).
				WillReturnError(sql.ErrNoRows)

			k, err := db.Get(genericMockJob.Id)
			assert.Error(t, err)
			assert.Nil(t, k)
		}
	}

}

func TestSaveAndGetAllJobs(t *testing.T) {
	db, m := NewTestDb()

	genericMockJobOne := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJobTwo := job.GetMockJobWithGenericSchedule(time.Now())

	jobOne, err := json.Marshal(genericMockJobOne)
	if assert.NoError(t, err) {
		jobTwo, err := json.Marshal(genericMockJobTwo)
		if assert.NoError(t, err) {

			m.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"jobs"}).AddRow(jobOne).AddRow(jobTwo))

			jobs, err := db.GetAll()
			assert.Nil(t, err)
			assert.Equal(t, 2, len(jobs))
		}
	}

}

func TestRealDb(t *testing.T) {

	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		mysqld, err := mysqltest.NewMysqld(nil)
		if assert.NoError(t, err) {
			dsn = mysqld.Datasource("test", "", "", 0)
			defer mysqld.Stop()
		} else {
			t.FailNow()
		}
	}

	db := New(dsn, nil)
	cache := job.NewLockFreeJobCache(db)

	genericMockJobOne := job.GetMockJobWithGenericSchedule(time.Now().Add(time.Hour))
	genericMockJobTwo := job.GetMockJobWithGenericSchedule(time.Now().Add(time.Hour))

	genericMockJobOne.Init(cache)
	genericMockJobTwo.Init(cache)

	t.Logf("Mock job one: %s", genericMockJobOne.Id)
	t.Logf("Mock job two: %s", genericMockJobTwo.Id)

	err := db.Save(genericMockJobOne)
	if assert.NoError(t, err) {

		err := db.Save(genericMockJobTwo)
		if assert.NoError(t, err) {

			jobs, err := db.GetAll()
			if assert.NoError(t, err) {

				assert.Equal(t, 2, len(jobs))

				if jobs[0].Id == genericMockJobTwo.Id {
					jobs[0], jobs[1] = jobs[1], jobs[0]
				}

				assert.WithinDuration(t, jobs[0].NextRunAt, genericMockJobOne.NextRunAt, 400*time.Microsecond)
				assert.Equal(t, jobs[0].Name, genericMockJobOne.Name)
				assert.Equal(t, jobs[0].Id, genericMockJobOne.Id)
				assert.Equal(t, jobs[0].Command, genericMockJobOne.Command)
				assert.Equal(t, jobs[0].Schedule, genericMockJobOne.Schedule)
				assert.Equal(t, jobs[0].Owner, genericMockJobOne.Owner)
				assert.Equal(t, jobs[0].Metadata.SuccessCount, genericMockJobOne.Metadata.SuccessCount)

				assert.WithinDuration(t, jobs[1].NextRunAt, genericMockJobTwo.NextRunAt, 400*time.Microsecond)
				assert.Equal(t, jobs[1].Name, genericMockJobTwo.Name)
				assert.Equal(t, jobs[1].Id, genericMockJobTwo.Id)
				assert.Equal(t, jobs[1].Command, genericMockJobTwo.Command)
				assert.Equal(t, jobs[1].Schedule, genericMockJobTwo.Schedule)
				assert.Equal(t, jobs[1].Owner, genericMockJobTwo.Owner)
				assert.Equal(t, jobs[1].Metadata.SuccessCount, genericMockJobTwo.Metadata.SuccessCount)

				job2, err := db.Get(genericMockJobTwo.Id)
				if assert.NoError(t, err) {

					assert.WithinDuration(t, job2.NextRunAt, genericMockJobTwo.NextRunAt, 400*time.Microsecond)
					assert.Equal(t, job2.Name, genericMockJobTwo.Name)
					assert.Equal(t, job2.Id, genericMockJobTwo.Id)
					assert.Equal(t, job2.Command, genericMockJobTwo.Command)
					assert.Equal(t, job2.Schedule, genericMockJobTwo.Schedule)
					assert.Equal(t, job2.Owner, genericMockJobTwo.Owner)
					assert.Equal(t, job2.Metadata.SuccessCount, genericMockJobTwo.Metadata.SuccessCount)

					t.Logf("Deleting job with id %s", job2.Id)

					err := db.Delete(job2.Id)
					if assert.NoError(t, err) {

						jobs, err := db.GetAll()
						if assert.NoError(t, err) {

							assert.Equal(t, 1, len(jobs))

							assert.WithinDuration(t, jobs[0].NextRunAt, genericMockJobOne.NextRunAt, 400*time.Microsecond)
							assert.Equal(t, jobs[0].Name, genericMockJobOne.Name)
							assert.Equal(t, jobs[0].Id, genericMockJobOne.Id)
							assert.Equal(t, jobs[0].Command, genericMockJobOne.Command)
							assert.Equal(t, jobs[0].Schedule, genericMockJobOne.Schedule)
							assert.Equal(t, jobs[0].Owner, genericMockJobOne.Owner)
							assert.Equal(t, jobs[0].Metadata.SuccessCount, genericMockJobOne.Metadata.SuccessCount)

						}
					}
				}
			}
		}
	}
}

func TestPersistEpsilon(t *testing.T) {

	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		mysqld, err := mysqltest.NewMysqld(nil)
		if assert.NoError(t, err) {
			dsn = mysqld.Datasource("test", "", "", 0)
			defer mysqld.Stop()
		} else {
			t.FailNow()
		}
	}

	db := New(dsn, nil)
	defer db.Close()

	cache := job.NewMemoryJobCache(db)

	mockJob := job.GetMockRecurringJobWithSchedule(time.Now().Add(time.Second*1), "PT1H")
	mockJob.Epsilon = "PT1H"
	mockJob.Command = "asdf"
	mockJob.Retries = 2

	err := mockJob.Init(cache)
	if assert.NoError(t, err) {

		err := db.Save(mockJob)
		if assert.NoError(t, err) {

			retrieved, err := db.GetAll()
			if assert.NoError(t, err) {

				retrieved[0].Run(cache)

			}
		}
	}
}

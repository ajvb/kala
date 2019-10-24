package mysql

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/lestrrat-go/test-mysqld"

	"github.com/jmoiron/sqlx"

	"github.com/ajvb/kala/job"

	"github.com/stretchr/testify/assert"
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

	genericMockJob := job.GetMockJobWithGenericSchedule()
	genericMockJob.Init(cache)

	job, err := json.Marshal(genericMockJob)
	if assert.NoError(t, err) {
		m.ExpectBegin()
		m.ExpectPrepare("insert .*").
			ExpectExec().
			WithArgs(genericMockJob.Id, string(job)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		m.ExpectCommit()
		err := db.Save(genericMockJob)
		if assert.NoError(t, err) {
			m.ExpectQuery("select .*").
				WithArgs(genericMockJob.Id).
				WillReturnRows(sqlmock.NewRows([]string{"job"}).AddRow(job))
			j, err := db.Get(genericMockJob.Id)
			if assert.Nil(t, err) {
				assert.WithinDuration(t, j.NextRunAt, genericMockJob.NextRunAt, 400*time.Microsecond)
				assert.Equal(t, j.Name, genericMockJob.Name)
				assert.Equal(t, j.Id, genericMockJob.Id)
				assert.Equal(t, j.Command, genericMockJob.Command)
				assert.Equal(t, j.Schedule, genericMockJob.Schedule)
				assert.Equal(t, j.Owner, genericMockJob.Owner)
				assert.Equal(t, j.Metadata.SuccessCount, genericMockJob.Metadata.SuccessCount)
			}
		}
	}
}

func TestDeleteJob(t *testing.T) {
	db, m := NewTestDb()

	cache := job.NewLockFreeJobCache(db)

	genericMockJob := job.GetMockJobWithGenericSchedule()
	genericMockJob.Init(cache)

	job, err := json.Marshal(genericMockJob)
	if assert.NoError(t, err) {

		m.ExpectBegin()
		m.ExpectPrepare("insert .*").
			ExpectExec().
			WithArgs(genericMockJob.Id, string(job)).
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

	genericMockJobOne := job.GetMockJobWithGenericSchedule()
	genericMockJobTwo := job.GetMockJobWithGenericSchedule()

	jobOne, err := json.Marshal(genericMockJobOne)
	if assert.NoError(t, err) {
		jobTwo, err := json.Marshal(genericMockJobTwo)
		if assert.NoError(t, err) {

			// aggregate := fmt.Sprintf(`[%s,%s]`, jobOne, jobTwo)
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

	db := New(dsn)
	cache := job.NewLockFreeJobCache(db)

	genericMockJobOne := job.GetMockJobWithGenericSchedule()
	genericMockJobTwo := job.GetMockJobWithGenericSchedule()

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

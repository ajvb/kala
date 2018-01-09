package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/ajvb/kala/job"

	"github.com/stretchr/testify/assert"
)

func NewTestDb() (*DB, sqlmock.Sqlmock) {
	connection, m, _ := sqlmock.New()
	var db = &DB{
		conn: connection,
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
			WithArgs(string(job)).
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
			WithArgs(string(job)).
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
		if assert.NoError(t, err) {
			jobTwo, err := json.Marshal(genericMockJobTwo)

			aggregate := fmt.Sprintf(`[%s,%s]`, jobOne, jobTwo)
			m.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"jobs"}).AddRow(aggregate))

			jobs, err := db.GetAll()
			assert.Nil(t, err)
			assert.Equal(t, 2, len(jobs))
		}
	}

}

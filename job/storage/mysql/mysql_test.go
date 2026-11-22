package mysql

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/ajvb/kala/v2/job"
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
				ok, err := genericMockJob.Compare(j2)
				assert.True(t, ok, err)
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

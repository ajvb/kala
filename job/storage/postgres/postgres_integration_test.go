//go:build integration

package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ajvb/kala/v2/internal/containers"
	"github.com/ajvb/kala/v2/job"
)

// One PostgreSQL container for all the tests of this package.
var postgresContainer = containers.NewShared(containers.Spec{
	Repository: "postgres",
	Tag:        "11",
	Env:        []string{"POSTGRES_PASSWORD=secret", "POSTGRES_USER=user_name", "POSTGRES_DB=dbname"},
	Port:       "5432/tcp",
	Ready: func(hostPort string) error {
		conn, err := sql.Open("postgres", realDSN(hostPort))
		if err != nil {
			return err
		}
		defer conn.Close()
		return conn.Ping()
	},
})

func realDSN(hostPort string) string {
	return fmt.Sprintf("postgres://user_name:secret@%s/dbname?sslmode=disable&connect_timeout=2", hostPort)
}

func TestMain(m *testing.M) {
	code := m.Run()
	if err := postgresContainer.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not remove the PostgreSQL container:", err)
	}
	os.Exit(code)
}

// newRealDB connects to the shared container (failing the test if there is no
// Docker daemon) and deletes every job, so each test starts empty.
func newRealDB(t *testing.T) *DB {
	t.Helper()
	db := New(realDSN(postgresContainer.HostPort(t)))
	_, err := db.conn.Exec("delete from " + TABLE_NAME)
	assert.NoError(t, err)
	return db
}

func TestRealPostgres(t *testing.T) {
	db := newRealDB(t)
	cache := job.NewLockFreeJobCache(db)
	defer db.Close()

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)

	err := db.Save(genericMockJob)
	if assert.NoError(t, err) {
		j2, err := db.Get(genericMockJob.Id)
		if assert.Nil(t, err) {
			ok, err := genericMockJob.Compare(j2)
			assert.True(t, ok, err)
			err = db.Delete(j2.Id)
			if assert.Nil(t, err) {
				jobs, _ := db.GetAll()
				assert.Equal(t, 0, len(jobs))
			}
		}
	}
}

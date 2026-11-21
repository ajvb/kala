//go:build integration

package mysql

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

// One MySQL container for all the tests of this package.
var mysqlContainer = containers.NewShared(containers.Spec{
	Repository: "mysql",
	Tag:        "5.7",
	Env:        []string{"MYSQL_ALLOW_EMPTY_PASSWORD=1", "MYSQL_DATABASE=test"},
	Port:       "3306/tcp",
	Ready: func(hostPort string) error {
		conn, err := sql.Open("mysql", realDSN(hostPort))
		if err != nil {
			return err
		}
		defer conn.Close()
		return conn.Ping()
	},
})

func realDSN(hostPort string) string {
	return fmt.Sprintf("root:@tcp(%s)/test?timeout=2s&readTimeout=2s", hostPort)
}

func TestMain(m *testing.M) {
	code := m.Run()
	if err := mysqlContainer.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not remove the MySQL container:", err)
	}
	os.Exit(code)
}

// newRealDB connects to the shared container (failing the test if there is no
// Docker daemon) and deletes every job, so each test starts empty.
func newRealDB(t *testing.T) *DB {
	t.Helper()
	db := New(realDSN(mysqlContainer.HostPort(t)), nil)
	_, err := db.conn.Exec("delete from " + TableName)
	assert.NoError(t, err)
	return db
}

func TestRealDb(t *testing.T) {

	db := newRealDB(t)
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

				ok, err := genericMockJobOne.Compare(jobs[0])
				assert.True(t, ok, err)

				ok, err = genericMockJobTwo.Compare(jobs[1])
				assert.True(t, ok, err)

				job2, err := db.Get(genericMockJobTwo.Id)
				if assert.NoError(t, err) {

					ok, err := genericMockJobTwo.Compare(job2)
					assert.True(t, ok, err)

					t.Logf("Deleting job with id %s", job2.Id)

					err = db.Delete(job2.Id)
					if assert.NoError(t, err) {

						jobs, err := db.GetAll()
						if assert.NoError(t, err) {

							assert.Equal(t, 1, len(jobs))

							ok, err := genericMockJobOne.Compare(jobs[0])
							assert.True(t, ok, err)

						}
					}
				}
			}
		}
	}
}

func TestPersistEpsilon(t *testing.T) {

	db := newRealDB(t)
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

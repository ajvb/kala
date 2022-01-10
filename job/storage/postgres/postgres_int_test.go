// +build integration

package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ajvb/kala/job"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"

	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3/docker"
	log "github.com/sirupsen/logrus"
)

var db *sql.DB
var dbDocker *DB

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "11",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=user_name",
			"POSTGRES_DB=dbname",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://user_name:secret@%s/dbname?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url: ", databaseUrl)

	resource.Expire(120) // Tell docker to hard kill the container in 120 seconds

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		db, err = sql.Open("postgres", databaseUrl)
		if err != nil {
			return err
		}

		// passive attempt to create table
		_, _ = db.Exec(fmt.Sprintf(`create table %s (job jsonb);`, TableName))

		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	dbDocker = &DB{
		conn: db,
	}

	//Run tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestRealPostgres(t *testing.T) {
	cache := job.NewLockFreeJobCache(dbDocker)
	defer dbDocker.Close()

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)

	j, err := json.Marshal(genericMockJob)

	fmt.Println(string(j))

	if assert.NoError(t, err) {
		err := dbDocker.Save(genericMockJob)
		if assert.NoError(t, err) {
			j2, err := dbDocker.Get(genericMockJob.Id)
			if assert.Nil(t, err) {
				assert.WithinDuration(t, j2.NextRunAt, genericMockJob.NextRunAt, 400*time.Microsecond)
				assert.Equal(t, j2.Name, genericMockJob.Name)
				assert.Equal(t, j2.Id, genericMockJob.Id)
				assert.Equal(t, j2.Command, genericMockJob.Command)
				assert.Equal(t, j2.Schedule, genericMockJob.Schedule)
				assert.Equal(t, j2.Owner, genericMockJob.Owner)
				assert.Equal(t, j2.Metadata.SuccessCount, genericMockJob.Metadata.SuccessCount)
				err := dbDocker.Delete(j2.Id)
				if assert.Nil(t, err) {
					jobs, _ := dbDocker.GetAll()
					assert.Equal(t, len(jobs), 0)
				}
			}
		}
	}

}

//go:build integration

package consul

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ajvb/kala/v2/internal/containers"
	"github.com/ajvb/kala/v2/job"

	"github.com/stretchr/testify/assert"
)

// One Consul container for all the tests of this package.
var consulContainer = containers.NewShared(containers.Spec{
	Repository: "hashicorp/consul",
	Tag:        "1.20",
	Cmd:        []string{"agent", "-dev", "-client=0.0.0.0"},
	Port:       "8500/tcp",
	Ready: func(hostPort string) error {
		// The port opens before a leader is elected, and writes fail until then.
		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://" + hostPort + "/v1/status/leader")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if strings.Trim(strings.TrimSpace(string(body)), `"`) == "" {
			return fmt.Errorf("no leader elected yet")
		}
		return nil
	},
})

func TestMain(m *testing.M) {
	code := m.Run()
	if err := consulContainer.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "could not remove the Consul container:", err)
	}
	os.Exit(code)
}

// newTestDB connects to the shared container (failing the test if there is no
// Docker daemon) and deletes every job, so each test starts empty.
func newTestDB(t *testing.T) *ConsulJobDB {
	t.Helper()
	db := New(consulContainer.HostPort(t))

	jobs, err := db.GetAll()
	assert.NoError(t, err)
	for _, j := range jobs {
		assert.NoError(t, db.Delete(j.Id))
	}
	return db
}

func TestSaveAndGetJob(t *testing.T) {
	db := newTestDB(t)
	cache := job.NewLockFreeJobCache(db)
	defer db.Close()

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)
	db.Save(genericMockJob)

	j, err := db.Get(genericMockJob.Id)
	assert.Nil(t, err)

	ok, err := genericMockJob.Compare(j)
	assert.True(t, ok, err)
}

func TestDeleteJob(t *testing.T) {
	db := newTestDB(t)
	cache := job.NewLockFreeJobCache(db)

	genericMockJob := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJob.Init(cache)
	db.Save(genericMockJob)

	// Make sure its there
	j, err := db.Get(genericMockJob.Id)
	assert.Nil(t, err)
	assert.Equal(t, j.Name, genericMockJob.Name)
	retrievedJob, err := cache.Get(genericMockJob.Id)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedJob)

	// Delete it
	err = genericMockJob.Delete(cache)
	assert.Nil(t, err)
	//
	// fmt.Printf("%#v", genericMockJob)
	k, err := db.Get(genericMockJob.Id)
	assert.Error(t, err)
	assert.Nil(t, k)
	retrievedJobTwo, err := cache.Get(genericMockJob.Id)
	assert.Error(t, err)
	assert.Nil(t, retrievedJobTwo)

	genericMockJob.Delete(cache)
}

func TestSaveAndGetAllJobs(t *testing.T) {
	db := newTestDB(t)
	cache := job.NewLockFreeJobCache(db)
	defer db.Close()

	genericMockJobOne := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJobOne.Init(cache)
	err := db.Save(genericMockJobOne)
	assert.NoError(t, err)

	genericMockJobTwo := job.GetMockJobWithGenericSchedule(time.Now())
	genericMockJobTwo.Init(cache)
	err = db.Save(genericMockJobTwo)
	assert.NoError(t, err)

	jobs, err := db.GetAll()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(jobs))
}

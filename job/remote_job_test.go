package job

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// newRemoteJob returns an initialized remote job whose first scheduled run is
// an hour away, so the only runs are the ones a test starts.
func newRemoteJob(t *testing.T, props RemoteProperties) (*Job, JobCache) {
	t.Helper()
	j := GetMockRemoteJob(props)
	j.Schedule = "R/" + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + "/PT1H"
	cache := NewMockCache()
	assert.NoError(t, j.Init(cache))
	t.Cleanup(func() { assert.NoError(t, j.Delete(cache)) })
	return j, cache
}

func successCount(j *Job) uint {
	j.lock.RLock()
	defer j.lock.RUnlock()
	return j.Metadata.SuccessCount
}

func TestRemoteJobRunner(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer testServer.Close()

	mockRemoteJob, cache := newRemoteJob(t, RemoteProperties{Url: testServer.URL})
	mockRemoteJob.Run(cache)

	assert.Equal(t, uint(1), successCount(mockRemoteJob))
}

func TestRemoteJobBadStatus(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "something failed", http.StatusInternalServerError)
	}))
	defer testServer.Close()

	mockRemoteJob, cache := newRemoteJob(t, RemoteProperties{Url: testServer.URL})
	mockRemoteJob.Run(cache)

	assert.Equal(t, uint(0), successCount(mockRemoteJob))
}

func TestRemoteJobBadStatusSuccess(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "something failed", http.StatusInternalServerError)
	}))
	defer testServer.Close()

	mockRemoteJob, cache := newRemoteJob(t, RemoteProperties{
		Url:                   testServer.URL,
		ExpectedResponseCodes: []int{500},
	})
	mockRemoteJob.Run(cache)

	assert.Equal(t, uint(1), successCount(mockRemoteJob))
}

// Runs of the same remote job can overlap. Each request gets the default
// Content-Type and the default expected status, and the job itself is left
// unchanged (writing its header map from two runs at once crashes the process).
func TestOverlappingRemoteJobRunsLeaveTheJobUnchanged(t *testing.T) {
	var (
		mu           sync.Mutex
		contentTypes []string
	)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		contentTypes = append(contentTypes, r.Header.Get("Content-Type"))
		mu.Unlock()
	}))
	defer testServer.Close()

	mockRemoteJob, cache := newRemoteJob(t, RemoteProperties{Url: testServer.URL})

	const runs = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range runs {
		wg.Go(func() {
			<-start
			mockRemoteJob.Run(cache)
		})
	}
	close(start)
	wg.Wait()

	for _, ct := range contentTypes {
		assert.Equal(t, "application/json", ct)
	}
	assert.Len(t, contentTypes, runs)
	mockRemoteJob.lock.RLock()
	defer mockRemoteJob.lock.RUnlock()
	assert.Nil(t, mockRemoteJob.RemoteProperties.Headers, "a run changed the job's headers")
	assert.Empty(t, mockRemoteJob.RemoteProperties.ExpectedResponseCodes, "a run changed the job's expected codes")
}

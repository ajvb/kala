//go:build !race
// +build !race

package job

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRemoteJobRunner(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer testServer.Close()

	mockRemoteJob := GetMockRemoteJob(RemoteProperties{
		Url: testServer.URL,
	})

	cache := NewMockCache()
	mockRemoteJob.Init(cache)
	cache.Start(0, 2*time.Second) // Retain 1 minute

	mockRemoteJob.Run(cache)

	mockRemoteJob.lock.RLock()
	assert.True(t, mockRemoteJob.Metadata.SuccessCount == 1)
	mockRemoteJob.lock.RUnlock()
}

func TestRemoteJobBadStatus(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "something failed", http.StatusInternalServerError)
	}))
	defer testServer.Close()

	mockRemoteJob := GetMockRemoteJob(RemoteProperties{
		Url: testServer.URL,
	})

	cache := NewMockCache()
	mockRemoteJob.Init(cache)
	cache.Start(0, 2*time.Second) // Retain 1 minute

	mockRemoteJob.Run(cache)
	assert.True(t, mockRemoteJob.Metadata.SuccessCount == 0)
}

func TestRemoteJobBadStatusSuccess(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "something failed", http.StatusInternalServerError)
	}))
	defer testServer.Close()

	mockRemoteJob := GetMockRemoteJob(RemoteProperties{
		Url:                   testServer.URL,
		ExpectedResponseCodes: []int{500},
	})

	cache := NewMockCache()
	mockRemoteJob.Init(cache)
	cache.Start(0, 2*time.Second) // Retain 1 minute

	mockRemoteJob.Run(cache)

	mockRemoteJob.lock.Lock()
	assert.True(t, mockRemoteJob.Metadata.SuccessCount == 1)
	mockRemoteJob.lock.Unlock()
}

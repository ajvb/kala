package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajvb/kala/v2/job"
	log "github.com/sirupsen/logrus"
)

// BenchmarkListJobs measures listing 1,000 jobs through the API handler.
func BenchmarkListJobs(b *testing.B) {
	previous := log.StandardLogger().Out
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(previous) })

	cache := job.NewMockCache()
	start := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 1000 {
		j := job.GetMockJobWithGenericSchedule(start)
		j.Id = fmt.Sprintf("job-%04d", i)
		if err := cache.Set(j); err != nil {
			b.Fatal(err)
		}
	}
	handler := HandleListJobsRequest(cache)

	for b.Loop() {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest(http.MethodGet, ApiJobPath, http.NoBody))
		if w.Code != http.StatusOK {
			b.Fatalf("status %d", w.Code)
		}
	}
}

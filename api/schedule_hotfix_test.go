package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ajvb/kala/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A schedule string that crashed v1 now gets HTTP 400 and the
// job cache is unchanged.
func TestAddJobWithInvalidScheduleReturns400AndLeavesCacheUnchanged(t *testing.T) {
	for _, schedule := range []string{
		"X/2030-01-01T00:00:00Z/PT1H", // the reproducer from the review
		"5/2030-01-01T00:00:00Z/PT1H",
		"R/2030-01-01T00:00:00Z/garbage",
		"RR5/2030-01-01T00:00:00Z/PT1H",
	} {
		cache := job.NewMockCache()
		body, err := json.Marshal(map[string]string{
			"name": "bad", "command": "echo hi", "schedule": schedule,
		})
		require.NoError(t, err)

		w, req := setupTestReq(t, "POST", ApiJobPath, body)
		require.NotPanics(t, func() { HandleAddJob(cache, "")(w, req) }, schedule)

		assert.Equal(t, http.StatusBadRequest, w.Code, schedule)
		var respErr apiError
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respErr), schedule)
		assert.Equal(t, "Error occurred when initializing the job", respErr.Error, schedule)
		assert.Empty(t, cache.GetAll().Jobs, "cache must be unchanged for %q", schedule)
	}
}

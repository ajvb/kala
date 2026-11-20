package job

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertLockUsable fails if the job's lock is left held (or otherwise broken).
func assertLockUsable(t *testing.T, j *Job) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		func() {
			j.lock.Lock()
			defer j.lock.Unlock()
		}()
		func() {
			j.lock.RLock()
			defer j.lock.RUnlock()
		}()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("job lock was left held after Init returned")
	}
}

// No schedule string may panic or leave a lock in the wrong
// state; an invalid schedule returns an error and leaves the cache unchanged.
func TestInvalidScheduleReturnsErrorAndLeavesCacheUnchanged(t *testing.T) {
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	cases := map[string]string{
		// The reproducer from the review: first part does not start with "R".
		"reproducer X prefix":     "X/2030-01-01T00:00:00Z/PT1H",
		"lowercase r":             "r/" + future + "/PT1H",
		"digits only repeat":      "5/" + future + "/PT1H",
		"empty repeat part":       "/" + future + "/PT1H",
		"repeat not a number":     "Rabc/" + future + "/PT1H",
		"repeat overflows int64":  "R99999999999999999999/" + future + "/PT1H",
		"bad start time":          "R/not-a-time/PT1H",
		"bad duration":            "R/" + future + "/garbage",
		"empty duration repeated": "R/" + future + "/",
		"two parts":               "R/" + future,
		"four parts":              "R/" + future + "/PT1H/extra",
		"double R repeat":         "RR5/" + future + "/PT1H",
		"start in the past":       "R/2000-01-01T00:00:00Z/PT1H",
		"bad epsilon":             "R/" + future + "/PT1H",
	}
	for name, schedule := range cases {
		schedule := schedule
		t.Run(name, func(t *testing.T) {
			cache := NewMockCache()
			j := GetMockJob()
			j.Schedule = schedule
			if strings.HasPrefix(name, "bad epsilon") {
				j.Epsilon = "not-a-duration"
			}

			var err error
			require.NotPanics(t, func() { err = j.Init(cache) })

			require.Error(t, err, "schedule %q must be rejected", schedule)
			if name == "reproducer X prefix" {
				assert.Contains(t, err.Error(), `repeat part "X"`, "error must name the offending part")
			}
			assert.Empty(t, cache.GetAll().Jobs, "cache must be unchanged after a rejected schedule")
			assert.Nil(t, j.jobTimer, "no timer may be started for a rejected schedule")
			assertLockUsable(t, j)
		})
	}
}

// Timers are not stopped in these tests: Timer.Stop on the mixer/clock timer
// can hang on current Go (it copies a time.Timer by value). The timers are
// for times far enough ahead that they never fire during the test run.
//
// Valid schedules, including the one-off R0/<start>/ form,
// must still be accepted (boundary against over-rejecting).
func TestValidSchedulesStillAccepted(t *testing.T) {
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	cases := map[string]struct {
		schedule string
		repeat   int64 // expected value worked out by hand from the schedule
	}{
		"repeat forever":       {"R/" + future + "/PT1H", -1},
		"repeat twice":         {"R2/" + future + "/PT1H", 2},
		"one-off empty period": {"R0/" + future + "/", 0},
		"no timezone":          {"R/" + strings.TrimSuffix(future, "Z") + "/PT1H", -1},
		// Legacy forms v1 accepts today (legacy forms). Pinned so a later change is deliberate.
		"legacy negative repeat": {"R-5/" + future + "/PT1H", -5},
		"legacy XR5 reads as R5": {"XR5/" + future + "/PT1H", 5},
	}
	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			cache := NewMockCache()
			j := GetMockJob()
			j.Schedule = c.schedule

			require.NoError(t, j.Init(cache))
			assert.Len(t, cache.GetAll().Jobs, 1)
			assert.NotNil(t, j.jobTimer)
			assert.Equal(t, c.repeat, j.timesToRepeat)
			assertLockUsable(t, j)
		})
	}
}

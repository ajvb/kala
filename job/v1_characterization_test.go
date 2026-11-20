package job

// Characterization tests: these record what v1 does TODAY, so that later work
// rests on facts. A failing test here means v1 behavior changed, not that v1
// is "right". Where v1 behavior looks surprising, a comment says so.
//
// They are deterministic: they call the decision code directly (Init,
// InitDelayDuration, GetWaitDuration, ShouldStartWaiting, Enable, retention)
// and never wait on a real timer, and they never call Timer.Stop (it can hang
// on current Go because mixer/clock copies a time.Timer by value).
// End-to-end timing with the real clock was observed by running a server
// (R2 ran 3 times; R0 and R-2 ran once; enabling a disabled job after a
// restart ran 0 times) and is also covered by the existing
// TestTwoTimesRecurringJobIsDoneAfterThirdRun.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timingTolerance = 5 * time.Second

// loadedJob mimics a stored job read back at startup: InitDelayDuration is
// called with checkTime=false, so a start time in the past is accepted.
func loadedJob(t *testing.T, schedule string) *Job {
	t.Helper()
	j := GetMockJob()
	j.Id = "loaded-job"
	j.Schedule = schedule
	require.NoError(t, j.InitDelayDuration(false))
	return j
}

func rfc3339Ago(d time.Duration) string {
	return time.Now().Add(-d).UTC().Format(time.RFC3339)
}

// What v1 does for a job whose start time has passed.
// Expected values are worked out by hand from the interval (PT1H = 1 hour).
func TestV1StartTimePassed(t *testing.T) {
	t.Run("never ran: waits one full interval, does not run immediately", func(t *testing.T) {
		j := loadedJob(t, "R/"+rfc3339Ago(5*time.Hour)+"/PT1H")
		assert.InDelta(t, float64(time.Hour), float64(j.GetWaitDuration()), float64(timingTolerance))
	})
	t.Run("ran 10 minutes ago: waits interval minus 10 minutes", func(t *testing.T) {
		j := loadedJob(t, "R/"+rfc3339Ago(5*time.Hour)+"/PT1H")
		j.Metadata.LastAttemptedRun = time.Now().Add(-10 * time.Minute)
		assert.InDelta(t, float64(50*time.Minute), float64(j.GetWaitDuration()), float64(timingTolerance))
	})
	t.Run("last run more than one interval ago: wait is negative (fires at once)", func(t *testing.T) {
		j := loadedJob(t, "R/"+rfc3339Ago(5*time.Hour)+"/PT1H")
		j.Metadata.LastAttemptedRun = time.Now().Add(-3 * time.Hour)
		assert.InDelta(t, float64(-2*time.Hour), float64(j.GetWaitDuration()), float64(timingTolerance))
	})
	t.Run("ResumeAtNextScheduledTime: next point on the original grid", func(t *testing.T) {
		// Start 2h30m ago with a 1h interval: grid points are 1.5h, 0.5h ago, then 30 minutes ahead.
		j := loadedJob(t, "R/"+rfc3339Ago(150*time.Minute)+"/PT1H")
		j.ResumeAtNextScheduledTime = true
		assert.InDelta(t, float64(30*time.Minute), float64(j.GetWaitDuration()), float64(timingTolerance))
	})
	t.Run("one-off R0: runs immediately (wait 0)", func(t *testing.T) {
		j := loadedJob(t, "R0/"+rfc3339Ago(time.Hour)+"/")
		assert.Equal(t, time.Duration(0), j.GetWaitDuration())
	})
	t.Run("Init (new job) rejects a start in the past", func(t *testing.T) {
		j := GetMockJob()
		j.Schedule = "R/" + rfc3339Ago(time.Hour) + "/PT1H"
		err := j.Init(NewMockCache())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be scheduled")
	})
	// Note: v1 does not have one rule for a passed start time: new jobs are rejected, loaded jobs wait one interval
	// from their last run (or from now, if they never ran), and a missed run
	// is never "caught up".
}

// runsBeforeStopping returns how many runs v1 performs for a schedule:
// Init always arms the first run, then after each run ShouldStartWaiting
// decides whether to continue, based only on len(Stats). Returns -1 when
// still continuing after limit runs ("forever").
func runsBeforeStopping(t *testing.T, schedule string, limit int) int {
	t.Helper()
	j := loadedJob(t, schedule)
	for n := 1; n <= limit; n++ {
		j.Stats = make([]*JobStat, n)
		if !j.ShouldStartWaiting() {
			return n
		}
	}
	return -1
}

// Legacy repeat forms. The counts match a real-clock run of a server
// (R2 ran 3 times; R0 and R-2 ran once).
func TestV1RepeatCounts(t *testing.T) {
	start := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	cases := []struct {
		repeat   string
		duration string
		runs     int // -1 = never stops
	}{
		{"R0", "PT1H", 1},
		{"R0", "", 1},
		{"R1", "PT1H", 2},
		{"R2", "PT1H", 3},
		{"R-2", "PT1H", 1},
		{"R-5", "PT1H", 1},
		// Surprise: -1 is the internal marker for "forever", so R-1 repeats forever.
		{"R-1", "PT1H", -1},
		{"R", "PT1H", -1},
	}
	for _, c := range cases {
		c := c
		t.Run(c.repeat+"/"+c.duration, func(t *testing.T) {
			assert.Equal(t, c.runs, runsBeforeStopping(t, c.repeat+"/"+start+"/"+c.duration, 50))
		})
	}
	// Note: R-n runs once for n >= 2, but R-1 runs forever in v1. R2 runs 3
	// times (N+1): the number is repeats after the first run, not total runs.
}

// The one-off form R0/<start>/ is accepted by v1; R0/<start> (no
// trailing slash) is not, and R0 skips the duration check entirely.
func TestV1OneOffForm(t *testing.T) {
	start := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	t.Run("R0/<start>/ is accepted, runs once, has no delay duration", func(t *testing.T) {
		j := GetMockJob()
		j.Schedule = "R0/" + start + "/"
		require.NoError(t, j.Init(NewMockCache()))
		assert.Equal(t, int64(0), j.timesToRepeat)
		assert.Nil(t, j.delayDuration)
		assert.Equal(t, 1, runsBeforeStopping(t, j.Schedule, 5))
	})
	t.Run("R0/<start> without the trailing slash is rejected by v1", func(t *testing.T) {
		j := GetMockJob()
		j.Schedule = "R0/" + start
		err := j.Init(NewMockCache())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Schedule not formatted correctly")
	})
	t.Run("R0 accepts a garbage duration because it is never parsed", func(t *testing.T) {
		j := GetMockJob()
		j.Schedule = "R0/" + start + "/not-a-duration"
		assert.NoError(t, j.Init(NewMockCache()))
	})
	// Note: v1 rejects R0/<start> (no slash). The ISO 8601 duration is not
	// checked for R0: v1 accepts R0/<start>/garbage.
}

// With stats retention on, does v1 exceed N+1 runs? Yes. v1 counts runs
// as len(Stats), and retention deletes old Stats, so the count can fall.
func TestV1RetentionBreaksRepeatLimit(t *testing.T) {
	t.Run("without retention R2 stops after 3 runs", func(t *testing.T) {
		j := loadedJob(t, "R2/"+rfc3339Ago(time.Hour)+"/PT1H")
		j.Stats = GetMockJobStats(time.Now(), 3)
		assert.False(t, j.ShouldStartWaiting())
	})
	t.Run("after retention removes old stats a finished R2 job is eligible to run again", func(t *testing.T) {
		j := loadedJob(t, "R2/"+rfc3339Ago(time.Hour)+"/PT1H")
		j.Stats = GetMockJobStats(time.Now().Add(-48*time.Hour), 3) // all 48h old
		require.False(t, j.ShouldStartWaiting())

		cache := NewMockCache()
		cache.retentionPeriod = time.Hour
		cache.compactJobStats(j)

		assert.Len(t, j.Stats, 0, "retention removed all three entries")
		assert.True(t, j.ShouldStartWaiting(), "v1 would schedule runs 4, 5, ... for an R2 job")
	})
	t.Run("retention shorter than the interval means the job never reaches its limit", func(t *testing.T) {
		j := loadedJob(t, "R2/"+rfc3339Ago(time.Hour)+"/PT1H")
		cache := NewMockCache()
		cache.retentionPeriod = time.Minute
		// Chronological history: one hour passes between runs (every earlier
		// stat is aged by an hour), retention runs, then the new run adds a stat.
		for run := 1; run <= 10; run++ {
			for _, st := range j.Stats {
				st.RanAt = st.RanAt.Add(-time.Hour)
			}
			cache.compactJobStats(j)
			j.Stats = append(j.Stats, &JobStat{RanAt: time.Now()})
			assert.Len(t, j.Stats, 1, "after run %d only the newest stat is kept", run)
			assert.True(t, j.ShouldStartWaiting(), "after run %d (more than the 3 an R2 job allows)", run)
		}
	})
	t.Run("the finished-run counter is not affected by retention", func(t *testing.T) {
		j := loadedJob(t, "R2/"+rfc3339Ago(time.Hour)+"/PT1H")
		j.Metadata.NumberOfFinishedRuns = 3
		j.Stats = GetMockJobStats(time.Now().Add(-48*time.Hour), 3)
		cache := NewMockCache()
		cache.retentionPeriod = time.Hour
		cache.compactJobStats(j)
		assert.Equal(t, uint(3), j.Metadata.NumberOfFinishedRuns)
	})
	// Note: v1 uses len(Stats) for the repeat limit, so retention changes it.
	// NumberOfFinishedRuns is a counter that retention does not change.
}

type fixedDB struct {
	MockDB
	jobs []*Job
}

func (f *fixedDB) GetAll() ([]*Job, error) { return f.jobs, nil }

// A job stored as disabled and loaded after a restart. Enabling it
// does nothing: no timer is armed, so it never runs.
func TestV1EnableAfterRestartDoesNotArmTrigger(t *testing.T) {
	// A start time shortly in the past, as it would be after a restart.
	j := loadedJob(t, "R/"+rfc3339Ago(time.Minute)+"/PT1H")
	j.Disabled = true

	// A stored job carries the NextRunAt it had before the restart.
	stale := time.Now().Add(-time.Hour)
	j.NextRunAt = stale

	db := &fixedDB{jobs: []*Job{j}}
	cache := NewLockFreeJobCache(db)
	// Note: Start also registers a signal handler goroutine that lives until
	// the test process ends (on SIGINT it exits 0). Accepted for this test; it
	// is the only way to run v1's real load path.
	cache.Start(0, 0)

	assert.Nil(t, j.jobTimer, "a disabled job is loaded without a timer")
	require.NoError(t, cache.Enable(j))

	assert.False(t, j.Disabled, "Enable clears the flag")
	assert.Nil(t, j.jobTimer, "but Enable arms nothing: it only restarts a job that already has a timer")
	assert.True(t, j.NextRunAt.Equal(stale), "and the stale next-run time is left as it was")
	// Note: enabling should arm the next trigger, but v1 does not for a job
	// loaded disabled at startup. The real-clock result was 0 runs after the
	// restart, against 3 for a no-restart control.
}

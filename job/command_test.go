package job

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ajvb/kala/v2/internal/fakecommands"
)

// A local job's command is parsed and handed to the CommandRunner of the cache
// the job was added to.
func TestJobRunsItsCommandWithTheCachesCommandRunner(t *testing.T) {
	clk := newTestClock()
	cmds := fakecommands.New()
	cache := NewMockCache(WithClock(clk), WithCommandRunner(cmds))

	j := GetMockJobWithGenericSchedule(clk.Now())
	j.Command = `echo "hello world" again`
	require.NoError(t, j.Init(cache))
	j.Run(cache)

	assert.Equal(t, [][]string{{"echo", "hello world", "again"}}, cmds.Ran())
	j.lock.RLock()
	defer j.lock.RUnlock()
	assert.Equal(t, uint(1), j.Metadata.SuccessCount)
}

// A cache keeps ExecRunner when given a nil runner; a cache that carries no
// runner, and a job that was never added to a cache, use ExecRunner too.
func TestExecRunnerIsTheDefault(t *testing.T) {
	assert.Equal(t, ExecRunner{}, NewLockFreeJobCache(&MockDB{}, WithCommandRunner(nil)).CommandRunner())
	assert.Equal(t, ExecRunner{}, NewMemoryJobCache(&MockDB{}, WithCommandRunner(nil)).CommandRunner())
	assert.Equal(t, ExecRunner{}, commandRunnerOf(cacheWithoutRunner{}))
	assert.Equal(t, ExecRunner{}, (&Job{}).commandRunner())
}

// cacheWithoutRunner is a JobCache that carries no CommandRunner.
type cacheWithoutRunner struct{ JobCache }

// Jobs loaded by a memory cache's Start take the cache's clock and runner.
func TestMemoryCacheStartGivesLoadedJobsItsClockAndRunner(t *testing.T) {
	clk := newTestClock()
	cmds := fakecommands.New()
	j := GetMockJobWithGenericSchedule(clk.Now())
	j.Id = "loaded"
	db := &MockDBGetAll{response: []*Job{j}}
	cache := NewMemoryJobCache(db, WithClock(clk), WithCommandRunner(cmds))

	cache.Start(0)

	assert.Same(t, clk, j.clock())
	assert.Same(t, cmds, j.commandRunner())
}

// failingSetDB is a JobDB whose Save always fails.
type failingSetDB struct{ MockDB }

func (failingSetDB) Save(*Job) error { return errors.New("storage is down") }

// When persisting fails, Enable reports the error, leaves the job disabled and
// starts no timer.
func TestEnableThatCannotPersistLeavesTheJobDisabled(t *testing.T) {
	clk := newTestClock()
	cache := NewLockFreeJobCache(&MockDB{}, WithClock(clk), WithCommandRunner(fakecommands.New()))
	j := GetMockJobWithGenericSchedule(clk.Now())
	require.NoError(t, j.Init(cache))
	require.NoError(t, j.Disable(cache))
	cache.jobDB = &failingSetDB{}
	cache.PersistOnWrite = true

	assert.Error(t, j.Enable(cache))

	j.lock.RLock()
	defer j.lock.RUnlock()
	assert.True(t, j.Disabled)
	clk.Advance(24 * time.Hour)
	assert.Equal(t, 0, clk.Fired(), "no timer was started")
}

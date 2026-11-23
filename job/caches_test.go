package job

import (
	"runtime"
	"testing"
	"time"

	"github.com/ajvb/kala/v2/internal/fakecommands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file contains tests that can be run against all types of JobCaches.

func TestCache(t *testing.T) {

	tt := []struct {
		c JobCache
	}{
		{
			c: NewLockFreeJobCache(&MockDB{}, WithCommandRunner(fakecommands.New())),
		},
		{
			c: NewMemoryJobCache(&MockDB{}, WithCommandRunner(fakecommands.New())),
		},
	}

	for _, row := range tt {
		t.Run("", func(t *testing.T) {
			testCache(t, row.c)
		})
	}

}

func testCache(t *testing.T, cache JobCache) {

	t.Run("TestCacheDeleteJobNotFound", func(t *testing.T) {
		err := cache.Delete("not-a-real-id")
		assert.Equal(t, ErrJobDoesntExist, err)
	})

	t.Run("TestCachePersist", func(t *testing.T) {
		err := cache.Persist()
		assert.NoError(t, err)
	})

}

func TestCachePersistence(t *testing.T) {

	mdb1a := NewMemoryDB()
	cache1a := NewMemoryJobCache(mdb1a, WithCommandRunner(fakecommands.New()))

	mdb1b := NewMemoryDB()
	cache1b := NewMemoryJobCache(mdb1b, WithCommandRunner(fakecommands.New()))
	cache1b.PersistOnWrite = true

	mdb2a := NewMemoryDB()
	cache2a := NewLockFreeJobCache(mdb2a, WithCommandRunner(fakecommands.New()))

	mdb2b := NewMemoryDB()
	cache2b := NewLockFreeJobCache(mdb2b, WithCommandRunner(fakecommands.New()))
	cache2b.PersistOnWrite = true

	tt := []struct {
		db            JobDB
		c             JobCache
		shouldPersist bool
	}{
		{
			db:            mdb1a,
			c:             cache1a,
			shouldPersist: false,
		},
		{
			db:            mdb1b,
			c:             cache1b,
			shouldPersist: true,
		},
		{
			db:            mdb2a,
			c:             cache2a,
			shouldPersist: false,
		},
		{
			db:            mdb2b,
			c:             cache2b,
			shouldPersist: true,
		},
	}

	for _, row := range tt {
		t.Run("", func(t *testing.T) {
			testCachePersistence(t, row.c, row.db, row.shouldPersist)
		})
	}

}

// This battery of tests ensures that a JobCache behaves appropriately from a persistence perspective.
// We verify that it's either persisting to the JobDB upon each creation/update/delete, or not, as appropriate.
// Note that deletes always propagate to the db, and return an error if that fails,
// but if transactional persistence is turned off the cache will still delete from itself.
func testCachePersistence(t *testing.T, cache JobCache, db JobDB, shouldPersist bool) {

	t.Run("testCachePersistence", func(t *testing.T) {

		// The job runs once at Init; wait for that run, so it cannot overlap
		// the disable, enable and delete below.
		j := GetMockJob()
		j.ranChan = make(chan struct{}, 1)
		assert.NoError(t, j.Init(cache)) // Saves the job
		awaitJobRan(t, j, 5*time.Second)

		saved, err := db.Get(j.Id)
		if shouldPersist {
			assert.NoError(t, err)
			assert.Equal(t, j.Id, saved.Id)
		} else {
			assert.IsType(t, ErrJobNotFound(""), err)
			assert.Equal(t, (*Job)(nil), saved)
		}

		t.Run("disable", func(t *testing.T) {
			j.Disable(cache)
			ret, err := db.Get(j.Id)
			if shouldPersist {
				assert.NoError(t, err)
				assert.Equal(t, true, ret.Disabled)
			} else {
				assert.IsType(t, ErrJobNotFound(""), err)
				assert.Equal(t, (*Job)(nil), ret)
			}

		})

		t.Run("enable", func(t *testing.T) {
			j.Enable(cache)
			ret, err := db.Get(j.Id)
			if shouldPersist {
				assert.NoError(t, err)
				assert.Equal(t, !shouldPersist, ret.Disabled)
			} else {
				assert.IsType(t, ErrJobNotFound(""), err)
				assert.Equal(t, (*Job)(nil), ret)
			}
		})

		t.Run("delete", func(t *testing.T) {

			// If we haven't been persisting, persist it to the DB now
			// because we need it there for this test.
			if !shouldPersist {
				assert.NoError(t, cache.Persist())
			}

			assert.NoError(t, cache.Delete(j.Id))
			ret, err := db.Get(j.Id)
			assert.IsType(t, ErrJobNotFound(""), err)
			assert.Equal(t, (*Job)(nil), ret)

			t.Run("errored", func(t *testing.T) {

				j := GetMockJob()
				j.ranChan = make(chan struct{}, 1)
				assert.NoError(t, j.Init(cache))
				awaitJobRan(t, j, 5*time.Second)
				if !shouldPersist {
					assert.NoError(t, cache.Persist())
				}
				assert.NoError(t, db.Delete(j.Id))

				assert.Error(t, cache.Delete(j.Id))
				ret, err := cache.Get(j.Id)
				if shouldPersist {
					assert.NoError(t, err)
					assert.Equal(t, j.Id, ret.Id)
				} else {
					assert.Error(t, err)
					assert.Equal(t, (*Job)(nil), ret)
				}

			})

		})

	})

}

// Deleting a job while it runs must not deadlock. The run holds the job's lock
// while it reads the cache to start the job's children, so Delete must not
// hold the cache's lock while it waits for the job's.
func TestDeletingARunningJobDoesNotDeadlock(t *testing.T) {
	for name, newCache := range map[string]func(CacheOption) JobCache{
		"memory":    func(o CacheOption) JobCache { return NewMemoryJobCache(&MockDB{}, o) },
		"lock-free": func(o CacheOption) JobCache { return NewLockFreeJobCache(&MockDB{}, o) },
	} {
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			cmds := fakecommands.New()
			cmds.Handle("block", func([]string) (string, error) {
				close(started)
				<-release
				return "", nil
			})
			cache := newCache(WithCommandRunner(cmds))

			parent := GetMockJobWithGenericSchedule(time.Now())
			parent.Command = "block"
			require.NoError(t, parent.Init(cache))
			child := GetMockJobWithGenericSchedule(time.Now())
			child.ParentJobs = []string{parent.Id}
			require.NoError(t, child.Init(cache))

			ran := make(chan struct{})
			go func() {
				parent.Run(cache)
				close(ran)
			}()
			<-started

			deleted := make(chan error, 1)
			go func() { deleted <- cache.Delete(parent.Id) }()
			// A waiting writer makes TryRLock fail: Delete is now waiting for
			// the job's lock while the run holds it.
			for parent.lock.TryRLock() {
				parent.lock.RUnlock()
				runtime.Gosched()
			}
			close(release)

			for _, done := range []struct {
				what string
				ch   <-chan struct{}
			}{{"Delete", errDone(deleted, t)}, {"the run", ran}} {
				select {
				case <-done.ch:
				case <-time.After(5 * time.Second):
					t.Fatalf("%s deadlocked", done.what)
				}
			}
			cache.(interface{ waitForCleanups() }).waitForCleanups()
		})
	}
}

// errDone returns a channel that is closed once errs delivers a nil error;
// a non-nil error fails the test.
func errDone(errs <-chan error, t *testing.T) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		assert.NoError(t, <-errs)
		close(done)
	}()
	return done
}

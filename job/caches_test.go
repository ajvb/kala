package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This file contains tests that can be run against all types of JobCaches.

func TestCache(t *testing.T) {

	tt := []struct {
		c JobCache
	}{
		{
			c: NewLockFreeJobCache(&MockDB{}),
		},
		{
			c: NewMemoryJobCache(&MockDB{}),
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
	cache1a := NewMemoryJobCache(mdb1a)

	mdb1b := NewMemoryDB()
	cache1b := NewMemoryJobCache(mdb1b)
	cache1b.PersistOnWrite = true

	mdb2a := NewMemoryDB()
	cache2a := NewLockFreeJobCache(mdb2a)

	mdb2b := NewMemoryDB()
	cache2b := NewLockFreeJobCache(mdb2b)
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

		j := GetMockJob()
		assert.NoError(t, j.Init(cache)) // Saves the job

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
				assert.NoError(t, j.Init(cache))
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

				for _, element := range cache.GetAll().Jobs {
					assert.NotEqual(t, element.Name, "mock_job")
				}

			})

		})

	})

}

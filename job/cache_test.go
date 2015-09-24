package job

import (
	"time"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCacheStart(t *testing.T) {
	cache := NewMockCache()
	cache.Start(time.Duration(time.Hour))
}

func TestCacheDeleteJobNotFound(t *testing.T) {
	cache := NewMockCache()
	err := cache.Delete("not-a-real-id")
	assert.Equal(t, JobDoesntExistErr, err)
}

func TestCachePersist(t *testing.T) {
	cache := NewMockCache()
	err := cache.Persist()
	assert.NoError(t, err)
}

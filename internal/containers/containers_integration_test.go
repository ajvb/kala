//go:build integration

package containers

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A container that an earlier, crashed test run left behind is removed by the
// next run, once it is old enough.
func TestRemoveStaleRemovesOnlyContainersOlderThanTheLimit(t *testing.T) {
	pool, err := Connect()
	if err != nil {
		failNoDocker(t, Spec{Repository: "hashicorp/consul", Tag: "1.20"}, err)
	}
	// A label of its own, so that this test never removes the containers of
	// other packages whose tests run at the same time.
	own := "io.kala.test.removestale"
	value := strconv.FormatInt(time.Now().UnixNano(), 10)
	c, err := start(pool, Spec{
		Repository: "hashicorp/consul", Tag: "1.20", Cmd: []string{"agent", "-dev"}, Port: "8500/tcp",
		Labels: map[string]string{own: value},
		Ready:  func(string) error { return nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.remove() })

	removeStale(pool, own+"="+value, time.Hour)
	_, err = pool.Client.InspectContainer(c.resource.Container.ID)
	assert.NoError(t, err, "a container younger than the limit must be left alone")

	removeStale(pool, own+"="+value, 0)
	_, err = pool.Client.InspectContainer(c.resource.Container.ID)
	assert.Error(t, err, "a container older than the limit must be removed")
}

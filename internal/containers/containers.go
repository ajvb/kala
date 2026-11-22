// Package containers starts throwaway Docker containers for tests that need a
// real database or service, and fails those tests with a clear message when no
// Docker daemon is available.
package containers

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	// readyTimeout is how long Run waits for a container to accept connections.
	readyTimeout = 120 * time.Second
	// label marks the containers started here, so that containers left behind by
	// a test run that crashed can be found and removed.
	label = "io.kala.test"
	// staleAfter is how old a labeled container must be before the next test
	// run removes it. It is much longer than any test run, so containers of
	// packages that are running at the same time are never touched.
	staleAfter = 30 * time.Minute
	// noDaemonReason starts the error that Connect returns when there is no
	// Docker daemon.
	noDaemonReason = "no Docker daemon is reachable"
)

// Spec describes a container.
type Spec struct {
	Repository string
	Tag        string
	Env        []string
	Cmd        []string
	// Labels are added to the container (the package's own label is always set).
	Labels map[string]string
	// Port is the container port to expose, for example "5432/tcp".
	Port string
	// Ready is called with the host and port of Port until it returns nil.
	Ready func(hostPort string) error
}

// Connect returns a client for the Docker daemon, or an error saying why there
// is none. It honors DOCKER_HOST like the docker command does. The Docker
// client's ping gives up after 30 seconds.
func Connect() (*dockertest.Pool, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("cannot create a Docker client: %w", err)
	}
	if err := pool.Client.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", noDaemonReason, err)
	}
	return pool, nil
}

// started is a running container.
type started struct {
	pool     *dockertest.Pool
	resource *dockertest.Resource
	hostPort string
}

// start runs the container and waits until it is ready.
func start(pool *dockertest.Pool, spec Spec) (*started, error) {
	image := spec.Repository + ":" + spec.Tag
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: spec.Repository,
		Tag:        spec.Tag,
		Env:        spec.Env,
		Cmd:        spec.Cmd,
		Labels:     labelsFor(spec),
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, fmt.Errorf("could not start the %s container: %w", image, err)
	}
	c := &started{pool: pool, resource: resource, hostPort: resource.GetHostPort(spec.Port)}
	pool.MaxWait = readyTimeout
	if err := pool.Retry(func() error { return spec.Ready(c.hostPort) }); err != nil {
		_ = c.remove() // the readiness error is the one to report
		return nil, fmt.Errorf("the %s container did not become ready at %s: %w", image, c.hostPort, err)
	}
	return c, nil
}

func labelsFor(spec Spec) map[string]string {
	labels := map[string]string{label: "1"}
	for k, v := range spec.Labels {
		labels[k] = v
	}
	return labels
}

func (c *started) remove() error { return c.pool.Purge(c.resource) }

// removeStale removes containers started here by an earlier test run that
// crashed before it could clean up (a panic in a test skips TestMain's cleanup).
// It is best effort: a failure only means the leftovers stay.
func removeStale(pool *dockertest.Pool, labelFilter string, olderThan time.Duration) {
	list, err := pool.Client.ListContainers(docker.ListContainersOptions{
		All:     true,
		Filters: map[string][]string{"label": {labelFilter}},
	})
	if err != nil {
		return
	}
	for i := range list {
		if time.Since(time.Unix(list[i].Created, 0)) > olderThan {
			_ = pool.Client.RemoveContainer(docker.RemoveContainerOptions{ID: list[i].ID, Force: true, RemoveVolumes: true})
		}
	}
}

// noDockerMessage is the failure message of a test that needs a container when
// there is no Docker daemon.
func noDockerMessage(spec Spec, err error) string {
	return fmt.Sprintf(
		"this integration test needs a %s:%s container but %v; run `go test ./...` to run only the unit tests",
		spec.Repository, spec.Tag, err,
	)
}

// failNoDocker fails the test because there is no Docker daemon. A test that
// needs a container never passes without one.
func failNoDocker(t *testing.T, spec Spec, err error) {
	t.Helper()
	t.Fatal(noDockerMessage(spec, err))
}

// Run starts the container for one test, waits until Ready succeeds and returns
// the host and port to connect to. The container is removed when the test ends.
// If there is no Docker daemon the test fails.
func Run(t *testing.T, spec Spec) string {
	t.Helper()
	pool, err := Connect()
	if err != nil {
		failNoDocker(t, spec, err)
	}
	removeStale(pool, label+"=1", staleAfter)
	c, err := start(pool, spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.remove(); err != nil {
			t.Errorf("could not remove the container: %v", err)
		}
	})
	return c.hostPort
}

// Shared is one container used by all the tests of a package, so that a slow
// database starts once. Start it lazily with HostPort and remove it with Close
// from TestMain after m.Run(). Tests that share a container must not run in
// parallel with each other: each one empties the database before it starts.
type Shared struct {
	spec     Spec
	once     sync.Once
	c        *started
	err      error
	noDaemon error
}

// NewShared describes a container; nothing is started yet.
func NewShared(spec Spec) *Shared { return &Shared{spec: spec} }

// HostPort starts the container on the first call and returns the host and port.
// If there is no Docker daemon, or the container cannot be started, the calling
// test fails.
func (s *Shared) HostPort(t *testing.T) string {
	t.Helper()
	s.once.Do(func() {
		pool, err := Connect()
		if err != nil {
			s.noDaemon = err
			return
		}
		removeStale(pool, label+"=1", staleAfter)
		s.c, s.err = start(pool, s.spec)
	})
	if s.noDaemon != nil {
		failNoDocker(t, s.spec, s.noDaemon)
	}
	if s.err != nil {
		t.Fatal(s.err)
	}
	return s.c.hostPort
}

// Close removes the container if it was started. Call it after m.Run has
// returned, when no test is running.
func (s *Shared) Close() error {
	if s.c == nil {
		return nil
	}
	return s.c.remove()
}

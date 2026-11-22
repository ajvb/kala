package containers

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const noDaemon = "unix:///nonexistent/docker.sock"

// needsContainer is the container the helper test asks for.
var needsContainer = Spec{Repository: "example", Tag: "1", Port: "1/tcp", Ready: func(string) error { return nil }}

func TestConnectSaysThereIsNoDockerDaemon(t *testing.T) {
	t.Setenv("DOCKER_HOST", noDaemon)

	pool, err := Connect()

	assert.Nil(t, pool)
	require.Error(t, err)
	assert.Contains(t, err.Error(), noDaemonReason)
}

// TestHelperNeedsDocker is run in a child process by TestRunFailsWhenThereIsNoDockerDaemon.
func TestHelperNeedsDocker(t *testing.T) {
	if os.Getenv("KALA_CONTAINERS_HELPER") == "" {
		t.Skip("only runs as the child of TestRunFailsWhenThereIsNoDockerDaemon")
	}
	Run(t, needsContainer)
}

// A test that needs a container must fail with the standard message when there
// is no Docker daemon, and never pass.
func TestRunFailsWhenThereIsNoDockerDaemon(t *testing.T) {
	t.Setenv("DOCKER_HOST", noDaemon)
	_, connectErr := Connect()
	require.Error(t, connectErr)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperNeedsDocker$", "-test.v")
	cmd.Env = append(os.Environ(), "KALA_CONTAINERS_HELPER=1", "DOCKER_HOST="+noDaemon)

	out, err := cmd.CombinedOutput()

	require.Error(t, err, "the child must fail:\n%s", out)
	assert.Contains(t, string(out), noDockerMessage(needsContainer, connectErr))
}

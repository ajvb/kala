//go:build integration

package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ExecRunner starts the program as a child process and returns its output; a
// program that is not installed is an error.
func TestExecRunnerStartsTheProgram(t *testing.T) {
	out, err := ExecRunner{}.Run("echo", "hello", "kala")
	require.NoError(t, err)
	assert.Equal(t, "hello kala\n", string(out))

	_, err = ExecRunner{}.Run("kala-test-program-that-does-not-exist")
	assert.Error(t, err)
}

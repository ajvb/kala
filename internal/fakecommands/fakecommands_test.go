package fakecommands

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPrograms(t *testing.T) {
	for _, c := range []struct {
		name    string
		args    []string
		out     string
		wantErr bool
	}{
		{name: "echo", args: []string{"a", "b"}, out: "a b"},
		{name: "true"},
		{name: "false", wantErr: true},
	} {
		out, err := New().Run(c.name, c.args...)
		assert.Equal(t, c.out, string(out), c.name)
		assert.Equal(t, c.wantErr, err != nil, c.name)
	}
}

func TestAProgramWithoutAHandlerIsNotFound(t *testing.T) {
	_, err := New().Run("not-a-program")
	assert.True(t, errors.Is(err, exec.ErrNotFound), "got %v", err)
}

func TestHandleReplacesAProgramAndRanRecordsEveryCommand(t *testing.T) {
	r := New()
	r.Handle("echo", func(args []string) (string, error) { return "replaced " + args[0], nil })
	r.Handle("greet", func(args []string) (string, error) { return "hi " + args[0], nil })

	out, err := r.Run("greet", "kala")
	require.NoError(t, err)
	assert.Equal(t, "hi kala", string(out))
	out, err = r.Run("echo", "x")
	require.NoError(t, err)
	assert.Equal(t, "replaced x", string(out))

	assert.Equal(t, [][]string{{"greet", "kala"}, {"echo", "x"}}, r.Ran())
}

func TestRunnerIsSafeForConcurrentUse(t *testing.T) {
	r := New()
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			r.Handle(fmt.Sprintf("p%d", i), func([]string) (string, error) { return "", nil })
			_, _ = r.Run("echo", "x")
			_ = r.Ran()
		})
	}
	wg.Wait()
	assert.Len(t, r.Ran(), 20)
}

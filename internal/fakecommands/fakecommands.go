// Package fakecommands provides a command runner for tests that starts no
// process. It satisfies job.CommandRunner.
package fakecommands

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Handler runs a fake program: it gets the program's arguments and returns
// its output, or an error if the program fails.
type Handler func(args []string) (string, error)

// Runner runs commands without starting a process. By default "echo" prints
// its arguments joined by spaces, "true" succeeds with no output, "false"
// fails, and any other program fails as if it were not installed. Handle
// changes what a program does. A Runner is safe for concurrent use; create
// one with New.
type Runner struct {
	mu       sync.Mutex
	handlers map[string]Handler
	ran      [][]string
}

// New returns a Runner with the default programs.
func New() *Runner {
	return &Runner{handlers: map[string]Handler{
		"echo":  func(args []string) (string, error) { return strings.Join(args, " "), nil },
		"true":  func([]string) (string, error) { return "", nil },
		"false": func([]string) (string, error) { return "", errors.New("exit status 1") },
	}}
}

// Handle makes the program name run h.
func (r *Runner) Handle(name string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = h
}

// Run records the command and runs the program's handler. A program without a
// handler fails with an error that wraps exec.ErrNotFound.
func (r *Runner) Run(name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	r.ran = append(r.ran, append([]string{name}, args...))
	h, ok := r.handlers[name]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("exec: %q: %w", name, exec.ErrNotFound)
	}
	out, err := h(args)
	return []byte(out), err
}

// Ran returns every command run so far, each as the program followed by its
// arguments, in the order they were run.
func (r *Runner) Ran() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]string(nil), r.ran...)
}

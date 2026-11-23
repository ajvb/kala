package job

import "os/exec"

// CommandRunner starts the program of a local job. name and args come from
// the job's command line after parsing and templating. Run returns the
// program's standard output and standard error together.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// ExecRunner runs each command as a child process. Caches use it unless
// WithCommandRunner gives them another.
type ExecRunner struct{}

// Run starts the program and waits for it to exit.
func (ExecRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput() //nolint:gosec // running the job's command is what a local job does
}

// commandRunnerProvider is implemented by caches that carry a CommandRunner.
// A job takes the runner of the cache it is added to.
type commandRunnerProvider interface {
	CommandRunner() CommandRunner
}

// commandRunnerOf returns the CommandRunner of the cache, or ExecRunner if the
// cache does not carry one.
func commandRunnerOf(cache JobCache) CommandRunner {
	if p, ok := cache.(commandRunnerProvider); ok {
		return p.CommandRunner()
	}
	return ExecRunner{}
}

// WithCommandRunner makes every job added to the cache start its command with
// r rather than as a child process. A nil r is ignored.
func WithCommandRunner(r CommandRunner) CacheOption {
	return func(o *cacheOptions) {
		if r != nil {
			o.cmdRunner = r
		}
	}
}

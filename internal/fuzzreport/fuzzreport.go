// Package fuzzreport helps fuzz targets fail without printing the input that
// made them fail.
//
// Fuzzing runs in public CI, where the logs can be read by anyone. A crashing
// input printed there would disclose a bug before it is fixed. A failing target
// therefore reports only what went wrong; Go still saves the input under
// testdata/fuzz/ on the machine that ran the fuzzer, where it can be read and
// reproduced.
package fuzzreport

import (
	"io"
	"runtime/debug"
	"testing"

	log "github.com/sirupsen/logrus"
)

// NoPanic runs f and fails the test if f panics. The failure names what
// panicked and includes the stack trace, but not the panic value, which can
// contain the input.
func NoPanic(t testing.TB, what string, f func()) {
	t.Helper()
	panicked := true
	var stack []byte
	func() {
		defer func() {
			if panicked {
				recover() //nolint:errcheck // the value is dropped on purpose
				stack = debug.Stack()
			}
		}()
		f()
		panicked = false
	}()
	if panicked {
		t.Fatalf("%s panicked\n%s", what, stack)
	}
}

// SilenceLogs discards the standard logrus logger's output until t (a fuzz
// target's *testing.F) ends, because log lines can contain the input. The
// previous output is then restored.
func SilenceLogs(t testing.TB) {
	previous := log.StandardLogger().Out
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(previous) })
}

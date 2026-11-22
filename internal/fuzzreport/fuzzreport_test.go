package fuzzreport

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

// recorder is a testing.TB that records a fatal failure instead of stopping
// the test that uses it.
type recorder struct {
	testing.TB
	failed  bool
	message string
}

func (r *recorder) Helper() {}

func (r *recorder) Fatalf(format string, args ...any) {
	r.failed = true
	r.message = fmt.Sprintf(format, args...)
}

func TestNoPanicFailsWithoutThePanicValue(t *testing.T) {
	r := &recorder{TB: t}
	NoPanic(r, "the parser", func() { panic("secret-input-value") })

	if !r.failed {
		t.Fatal("a panic did not fail the test")
	}
	if !strings.HasPrefix(r.message, "the parser panicked\n") {
		t.Errorf("the failure does not say what panicked: %q", r.message)
	}
	if strings.Contains(r.message, "secret-input-value") {
		t.Error("the failure contains the panic value")
	}
}

func TestNoPanicPassesWhenNothingPanics(t *testing.T) {
	r := &recorder{TB: t}
	ran := false
	NoPanic(r, "the parser", func() { ran = true })

	if !ran || r.failed {
		t.Fatalf("ran=%v failed=%v, want ran and not failed", ran, r.failed)
	}
}

func TestSilenceLogsDiscardsLogOutputUntilTheEnd(t *testing.T) {
	var buf bytes.Buffer
	previous := log.StandardLogger().Out
	log.SetOutput(&buf)
	defer log.SetOutput(previous)

	t.Run("silenced", func(t *testing.T) {
		SilenceLogs(t)
		log.Error("secret-input-value")
	})
	if buf.Len() != 0 {
		t.Fatal("a log line was written while the logs were silenced")
	}

	log.Error("after")
	if !strings.Contains(buf.String(), "after") {
		t.Error("the log output was not restored")
	}
}

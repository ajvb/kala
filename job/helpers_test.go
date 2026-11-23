package job

import (
	"testing"
	"time"
)

// parseTime parses a time written as "2006-Jan-02 15:04", in UTC.
func parseTime(t *testing.T, value string) time.Time {
	t.Helper()
	now, err := time.Parse("2006-Jan-02 15:04", value)
	if err != nil {
		t.Fatal(err)
	}
	return now
}

// parseTimeInLocation parses a time written as "2006-Jan-02 15:04" in the
// named time zone.
func parseTimeInLocation(t *testing.T, value string, location string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation(location)
	if err != nil {
		t.Fatal(err)
	}
	now, err := time.ParseInLocation("2006-Jan-02 15:04", value, loc)
	if err != nil {
		t.Fatal(err)
	}
	return now
}

// awaitJobRan waits until the job reports a finished run on its ranChan, and
// fails the test if that takes longer than timeout.
func awaitJobRan(t *testing.T, j *Job, timeout time.Duration) {
	t.Helper()
	select {
	case <-j.ranChan:
	case <-time.After(timeout):
		t.Fatal("Job failed to run")
	}
}

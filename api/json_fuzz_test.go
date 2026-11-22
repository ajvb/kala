package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajvb/kala/v2/internal/fuzzreport"
	"github.com/ajvb/kala/v2/job"
)

// For any request body, decoding a new job must not panic and must return
// either an error and no job, or a job. It only decodes: it never initializes
// the job, because a job runs its command. Failures do not print the input
// (fuzzreport).
func FuzzUnmarshalNewJob(f *testing.F) {
	for _, seed := range []string{
		`{"name":"a","command":"echo hi","schedule":"R/2030-01-01T00:00:00Z/PT1H"}`,
		`{"name":"a","command":"echo hi","owner":"x","retries":2,"epsilon":"PT1H"}`,
		`{"on_success":"x","retries":-1,"run_as_user":"u"}`,
		`{}`, `[]`, `null`, `"x"`, `{"name":`, ``,
	} {
		f.Add([]byte(seed))
	}
	fuzzreport.SilenceLogs(f)

	f.Fuzz(func(t *testing.T, body []byte) {
		req := httptest.NewRequest(http.MethodPost, ApiJobPath, bytes.NewReader(body))

		var (
			got *job.Job
			err error
		)
		fuzzreport.NoPanic(t, "unmarshalNewJob", func() { got, err = unmarshalNewJob(req) })
		if err != nil && got != nil {
			t.Fatal("unmarshalNewJob returned an error and a job")
		}
		if err == nil && got == nil {
			t.Fatal("unmarshalNewJob returned neither an error nor a job")
		}
	})
}

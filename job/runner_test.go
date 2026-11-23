package job

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajvb/kala/v2/internal/fakecommands"
	"github.com/stretchr/testify/assert"
)

func TestTemplatize(t *testing.T) {

	t.Run("command", func(t *testing.T) {

		t.Run("raw", func(t *testing.T) {
			j := &Job{
				Name:      "mock_job",
				Command:   "echo mr.{{$.Owner}}",
				Owner:     "jedi@example.com",
				cmdRunner: fakecommands.New(),
			}
			r := JobRunner{
				job: j,
			}
			out, err := r.LocalRun()
			assert.NoError(t, err)
			assert.Equal(t, "mr.{{$.Owner}}", out)
		})

		t.Run("templated", func(t *testing.T) {
			j := &Job{
				Name:               "mock_job",
				Command:            "echo mr.{{$.Owner}}",
				Owner:              "jedi@example.com",
				TemplateDelimiters: "{{ }}",
				cmdRunner:          fakecommands.New(),
			}
			r := JobRunner{
				job: j,
			}
			out, err := r.LocalRun()
			assert.NoError(t, err)
			assert.Equal(t, "mr.jedi@example.com", out)
		})

	})

	t.Run("url", func(t *testing.T) {

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			val := r.URL.Query().Get("val")
			_, err := w.Write([]byte(val))
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			w.WriteHeader(200)
		}))

		t.Run("raw", func(t *testing.T) {
			j := &Job{
				Name:  "mock_job",
				Owner: "jedi@example.com",
				RemoteProperties: RemoteProperties{
					Url: "http://" + srv.Listener.Addr().String() + "/path?val=a_{{$.Name}}",
				},
			}
			r := JobRunner{
				job: j,
			}
			out, err := r.RemoteRun()
			assert.NoError(t, err)
			assert.Equal(t, "a_{{$.Name}}", out)
		})

		t.Run("templated", func(t *testing.T) {
			j := &Job{
				Name:  "mock_job",
				Owner: "jedi@example.com",
				RemoteProperties: RemoteProperties{
					Url: "http://" + srv.Listener.Addr().String() + "/path?val=a_{{$.Name}}",
				},
				TemplateDelimiters: "{{ }}",
			}
			r := JobRunner{
				job: j,
			}
			out, err := r.RemoteRun()
			assert.NoError(t, err)
			assert.Equal(t, "a_mock_job", out)
		})

	})

	t.Run("body", func(t *testing.T) {

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			w.Write(b)
			w.WriteHeader(200)
		}))

		t.Run("raw", func(t *testing.T) {
			j := &Job{
				Name:  "mock_job",
				Owner: "jedi@example.com",
				RemoteProperties: RemoteProperties{
					Url:  "http://" + srv.Listener.Addr().String() + "/path",
					Body: `{"hello": "world", "foo": "young-${$.Owner}"}`,
				},
			}
			r := JobRunner{
				job: j,
			}
			out, err := r.RemoteRun()
			assert.NoError(t, err)
			assert.Equal(t, `{"hello": "world", "foo": "young-${$.Owner}"}`, out)
		})

		t.Run("templated", func(t *testing.T) {
			j := &Job{
				Name:  "mock_job",
				Owner: "jedi@example.com",
				RemoteProperties: RemoteProperties{
					Url:  "http://" + srv.Listener.Addr().String() + "/path",
					Body: `{"hello": "world", "foo": "young-${$.Owner}"}`,
				},
				TemplateDelimiters: "${ }",
			}
			r := JobRunner{
				job: j,
			}
			out, err := r.RemoteRun()
			assert.NoError(t, err)
			assert.Equal(t, `{"hello": "world", "foo": "young-jedi@example.com"}`, out)
		})

	})

}

package metrics

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	r := mux.NewRouter()
	m := NewMetrics("test-version")
	r.Handle("/metrics", m.Handler())
	ts := httptest.NewServer(r)
	_, req := setupTestReq(t, "GET", ts.URL+"/metrics", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	resp.Body.Close()
	assert.Contains(t, string(body), "start_time_seconds")
}

// setupTestReq constructs the writer recorder and request obj for use in tests
func setupTestReq(t assert.TestingT, method, path string, data []byte) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, path, bytes.NewReader(data))
	assert.NoError(t, err)
	return w, req
}

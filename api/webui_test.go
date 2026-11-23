package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ajvb/kala/v2/job"
	"github.com/ajvb/kala/v2/webui"
	"github.com/stretchr/testify/assert"
)

// /webui/ is served by the webui package: Kala's own files byte for byte when
// the build contains the libraries, otherwise the "not available" page with
// status 503. Files that are not part of the interface are never served.
func TestWebUIRoutesToTheEmbeddedInterface(t *testing.T) {
	server := MakeServer("", job.NewMockCache(), "", false)
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		server.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		return w
	}

	index, err := os.ReadFile("../webui/index.html")
	if !assert.NoError(t, err) {
		return
	}
	w := get("/webui/")
	if webui.LibrariesPresent(webui.Files) == nil {
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, index, w.Body.Bytes())
	} else {
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "The web interface is not available")
	}

	for _, path := range []string{"/webui/embed.go", "/webui/package.json", "/webui/LIBRARIES.md"} {
		assert.Contains(t, []int{http.StatusNotFound, http.StatusServiceUnavailable}, get(path).Code, path)
	}
}

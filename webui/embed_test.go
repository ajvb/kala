package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testIndex = `<link href="vendor/lib/a.css" rel="stylesheet"><script src="vendor/lib/a.js"></script>`

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	return w
}

func TestHandlerServesTheInterfaceWhenTheLibrariesArePresent(t *testing.T) {
	h := handler(fstest.MapFS{
		"index.html":       {Data: []byte(testIndex)},
		"vendor/lib/a.css": {Data: []byte("css")},
		"vendor/lib/a.js":  {Data: []byte("js")},
	})

	w := get(h, "/")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, testIndex, w.Body.String())
	assert.Equal(t, "js", get(h, "/vendor/lib/a.js").Body.String())
}

// Without its libraries the interface cannot work, so every path gets a page
// that says so.
func TestHandlerSaysTheInterfaceIsUnavailableWithoutItsLibraries(t *testing.T) {
	for name, fsys := range map[string]fstest.MapFS{
		"not installed": {"index.html": {Data: []byte(testIndex)}, "vendor/README.md": {Data: []byte("x")}},
		"one file missing": {
			"index.html":       {Data: []byte(testIndex)},
			"vendor/lib/a.css": {Data: []byte("css")},
		},
	} {
		h := handler(fsys)
		for _, path := range []string{"/", "/vendor/lib/a.css"} {
			w := get(h, path)
			assert.Equal(t, http.StatusServiceUnavailable, w.Code, "%s: %s", name, path)
			assert.Contains(t, w.Body.String(), "The web interface is not available", "%s: %s", name, path)
		}
	}
}

// Every library in package.json is listed in LIBRARIES.md with its version,
// and every vendor file that index.html loads is one that "make webui-libs"
// copies.
func TestLibrariesAreListedAndProvideWhatIndexLoads(t *testing.T) {
	raw, err := os.ReadFile("package.json")
	require.NoError(t, err)
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	require.NoError(t, json.Unmarshal(raw, &pkg))
	require.NotEmpty(t, pkg.Dependencies)
	inventory, err := os.ReadFile("LIBRARIES.md")
	require.NoError(t, err)
	for name, version := range pkg.Dependencies {
		row := regexp.MustCompile("(?m)^\\|.*`" + regexp.QuoteMeta(name) + "`.*\\| " + regexp.QuoteMeta(version) + " \\|")
		assert.True(t, row.Match(inventory), "%s %s is not in LIBRARIES.md", name, version)
	}

	makefile, err := os.ReadFile("../Makefile")
	require.NoError(t, err)
	copied := map[string]bool{}
	for _, m := range regexp.MustCompile(`webui/(vendor/\S+)`).FindAllSubmatch(makefile, -1) {
		copied[string(m[1])] = true
	}
	index, err := os.ReadFile("index.html")
	require.NoError(t, err)
	refs := libraryRef.FindAllSubmatch(index, -1)
	require.NotEmpty(t, refs, "index.html loads nothing from vendor/")
	for _, ref := range refs {
		assert.True(t, copied[string(ref[1])], "index.html loads %s, which make webui-libs does not copy", ref[1])
	}
}

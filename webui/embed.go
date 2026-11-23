// Package webui holds the files of Kala's web interface, embedded in the
// binary, and serves them.
package webui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"

	log "github.com/sirupsen/logrus"
)

// Files holds the web interface: index.html, the icons and manifest, the css
// and js directories, and the third-party libraries in vendor, which
// "make webui-libs" installs with npm at build time.
//
//go:embed index.html site.webmanifest favicon.ico *.png css js vendor
var Files embed.FS

// libraryRef matches a file that index.html loads from vendor.
var libraryRef = regexp.MustCompile(`(?:src|href)="(vendor/[^"]+)"`)

// unavailablePage is served instead of the web interface when the build does
// not contain its libraries.
const unavailablePage = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Kala Web Interface</title></head>
<body>
<h1>The web interface is not available</h1>
<p>This Kala binary was built without the web interface's JavaScript and CSS
libraries. Build it with <code>make bin/kala</code> (which needs npm), or use
a release binary or the container image. The API is not affected.</p>
</body>
</html>
`

// Handler serves the web interface, or, if the build does not contain its
// libraries, a page saying so with status 503 for every path.
func Handler() http.Handler {
	return handler(Files)
}

func handler(fsys fs.FS) http.Handler {
	if err := LibrariesPresent(fsys); err != nil {
		log.Warnf("The web interface is disabled: %v", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(unavailablePage))
		})
	}
	return http.FileServer(http.FS(fsys))
}

// LibrariesPresent reports whether fsys holds every library file that its
// index.html loads from vendor.
func LibrariesPresent(fsys fs.FS) error {
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return err
	}
	refs := libraryRef.FindAllSubmatch(index, -1)
	if len(refs) == 0 {
		return fmt.Errorf("index.html loads no libraries from vendor")
	}
	for _, ref := range refs {
		if _, err := fs.Stat(fsys, string(ref[1])); err != nil {
			return fmt.Errorf("the libraries were not installed (make webui-libs): %s is missing", ref[1])
		}
	}
	return nil
}

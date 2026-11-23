// Package version reports what Kala binary is running: its version, the
// commit and date it was built from, and the Go version.
//
// A release build sets Version, Commit and BuildDate with the linker, for
// example:
//
//	go build -trimpath -ldflags "-X github.com/ajvb/kala/v2/internal/version.Version=v2.0.0 ..."
//
// The Makefile and the Dockerfile do this. A plain go build leaves Version
// empty, which Get reports as "dev", and takes the commit from the build
// information Go records when building in a git checkout.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Set with -ldflags "-X ..." by a release build.
var (
	Version   string
	Commit    string
	BuildDate string
)

// Unset is what Info reports for a value the build did not record.
const Unset = "unknown"

// Info describes the running binary.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
}

// Get returns the version information of the running binary.
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
	if info.Version == "" {
		info.Version = "dev"
	}
	if info.Commit == "" {
		info.Commit = commitFromBuildInfo()
	}
	if info.BuildDate == "" {
		info.BuildDate = Unset
	}
	return info
}

// commitFromBuildInfo returns the commit Go recorded at build time, with
// "-dirty" if the checkout had uncommitted changes, or Unset.
func commitFromBuildInfo() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return Unset
	}
	var revision, modified string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if revision == "" {
		return Unset
	}
	if modified == "true" {
		revision += "-dirty"
	}
	return revision
}

// String formats the information as `kala version` prints it.
func (i Info) String() string {
	return fmt.Sprintf("kala %s\ncommit:     %s\nbuild date: %s\ngo:         %s\n",
		i.Version, i.Commit, i.BuildDate, i.GoVersion)
}

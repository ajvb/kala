package version

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReportsTheValuesSetAtBuildTime(t *testing.T) {
	defer func(v, c, d string) { Version, Commit, BuildDate = v, c, d }(Version, Commit, BuildDate)
	Version, Commit, BuildDate = "v2.0.0", "0123abc", "2026-11-22T00:00:00Z"

	assert.Equal(t, Info{
		Version:   "v2.0.0",
		Commit:    "0123abc",
		BuildDate: "2026-11-22T00:00:00Z",
		GoVersion: runtime.Version(),
	}, Get())
}

func TestGetWithoutBuildValuesSaysDevAndUnknown(t *testing.T) {
	defer func(v, c, d string) { Version, Commit, BuildDate = v, c, d }(Version, Commit, BuildDate)
	Version, Commit, BuildDate = "", "", ""

	info := Get()
	assert.Equal(t, "dev", info.Version)
	assert.Equal(t, Unset, info.BuildDate)
	assert.NotEmpty(t, info.Commit)
}

func TestStringPrintsEveryField(t *testing.T) {
	info := Info{Version: "v2.0.0", Commit: "0123abc", BuildDate: "2026-11-22T00:00:00Z", GoVersion: "go1.27.4"}
	assert.Equal(t, "kala v2.0.0\ncommit:     0123abc\nbuild date: 2026-11-22T00:00:00Z\ngo:         go1.27.4\n", info.String())
}

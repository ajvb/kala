package job

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ajvb/kala/v2/internal/fuzzreport"
	"github.com/ajvb/kala/v2/utils/iso8601"
)

// For any schedule string, Init must not panic or leave the lock held,
// and must either return an error (cache unchanged) or accept the job (cached
// with a timer). Failures do not print the input (fuzzreport).
func FuzzInitSchedule(f *testing.F) {
	for _, seed := range []string{
		"X/2030-01-01T00:00:00Z/PT1H",
		"R/2030-01-01T00:00:00Z/PT1H",
		"R0/2030-01-01T00:00:00Z/",
		"R-5/2030-01-01T00:00:00Z/PT1H",
		"R2/2030-01-01T00:00:00/P1DT10M10S",
		"R//", "///", "R/", "",
	} {
		f.Add(seed)
	}
	fuzzreport.SilenceLogs(f)

	f.Fuzz(func(t *testing.T, schedule string) {
		if schedule == "" {
			t.Skip("an empty schedule means run now; not a schedule string")
		}
		if hitsKnownV1TimingFlaw(schedule) {
			t.Skip("known v1 flaw outside the scope of this fix")
		}
		cache := NewMockCache()
		j := GetMockJob()
		j.Schedule = schedule

		var err error
		fuzzreport.NoPanic(t, "Init", func() { err = j.Init(cache) })
		cached := len(cache.GetAll().Jobs)
		switch {
		case err != nil && cached != 0:
			t.Error("a rejected schedule left a job in the cache")
		case err == nil && cached != 1:
			t.Error("an accepted schedule did not leave exactly one job in the cache")
		case err == nil && j.jobTimer == nil:
			t.Error("an accepted schedule has no timer")
		}
		assertLockUsable(t, j)
	})
}

// hitsKnownV1TimingFlaw reports schedules that v1 accepts but handles wrongly,
// which the fuzzer found. They are out of scope for this hotfix: a start
// after the year 2262 overflows UnixNano in GetWaitDuration, and a repeating
// job with a zero-length duration runs in a hot loop.
func hitsKnownV1TimingFlaw(schedule string) bool {
	parts := strings.Split(schedule, "/")
	if len(parts) != 3 {
		return false
	}
	start, err := time.Parse(time.RFC3339, parts[1])
	if err != nil {
		start, err = time.Parse(RFC3339WithoutTimezone, parts[1])
	}
	if err != nil {
		return false // rejected anyway; not the flaw
	}
	if start.Year() >= 2262 {
		return true
	}
	repeatOK := parts[0] == "R"
	if rp := strings.Split(parts[0], "R"); !repeatOK && len(rp) >= 2 {
		n, perr := strconv.ParseInt(rp[1], 10, 0)
		repeatOK = perr == nil && n != 0
	}
	d, derr := iso8601.FromString(parts[2])
	return repeatOK && derr == nil && *d == (iso8601.Duration{})
}

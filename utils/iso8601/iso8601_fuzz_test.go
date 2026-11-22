package iso8601_test

import (
	"testing"
	"time"

	"github.com/ajvb/kala/v2/internal/fuzzreport"
	"github.com/ajvb/kala/v2/utils/iso8601"
)

// For any string, FromString must not panic and must return either an error
// and no duration, or a duration that can be printed, parsed again to the same
// value, and added to a time. Failures do not print the input (fuzzreport).
func FuzzFromString(f *testing.F) {
	for _, seed := range []string{
		"P1Y2M3DT4H5M6S", "P1W", "P2M", "PT1H", "PT0S", "P0D",
		"P", "PT", "", "asdf", "P-1D", "P99999999999999999999D",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		var (
			d   *iso8601.Duration
			err error
		)
		fuzzreport.NoPanic(t, "FromString", func() { d, err = iso8601.FromString(s) })
		if err != nil {
			if d != nil {
				t.Fatal("FromString returned an error and a duration")
			}
			return
		}
		if d == nil {
			t.Fatal("FromString returned neither an error nor a duration")
		}
		fuzzreport.NoPanic(t, "the duration's methods", func() {
			d.Add(time.Unix(0, 0).UTC())
			d.HasTimePart()
		})

		if d.IsZero() {
			return // prints as "P", which is not a valid duration
		}
		var again *iso8601.Duration
		fuzzreport.NoPanic(t, "printing and parsing again", func() { again, err = iso8601.FromString(d.String()) })
		if err != nil {
			t.Fatal("an accepted duration does not parse after printing")
		}
		if *again != *d {
			t.Fatal("an accepted duration parses to a different value after printing")
		}
	})
}

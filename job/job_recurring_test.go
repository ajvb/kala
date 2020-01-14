// Tests for recurring jobs firing appropriately.
package job

import (
	"fmt"
	"testing"
	"time"

	"github.com/mixer/clock"
	"github.com/stretchr/testify/assert"
)

var recurTableTests = []struct {
	Name        string
	Location    string
	Start       string
	Interval    string
	Checkpoints []string
}{
	{
		Name:     "Daily",
		Location: "America/Los_Angeles",
		Start:    "2020-Jan-13 14:09",
		Interval: "P1D",
		Checkpoints: []string{
			"2020-Jan-14 14:09",
			"2020-Jan-15 14:09",
			"2020-Jan-16 14:09",
		},
	},
	{
		Name:     "Daily across DST boundary",
		Location: "America/Los_Angeles",
		Start:    "2020-Mar-05 14:09",
		Interval: "P1D",
		Checkpoints: []string{
			"2020-Mar-06 14:09",
			"2020-Mar-07 14:09",
			"2020-Mar-08 14:09",
			"2020-Mar-09 14:09",
		},
	},
	{
		Name:     "24 Hourly across DST boundary",
		Location: "America/Los_Angeles",
		Start:    "2020-Mar-05 14:09",
		Interval: "PT24H",
		Checkpoints: []string{
			"2020-Mar-06 14:09",
			"2020-Mar-07 14:09",
			"2020-Mar-08 15:09",
			"2020-Mar-09 15:09",
		},
	},
	{
		Name:     "Weekly",
		Location: "America/Los_Angeles",
		Start:    "2020-Jan-13 14:09",
		Interval: "P1W",
		Checkpoints: []string{
			"2020-Jan-20 14:09",
			"2020-Jan-27 14:09",
			"2020-Feb-03 14:09",
		},
	},
	{
		Name:     "Monthly",
		Location: "America/Los_Angeles",
		Start:    "2020-Jan-20 14:09",
		Interval: "P1M",
		Checkpoints: []string{
			"2020-Feb-20 14:09",
			"2020-Mar-20 14:09",
			"2020-Apr-20 14:09",
			"2020-May-20 14:09",
			"2020-Jun-20 14:09",
			"2020-Jul-20 14:09",
			"2020-Aug-20 14:09",
			"2020-Sep-20 14:09",
			"2020-Oct-20 14:09",
			"2020-Nov-20 14:09",
			"2020-Dec-20 14:09",
			"2021-Jan-20 14:09",
		},
	},
	{
		Name:     "Monthly with Normalization",
		Location: "America/Los_Angeles",
		Start:    "2020-Jul-31 14:09",
		Interval: "P1M",
		Checkpoints: []string{
			"2020-Aug-31 14:09",
			"2020-Oct-01 14:09",
			"2020-Nov-01 14:09",
		},
	},
	{
		Name:     "Yearly across Leap Year boundary",
		Location: "America/Los_Angeles",
		Start:    "2020-Jan-20 14:09",
		Interval: "P1Y",
		Checkpoints: []string{
			"2021-Jan-20 14:09",
			"2022-Jan-20 14:09",
			"2023-Jan-20 14:09",
			"2024-Jan-20 14:09",
			"2025-Jan-20 14:09",
		},
	},
}

func TestRecur(t *testing.T) {

	for _, testStruct := range recurTableTests {

		func() {

			now := parseTimeInLocation(t, testStruct.Start, testStruct.Location)

			clk := clock.NewMockClock(now)

			start := now.Add(time.Second * 5)
			j := GetMockRecurringJobWithSchedule(start, testStruct.Interval)
			j.clk.SetClock(clk)

			cache := NewMockCache()
			j.Init(cache)

			checkpoints := append([]string{testStruct.Start}, testStruct.Checkpoints...)

			for i, chk := range checkpoints {

				clk.SetTime(parseTimeInLocation(t, chk, testStruct.Location))
				briefPause()

				j.lock.RLock()
				assert.Equal(t, i, int(j.Metadata.SuccessCount), fmt.Sprintf("1st Test of %s index %d", testStruct.Name, i))
				j.lock.RUnlock()

				clk.AddTime(time.Second * 6)
				briefPause()

				j.lock.RLock()
				assert.Equal(t, i+1, int(j.Metadata.SuccessCount), fmt.Sprintf("2nd Test of %s index %d", testStruct.Name, i))
				j.lock.RUnlock()
			}

		}()

	}

}

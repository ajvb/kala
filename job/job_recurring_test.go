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
	Start       string
	Interval    string
	Checkpoints []string
}{
	{
		Name:     "Daily",
		Start:    "2020-Jan-13 14:09 PST",
		Interval: "P1D",
		Checkpoints: []string{
			"2020-Jan-14 14:09 PST",
			"2020-Jan-15 14:09 PST",
			"2020-Jan-16 14:09 PST",
		},
	},
	{
		Name:     "Daily with DST",
		Start:    "2020-Mar-05 14:09 PST",
		Interval: "P1D",
		Checkpoints: []string{
			"2020-Mar-06 14:09 PST",
			"2020-Mar-07 14:09 PST",
			"2020-Mar-08 14:09 PDT",
			"2020-Mar-09 14:09 PDT",
		},
	},
	{
		Name:     "24 Hourly with DST",
		Start:    "2020-Mar-05 14:09 PST",
		Interval: "PT24H",
		Checkpoints: []string{
			"2020-Mar-06 14:09 PST",
			"2020-Mar-07 14:09 PST",
			"2020-Mar-08 15:09 PDT",
			"2020-Mar-09 15:09 PDT",
		},
	},
	{
		Name:     "Weekly",
		Start:    "2020-Jan-13 14:09 PST",
		Interval: "P1W",
		Checkpoints: []string{
			"2020-Jan-20 14:09 PST",
			"2020-Jan-27 14:09 PST",
			"2020-Feb-03 14:09 PST",
		},
	},
	{
		Name:     "Monthly",
		Start:    "2020-Jan-20 14:09 PST",
		Interval: "P1M",
		Checkpoints: []string{
			"2020-Feb-20 14:09 PST",
			"2020-Mar-20 14:09 PDT",
			"2020-Apr-20 14:09 PDT",
			"2020-May-20 14:09 PDT",
			"2020-Jun-20 14:09 PDT",
			"2020-Jul-20 14:09 PDT",
			"2020-Aug-20 14:09 PDT",
			"2020-Sep-20 14:09 PDT",
			"2020-Oct-20 14:09 PDT",
			"2020-Nov-20 14:09 PST",
			"2020-Dec-20 14:09 PST",
			"2021-Jan-20 14:09 PST",
		},
	},
	{
		Name:     "Monthly with Normalization",
		Start:    "2020-Jul-31 14:09 PDT",
		Interval: "P1M",
		Checkpoints: []string{
			"2020-Aug-31 14:09 PDT",
			"2020-Oct-01 14:09 PDT",
			"2020-Nov-01 14:09 PST",
		},
	},
	{
		Name:     "Yearly",
		Start:    "2020-Jan-20 14:09 PST",
		Interval: "P1Y",
		Checkpoints: []string{
			"2021-Jan-20 14:09 PST",
			"2022-Jan-20 14:09 PST",
			"2023-Jan-20 14:09 PST",
			"2024-Jan-20 14:09 PST",
			"2025-Jan-20 14:09 PST",
		},
	},
}

func TestRecur(t *testing.T) {

	for _, testStruct := range recurTableTests {

		func() {

			now := parseTime(t, testStruct.Start)

			clk := clock.NewMockClock(now)

			start := now.Add(time.Second * 5)
			j := GetMockRecurringJobWithSchedule(start, testStruct.Interval)
			j.clk.SetClock(clk)

			cache := NewMockCache()
			j.Init(cache)

			checkpoints := append([]string{testStruct.Start}, testStruct.Checkpoints...)

			for i, chk := range checkpoints {

				clk.SetTime(parseTime(t, chk))
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

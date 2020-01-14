// Tests for recurring jobs firing appropriately.
package job

import (
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
		Start:    "2020-Jan-13 14:09",
		Interval: "P1D",
		Checkpoints: []string{
			"2020-Jan-14 14:09",
			"2020-Jan-15 14:09",
			"2020-Jan-16 14:09",
		},
	},
	{
		Name:     "Weekly",
		Start:    "2020-Jan-13 14:09",
		Interval: "P1W",
		Checkpoints: []string{
			"2020-Jan-20 14:09",
			"2020-Jan-27 14:09",
			"2020-Feb-03 14:09",
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
				assert.Equal(t, i, int(j.Metadata.SuccessCount))

				clk.AddTime(time.Second * 6)
				briefPause()
				assert.Equal(t, i+1, int(j.Metadata.SuccessCount))
			}

		}()

	}

}

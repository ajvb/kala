package job

import (
	"testing"
	"time"

	"github.com/ajvb/kala/v2/clock"
	"github.com/ajvb/kala/v2/utils/iso8601"
)

// BenchmarkGetWaitDuration measures the next-run computation: a first run in
// the future, a repeat after the last run, and a job that resumes at its next
// scheduled time after a day of one-minute runs were missed.
func BenchmarkGetWaitDuration(b *testing.B) {
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	everyHour, err := iso8601.FromString("PT1H")
	if err != nil {
		b.Fatal(err)
	}
	everyMinute, err := iso8601.FromString("PT1M")
	if err != nil {
		b.Fatal(err)
	}

	cases := map[string]*Job{
		"first run in the future": {
			scheduleTime: now.Add(time.Hour), delayDuration: everyHour, timesToRepeat: -1,
		},
		"repeat after the last run": {
			scheduleTime: now.Add(-2 * time.Hour), delayDuration: everyHour, timesToRepeat: -1,
			Metadata: Metadata{LastAttemptedRun: now.Add(-30 * time.Minute)},
		},
		"resume after a missed day of minute runs": {
			scheduleTime: now.Add(-24 * time.Hour), delayDuration: everyMinute, timesToRepeat: -1,
			ResumeAtNextScheduledTime: true,
		},
	}
	for name, j := range cases {
		j.clk = clock.NewFake(now)
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				j.GetWaitDuration()
			}
		})
	}
}

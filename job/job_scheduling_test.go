// Tests for scheduling code.
package job

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"testing"
)

var getWaitDurationTableTests = []struct {
	Job              *Job
	JobFunc          func() *Job
	ExpectedDuration time.Duration
	Name             string
}{
	{
		JobFunc: func() *Job {
			return &Job{
				// Schedule time is passed
				Schedule: "R/2015-10-17T11:44:54.389361-07:00/PT10S",
				Metadata: Metadata{
					LastAttemptedRun: time.Now(),
				},
			}
		},
		ExpectedDuration: 10 * time.Second,
		Name:             "Passed time, 10 seconds",
	},
	{
		JobFunc: func() *Job {
			return &Job{
				// Schedule time is passed
				Schedule: "R/2015-10-17T11:44:54.389361-07:00/PT1M",
				Metadata: Metadata{
					LastAttemptedRun: time.Now(),
				},
			}
		},
		ExpectedDuration: time.Minute,
		Name:             "Passed time, 1 minute",
	},
	{
		JobFunc: func() *Job {
			return &Job{
				// Schedule time is passed
				Schedule: "R/2015-10-17T11:44:54.389361-07:00/P1DT",
				Metadata: Metadata{
					LastAttemptedRun: time.Now(),
				},
			}
		},
		ExpectedDuration: 24 * time.Hour,
		Name:             "Passed time, 1 day",
	},
}

func TestGetWatiDuration(t *testing.T) {
	for _, testStruct := range getWaitDurationTableTests {
		testStruct.Job = testStruct.JobFunc()
		err := testStruct.Job.InitDelayDuration(false)
		assert.NoError(t, err)
		actualDuration := testStruct.Job.GetWaitDuration()
		log.Warnf("LastAttempted: %s", testStruct.Job.Metadata.LastAttemptedRun)
		assert.InDelta(t, float64(testStruct.ExpectedDuration), float64(actualDuration), float64(time.Millisecond*50), "Test of "+testStruct.Name)
	}
}

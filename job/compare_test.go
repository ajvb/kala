package job

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var compareTime = time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)

// newComparableJob returns a job with every kind of exported field set.
func newComparableJob() *Job {
	j := GetMockJobWithGenericSchedule(compareTime)
	j.Id = "id-1"
	j.NextRunAt = compareTime
	j.ParentJobs = []string{"parent"}
	j.Metadata.SuccessCount = 3
	j.Metadata.LastSuccess = compareTime
	j.RemoteProperties = RemoteProperties{Url: "http://example.com", Headers: http.Header{"A": {"1"}}, ExpectedResponseCodes: []int{200}}
	j.Stats = []*JobStat{{JobId: "id-1", RanAt: compareTime, Success: true}}
	return j
}

func TestCompareIdenticalJobsAreEqual(t *testing.T) {
	ok, err := newComparableJob().Compare(newComparableJob())

	assert.True(t, ok)
	assert.NoError(t, err)
}

func TestCompareNamesEachExportedFieldThatDiffers(t *testing.T) {
	changes := map[string]func(*Job){
		"Name":                      func(j *Job) { j.Name = "other" },
		"Id":                        func(j *Job) { j.Id = "other" },
		"Command":                   func(j *Job) { j.Command = "other" },
		"Owner":                     func(j *Job) { j.Owner = "other" },
		"Disabled":                  func(j *Job) { j.Disabled = true },
		"Schedule":                  func(j *Job) { j.Schedule = "other" },
		"Retries":                   func(j *Job) { j.Retries++ },
		"NextRunAt":                 func(j *Job) { j.NextRunAt = j.NextRunAt.Add(time.Nanosecond) },
		"ParentJobs[0]":             func(j *Job) { j.ParentJobs[0] = "other" },
		"ParentJobs":                func(j *Job) { j.ParentJobs = append(j.ParentJobs, "extra") },
		"Metadata.SuccessCount":     func(j *Job) { j.Metadata.SuccessCount++ },
		"Metadata.LastSuccess":      func(j *Job) { j.Metadata.LastSuccess = j.Metadata.LastSuccess.Add(time.Second) },
		"RemoteProperties.Url":      func(j *Job) { j.RemoteProperties.Url = "other" },
		"RemoteProperties.Headers":  func(j *Job) { j.RemoteProperties.Headers = http.Header{"A": {"2"}} },
		"Stats[0].Success":          func(j *Job) { j.Stats[0].Success = false },
		"Stats":                     func(j *Job) { j.Stats = nil },
		"RemoteProperties.Timeout":  func(j *Job) { j.RemoteProperties.Timeout = 9 },
		"ResumeAtNextScheduledTime": func(j *Job) { j.ResumeAtNextScheduledTime = true },
		"RemoteProperties.ExpectedResponseCodes[0]": func(j *Job) { j.RemoteProperties.ExpectedResponseCodes = []int{201} },
	}
	for field, change := range changes {
		t.Run(field, func(t *testing.T) {
			other := newComparableJob()
			change(other)

			ok, err := newComparableJob().Compare(other)

			assert.False(t, ok)
			require.Error(t, err)
			assert.Contains(t, err.Error(), field+":")
		})
	}
}

func TestCompareReportsEveryDifferenceInOneError(t *testing.T) {
	other := newComparableJob()
	other.Name = "other"
	other.Metadata.SuccessCount = 9

	ok, err := newComparableJob().Compare(other)

	assert.False(t, ok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Name:")
	assert.Contains(t, err.Error(), "Metadata.SuccessCount: 3 != 9")
}

func TestCompareIgnoresUnexportedFields(t *testing.T) {
	other := newComparableJob()
	other.timesToRepeat = 5
	other.succeedInstantly = true
	other.scheduleTime = compareTime.Add(time.Hour)

	ok, err := newComparableJob().Compare(other)

	assert.True(t, ok)
	assert.NoError(t, err)
}

func TestCompareTimesAreComparedAsInstants(t *testing.T) {
	other := newComparableJob()
	other.NextRunAt = compareTime.In(time.FixedZone("somewhere", -8*60*60))

	ok, err := newComparableJob().Compare(other)

	assert.True(t, ok)
	assert.NoError(t, err)
}

func TestCompareTreatsANilSliceOrMapAsEqualToAnEmptyOne(t *testing.T) {
	a, b := newComparableJob(), newComparableJob()
	a.ParentJobs, b.ParentJobs = nil, []string{}
	a.RemoteProperties.Headers, b.RemoteProperties.Headers = nil, http.Header{}

	ok, err := a.Compare(b)

	assert.True(t, ok)
	assert.NoError(t, err)
}

func TestCompareAJobWithItselfIsEqualAndANilJobIsAnError(t *testing.T) {
	j := newComparableJob()

	ok, err := j.Compare(j) //nolint:gocritic // a job compared with itself is the case under test
	assert.True(t, ok)
	assert.NoError(t, err)

	ok, err = j.Compare(nil)
	assert.False(t, ok)
	assert.Error(t, err)

	var none *Job
	ok, err = none.Compare(j)
	assert.False(t, ok)
	assert.Error(t, err)
}

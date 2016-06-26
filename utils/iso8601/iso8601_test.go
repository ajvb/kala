package iso8601_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cescoferraro/kala/utils/iso8601"

	"github.com/stretchr/testify/assert"
)

func TestFromString(t *testing.T) {
	t.Parallel()

	// test with bad format
	_, err := iso8601.FromString("asdf")
	assert.Equal(t, err, iso8601.ErrBadFormat)

	// test with good full string
	dur, err := iso8601.FromString("P1Y2M3DT4H5M6S")
	assert.Nil(t, err)
	assert.Equal(t, 1, dur.Years)
	assert.Equal(t, 2, dur.Months)
	assert.Equal(t, 3, dur.Days)
	assert.Equal(t, 4, dur.Hours)
	assert.Equal(t, 5, dur.Minutes)
	assert.Equal(t, 6, dur.Seconds)

	// test with good week string
	dur, err = iso8601.FromString("P1W")
	assert.Nil(t, err)
	assert.Equal(t, 1, dur.Weeks)

	// test with 2M
	dur, err = iso8601.FromString("P2M")
	assert.Nil(t, err)
	assert.Equal(t, 0, dur.Years)
	assert.Equal(t, 2, dur.Months)
	assert.Equal(t, 0, dur.Days)
	assert.Equal(t, 0, dur.Hours)
	assert.Equal(t, 0, dur.Minutes)
	assert.Equal(t, 0, dur.Seconds)

	// test with invalid
	dur, err = iso8601.FromString("PT")
	assert.Nil(t, dur)
	assert.Equal(t, err.Error(), "invalid ISO 8601 duration spec PT")

	// test with 4h
	dur, err = iso8601.FromString("PT4H")
	assert.Nil(t, err)
	assert.Equal(t, 0, dur.Years)
	assert.Equal(t, 0, dur.Months)
	assert.Equal(t, 0, dur.Days)
	assert.Equal(t, 4, dur.Hours)
	assert.Equal(t, 0, dur.Minutes)
	assert.Equal(t, 0, dur.Seconds)

	// test with 5m
	dur, err = iso8601.FromString("PT5M")
	assert.Nil(t, err)
	assert.Equal(t, 0, dur.Years)
	assert.Equal(t, 0, dur.Months)
	assert.Equal(t, 0, dur.Days)
	assert.Equal(t, 0, dur.Hours)
	assert.Equal(t, 5, dur.Minutes)
	assert.Equal(t, 0, dur.Seconds)

	// test with 6s
	dur, err = iso8601.FromString("PT6S")
	assert.Nil(t, err)
	assert.Equal(t, 0, dur.Years)
	assert.Equal(t, 0, dur.Months)
	assert.Equal(t, 0, dur.Days)
	assert.Equal(t, 0, dur.Hours)
	assert.Equal(t, 0, dur.Minutes)
	assert.Equal(t, 6, dur.Seconds)
}

func TestString(t *testing.T) {
	t.Parallel()

	// test empty
	d := iso8601.Duration{}
	assert.Equal(t, d.String(), "P")

	// test only larger-than-day
	d = iso8601.Duration{Years: 1, Days: 2}
	assert.Equal(t, d.String(), "P1Y2D")

	// test only smaller-than-day
	d = iso8601.Duration{Hours: 1, Minutes: 2, Seconds: 3}
	assert.Equal(t, d.String(), "PT1H2M3S")

	// test full format
	d = iso8601.Duration{Years: 1, Months: 2, Days: 3, Hours: 4, Minutes: 5, Seconds: 6}
	assert.Equal(t, d.String(), "P1Y2M3DT4H5M6S")

	// test week format
	d = iso8601.Duration{Weeks: 1}
	assert.Equal(t, d.String(), "P1W")
}

func TestToDuration(t *testing.T) {
	t.Parallel()

	d := iso8601.Duration{Years: 1}
	assert.Equal(t, d.ToDuration(), time.Hour*24*365)

	d = iso8601.Duration{Weeks: 1}
	assert.Equal(t, d.ToDuration(), time.Hour*24*7)

	d = iso8601.Duration{Days: 1}
	assert.Equal(t, d.ToDuration(), time.Hour*24)

	d = iso8601.Duration{Hours: 1}
	assert.Equal(t, d.ToDuration(), time.Hour)

	d = iso8601.Duration{Minutes: 1}
	assert.Equal(t, d.ToDuration(), time.Minute)

	d = iso8601.Duration{Seconds: 1}
	assert.Equal(t, d.ToDuration(), time.Second)

	d = iso8601.Duration{Months: 2}
	fmt.Println(d.ToDuration())
}

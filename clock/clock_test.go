package clock

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Both implementations satisfy the interface.
var (
	_ Clock = Real{}
	_ Clock = (*Fake)(nil)
)

var epoch = time.Date(2030, time.January, 1, 12, 0, 0, 0, time.UTC)

func TestRealNowIsTheCurrentTime(t *testing.T) {
	before := time.Now()
	got := Real{}.Now()
	after := time.Now()

	assert.False(t, got.Before(before), "Now went backwards: %v < %v", got, before)
	assert.False(t, got.After(after), "Now is in the future: %v > %v", got, after)
}

func TestRealAfterFuncRunsTheFunction(t *testing.T) {
	ran := make(chan struct{})
	Real{}.AfterFunc(time.Millisecond, func() { close(ran) })

	select {
	case <-ran:
	case <-time.After(10 * time.Second):
		t.Fatal("the function did not run")
	}
}

func TestRealStopPreventsTheFunctionFromRunning(t *testing.T) {
	var ran atomic.Bool
	timer := Real{}.AfterFunc(time.Hour, func() { ran.Store(true) })

	assert.True(t, timer.Stop(), "first Stop of a pending timer reports true")
	assert.False(t, timer.Stop(), "second Stop reports false: nothing left to stop")
	assert.False(t, ran.Load())
}

func TestRealStopAfterTheFunctionRanReportsFalse(t *testing.T) {
	ran := make(chan struct{})
	timer := Real{}.AfterFunc(time.Millisecond, func() { close(ran) })
	select {
	case <-ran:
	case <-time.After(10 * time.Second):
		t.Fatal("the function did not run")
	}

	assert.False(t, timer.Stop())
}

// Stopping many timers from many goroutines must neither hang nor crash. (The
// timer of the clock library this replaces copied a time.Timer by value and was
// seen to hang or abort the process on current Go when stopped.)
func TestRealStopManyTimersConcurrently(t *testing.T) {
	const goroutines, perGoroutine = 8, 500
	var wg sync.WaitGroup
	var stopped atomic.Int64
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				timer := Real{}.AfterFunc(time.Hour, func() {})
				if timer.Stop() {
					stopped.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(goroutines*perGoroutine), stopped.Load())
}

func TestFakeStartsAtTheGivenTimeAndOnlyMovesWhenAdvanced(t *testing.T) {
	f := NewFake(epoch)
	assert.True(t, f.Now().Equal(epoch))

	f.Advance(90 * time.Second)
	assert.True(t, f.Now().Equal(epoch.Add(90*time.Second)), "got %v", f.Now())
}

func TestFakeFiresTimersAtTheirDueTimeAndNotBefore(t *testing.T) {
	f := NewFake(epoch)
	var fired []string
	f.AfterFunc(10*time.Second, func() { fired = append(fired, "ten") })

	f.Advance(10*time.Second - time.Nanosecond)
	assert.Empty(t, fired, "one nanosecond before the due time")

	f.Advance(time.Nanosecond)
	assert.Equal(t, []string{"ten"}, fired, "exactly at the due time")

	f.Advance(time.Hour)
	assert.Equal(t, []string{"ten"}, fired, "a timer fires only once")
}

func TestFakeFiresTimersInTimeOrderAndTiesInCreationOrder(t *testing.T) {
	f := NewFake(epoch)
	var fired []string
	add := func(name string, d time.Duration) {
		f.AfterFunc(d, func() { fired = append(fired, name) })
	}
	add("c", 3*time.Second)
	add("a", 1*time.Second)
	add("b1", 2*time.Second)
	add("b2", 2*time.Second)
	add("b3", 2*time.Second)

	f.Advance(time.Minute)

	assert.Equal(t, []string{"a", "b1", "b2", "b3", "c"}, fired)
}

func TestFakeNowInsideAFunctionIsItsDueTime(t *testing.T) {
	f := NewFake(epoch)
	seen := map[string]time.Time{}
	f.AfterFunc(2*time.Second, func() { seen["two"] = f.Now() })
	f.AfterFunc(5*time.Second, func() { seen["five"] = f.Now() })

	f.Advance(time.Minute)

	assert.True(t, seen["two"].Equal(epoch.Add(2*time.Second)), "got %v", seen["two"])
	assert.True(t, seen["five"].Equal(epoch.Add(5*time.Second)), "got %v", seen["five"])
	assert.True(t, f.Now().Equal(epoch.Add(time.Minute)), "after Advance the clock is at the end of the window")
}

func TestFakeTimerCreatedByAFunctionFiresInOrderWithinTheSameAdvance(t *testing.T) {
	f := NewFake(epoch)
	var fired []string
	f.AfterFunc(1*time.Second, func() {
		fired = append(fired, "first")
		// Due at 1s + 2s = 3s, which is inside the 4s window, after "mid" (2s).
		f.AfterFunc(2*time.Second, func() { fired = append(fired, "child") })
		// Due at 1s + 10s = 11s, outside the window.
		f.AfterFunc(10*time.Second, func() { fired = append(fired, "late") })
	})
	f.AfterFunc(2*time.Second, func() { fired = append(fired, "mid") })

	f.Advance(4 * time.Second)
	assert.Equal(t, []string{"first", "mid", "child"}, fired)

	f.Advance(7 * time.Second) // now at 11s
	assert.Equal(t, []string{"first", "mid", "child", "late"}, fired)
}

func TestFakeStop(t *testing.T) {
	t.Run("a stopped timer never fires and Stop reports true once", func(t *testing.T) {
		f := NewFake(epoch)
		var ran bool
		timer := f.AfterFunc(time.Second, func() { ran = true })

		assert.True(t, timer.Stop())
		assert.False(t, timer.Stop())
		f.Advance(time.Hour)
		assert.False(t, ran)
	})
	t.Run("Stop after the timer fired reports false", func(t *testing.T) {
		f := NewFake(epoch)
		timer := f.AfterFunc(time.Second, func() {})
		f.Advance(time.Second)

		assert.False(t, timer.Stop())
	})
	t.Run("a function can stop a later timer in the same Advance", func(t *testing.T) {
		f := NewFake(epoch)
		var fired []string
		var later Timer
		f.AfterFunc(1*time.Second, func() {
			fired = append(fired, "early")
			assert.True(t, later.Stop())
		})
		later = f.AfterFunc(2*time.Second, func() { fired = append(fired, "later") })

		f.Advance(time.Minute)
		assert.Equal(t, []string{"early"}, fired)
	})
}

func TestFakeZeroAndNegativeDurationsFireOnTheNextAdvance(t *testing.T) {
	f := NewFake(epoch)
	var fired []string
	f.AfterFunc(0, func() { fired = append(fired, "zero") })
	f.AfterFunc(-time.Hour, func() { fired = append(fired, "negative") })
	assert.Empty(t, fired, "nothing runs until the clock is advanced")

	f.Advance(0)

	assert.Equal(t, []string{"zero", "negative"}, fired)
	assert.True(t, f.Now().Equal(epoch), "Advance(0) does not move the clock")
}

func TestFakeAdvanceRejectsANegativeDuration(t *testing.T) {
	f := NewFake(epoch)
	assert.PanicsWithValue(t, "clock: Fake.Advance called with a negative duration", func() {
		f.Advance(-time.Second)
	})
}

// The fake is safe to use from several goroutines at once (run with -race).
// Every timer that Stop reported as stopped must not fire, and every other
// timer must fire exactly once by the end.
func TestFakeIsSafeForConcurrentUse(t *testing.T) {
	const goroutines, perGoroutine = 4, 200
	f := NewFake(epoch)
	var fired, stopped atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				timer := f.AfterFunc(time.Duration(i)*time.Millisecond, func() { fired.Add(1) })
				if i%3 == 0 && timer.Stop() {
					stopped.Add(1)
				}
				_ = f.Now()
			}
		}()
	}
	for i := 0; i < 50; i++ {
		f.Advance(10 * time.Millisecond)
	}
	wg.Wait()
	f.Advance(time.Hour)

	total := int64(goroutines * perGoroutine)
	assert.Equal(t, total-stopped.Load(), fired.Load())
	assert.Positive(t, stopped.Load(), "the test stopped no timer, so it proves nothing about Stop")
}

func TestFakeStopFromInsideItsOwnFunctionReportsFalse(t *testing.T) {
	f := NewFake(epoch)
	var timer Timer
	var stopResult = true
	timer = f.AfterFunc(time.Second, func() { stopResult = timer.Stop() })

	f.Advance(time.Second)

	assert.False(t, stopResult, "the function had already started")
}

func TestFakeZeroDurationTimerCreatedByAFunctionRunsInTheSameAdvance(t *testing.T) {
	f := NewFake(epoch)
	var fired []string
	f.AfterFunc(time.Second, func() {
		fired = append(fired, "outer")
		f.AfterFunc(0, func() { fired = append(fired, "inner at "+f.Now().Sub(epoch).String()) })
	})

	f.Advance(time.Second)

	assert.Equal(t, []string{"outer", "inner at 1s"}, fired)
}

func TestFakeRejectedNegativeAdvanceChangesNothing(t *testing.T) {
	f := NewFake(epoch)
	var ran bool
	f.AfterFunc(time.Second, func() { ran = true })

	assert.Panics(t, func() { f.Advance(-time.Second) })

	assert.True(t, f.Now().Equal(epoch))
	f.Advance(time.Second)
	assert.True(t, ran, "the pending timer is untouched")
}

func TestFakeAPanickingFunctionLeavesTheOtherTimersPending(t *testing.T) {
	f := NewFake(epoch)
	var later bool
	f.AfterFunc(time.Second, func() { panic("boom") })
	f.AfterFunc(2*time.Second, func() { later = true })

	assert.PanicsWithValue(t, "boom", func() { f.Advance(time.Minute) })
	assert.True(t, f.Now().Equal(epoch.Add(time.Second)), "the clock stays at the panicking timer's due time")
	assert.False(t, later)

	f.Advance(time.Minute)
	assert.True(t, later, "the timer that had not run is still pending")
}

// Two Advance calls at once: the clock never ends up earlier than the furthest
// target. The first call is held inside a function while the second one, with
// a later target, completes.
func TestFakeConcurrentAdvanceNeverMovesTimeBackwards(t *testing.T) {
	f := NewFake(epoch)
	inside := make(chan struct{})
	release := make(chan struct{})
	f.AfterFunc(time.Second, func() {
		close(inside)
		<-release
	})

	first := make(chan struct{})
	go func() {
		defer close(first)
		f.Advance(10 * time.Second) // target epoch+10s, blocked in the function above
	}()
	<-inside

	// The clock is at epoch+1s (the first call is inside its function), so this
	// call's target is epoch+1s+1h. It completes while the first call is held.
	f.Advance(time.Hour)
	close(release)
	<-first

	want := epoch.Add(time.Second + time.Hour)
	assert.True(t, f.Now().Equal(want), "got %v, want %v", f.Now(), want)
}

func TestFakeFiredCountsTimersThatRanAndNotStoppedOnes(t *testing.T) {
	f := NewFake(epoch)
	f.AfterFunc(time.Second, func() {})
	stopped := f.AfterFunc(2*time.Second, func() {})
	f.AfterFunc(3*time.Second, func() {})
	assert.Equal(t, 0, f.Fired())

	f.Advance(time.Second)
	assert.Equal(t, 1, f.Fired(), "the first timer ran during Advance")

	stopped.Stop()
	f.Advance(time.Second)
	assert.Equal(t, 1, f.Fired(), "a stopped timer does not count")

	f.Advance(time.Hour)
	assert.Equal(t, 2, f.Fired())
}

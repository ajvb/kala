package integrate

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/client"
	"github.com/ajvb/kala/job"
	"github.com/mixer/clock"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationTest(t *testing.T) {

	jobDB := &job.MockDB{}
	cache := job.NewLockFreeJobCache(jobDB)

	clk := clock.NewMockClock()
	cache.Clock.SetClock(clk)

	addr := freeTCPAddr(t)

	kalaApi := api.MakeServer(addr, cache, jobDB, "default@example.com")
	go kalaApi.ListenAndServe()
	kalaClient := client.New("http://" + addr)

	var hitCount int
	var lock sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		hitCount++
		lock.Unlock()
		w.WriteHeader(200)
	}))

	scheduleTime := clk.Now().Add(time.Minute * 3)
	parsedTime := scheduleTime.Format(time.RFC3339)
	delay := "PT5M"
	scheduleStr := fmt.Sprintf("R/%s/%s", parsedTime, delay)
	body := &job.Job{
		Name:     "Increment HitCount",
		Schedule: scheduleStr,
		JobType:  job.RemoteJob,
		RemoteProperties: job.RemoteProperties{
			Url:                   "http://" + srv.Listener.Addr().String(),
			Method:                http.MethodGet,
			ExpectedResponseCodes: []int{200},
		},
	}
	kalaClient.CreateJob(body)

	clk.AddTime(time.Minute)
	time.Sleep(time.Millisecond * 250)
	lock.Lock()
	assert.Equal(t, 0, hitCount)
	lock.Unlock()

	clk.AddTime(time.Minute * 3)
	time.Sleep(time.Millisecond * 500)
	lock.Lock()
	assert.Equal(t, 1, hitCount)
	lock.Unlock()

	clk.AddTime(time.Minute * 5)
	time.Sleep(time.Millisecond * 500)
	lock.Lock()
	assert.Equal(t, 2, hitCount)
	lock.Unlock()

}

func freeTCPAddr(t *testing.T) string {
	addr := &net.TCPAddr{
		IP: net.IPv4(127, 0, 0, 1),
	}

	lis, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	return lis.Addr().String()
}

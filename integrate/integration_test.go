package integrate

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/client"
	"github.com/ajvb/kala/job"
	"github.com/mixer/clock"
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

	hit := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit <- struct{}{}
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
	_, err := kalaClient.CreateJob(body)
	if err != nil {
		t.Fatal(err)
	}

	clk.AddTime(time.Minute)
	select {
	case <-hit:
		t.Fatalf("Did not expect job to have run yet.")
	case <-time.After(time.Second):
	}

	clk.AddTime(time.Minute)
	select {
	case <-hit:
		t.Fatalf("Still did not expect job to have run yet.")
	case <-time.After(time.Second):
	}

	clk.AddTime(time.Minute * 3)
	select {
	case <-hit:
	case <-time.After(time.Second * 5):
		t.Fatalf("Expected job to have run.")
	}

	clk.AddTime(time.Minute * 5)
	select {
	case <-hit:
	case <-time.After(time.Second * 5):
		t.Fatalf("Expected job to have run again.")
	}

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

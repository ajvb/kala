package integrate

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajvb/kala/client/v2"
	"github.com/ajvb/kala/v2/api"
	"github.com/ajvb/kala/v2/clock"
	"github.com/ajvb/kala/v2/job"
)

func TestIntegrationTest(t *testing.T) {

	jobDB := &job.MockDB{}
	clk := clock.NewFake(time.Now())
	cache := job.NewLockFreeJobCache(jobDB, job.WithClock(clk))

	// The listener is open before the client's first request, so the request
	// cannot arrive before the server is ready.
	listener := newLocalListener(t)
	addr := listener.Addr().String()

	kalaApi := api.MakeServer(addr, cache, "default@example.com", false)
	// Serve returns http.ErrServerClosed when the server is closed at the end.
	go func() { _ = kalaApi.Serve(listener) }()
	t.Cleanup(func() { _ = kalaApi.Close() })
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

	clk.Advance(time.Minute)
	select {
	case <-hit:
		t.Fatalf("Did not expect job to have run yet.")
	case <-time.After(time.Second):
	}

	clk.Advance(time.Minute)
	select {
	case <-hit:
		t.Fatalf("Still did not expect job to have run yet.")
	case <-time.After(time.Second):
	}

	clk.Advance(time.Minute * 3)
	select {
	case <-hit:
	case <-time.After(time.Second * 5):
		t.Fatalf("Expected job to have run.")
	}

	// The job schedules its next run after the current run finishes, which
	// happens in the background. A timer that is already due fires on the next
	// Advance, so keep advancing (by nothing) until the second run happens.
	clk.Advance(time.Minute * 5)
	deadline := time.After(time.Second * 5)
	for ran := false; !ran; {
		select {
		case <-hit:
			ran = true
		case <-deadline:
			t.Fatalf("Expected job to have run again.")
		case <-time.After(time.Millisecond * 10):
			clk.Advance(0)
		}
	}

}

// newLocalListener opens a listener on a free local port.
func newLocalListener(t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			t.Fatalf("failed to listen on a port: %v", err)
		}
	}
	return l
}

package test_utils

import (
	"fmt"
	"net/http/httptest"
	"time"

	"github.com/ajvb/kala/api"

	"github.com/gorilla/mux"
)

func NewTestServer() *httptest.Server {
	r := mux.NewRouter()
	api.SetupApiRoutes(r)
	return httptest.NewServer(r)
}

func NewJobMap() map[string]string {
	scheduleTime := time.Now().Add(time.Minute * 5)
	repeat := 1
	delay := "P1DT10M10S"
	parsedTime := scheduleTime.Format(time.RFC3339)
	scheduleStr := fmt.Sprintf("R%d/%s/%s", repeat, parsedTime, delay)

	return map[string]string{
		"schedule": scheduleStr,
		"name":     "mock_job",
		"command":  "bash -c 'date'",
		"owner":    "aj@ajvb.me",
	}
}

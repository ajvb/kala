package api

import (
	"fmt"
	"time"

	//"github.com/stretchr/testify/assert"
	"testing"
)

func generateNewJobMap() map[string]string {
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

func TestHandleAddJob(t *testing.T) {
}
func TestHandleAddJobBadJson(t *testing.T) {
}

func TestHandleJobRequest(t *testing.T) {
}
func TestHandleJobRequestNotFound(t *testing.T) {
}

func TestHandleListJobStatsRequest(t *testing.T) {
}
func TestHandleListJobStatsRequestNotFound(t *testing.T) {
}

func TestHandleListJobsRequest(t *testing.T) {
}

func TestHandleStartJobRequest(t *testing.T) {
}
func TestHandleStartJobRequestNotFound(t *testing.T) {
}

func TestHandleKalaStatsRequest(t *testing.T) {
}

func TestHandleNotFound(t *testing.T) {
}

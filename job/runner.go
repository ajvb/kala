package job

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/mattn/go-shellwords"
	log "github.com/sirupsen/logrus"
)

type JobRunner struct {
	job  *Job
	meta Metadata

	numberOfAttempts uint
	currentRetries   uint
	currentStat      *JobStat
}

var (
	ErrJobDisabled       = errors.New("Job cannot run, as it is disabled")
	ErrCmdIsEmpty        = errors.New("Job Command is empty.")
	ErrJobTypeInvalid    = errors.New("Job Type is not valid.")
	ErrInvalidDelimiters = errors.New("Job has invalid templating delimiters.")
)

// Run calls the appropriate run function, collects metadata around the success
// or failure of the Job's execution, and schedules the next run.
func (j *JobRunner) Run(cache JobCache) (*JobStat, Metadata, error) {
	j.job.lock.RLock()
	defer j.job.lock.RUnlock()

	j.meta.LastAttemptedRun = j.job.clk.Time().Now()

	if j.job.Disabled {
		log.Infof("Job %s tried to run, but exited early because its disabled.", j.job.Name)
		return nil, j.meta, ErrJobDisabled
	}

	log.Infof("Job %s:%s started.", j.job.Name, j.job.Id)

	j.runSetup()

	var out string
	for {
		var err error
		switch {
		case j.job.succeedInstantly:
			out = "Job succeeded instantly for test purposes."
		case j.job.JobType == LocalJob:
			out, err = j.LocalRun()
		case j.job.JobType == RemoteJob:
			out, err = j.RemoteRun()
		default:
			err = ErrJobTypeInvalid
		}

		if err != nil {
			// Log Error in Metadata
			// TODO - Error Reporting, email error
			log.Errorln("Error running job:", j.currentStat.JobId)
			log.Errorln(err)

			j.meta.ErrorCount++
			j.meta.LastError = j.job.clk.Time().Now()

			// Handle retrying
			if j.shouldRetry() {
				j.currentRetries--
				continue
			}

			j.collectStats(false)
			j.meta.NumberOfFinishedRuns++

			// TODO: Wrap error into something better.
			return j.currentStat, j.meta, err
		} else {
			break
		}
	}

	log.Infof("Job %s:%s finished.", j.job.Name, j.job.Id)
	log.Debugf("Job %s:%s output: %s", j.job.Name, j.job.Id, out)
	j.meta.SuccessCount++
	j.meta.NumberOfFinishedRuns++
	j.meta.LastSuccess = j.job.clk.Time().Now()

	j.collectStats(true)

	// Run Dependent Jobs
	if len(j.job.DependentJobs) != 0 {
		for _, id := range j.job.DependentJobs {
			newJob, err := cache.Get(id)
			if err != nil {
				log.Errorf("Error retrieving dependent job with id of %s", id)
			} else {
				newJob.Run(cache)
			}
		}
	}

	return j.currentStat, j.meta, nil
}

// LocalRun executes the Job's local shell command
func (j *JobRunner) LocalRun() (string, error) {
	return j.runCmd()
}

// RemoteRun sends a http request, and checks if the response is valid in time,
func (j *JobRunner) RemoteRun() (string, error) {
	// Calculate a response timeout
	timeout := j.responseTimeout()

	ctx := context.Background()
	if timeout > 0 {
		var cncl func()
		ctx, cncl = context.WithTimeout(ctx, timeout)
		defer cncl()
	}

	// Get the actual url and body we're going to be using,
	// including any necessary templating.
	url, err := j.tryTemplatize(j.job.RemoteProperties.Url)
	if err != nil {
		return "", fmt.Errorf("Error templatizing url: %v", err)
	}
	body, err := j.tryTemplatize(j.job.RemoteProperties.Body)
	if err != nil {
		return "", fmt.Errorf("Error templatizing body: %v", err)
	}

	// Normalize the method passed by the user
	method := strings.ToUpper(j.job.RemoteProperties.Method)
	bodyBuffer := &bytes.Buffer{}
	bodyBuffer = bytes.NewBufferString(body)
	fmt.Println(bodyBuffer.String())
	req, err := http.NewRequest(method, url, bodyBuffer)
	if err != nil {
		return "", err
	}

	// Set default or user's passed headers
	j.setHeaders(req)

	// Do the request
	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	// Check if we got any of the status codes the user asked for
	if j.checkExpected(res.StatusCode) {
		return string(b), nil
	} else {
		return "", errors.New(res.Status + string(b))
	}
}

func initShParser() *shellwords.Parser {
	shParser := shellwords.NewParser()
	shParser.ParseEnv = true
	shParser.ParseBacktick = true
	return shParser
}

func (j *JobRunner) runCmd() (string, error) {
	j.numberOfAttempts++

	// Get the actual command we're going to be running,
	// including any necessary templating.
	cmdText, err := j.tryTemplatize(j.job.Command)
	if err != nil {
		return "", fmt.Errorf("Error templatizing command: %v", err)
	}

	// Execute command
	shParser := initShParser()
	args, err := shParser.Parse(cmdText)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", ErrCmdIsEmpty
	}

	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // That's the job description
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (j *JobRunner) tryTemplatize(content string) (string, error) {
	delims := j.job.TemplateDelimiters

	if delims == "" {
		return content, nil
	}

	split := strings.Split(delims, " ")
	if len(split) != 2 { //nolint:gomnd
		return "", ErrInvalidDelimiters
	}

	left, right := split[0], split[1]
	if left == "" || right == "" {
		return "", ErrInvalidDelimiters
	}

	t, err := template.New("tmpl").Delims(left, right).Parse(content)
	if err != nil {
		return "", fmt.Errorf("Error parsing template: %v", err)
	}

	b := bytes.NewBuffer(nil)
	if err := t.Execute(b, j.job); err != nil {
		return "", fmt.Errorf("Error executing template: %v", err)
	}

	return b.String(), nil
}

func (j *JobRunner) shouldRetry() bool {
	// Check number of retries left
	if j.currentRetries == 0 {
		return false
	}

	// Check Epsilon
	if j.job.Epsilon != "" && j.job.Schedule != "" {
		if !j.job.epsilonDuration.IsZero() {
			timeSinceStart := j.job.clk.Time().Now().Sub(j.job.NextRunAt)
			timeLeftToRetry := j.job.epsilonDuration.RelativeTo(j.job.clk.Time().Now()) - timeSinceStart
			if timeLeftToRetry < 0 {
				return false
			}
		}
	}

	return true
}

func (j *JobRunner) runSetup() {
	// Setup Job Stat
	j.currentStat = NewJobStat(j.job.Id)

	// Init retries
	j.currentRetries = j.job.Retries
}

func (j *JobRunner) collectStats(success bool) {
	j.currentStat.ExecutionDuration = j.job.clk.Time().Now().Sub(j.currentStat.RanAt)
	j.currentStat.Success = success
	j.currentStat.NumberOfRetries = j.job.Retries - j.currentRetries
}

func (j *JobRunner) checkExpected(statusCode int) bool {
	// If no expected response codes passed, add 200 status code as expected
	if len(j.job.RemoteProperties.ExpectedResponseCodes) == 0 {
		j.job.RemoteProperties.ExpectedResponseCodes = append(j.job.RemoteProperties.ExpectedResponseCodes, 200)
	}
	for _, expected := range j.job.RemoteProperties.ExpectedResponseCodes {
		if expected == statusCode {
			return true
		}
	}

	return false
}

// responseTimeout sets a default timeout if none specified
func (j *JobRunner) responseTimeout() time.Duration {
	responseTimeout := j.job.RemoteProperties.Timeout
	if responseTimeout == 0 {

		// set default to 30 seconds
		responseTimeout = 30
	}
	return time.Duration(responseTimeout) * time.Second
}

// setHeaders sets default and user specific headers to the http request
func (j *JobRunner) setHeaders(req *http.Request) {
	if j.job.RemoteProperties.Headers == nil {
		j.job.RemoteProperties.Headers = http.Header{}
	}
	// A valid assumption is that the user is sending something in json cause we're past 2017
	if j.job.RemoteProperties.Headers["Content-Type"] == nil {
		j.job.RemoteProperties.Headers["Content-Type"] = []string{"application/json"}
	}
	req.Header = j.job.RemoteProperties.Headers
}

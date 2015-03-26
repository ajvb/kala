package job

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"../utils/iso8601"

	"github.com/222Labs/common/go/logging"
	"github.com/boltdb/bolt"
	"github.com/nu7hatch/gouuid"
)

var (
	AllJobs             = make(map[string]*Job)
	SaveAllJobsWaitTime = time.Duration(5 * time.Second)

	db = getDB()

	jobBucket = []byte("jobs")

	log = logging.GetLogger("kala")
)

func init() {
	// Prep cache
	allJobs, err := GetAllJobs()
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range allJobs {
		AllJobs[v.Id] = v
	}
	// Occasionally, save items in cache to db.
	go SaveAllJobsEvery(SaveAllJobsWaitTime)
}

func getDB() *bolt.DB {
	database, err := bolt.Open("jobdb.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	return database
}

func StartWatchingAllJobs() error {
	allJobs, err := GetAllJobs()
	if err != nil {
		return err
	}

	for _, v := range allJobs {
		go v.StartWaiting()
	}

	return nil
}

func GetAllJobs() ([]*Job, error) {
	allJobs := make([]*Job, 0)

	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(jobBucket)
		if err != nil {
			return err
		}

		err = bucket.ForEach(func(k, v []byte) error {
			buffer := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buffer)
			j := new(Job)
			err := dec.Decode(j)

			allJobs = append(allJobs, j)

			return err
		})

		return err
	})

	return allJobs, err
}

func GetJob(id string) (*Job, error) {
	j := new(Job)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(jobBucket)

		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("Job with id of %s not found.", id)
		}

		buffer := bytes.NewBuffer(v)
		dec := gob.NewDecoder(buffer)
		err := dec.Decode(j)

		return err
	})
	if err != nil {
		return nil, err
	}

	j.Init()
	j.Id = id
	return j, err
}

func (j *Job) Delete() {
	j.Disable()
	delete(AllJobs, j.Id)
	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(jobBucket)
		bucket.Delete([]byte(j.Id))
		return nil
	})
}

func (j *Job) Save() error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(jobBucket)
		if err != nil {
			return err
		}

		buffer := new(bytes.Buffer)
		enc := gob.NewEncoder(buffer)
		err = enc.Encode(j)
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(j.Id), buffer.Bytes())
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func SaveAllJobs() error {
	for _, v := range AllJobs {
		err := v.Save()
		if err != nil {
			return err
		}
	}
	return nil
}

func SaveAllJobsEvery(waitTime time.Duration) {
	for {
		time.Sleep(waitTime)
		go SaveAllJobs()
	}
}

type Job struct {
	Name string `json:"name"`
	Id   string `json:"id"`

	// Command to run
	// e.g. "bash /path/to/my/script.sh"
	Command string `json:"command"`

	// Email of the owner of this job
	// e.g. "admin@example.com"
	Owner string `json:"owner"`

	// Is this job disabled?
	Disabled bool `json:"disabled"`

	// Jobs that are dependent upon this one.
	// Will be run after this job runs.
	DependentJobs []string `json:"dependent_jobs"`
	ParentJobs    []string `json:"parent_jobs"`

	// ISO 8601 String
	// e.g. "R/2014-03-08T20:00:00.000Z/PT2H"
	Schedule     string `json:"schedule"`
	scheduleTime time.Time
	// ISO 8601 Duration struct, used for scheduling
	// job after each run.
	delayDuration *iso8601.Duration

	// Number of times to schedule this job after the
	// first run.
	timesToRepeat int64

	// Number of times to retry on failed attempt for each run.
	Retries         uint `json:"retries"`
	currentRetries  uint
	Epsilon         string `json:"epsilon"`
	epsilonDuration *iso8601.Duration

	// Meta data about successful and failed runs.
	SuccessCount     uint      `json:"success_count"`
	LastSuccess      time.Time `json:"last_success"`
	ErrorCount       uint      `json:"error_count"`
	LastError        time.Time `json:"last_error"`
	LastAttemptedRun time.Time `json:"last_attempted_run"`

	jobTimer  *time.Timer
	nextRunAt time.Time

	// TODO
	// RunAsUser string `json:""`
	// EnvironmentVariables map[string]string `json:""`
}

// Init() fills in the protected fields and parses the iso8601 notation.
func (j *Job) Init() error {
	u4, err := uuid.NewV4()
	if err != nil {
		log.Error("Error occured when generating uuid: %s", err)
		return err
	}
	j.Id = u4.String()

	if len(j.ParentJobs) != 0 {
		// Add new job to parent jobs
		for _, p := range j.ParentJobs {
			AllJobs[p].DependentJobs = append(AllJobs[p].DependentJobs, j.Id)
		}
		return nil
	}

	if j.Schedule == "" {
		// If schedule is empty, its a one-off job.
		go j.Run()
		return nil
	}

	splitTime := strings.Split(j.Schedule, "/")
	if len(splitTime) != 3 {
		return fmt.Errorf("Schedule not formatted correctly. Should look like: R/2014-03-08T20:00:00Z/PT2H")
	}

	// Handle Repeat Amount
	if splitTime[0] == "R" {
		// Repeat forever
		j.timesToRepeat = -1
	} else {
		j.timesToRepeat, err = strconv.ParseInt(strings.Split(splitTime[0], "R")[1], 10, 0)
		if err != nil {
			log.Error("Error converting timesToRepeat to an int: %s", err)
			return err
		}
	}
	log.Debug("timesToRepeat: %d", j.timesToRepeat)

	j.scheduleTime, err = time.Parse(time.RFC3339, splitTime[1])
	if err != nil {
		log.Error("Error converting scheduleTime to a time.Time: %s", err)
		return err
	}
	if (time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())) < 0 {
		return fmt.Errorf("Schedule time has passed.")
	}
	log.Debug("Schedule Time: %s", j.scheduleTime)

	j.delayDuration, err = iso8601.FromString(splitTime[2])
	if err != nil {
		log.Error("Error converting delayDuration to a iso8601.Duration: %s", err)
		return err
	}
	log.Debug("Delay Duration: %s", j.delayDuration.ToDuration())

	if j.Epsilon != "" {
		j.epsilonDuration, err = iso8601.FromString(j.Epsilon)
		if err != nil {
			log.Error("Error converting j.Epsilon to iso8601.Duration: %s", err)
			return err
		}
	}

	j.StartWaiting()

	return nil
}

// StartWaiting begins a timer for when it should execute the Jobs .Run() method.
func (j *Job) StartWaiting() {
	waitDuration := time.Duration(j.scheduleTime.UnixNano() - time.Now().UnixNano())
	log.Debug("Wait Duration initial: %s", waitDuration)
	if waitDuration < 0 {
		// Needs to be recalculated each time because of Months.
		waitDuration = j.delayDuration.ToDuration()
	}
	log.Info("Job Scheduled to run in: %s", waitDuration)
	j.nextRunAt = time.Now().Add(waitDuration)
	j.jobTimer = time.AfterFunc(waitDuration, j.Run)
}

func (j *Job) Disable() {
	// TODO - revisit error handling
	//hasBeenStopped := j.jobTimer.Stop()
	_ = j.jobTimer.Stop()
	j.Disabled = true
}

// Run() executes the Job's command, collects metadata around the success
// or failure of the Job's execution, and schedules the next run.
func (j *Job) Run() {
	log.Info("Job %s running", j.Name)

	// Schedule next run
	if j.timesToRepeat != 0 {
		j.timesToRepeat -= 1
		go j.StartWaiting()
	}

	j.LastAttemptedRun = time.Now()

	// TODO - Make thread safe
	// Init retries
	if j.currentRetries == 0 && j.Retries != 0 {
		j.currentRetries = j.Retries
	}

	// Execute command
	args := strings.Split(j.Command, " ")
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()
	if err != nil {
		log.Error("Run Command got an Error: %s", err)
		j.ErrorCount += 1
		j.LastError = time.Now()
		// Handle retrying
		if j.currentRetries != 0 {
			if j.epsilonDuration.ToDuration() == 0 {
				timeLeftToRetry := time.Duration(j.epsilonDuration.ToDuration()) - time.Duration(time.Now().UnixNano()-j.nextRunAt.UnixNano())
				if timeLeftToRetry < 0 {
					// TODO - Make thread safe
					// Reset retries and exit.
					j.currentRetries = 0
					return
				}
			}
			j.currentRetries -= 0
			j.Run()
		}
		return
	}

	log.Info("%s was successful!", j.Name)
	j.SuccessCount += 1
	j.LastSuccess = time.Now()

	// Run Dependent Jobs
	if len(j.DependentJobs) != 0 {
		for _, id := range j.DependentJobs {
			go AllJobs[id].Run()
		}
	}
}

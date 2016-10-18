package consul

import (
	"github.com/ajvb/kala/job"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testJob struct {
	Job   *job.Job
	Bytes []byte
}

// testJobs initializes n testJobs
func initTestJobs(n int, cache job.JobCache) []testJob {
	testJobs := []testJob{}

	for i := 0; i < n; i++ {
		j := job.GetMockJobWithGenericSchedule()
		j.Init(cache)

		bytes, err := j.Bytes()
		if err != nil {
			panic(err)
		}
		t := testJob{
			Job:   j,
			Bytes: bytes,
		}

		testJobs = append(testJobs, t)
	}

	return testJobs
}

func TestSaveJob(t *testing.T) {
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	d := New(srv1.HTTPAddr)
	cache := job.NewMemoryJobCache(d)
	testJobs := initTestJobs(3, cache)

	for _, j := range testJobs {
		err := d.Save(j.Job)
		assert.Nil(t, err)
	}
}

func TestGetJob(t *testing.T) {
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	d := New(srv1.HTTPAddr)
	cache := job.NewMemoryJobCache(d)
	testJobs := initTestJobs(3, cache)

	for _, j := range testJobs {
		d.Save(j.Job)
	}

	for _, j := range testJobs {
		kala_job, err := d.Get(j.Job.Id)
		assert.Nil(t, err)

		assert.WithinDuration(t, kala_job.NextRunAt, j.Job.NextRunAt, 100*time.Microsecond)
		assert.Equal(t, kala_job.Name, j.Job.Name)
		assert.Equal(t, kala_job.Id, j.Job.Id)
		assert.Equal(t, kala_job.Command, j.Job.Command)
		assert.Equal(t, kala_job.Schedule, j.Job.Schedule)
		assert.Equal(t, kala_job.Owner, j.Job.Owner)
		assert.Equal(t, kala_job.Metadata.SuccessCount, j.Job.Metadata.SuccessCount)
	}
}

func TestGetAllJobs(t *testing.T) {
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	d := New(srv1.HTTPAddr)
	cache := job.NewMemoryJobCache(d)
	testJobs := initTestJobs(3, cache)

	jobs, err := d.GetAll()
	assert.Nil(t, err)
	assert.Empty(t, jobs)

	for _, j := range testJobs {
		d.Save(j.Job)
	}

	jobs, err = d.GetAll()
	assert.Nil(t, err)
	assert.NotEmpty(t, jobs)
}

func TestDeleteJob(t *testing.T) {
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	d := New(srv1.HTTPAddr)
	cache := job.NewMemoryJobCache(d)
	testJobs := initTestJobs(3, cache)

	err := d.Delete("not_a_real_id")
	assert.Nil(t, err)

	for _, j := range testJobs {
		d.Save(j.Job)
	}

	err = d.Delete(testJobs[0].Job.Id)
	assert.Nil(t, err)

	jobs, err := d.GetAll()
	assert.EqualValues(t, 2, len(jobs))
}

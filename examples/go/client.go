package client

import (
	"log"

	"github.com/ajvb/kala/client"
	"github.com/ajvb/kala/job"
)

func main() {
	c := client.New("http://127.0.0.1:8000")
	body := &job.Job{
		Schedule: "R2/2016-06-30T16:25:16.828696-07:00/PT10S",
		Name:     "test_job",
		Command:  "bash -c 'date'",
	}
	id, err := c.CreateJob(body)
	log.Println(id, err)

	hey, err := c.GetAllJobs()
	log.Println(hey, err)
}

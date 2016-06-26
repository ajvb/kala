package main

import (
	"log"
	"time"

	"github.com/cescoferraro/kala/client"
	"github.com/cescoferraro/kala/job"
)

func main() {

	c := client.New("http://127.0.0.1:8000")
	body := &job.Job{
		Schedule: getSchedule(),
		Name:     "cescojob",
		Command:  "touch teste.txt",
	}
	id, err := c.CreateJob(body)
	log.Println(id, err)

}

func getSchedule() string {
	thisTime := time.Now().Add(2 * time.Second).Format(time.RFC3339Nano)

	return "R1/" + thisTime + "/PT10S"
}

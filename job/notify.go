package job

import (
	"fmt"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	log "github.com/sirupsen/logrus"
)

func notify(toAddress string, subject string, msg string) error {
	from := mail.NewEmail("Scheduler Service", "no-reply@nextiva.com")
	to := mail.NewEmail(toAddress, toAddress)
	message := mail.NewSingleEmail(from, subject, to, msg, msg)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)

	if err != nil {
		log.Errorf("Error sending email: %s", err)
		return err
	}

	log.Debugf("Email sent with status %d, body %s, headers, %s", response.StatusCode, response.Body, response.Headers)
	return nil
}

func notifyOfJobFailure(j *Job, run *JobStat) error {
	subject := fmt.Sprintf("Job %s Failed", j.Name)

	url := fmt.Sprintf("<a href=\"http://kalaurl/webui/job/execution/%s\">Job Run Link</a>", run.Id)
	msg := fmt.Sprintf("Hi!  Please be advised that your job failed.  Details: %+v Job Run: %s", run, url)

	return notify(j.Owner, subject, msg)
}

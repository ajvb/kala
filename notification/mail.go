package notification

import (
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"
)

type Mailer struct {
	Username   string
	Host       string
	secret     string
	identity   string
	password   string
	serverAddr string
	from       string

	auth smtp.Auth
}

func NewMailerPlainAuth(serverAddr, from, username, password, identity, host string) *Mailer {
	m := &Mailer{
		Username:   username,
		Host:       host,
		identity:   identity,
		password:   password,
		serverAddr: serverAddr,
		from:       from,
	}
	m.auth = smtp.PlainAuth(identity, username, password, host)
	return m
}

type TemplateData struct {
	From    string
	To      []string
	Subject string
	Body    string
}

const emailTemplate = `
From: {{ .From }}
To: {{ .To }}
Subject: {{ .Subject }}

{{ .Body }}
`

func generateTemplate(from string, to []string, subject, body string) (bytes.Buffer, error) {
	var doc bytes.Buffer
	ctx := &TemplateData{
		From:    from,
		To:      to,
		Subject: subject,
		Body:    body,
	}
	t := template.New("emailTemplate")
	t, err := t.Parse(emailTemplate)
	if err != nil {
		log.Error("Error parsing emailTemplate. Err: %s", err)
		return doc, err
	}
	err = t.Execute(&doc, ctx)
	if err != nil {
		log.Error("Error rendering template. Err: %s", err)
		return doc, err
	}
	return doc, nil
}

func SendErrorReport(mailer *Mailer, jobErr error, jobName string, to []string) error {
	subject := fmt.Sprintf("Kala Err Report on %s", jobName)
	body := fmt.Sprintf("There has been an error on a job named %s. Err: %s", jobName, jobErr)
	msg, err := generateTemplate(mailer.from, to, subject, body)
	if err != nil {
		log.Error("Error generating template. Err: %s", err)
		return err
	}
	return SendMail(mailer, to, msg.Bytes())
}

func SendMail(mailer *Mailer, to []string, msg []byte) error {
	return smtp.SendMail(mailer.serverAddr, mailer.auth, mailer.from, to, msg)
}

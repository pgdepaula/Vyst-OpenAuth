package smtp

import (
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
)

type SMTPAdapter struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func NewSMTPAdapter(host, port, user, password, from string) *SMTPAdapter {
	return &SMTPAdapter{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		From:     from,
	}
}

func (s *SMTPAdapter) SendEmail(to, subject, body string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("invalid recipient email address: %w", err)
	}
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid email subject")
	}
	if strings.ContainsAny(s.From, "\r\n") {
		return fmt.Errorf("invalid sender email address")
	}

	auth := smtp.PlainAuth("", s.User, s.Password, s.Host)
	addr := fmt.Sprintf("%s:%s", s.Host, s.Port)

	headers := make(map[string]string)
	headers["From"] = s.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	return smtp.SendMail(addr, auth, s.From, []string{to}, []byte(message))
}

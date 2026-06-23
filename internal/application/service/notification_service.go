package service

import (
	"log/slog"
)

type NotificationService struct {
	emailSender interface {
		SendEmail(to, subject, body string) error
	}
	smsSender interface {
		SendSMS(to, content string) error
	}
}

func NewNotificationService(emailSender interface {
	SendEmail(to, subject, body string) error
}, smsSender interface {
	SendSMS(to, content string) error
}) *NotificationService {
	return &NotificationService{
		emailSender: emailSender,
		smsSender:   smsSender,
	}
}

func (s *NotificationService) SendEmail(to, subject, body string) error {
	// Async execution
	go func() {
		err := s.emailSender.SendEmail(to, subject, body)
		if err != nil {
			slog.Error("Failed to send email", "to", to, "error", err)
		}
	}()
	return nil
}

func (s *NotificationService) SendSMS(to, content string) error {
	// Async execution with fail-safe logging
	go func() {
		err := s.smsSender.SendSMS(to, content)
		if err != nil {
			slog.Error("Failed to send SMS", "to", to, "error", err)
		}
	}()
	return nil
}

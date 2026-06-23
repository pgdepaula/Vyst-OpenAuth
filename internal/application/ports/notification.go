package ports

type NotificationService interface {
	SendEmail(to, subject, body string) error
	SendSMS(to, content string) error
}

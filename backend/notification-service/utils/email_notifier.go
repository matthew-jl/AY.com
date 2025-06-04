package utils

import (
	"log"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

var (
	smtpHostNotif       string
	smtpPortNotif       int
	smtpUserNotif       string
	smtpPasswordNotif   string
	smtpSenderMailNotif string
)

func InitEmailNotifier() {
	smtpHostNotif = os.Getenv("SMTP_HOST")
	smtpUserNotif = os.Getenv("SMTP_USER")
	smtpPasswordNotif = os.Getenv("SMTP_PASSWORD")
	smtpSenderMailNotif = os.Getenv("SMTP_SENDER_EMAIL")
	portStr := os.Getenv("SMTP_PORT")
	if port, err := strconv.Atoi(portStr); err == nil { smtpPortNotif = port
	} else { smtpPortNotif = 587 }

	if smtpHostNotif == "" { log.Println("NotificationService: Email config missing.") }
}

func SendNotificationEmail(toEmail, subject, body string) error {
	if smtpHostNotif == "" {
		log.Printf("Email sending skipped for subject '%s': SMTP host not configured.", subject)
		return nil
	}
	m := gomail.NewMessage()
	m.SetHeader("From", smtpSenderMailNotif); m.SetHeader("To", toEmail)
	m.SetHeader("Subject", subject); m.SetBody("text/plain", body)
	d := gomail.NewDialer(smtpHostNotif, smtpPortNotif, smtpUserNotif, smtpPasswordNotif)
	log.Printf("Attempting to send notification email to %s, Subject: %s", toEmail, subject)
	if err := d.DialAndSend(m); err != nil {
		log.Printf("ERROR: Failed to send notification email to %s: %v", toEmail, err)
		return err
	}
	log.Printf("Notification email sent successfully to %s", toEmail)
	return nil
}
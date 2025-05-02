package utils

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

var (
	smtpHost       string
	smtpPort       int
	smtpUser       string
	smtpPassword   string
	smtpSenderMail string
)

func init() {
	smtpHost = os.Getenv("SMTP_HOST")
	smtpUser = os.Getenv("SMTP_USER")
	smtpPassword = os.Getenv("SMTP_PASSWORD")
	smtpSenderMail = os.Getenv("SMTP_SENDER_EMAIL")

	portStr := os.Getenv("SMTP_PORT")
	if port, err := strconv.Atoi(portStr); err == nil {
		smtpPort = port
	} else {
		log.Printf("Warning: Invalid or missing SMTP_PORT, using default 587. Error: %v", err)
		smtpPort = 587
	}

	if smtpHost == "" || smtpUser == "" || smtpPassword == "" || smtpSenderMail == "" {
		log.Println("Warning: Email configuration missing in environment variables. Email sending will likely fail.")
	}
}

func SendVerificationEmail(toEmail, code string) error {
	if smtpHost == "" {
		log.Println("Email sending skipped: SMTP host not configured.")
		return nil // don't block registration if email isn't configured
	}

	m := gomail.NewMessage()
	m.SetHeader("From", smtpSenderMail)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Verify Your AY.com Account")
	m.SetBody("text/plain", fmt.Sprintf("Welcome to AY.com!\n\nYour verification code is: %s\n\nThis code is valid for a limited time.", code))

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)

	d.TLSConfig = &tls.Config{
        ServerName: smtpHost,
		InsecureSkipVerify: false,
        MinVersion: tls.VersionTLS12,
    }

	log.Printf("Attempting to send verification email to %s via %s:%d", toEmail, smtpHost, smtpPort)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("ERROR: Failed to send verification email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	log.Printf("Verification email sent successfully to %s", toEmail)
	return nil
}

func SendWelcomeEmail(toEmail, name string) error {
	if smtpHost == "" {
		log.Println("Welcome email sending skipped: SMTP host not configured.")
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", smtpSenderMail)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Welcome to AY.com!")
	m.SetBody("text/plain", fmt.Sprintf("Hi %s,\n\nWelcome aboard!\n\nYour AY.com account has been successfully verified and activated.\n\nEnjoy the platform!\n\nThe AY.com Team", name))

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)
	log.Printf("Attempting to send welcome email to %s", toEmail)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("ERROR: Failed to send welcome email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	log.Printf("Welcome email sent successfully to %s", toEmail)
	return nil
}
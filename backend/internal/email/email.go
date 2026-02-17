package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

func (c *Config) IsConfigured() bool {
	return c.Host != "" && c.From != ""
}

func (c *Config) SendVerification(to, token, appURL string) error {
	if !c.IsConfigured() {
		return fmt.Errorf("email not configured")
	}

	verifyURL := strings.TrimRight(appURL, "/") + "/verify?token=" + token

	subject := "Verify your email - PUC Redyr Golf Scoring"
	body := fmt.Sprintf(
		"Welcome to PUC Redyr Golf Scoring!\r\n\r\n"+
			"Click the link below to verify your email address:\r\n\r\n"+
			"%s\r\n\r\n"+
			"If you did not create this account, you can ignore this email.",
		verifyURL,
	)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		c.From, to, subject, body,
	)

	addr := c.Host + ":" + c.Port

	var auth smtp.Auth
	if c.User != "" {
		auth = smtp.PlainAuth("", c.User, c.Pass, c.Host)
	}

	return smtp.SendMail(addr, auth, c.From, []string{to}, []byte(msg))
}

func (c *Config) SendNewUserNotification(adminEmails []string, userName, userEmail, appURL string) error {
	if !c.IsConfigured() || len(adminEmails) == 0 {
		return fmt.Errorf("email not configured or no admin emails")
	}

	manageURL := strings.TrimRight(appURL, "/") + "/admin/users"

	subject := "New user registration - PUC Redyr Golf Scoring"
	body := fmt.Sprintf(
		"A new user has registered and is awaiting approval:\r\n\r\n"+
			"Name: %s\r\nEmail: %s\r\n\r\n"+
			"Review and approve at:\r\n%s",
		userName, userEmail, manageURL,
	)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		c.From, strings.Join(adminEmails, ", "), subject, body,
	)

	addr := c.Host + ":" + c.Port

	var auth smtp.Auth
	if c.User != "" {
		auth = smtp.PlainAuth("", c.User, c.Pass, c.Host)
	}

	return smtp.SendMail(addr, auth, c.From, adminEmails, []byte(msg))
}

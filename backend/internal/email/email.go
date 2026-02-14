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

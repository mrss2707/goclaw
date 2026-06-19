package userauth

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	FromName string
}

type SMTPClient struct {
	config SMTPConfig
}

func NewSMTPClient(cfg SMTPConfig) *SMTPClient {
	return &SMTPClient{config: cfg}
}

func (c *SMTPClient) SendEmail(to, subject, htmlBody string) error {
	fromName := c.config.FromName
	if fromName == "" {
		fromName = "GoClaw"
	}
	from := c.config.Username

	headers := []string{
		"From: " + fromName + " <" + from + ">",
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		htmlBody,
	}
	msg := strings.Join(headers, "\r\n")

	addr := c.config.Host + ":" + c.config.Port
	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		slog.Warn("userauth.smtp.send_failed", "to", maskEmail(to), "error", err)
		return fmt.Errorf("send email: %w", err)
	}
	slog.Info("userauth.smtp.sent", "to", maskEmail(to))
	return nil
}

func (c *SMTPClient) SendVerificationCode(to, code string) error {
	subject := "Your GoClaw verification code: " + code
	body := verificationEmailTemplate(code)
	return c.SendEmail(to, subject, body)
}

func (c *SMTPClient) SendPasswordReset(to, code string) error {
	subject := "GoClaw password reset code: " + code
	body := passwordResetTemplate(code)
	return c.SendEmail(to, subject, body)
}

func GenerateVerificationCode() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: " + err.Error())
	}
	n := int(b[0])<<40 | int(b[1])<<32 | int(b[2])<<24 | int(b[3])<<16 | int(b[4])<<8 | int(b[5])
	return fmt.Sprintf("%06d", n%1000000)
}

func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) <= 2 {
		return email
	}
	return parts[0][:2] + "***@" + parts[1]
}

func verificationEmailTemplate(code string) string {
	return `<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;padding:20px">
<h2>Verify your email</h2>
<p>Your verification code is:</p>
<h1 style="font-size:36px;letter-spacing:8px">` + code + `</h1>
<p>This code expires in 15 minutes.</p>
<p>If you did not create a GoClaw account, ignore this email.</p>
</body></html>`
}

func passwordResetTemplate(code string) string {
	return `<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;padding:20px">
<h2>Reset your password</h2>
<p>Your password reset code is:</p>
<h1 style="font-size:36px;letter-spacing:8px">` + code + `</h1>
<p>This code expires in 15 minutes.</p>
<p>If you did not request a password reset, ignore this email.</p>
</body></html>`
}

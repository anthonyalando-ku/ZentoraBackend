// internal/service/email/service.go
package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailSender handles outgoing emails via SMTP.
type EmailSender struct {
	smtpHost string
	smtpPort string
	username string
	password string
	fromName string
	secure   bool
}

// NewEmailSender creates a new SMTP email sender.
func NewEmailSender(host, port, user, pass, fromName string, secure bool) *EmailSender {
	return &EmailSender{
		smtpHost: host,
		smtpPort: port,
		username: user,
		password: pass,
		fromName: fromName,
		secure:   secure,
	}
}

// Send sends an email with a subject and body (HTML supported).
func (e *EmailSender) Send(to, subject, bodyHTML string) error {
	from := fmt.Sprintf("%s <%s>", e.fromName, e.username)
	msg := []byte(
		fmt.Sprintf("From: %s\r\n", from) +
			fmt.Sprintf("To: %s\r\n", to) +
			fmt.Sprintf("Subject: %s\r\n", subject) +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"utf-8\"\r\n" +
			"\r\n" +
			buildHTMLTemplate(bodyHTML),
	)

	serverAddr := e.smtpHost + ":" + e.smtpPort

	// TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         e.smtpHost,
	}

	if e.secure {
		// Port 465 - implicit TLS
		conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls dial failed: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.smtpHost)
		if err != nil {
			return fmt.Errorf("smtp client failed: %w", err)
		}
		defer client.Quit()

		auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		if err := e.sendMail(client, from, to, msg); err != nil {
			return err
		}
		return nil
	}

	// Port 587 - STARTTLS
	auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
	if err := smtp.SendMail(serverAddr, auth, e.username, []string{to}, msg); err != nil {
		return fmt.Errorf("send mail failed: %w", err)
	}

	return nil
}

func (e *EmailSender) sendMail(client *smtp.Client, from, to string, msg []byte) error {
	if err := client.Mail(e.username); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close failed: %w", err)
	}
	return nil
}


// buildHTMLTemplate wraps a given body into a branded AuthY email layout.
func buildHTMLTemplate(content string) string {
	header := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8" />
		<title>AuthY</title>
		<style>
			body { font-family: Arial, sans-serif; background-color: #f6f8fa; padding: 30px; }
			.container { max-width: 600px; margin: auto; background: #fff; border-radius: 10px; overflow: hidden; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
			.header { background: #004aad; color: white; text-align: center; padding: 20px; font-size: 22px; font-weight: bold; }
			.footer { background: #f1f1f1; color: #555; text-align: center; padding: 15px; font-size: 13px; }
			.body { padding: 25px; color: #333; line-height: 1.6; }
			a.button { display: inline-block; background: #004aad; color: white; padding: 10px 20px; border-radius: 5px; text-decoration: none; }
		</style>
	</head>
	<body>
	<div class="container">
		<div class="header">AuthY</div>
		<div class="body">
	`

	footer := `
		</div>
		<div class="footer">
			<p>© 2025 AuthY. All rights reserved.</p>
		</div>
	</div>
	</body>
	</html>
	`

	return header + strings.TrimSpace(content) + footer
}

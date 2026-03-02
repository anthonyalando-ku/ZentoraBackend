package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// Brand colours
const (
	colorBlue   = "#004b8f"
	colorOrange = "#df7412"
	colorDark   = "#1a1a2e"
	colorBg     = "#f9f8f7"
	colorWhite  = "#ffffff"
	colorMuted  = "#6b7280"
	colorBorder = "#e5e7eb"
)

type EmailSender struct {
	smtpHost string
	smtpPort string
	username string
	password string
	fromName string
	secure   bool
	baseURL  string
	logoURL  string
}

func NewEmailSender(host, port, user, pass, fromName, baseURL, logoURL string, secure bool) *EmailSender {
	return &EmailSender{
		smtpHost: host,
		smtpPort: port,
		username: user,
		password: pass,
		fromName: fromName,
		secure:   secure,
		baseURL:  baseURL,
		logoURL:  logoURL,
	}
}

func (e *EmailSender) Send(to, subject, bodyHTML string) error {
	from := fmt.Sprintf("%s <%s>", e.fromName, e.username)
	msg := []byte(
		fmt.Sprintf("From: %s\r\n", from) +
			fmt.Sprintf("To: %s\r\n", to) +
			fmt.Sprintf("Subject: %s\r\n", subject) +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"utf-8\"\r\n" +
			"\r\n" +
			e.buildHTMLTemplate(bodyHTML),
	)

	serverAddr := e.smtpHost + ":" + e.smtpPort
	tlsConfig := &tls.Config{InsecureSkipVerify: false, ServerName: e.smtpHost}

	if e.secure {
		conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls dial: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.smtpHost)
		if err != nil {
			return fmt.Errorf("smtp client: %w", err)
		}
		defer client.Quit()

		if err := client.Auth(smtp.PlainAuth("", e.username, e.password, e.smtpHost)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		return e.sendMail(client, to, msg)
	}

	auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
	if err := smtp.SendMail(serverAddr, auth, e.username, []string{to}, msg); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}

func (e *EmailSender) sendMail(client *smtp.Client, to string, msg []byte) error {
	if err := client.Mail(e.username); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return w.Close()
}

// buildHTMLTemplate wraps content in the Zentora branded email shell.
// The logo is served from the static assets path configured at startup.
func (e *EmailSender) buildHTMLTemplate(content string) string {
	logoSrc := e.logoURL
	if logoSrc == "" {
		logoSrc = e.baseURL + "/static/assets/zentora_logo_clear.png"
	}

	header := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Zentora</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: 'Segoe UI', Arial, sans-serif;
      background-color: %s;
      padding: 32px 16px;
      color: %s;
    }
    .wrapper {
      max-width: 600px;
      margin: 0 auto;
    }
    /* ── Header ── */
    .email-header {
      background-color: %s;
      border-radius: 12px 12px 0 0;
      padding: 28px 32px;
      text-align: center;
    }
    .email-header img {
      height: 44px;
      width: auto;
    }
    /* ── Body ── */
    .email-body {
      background-color: %s;
      padding: 36px 40px;
      line-height: 1.7;
      font-size: 15px;
      border-left: 1px solid %s;
      border-right: 1px solid %s;
    }
    .email-body h1, .email-body h2 {
      color: %s;
      font-size: 20px;
      margin-bottom: 16px;
    }
    .email-body p {
      margin-bottom: 14px;
      color: %s;
    }
    /* ── Primary button ── */
    .btn-primary {
      display: inline-block;
      background-color: %s;
      color: %s !important;
      text-decoration: none;
      padding: 13px 28px;
      border-radius: 8px;
      font-weight: 600;
      font-size: 15px;
      margin: 20px 0;
      letter-spacing: 0.3px;
    }
    /* ── Accent button ── */
    .btn-accent {
      display: inline-block;
      background-color: %s;
      color: %s !important;
      text-decoration: none;
      padding: 13px 28px;
      border-radius: 8px;
      font-weight: 600;
      font-size: 15px;
      margin: 20px 0;
    }
    /* ── Info / alert boxes ── */
    .box {
      border-radius: 8px;
      padding: 18px 20px;
      margin: 20px 0;
      font-size: 14px;
    }
    .box-info {
      background-color: #e8f0fb;
      border-left: 4px solid %s;
    }
    .box-warning {
      background-color: #fef3e2;
      border-left: 4px solid %s;
    }
    .box-success {
      background-color: #ecfdf5;
      border-left: 4px solid #10b981;
    }
    .box-danger {
      background-color: #fef2f2;
      border-left: 4px solid #ef4444;
    }
    .box ul, .box ol {
      padding-left: 20px;
      margin-top: 8px;
    }
    .box li {
      margin-bottom: 4px;
    }
    /* ── OTP box ── */
    .otp-wrapper {
      text-align: center;
      margin: 28px 0;
    }
    .otp-code {
      display: inline-block;
      font-size: 40px;
      font-weight: 700;
      letter-spacing: 12px;
      color: %s;
      font-family: 'Courier New', monospace;
      background-color: #e8f0fb;
      padding: 20px 32px;
      border-radius: 12px;
      border: 2px dashed %s;
    }
    /* ── Divider ── */
    .divider {
      border: none;
      border-top: 1px solid %s;
      margin: 24px 0;
    }
    /* ── Credentials block ── */
    .credentials {
      background-color: #f0f4ff;
      border-left: 4px solid %s;
      border-radius: 8px;
      padding: 18px 20px;
      margin: 20px 0;
      font-size: 14px;
    }
    .credentials p {
      margin-bottom: 6px;
    }
    code {
      background-color: #e5e7eb;
      padding: 2px 7px;
      border-radius: 4px;
      font-family: 'Courier New', monospace;
      font-size: 13px;
      color: %s;
    }
    /* ── Footer ── */
    .email-footer {
      background-color: %s;
      border: 1px solid %s;
      border-top: none;
      border-radius: 0 0 12px 12px;
      padding: 20px 32px;
      text-align: center;
      font-size: 12px;
      color: %s;
      line-height: 1.6;
    }
    .email-footer a {
      color: %s;
      text-decoration: none;
    }
  </style>
</head>
<body>
<div class="wrapper">
  <div class="email-header">
    <img src="%s" alt="Zentora" />
  </div>
  <div class="email-body">`,
		colorBg,       // body bg
		colorDark,     // body text
		colorBlue,     // header bg
		colorWhite,    // body bg
		colorBorder,   // border left
		colorBorder,   // border right
		colorBlue,     // h1/h2 colour
		colorDark,     // p colour
		colorBlue,     // btn-primary bg
		colorWhite,    // btn-primary text
		colorOrange,   // btn-accent bg
		colorWhite,    // btn-accent text
		colorBlue,     // box-info border
		colorOrange,   // box-warning border
		colorBlue,     // otp-code colour
		colorBlue,     // otp-code border
		colorBorder,   // divider colour
		colorBlue,     // credentials border
		colorDark,     // code text
		colorBg,       // footer bg
		colorBorder,   // footer border
		colorMuted,    // footer text
		colorBlue,     // footer link
		logoSrc,       // logo img src
	)

	footer := fmt.Sprintf(`
  </div>
  <div class="email-footer">
    <p>This is an automated message — please do not reply directly to this email.</p>
    <p style="margin-top:8px;">
      Need help? <a href="mailto:support@zentora.com">support@zentora.com</a>
    </p>
    <hr class="divider" style="margin:14px 0;" />
    <p style="color:%s;">© 2026 Zentora. All rights reserved.</p>
  </div>
</div>
</body>
</html>`, colorMuted)

	return header + "\n" + strings.TrimSpace(content) + "\n" + footer
}
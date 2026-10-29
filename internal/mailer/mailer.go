// Package mailer sends plain-text emails over SMTP. Any provider works (Brevo, Mailgun, Gmail with an
// app password, ...): they all speak SMTP, so there's no SDK to learn.
package mailer

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Mailer struct {
	host     string // e.g. smtp-relay.brevo.com. Empty = dev mode, see Send
	port     string // 587: the port that starts plain and upgrades to TLS (STARTTLS)
	username string
	password string
	from     string // the address emails come from, e.g. no-reply@yourschool.com
}

func New(host, port, username, password, from string) *Mailer {
	return &Mailer{host: host, port: port, username: username, password: password, from: from}
}

// Send delivers one email. With no SMTP host configured it only prints the email to the server log, so
// the forgot-password flow can be tried without an email account. Never run production like that:
// reset codes would sit in the logs.
func (m *Mailer) Send(to, subject, body string) error {
	if m.host == "" {
		slog.Warn("SMTP_HOST not set, printing email instead of sending it", "to", to, "subject", subject, "body", body)
		return nil
	}
	// a header can't contain a line break; one in the address would let it add headers of its own
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("invalid recipient address")
	}

	// smtp.SendMail has no timeout, so a stuck mail server would hang the request forever.
	// Dialing ourselves lets us put a deadline on the whole conversation.
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(m.host, m.port), 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	c, err := smtp.NewClient(conn, m.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp newclient: %w", err)
	}
	defer c.Close()

	// encrypt before logging in. PlainAuth also refuses to send the password over a connection that isn't TLS.
	if err := c.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}
	if err := c.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(m.from); err != nil {
		return fmt.Errorf("smtp from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp to: %w", err)
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	// an email is headers, a blank line, then the body. SMTP wants \r\n line endings.
	msg := "From: " + m.from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		strings.ReplaceAll(body, "\n", "\r\n")
	if _, err := wc.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return c.Quit()
}

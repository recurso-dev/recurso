package notification

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

type SMTPNotifier struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPNotifier(host, port, username, password, from string) port.Notifier {
	return &SMTPNotifier{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (n *SMTPNotifier) SendEmail(ctx context.Context, to string, subject string, body string) error {
	addr := fmt.Sprintf("%s:%s", n.host, n.port)
	auth := smtp.PlainAuth("", n.username, n.password, n.host)

	// If no authentication (e.g. Mailhog), auth can be nil if the server allows it,
	// but net/smtp requires auth for non-local or specific setups.
	// Mailhog allows no auth. We'll handle that if username is empty.
	if n.username == "" {
		auth = nil
	}

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=\"utf-8\"\r\n"+
		"\r\n"+
		"%s\r\n", to, subject, body))

	if err := smtp.SendMail(addr, auth, n.from, []string{to}, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

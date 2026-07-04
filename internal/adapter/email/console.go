package email

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// ConsoleSender implements EmailSender by logging to console (for development)
type ConsoleSender struct{}

func NewConsoleSender() *ConsoleSender {
	return &ConsoleSender{}
}

func (s *ConsoleSender) Send(ctx context.Context, msg port.EmailMessage) error {
	log.Printf(`
================================================================================
📧 EMAIL SENT (Console Mode)
================================================================================
To:      %s
Subject: %s
Time:    %s
--------------------------------------------------------------------------------
%s
================================================================================
`, msg.To, msg.Subject, time.Now().Format(time.RFC3339), msg.HTMLBody)

	return nil
}

// LoggingSender wraps another sender and logs all emails
type LoggingSender struct {
	inner port.EmailSender
}

func NewLoggingSender(inner port.EmailSender) *LoggingSender {
	return &LoggingSender{inner: inner}
}

func (s *LoggingSender) Send(ctx context.Context, msg port.EmailMessage) error {
	log.Printf("[EMAIL] Sending to %s: %s", msg.To, msg.Subject)

	err := s.inner.Send(ctx, msg)
	if err != nil {
		log.Printf("[EMAIL] Failed to send to %s: %v", msg.To, err)
		return fmt.Errorf("email send failed: %w", err)
	}

	log.Printf("[EMAIL] Successfully sent to %s", msg.To)
	return nil
}

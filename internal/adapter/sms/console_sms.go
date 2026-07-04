package sms

import (
	"context"
	"log"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// ConsoleSMSSender logs SMS messages to console (for development)
type ConsoleSMSSender struct{}

func NewConsoleSMSSender() port.SMSSender {
	return &ConsoleSMSSender{}
}

func (s *ConsoleSMSSender) Send(ctx context.Context, msg port.SMSMessage) error {
	log.Printf("[SMS Console] To: %s | Body: %s", msg.To, msg.Body)
	return nil
}

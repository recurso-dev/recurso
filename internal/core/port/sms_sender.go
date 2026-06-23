package port

import "context"

// SMSMessage represents an SMS to be sent
type SMSMessage struct {
	To   string
	Body string
}

// SMSSender sends SMS messages
type SMSSender interface {
	Send(ctx context.Context, msg SMSMessage) error
}

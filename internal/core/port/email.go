package port

import "context"

// EmailMessage represents an email to be sent
type EmailMessage struct {
	To          string
	ToName      string
	Subject     string
	HTMLBody    string
	TextBody    string
	ReplyTo     string
	Attachments []EmailAttachment
}

// EmailAttachment represents a file attachment
type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// EmailSender is the interface for sending emails
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

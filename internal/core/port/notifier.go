package port

import "context"

type Notifier interface {
	SendEmail(ctx context.Context, to string, subject string, body string) error
}

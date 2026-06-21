package notification

import (
	"context"
	"log"

	"github.com/recur-so/recurso/internal/core/port"
)

type ConsoleNotifier struct{}

func NewConsoleNotifier() port.Notifier {
	return &ConsoleNotifier{}
}

func (n *ConsoleNotifier) SendEmail(ctx context.Context, to string, subject string, body string) error {
	log.Printf("================ MOCK EMAIL ================")
	log.Printf("TO: %s", to)
	log.Printf("SUBJECT: %s", subject)
	log.Printf("BODY: %s", body)
	log.Printf("============================================")
	return nil
}

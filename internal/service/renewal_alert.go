package service

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// renewalAlertCooldown bounds a persistent failure to one email an hour per
// process, against a sweep that runs every few minutes.
const renewalAlertCooldown = time.Hour

// renewalAlertSampleErrors is how many errors an alert quotes.
const renewalAlertSampleErrors = 3

// RenewalFailureAlerter emails the operator when scheduled renewals fail.
// Per-subscription failures were only logged, which is how a total renewal
// outage once went unnoticed.
type RenewalFailureAlerter struct {
	sender port.EmailSender
	to     string
	now    func() time.Time

	mu       sync.Mutex
	lastSent time.Time
}

// NewRenewalFailureAlerter sends to the given address through sender.
func NewRenewalFailureAlerter(sender port.EmailSender, to string) *RenewalFailureAlerter {
	return &RenewalFailureAlerter{sender: sender, to: strings.TrimSpace(to), now: time.Now}
}

// RenewalsFailed reports a sweep's failures. failed of claimed subscriptions
// failed, or claimed is 0 when the sweep couldn't claim any at all. A failed
// send is logged and retried on the next failing sweep.
func (a *RenewalFailureAlerter) RenewalsFailed(ctx context.Context, failed, claimed int, errs []error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	if !a.lastSent.IsZero() && now.Sub(a.lastSent) < renewalAlertCooldown {
		return
	}

	subject := fmt.Sprintf("Recurso: %d of %d scheduled renewals failed", failed, claimed)
	summary := fmt.Sprintf("%d of %d subscriptions due for renewal failed in the last sweep.", failed, claimed)
	if claimed == 0 {
		subject = "Recurso: the renewal sweep can't claim due subscriptions"
		summary = "The renewal sweep failed before it could renew anything, so nothing due is being billed."
	}
	var lines strings.Builder
	var items strings.Builder
	for i, err := range errs {
		if i == renewalAlertSampleErrors {
			break
		}
		lines.WriteString("- " + err.Error() + "\n")
		items.WriteString("<li>" + html.EscapeString(err.Error()) + "</li>")
	}
	const followUp = "Failed subscriptions retry when their lease lapses. Check the API logs. You won't get another alert for an hour."

	err := a.sender.Send(ctx, port.EmailMessage{
		To:       a.to,
		Subject:  subject,
		TextBody: summary + "\n\n" + lines.String() + "\n" + followUp,
		HTMLBody: "<p>" + summary + "</p><ul>" + items.String() + "</ul><p>" + followUp + "</p>",
	})
	if err != nil {
		slog.ErrorContext(ctx, "renewal failure alert could not be sent", "error", err)
		return
	}
	a.lastSent = now
}

// Package alerting delivers operational alerts to the operator via an
// outbound webhook. It is deliberately minimal: one POST per alert, a hard
// 10-second timeout, no retries and no queue — alerting must never block or
// loop the caller. Configure with ALERT_WEBHOOK_URL (and, optionally,
// ALERT_WEBHOOK_FORMAT=slack to emit Slack incoming-webhook payloads).
package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Severity classifies how urgent an alert is.
type Severity string

const (
	// SeverityCritical means money movement or the system of record is at
	// risk (e.g. Postgres unreachable). Act immediately.
	SeverityCritical Severity = "critical"
	// SeverityWarning means an optional component is degraded (e.g. Redis
	// or TigerBeetle down) — the system keeps working with reduced guarantees.
	SeverityWarning Severity = "warning"
	// SeverityInfo is for non-actionable notices such as recoveries.
	SeverityInfo Severity = "info"
)

// Payload formats accepted in ALERT_WEBHOOK_FORMAT.
const (
	// FormatJSON posts {"severity","title","body","source","timestamp"}.
	FormatJSON = "json"
	// FormatSlack posts {"text": "..."} — compatible with Slack (and
	// Mattermost/Discord-compatible) incoming webhooks.
	FormatSlack = "slack"
)

// source identifies this system in outbound alert payloads.
const source = "recurso"

// Alerter sends operational alerts. Implementations must be safe for
// concurrent use and must never block beyond a short, bounded timeout.
// A returned error means the alert may not have been delivered; callers
// should log it and move on — never retry in a loop.
type Alerter interface {
	Send(ctx context.Context, severity Severity, title, body string) error
}

// WebhookAlerter POSTs each alert to a configured webhook URL exactly once.
type WebhookAlerter struct {
	url    string
	format string
	client *http.Client
}

// NewWebhookAlerter builds a webhook alerter for the given URL. format is
// FormatJSON or FormatSlack; anything else falls back to FormatJSON.
func NewWebhookAlerter(url, format string) *WebhookAlerter {
	if format != FormatSlack {
		format = FormatJSON
	}
	return &WebhookAlerter{
		url:    url,
		format: format,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewFromEnv builds an Alerter from ALERT_WEBHOOK_URL and
// ALERT_WEBHOOK_FORMAT. When ALERT_WEBHOOK_URL is unset it returns a
// NoopAlerter, so callers can wire alerting unconditionally.
func NewFromEnv() Alerter {
	url := os.Getenv("ALERT_WEBHOOK_URL")
	if url == "" {
		return NoopAlerter{}
	}
	return NewWebhookAlerter(url, os.Getenv("ALERT_WEBHOOK_FORMAT"))
}

// Send delivers one alert with a single POST (no retries). It returns an
// error when the webhook is unreachable or responds with a non-2xx status.
func (a *WebhookAlerter) Send(ctx context.Context, severity Severity, title, body string) error {
	var payload any
	switch a.format {
	case FormatSlack:
		payload = map[string]string{
			"text": fmt.Sprintf("[%s] %s — %s", strings.ToUpper(string(severity)), title, body),
		}
	default:
		payload = map[string]string{
			"severity":  string(severity),
			"title":     title,
			"body":      body,
			"source":    source,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("alert webhook: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("alert webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("alert webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("alert webhook: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// NoopAlerter silently drops alerts. Used when ALERT_WEBHOOK_URL is unset.
type NoopAlerter struct{}

// Send discards the alert and always succeeds.
func (NoopAlerter) Send(context.Context, Severity, string, string) error { return nil }

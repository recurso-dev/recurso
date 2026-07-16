package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/httpsafe"
)

type WebhookWorker struct {
	deliveryRepo port.EventDeliveryRepository
	endpointRepo port.WebhookEndpointRepository
	eventRepo    port.EventRepository
	httpClient   *http.Client
}

func NewWebhookWorker(
	deliveryRepo port.EventDeliveryRepository,
	endpointRepo port.WebhookEndpointRepository,
	eventRepo port.EventRepository,
) *WebhookWorker {
	return &WebhookWorker{
		deliveryRepo: deliveryRepo,
		endpointRepo: endpointRepo,
		eventRepo:    eventRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			// Do not follow redirects — a public URL could 302 to an internal
			// target, bypassing the create-time SSRF check. Treat the 3xx as the
			// final response instead.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
			// Re-validate the resolved IP at connect time so a host that rebinds to
			// a private/loopback/link-local address after creation is still blocked.
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 10 * time.Second,
					Control: httpsafe.DialControl,
				}).DialContext,
			},
		},
	}
}

func (w *WebhookWorker) Start(ctx context.Context) {
	slog.Info("webhook worker started")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("webhook worker stopping")
			return
		case <-ticker.C:
			w.processDeliveries(ctx)
		}
	}
}

func (w *WebhookWorker) processDeliveries(ctx context.Context) {
	deliveries, err := w.deliveryRepo.ListPending(ctx, 10) // Process 10 at a time
	if err != nil {
		slog.Error("failed to fetch pending webhooks", "error", err)
		return
	}

	for _, delivery := range deliveries {
		w.deliver(ctx, delivery)
	}
}

func (w *WebhookWorker) deliver(ctx context.Context, delivery *domain.EventDelivery) {
	// Get Event
	event, err := w.eventRepo.GetByID(ctx, delivery.EventID)
	if err != nil {
		slog.Error("failed to fetch event", "event_id", delivery.EventID, "error", err)
		return // Can't proceed
	}

	// Get Endpoint
	endpoint, err := w.endpointRepo.GetByID(ctx, delivery.WebhookEndpointID)
	if err != nil {
		slog.Error("failed to fetch endpoint", "endpoint_id", delivery.WebhookEndpointID, "error", err)
		return
	}

	// Prepare Request
	payload, _ := json.Marshal(event)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewBuffer(payload))
	if err != nil {
		w.failDelivery(ctx, delivery, 0, err.Error())
		return
	}

	// Sign Request
	signature := computeSignature(endpoint.Secret, payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Recurso-Signature", signature)
	req.Header.Set("X-Recurso-Event-ID", event.ID.String())

	// Send
	resp, err := w.httpClient.Do(req)
	if err != nil {
		// Transport failure: no HTTP status to record.
		w.retryDelivery(ctx, delivery, 0, err.Error())
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body (up to 1KB)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	// Handle Response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		now := time.Now()
		delivery.DeliveredAt = &now
		delivery.StatusCode = resp.StatusCode
		delivery.ResponseBody = string(body)
		if err := w.deliveryRepo.Update(ctx, delivery); err != nil {
			slog.Error("failed to update delivery", "delivery_id", delivery.ID, "error", err)
		}
	} else {
		w.retryDelivery(ctx, delivery, resp.StatusCode, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}
}

func (w *WebhookWorker) retryDelivery(ctx context.Context, delivery *domain.EventDelivery, code int, reason string) {
	delivery.Attempt++
	delivery.StatusCode = code // last response code (0 for transport errors)
	delivery.ResponseBody = reason

	if delivery.Attempt >= 5 {
		// Max attempts exhausted — mark as delivered (failed) so ListPending stops picking it up
		now := time.Now()
		delivery.DeliveredAt = &now
		if err := w.deliveryRepo.Update(ctx, delivery); err != nil {
			slog.Error("failed to update delivery", "delivery_id", delivery.ID, "error", err)
		}
		return
	}

	// Exponential backoff: 2^attempt * 30s, capped at 24h
	backoff := time.Duration(math.Min(
		float64(time.Duration(1<<uint(delivery.Attempt))*30*time.Second),
		float64(24*time.Hour),
	))
	nextRetry := time.Now().Add(backoff)
	delivery.NextRetryAt = &nextRetry

	if err := w.deliveryRepo.Update(ctx, delivery); err != nil {
		slog.Error("failed to update delivery", "delivery_id", delivery.ID, "error", err)
	}
}

func (w *WebhookWorker) failDelivery(ctx context.Context, delivery *domain.EventDelivery, code int, reason string) {
	delivery.StatusCode = code
	delivery.ResponseBody = reason
	if err := w.deliveryRepo.Update(ctx, delivery); err != nil {
		slog.Error("failed to update delivery", "delivery_id", delivery.ID, "error", err)
	}
}

func computeSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
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
		},
	}
}

func (w *WebhookWorker) Start(ctx context.Context) {
	log.Println("Webhook Worker started")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Webhook Worker stopping")
			return
		case <-ticker.C:
			w.processDeliveries(ctx)
		}
	}
}

func (w *WebhookWorker) processDeliveries(ctx context.Context) {
	// 1. Fetch pending deliveries (This requires a method in repo that likely doesn't exist yet in the interface shown in previous turns)
	// We might need to extend the repository interface or implementation.
	// For MVP, since we don't have GetPendingDeliveries in the interface shown previously,
	// we will assume we need to implement polling or rely on a "ListPending" method.
	// However, looking at the previous file view of `webhook_repository.go`, `EventDeliveryRepository` only had:
	// Create, Update, ListByEventID.
	// It is missing `ListPending`. I need to add that first?
	// Or, I can iterate recently created events and check their delivery status? No, that's inefficient.

	// WAIT: I should have caught this in planning.
	// I'll add methods to the repo interface via a separate step or assume I'm updating it here?
	// I'll assume I can update the repository. But for now, I'll implement the logic assuming the method exists
	// and then I will update the repo files.

	// Just kidding, I can't pass compilation if interface doesn't match.
	// Let's implement the worker logic assuming `ListPending` exists, and I will strictly follow up to update the repo.

	deliveries, err := w.deliveryRepo.ListPending(ctx, 10) // Process 10 at a time
	if err != nil {
		log.Printf("Error fetching pending webhooks: %v", err)
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
		log.Printf("Error fetching event %v: %v", delivery.EventID, err)
		return // Can't proceed
	}

	// Get Endpoint
	endpoint, err := w.endpointRepo.GetByID(ctx, delivery.WebhookEndpointID)
	if err != nil {
		log.Printf("Error fetching endpoint %v: %v", delivery.WebhookEndpointID, err)
		return
	}

	// Prepare Request
	payload, _ := json.Marshal(event)
	req, err := http.NewRequest("POST", endpoint.URL, bytes.NewBuffer(payload))
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
		w.retryDelivery(ctx, delivery, err.Error())
		return
	}
	defer resp.Body.Close()

	// Handle Response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		now := time.Now()
		delivery.DeliveredAt = &now
		delivery.StatusCode = resp.StatusCode
		delivery.ResponseBody = "OK" // Simplified
		w.deliveryRepo.Update(ctx, delivery)
	} else {
		w.retryDelivery(ctx, delivery, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}
}

func (w *WebhookWorker) retryDelivery(ctx context.Context, delivery *domain.EventDelivery, reason string) {
	delivery.Attempt++
	delivery.StatusCode = 0
	delivery.ResponseBody = reason

	if delivery.Attempt >= 5 {
		// Consistently failing, stop retrying? Or just schedule for very late?
		// For now, we update it but NextRetryAt logic (which handles the backoff)
		// implies we need a NextRetryAt field in domain.EventDelivery.
		// Checking domain model: `Attempts` exists. `DeliveredAt` exists.
		// `NextRetryAt` was NOT in the viewed domain model.
		// I'll add `NextRetryAt` to domain model too.

		// For now, simple implementation: just update.
	}

	// For "NextRetryAt", we need that field to query effectively.
	// If it's not in DB, we rely on "LastUpdated" + "Attempt" count logic in the SQL query.

	w.deliveryRepo.Update(ctx, delivery)
}

func (w *WebhookWorker) failDelivery(ctx context.Context, delivery *domain.EventDelivery, code int, reason string) {
	delivery.StatusCode = code
	delivery.ResponseBody = reason
	w.deliveryRepo.Update(ctx, delivery)
}

func computeSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

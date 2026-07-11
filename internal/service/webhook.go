package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/url"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// WebhookService handles webhook endpoint management and event publishing
type WebhookService struct {
	endpointRepo port.WebhookEndpointRepository
	eventRepo    port.EventRepository
	deliveryRepo port.EventDeliveryRepository
}

func NewWebhookService(
	endpointRepo port.WebhookEndpointRepository,
	eventRepo port.EventRepository,
	deliveryRepo port.EventDeliveryRepository,
) *WebhookService {
	return &WebhookService{
		endpointRepo: endpointRepo,
		eventRepo:    eventRepo,
		deliveryRepo: deliveryRepo,
	}
}

// CreateEndpointInput represents the input for creating a webhook endpoint
type CreateEndpointInput struct {
	TenantID uuid.UUID
	URL      string
	Events   []string
}

// CreateEndpoint creates a new webhook endpoint for a tenant
func (s *WebhookService) CreateEndpoint(ctx context.Context, input CreateEndpointInput) (*domain.WebhookEndpoint, error) {
	// Validate URL
	if _, err := url.ParseRequestURI(input.URL); err != nil {
		return nil, ErrInvalidWebhookURL
	}

	// Validate events
	if len(input.Events) == 0 {
		return nil, ErrNoEventsSubscribed
	}

	// Generate signing secret
	secret, err := generateSecret()
	if err != nil {
		return nil, err
	}

	endpoint := &domain.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: input.TenantID,
		URL:      input.URL,
		Secret:   secret,
		Events:   input.Events,
		Status:   "active",
	}

	if err := s.endpointRepo.Create(ctx, endpoint); err != nil {
		return nil, err
	}

	return endpoint, nil
}

// ListEndpoints returns all webhook endpoints for a tenant
func (s *WebhookService) ListEndpoints(ctx context.Context, tenantID uuid.UUID) ([]*domain.WebhookEndpoint, error) {
	return s.endpointRepo.ListByTenantID(ctx, tenantID)
}

// GetEndpoint returns a single webhook endpoint by ID
func (s *WebhookService) GetEndpoint(ctx context.Context, id uuid.UUID) (*domain.WebhookEndpoint, error) {
	return s.endpointRepo.GetByID(ctx, id)
}

// DeleteEndpoint removes a webhook endpoint
func (s *WebhookService) DeleteEndpoint(ctx context.Context, id uuid.UUID) error {
	return s.endpointRepo.Delete(ctx, id)
}

// UpdateEndpointStatus updates the status (active/inactive) of a webhook endpoint
func (s *WebhookService) UpdateEndpointStatus(ctx context.Context, id uuid.UUID, status string) error {
	endpoint, err := s.endpointRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	endpoint.Status = status
	return s.endpointRepo.Update(ctx, endpoint)
}

// PublishEventInput represents input for publishing an event
type PublishEventInput struct {
	TenantID   uuid.UUID
	Type       string
	ObjectType string
	ObjectID   uuid.UUID
	Data       map[string]interface{}
}

// PublishEvent creates an event and queues it for delivery to subscribed endpoints
func (s *WebhookService) PublishEvent(ctx context.Context, input PublishEventInput) (*domain.Event, error) {
	event := &domain.Event{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		Type:       input.Type,
		ObjectType: input.ObjectType,
		ObjectID:   input.ObjectID,
		Data:       input.Data,
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	// Create EventDelivery records for all matching endpoints (best-effort)
	endpoints, err := s.endpointRepo.GetByTenantAndEventType(ctx, input.TenantID, input.Type)
	if err != nil {
		slog.Error("failed to fetch webhook endpoints", "error", err, "tenant_id", input.TenantID, "event_type", input.Type)
	}
	for _, endpoint := range endpoints {
		delivery := &domain.EventDelivery{
			ID:                uuid.New(),
			EventID:           event.ID,
			WebhookEndpointID: endpoint.ID,
			Attempt:           0,
			NextRetryAt:       nil, // immediate first attempt
		}
		if err := s.deliveryRepo.Create(ctx, delivery); err != nil {
			slog.Error("failed to create webhook delivery", "error", err, "event_id", event.ID, "endpoint_id", endpoint.ID)
		}
	}

	return event, nil
}

// ListEvents returns events for a tenant with pagination
func (s *WebhookService) ListEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Event, error) {
	return s.eventRepo.ListByTenantID(ctx, tenantID, limit, offset)
}

// GetEvent returns a single event by ID
func (s *WebhookService) GetEvent(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	return s.eventRepo.GetByID(ctx, id)
}

// EventDeliveryDetail pairs a delivery with the endpoint URL it targets.
type EventDeliveryDetail struct {
	Delivery    *domain.EventDelivery
	EndpointURL string
}

// getTenantEvent loads an event and enforces tenant ownership.
// A missing event and a cross-tenant event are indistinguishable to callers.
func (s *WebhookService) getTenantEvent(ctx context.Context, tenantID, eventID uuid.UUID) (*domain.Event, error) {
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	if event == nil || event.TenantID != tenantID {
		return nil, ErrEventNotFound
	}
	return event, nil
}

// getTenantEndpoint loads a webhook endpoint and enforces tenant ownership.
func (s *WebhookService) getTenantEndpoint(ctx context.Context, tenantID, endpointID uuid.UUID) (*domain.WebhookEndpoint, error) {
	endpoint, err := s.endpointRepo.GetByID(ctx, endpointID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEndpointNotFound
		}
		return nil, err
	}
	if endpoint == nil || endpoint.TenantID != tenantID {
		return nil, ErrEndpointNotFound
	}
	return endpoint, nil
}

// ListEventDeliveries returns delivery attempts for a tenant's event across
// all endpoints, enriched with each endpoint's URL.
func (s *WebhookService) ListEventDeliveries(ctx context.Context, tenantID, eventID uuid.UUID) ([]*EventDeliveryDetail, error) {
	if _, err := s.getTenantEvent(ctx, tenantID, eventID); err != nil {
		return nil, err
	}

	deliveries, err := s.deliveryRepo.ListByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	urls := make(map[uuid.UUID]string)
	details := make([]*EventDeliveryDetail, 0, len(deliveries))
	for _, d := range deliveries {
		endpointURL, ok := urls[d.WebhookEndpointID]
		if !ok {
			// Best-effort URL lookup; a delivery row is still useful without it.
			if endpoint, epErr := s.endpointRepo.GetByID(ctx, d.WebhookEndpointID); epErr == nil && endpoint != nil {
				endpointURL = endpoint.URL
			}
			urls[d.WebhookEndpointID] = endpointURL
		}
		details = append(details, &EventDeliveryDetail{Delivery: d, EndpointURL: endpointURL})
	}
	return details, nil
}

// ListEndpointDeliveries returns recent deliveries for one of the tenant's
// webhook endpoints, optionally filtered by derived status.
func (s *WebhookService) ListEndpointDeliveries(ctx context.Context, tenantID, endpointID uuid.UUID, status string, limit, offset int) ([]*domain.EventDelivery, *domain.WebhookEndpoint, error) {
	switch status {
	case "", domain.DeliveryStatusPending, domain.DeliveryStatusSucceeded, domain.DeliveryStatusFailed:
	default:
		return nil, nil, ErrInvalidDeliveryStatus
	}

	endpoint, err := s.getTenantEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		return nil, nil, err
	}

	deliveries, err := s.deliveryRepo.ListByEndpointID(ctx, endpointID, status, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	return deliveries, endpoint, nil
}

// RedeliverEvent re-enqueues delivery of an event to every active endpoint
// currently subscribed to its type. Existing delivery rows are reset so the
// worker's ListPending query (delivered_at IS NULL AND next_retry_at due)
// picks them up again; endpoints without a row get a fresh one. Calling it
// repeatedly resets the same rows rather than duplicating them. Returns the
// number of deliveries queued.
func (s *WebhookService) RedeliverEvent(ctx context.Context, tenantID, eventID uuid.UUID) (int, error) {
	event, err := s.getTenantEvent(ctx, tenantID, eventID)
	if err != nil {
		return 0, err
	}

	existing, err := s.deliveryRepo.ListByEventID(ctx, eventID)
	if err != nil {
		return 0, err
	}
	byEndpoint := make(map[uuid.UUID]*domain.EventDelivery, len(existing))
	for _, d := range existing {
		if _, ok := byEndpoint[d.WebhookEndpointID]; !ok {
			// ListByEventID is newest-first; keep the most recent row per endpoint.
			byEndpoint[d.WebhookEndpointID] = d
		}
	}

	endpoints, err := s.endpointRepo.GetByTenantAndEventType(ctx, tenantID, event.Type)
	if err != nil {
		return 0, err
	}

	queued := 0
	for _, endpoint := range endpoints {
		if delivery, ok := byEndpoint[endpoint.ID]; ok {
			delivery.StatusCode = 0
			delivery.ResponseBody = ""
			delivery.Attempt = 0
			delivery.NextRetryAt = nil // immediate retry
			delivery.DeliveredAt = nil
			if err := s.deliveryRepo.Update(ctx, delivery); err != nil {
				return queued, err
			}
		} else {
			fresh := &domain.EventDelivery{
				ID:                uuid.New(),
				EventID:           event.ID,
				WebhookEndpointID: endpoint.ID,
				Attempt:           0,
				NextRetryAt:       nil, // immediate first attempt
			}
			if err := s.deliveryRepo.Create(ctx, fresh); err != nil {
				return queued, err
			}
		}
		queued++
	}
	return queued, nil
}

// Helper to generate a random signing secret
func generateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(bytes), nil
}

// Errors
type WebhookError string

func (e WebhookError) Error() string {
	return string(e)
}

const (
	ErrInvalidWebhookURL     = WebhookError("invalid webhook URL")
	ErrNoEventsSubscribed    = WebhookError("at least one event type must be subscribed")
	ErrEventNotFound         = WebhookError("event not found")
	ErrEndpointNotFound      = WebhookError("webhook endpoint not found")
	ErrInvalidDeliveryStatus = WebhookError("status must be one of: pending, succeeded, failed")
)

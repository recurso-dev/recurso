package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/url"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
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
	ErrInvalidWebhookURL  = WebhookError("invalid webhook URL")
	ErrNoEventsSubscribed = WebhookError("at least one event type must be subscribed")
)

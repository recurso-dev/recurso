package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// WebhookEndpointRepository defines operations for managing webhook endpoints
type WebhookEndpointRepository interface {
	Create(ctx context.Context, endpoint *domain.WebhookEndpoint) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookEndpoint, error)
	ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*domain.WebhookEndpoint, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Update(ctx context.Context, endpoint *domain.WebhookEndpoint) error
	// GetByTenantAndEventType returns active endpoints subscribed to a specific event type
	GetByTenantAndEventType(ctx context.Context, tenantID uuid.UUID, eventType string) ([]*domain.WebhookEndpoint, error)
}

// EventRepository defines operations for managing events
type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error)
	ListByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Event, error)
}

// EventDeliveryRepository defines operations for tracking event deliveries
type EventDeliveryRepository interface {
	Create(ctx context.Context, delivery *domain.EventDelivery) error
	Update(ctx context.Context, delivery *domain.EventDelivery) error
	ListByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.EventDelivery, error)
	// ListByEndpointID returns recent deliveries for a webhook endpoint,
	// newest first. status filters by derived delivery status
	// (pending/succeeded/failed); empty string returns all.
	ListByEndpointID(ctx context.Context, endpointID uuid.UUID, status string, limit, offset int) ([]*domain.EventDelivery, error)
	ListPending(ctx context.Context, limit int) ([]*domain.EventDelivery, error)
	// ClaimPending atomically leases up to `limit` due deliveries for the calling
	// worker instance, advancing next_retry_at by `lease` so a second instance
	// can't pick the same rows and double-deliver them (ADR-003).
	ClaimPending(ctx context.Context, lease time.Duration, limit int) ([]*domain.EventDelivery, error)
}

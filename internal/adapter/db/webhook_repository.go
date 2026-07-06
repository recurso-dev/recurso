package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// WebhookEndpointRepository implements port.WebhookEndpointRepository
type WebhookEndpointRepository struct {
	db *sql.DB
}

func NewWebhookEndpointRepository(db *sql.DB) *WebhookEndpointRepository {
	return &WebhookEndpointRepository{db: db}
}

func (r *WebhookEndpointRepository) Create(ctx context.Context, endpoint *domain.WebhookEndpoint) error {
	query := `
		INSERT INTO webhook_endpoints (id, tenant_id, url, secret, events, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		endpoint.ID,
		endpoint.TenantID,
		endpoint.URL,
		endpoint.Secret,
		pq.Array(endpoint.Events),
		endpoint.Status,
	)
	return err
}

func (r *WebhookEndpointRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookEndpoint, error) {
	query := `
		SELECT id, tenant_id, url, secret, events, status, created_at, updated_at
		FROM webhook_endpoints WHERE id = $1
	`
	var endpoint domain.WebhookEndpoint
	var events pq.StringArray
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&endpoint.ID,
		&endpoint.TenantID,
		&endpoint.URL,
		&endpoint.Secret,
		&events,
		&endpoint.Status,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	endpoint.Events = []string(events)
	return &endpoint, nil
}

func (r *WebhookEndpointRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*domain.WebhookEndpoint, error) {
	query := `
		SELECT id, tenant_id, url, secret, events, status, created_at, updated_at
		FROM webhook_endpoints WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var endpoints []*domain.WebhookEndpoint
	for rows.Next() {
		var endpoint domain.WebhookEndpoint
		var events pq.StringArray
		if err := rows.Scan(
			&endpoint.ID,
			&endpoint.TenantID,
			&endpoint.URL,
			&endpoint.Secret,
			&events,
			&endpoint.Status,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		); err != nil {
			return nil, err
		}
		endpoint.Events = []string(events)
		endpoints = append(endpoints, &endpoint)
	}
	return endpoints, nil
}

func (r *WebhookEndpointRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webhook_endpoints WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *WebhookEndpointRepository) Update(ctx context.Context, endpoint *domain.WebhookEndpoint) error {
	query := `
		UPDATE webhook_endpoints 
		SET url = $2, events = $3, status = $4, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		endpoint.ID,
		endpoint.URL,
		pq.Array(endpoint.Events),
		endpoint.Status,
	)
	return err
}

func (r *WebhookEndpointRepository) GetByTenantAndEventType(ctx context.Context, tenantID uuid.UUID, eventType string) ([]*domain.WebhookEndpoint, error) {
	query := `
		SELECT id, tenant_id, url, secret, events, status, created_at, updated_at
		FROM webhook_endpoints 
		WHERE tenant_id = $1 AND status = 'active' AND $2 = ANY(events)
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, eventType)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var endpoints []*domain.WebhookEndpoint
	for rows.Next() {
		var endpoint domain.WebhookEndpoint
		var events pq.StringArray
		if err := rows.Scan(
			&endpoint.ID,
			&endpoint.TenantID,
			&endpoint.URL,
			&endpoint.Secret,
			&events,
			&endpoint.Status,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		); err != nil {
			return nil, err
		}
		endpoint.Events = []string(events)
		endpoints = append(endpoints, &endpoint)
	}
	return endpoints, nil
}

// EventRepository implements port.EventRepository
type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *domain.Event) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO events (id, tenant_id, type, object_type, object_id, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		event.TenantID,
		event.Type,
		event.ObjectType,
		event.ObjectID,
		dataJSON,
	)
	return err
}

func (r *EventRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	query := `
		SELECT id, tenant_id, type, object_type, object_id, data, created_at
		FROM events WHERE id = $1
	`
	var event domain.Event
	var dataJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID,
		&event.TenantID,
		&event.Type,
		&event.ObjectType,
		&event.ObjectID,
		&dataJSON,
		&event.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *EventRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, tenant_id, type, object_type, object_id, data, created_at
		FROM events WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*domain.Event
	for rows.Next() {
		var event domain.Event
		var dataJSON []byte
		if err := rows.Scan(
			&event.ID,
			&event.TenantID,
			&event.Type,
			&event.ObjectType,
			&event.ObjectID,
			&dataJSON,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
			return nil, err
		}
		events = append(events, &event)
	}
	return events, nil
}

// EventDeliveryRepository implements port.EventDeliveryRepository
type EventDeliveryRepository struct {
	db *sql.DB
}

func NewEventDeliveryRepository(db *sql.DB) *EventDeliveryRepository {
	return &EventDeliveryRepository{db: db}
}

func (r *EventDeliveryRepository) Create(ctx context.Context, delivery *domain.EventDelivery) error {
	query := `
		INSERT INTO event_deliveries (id, event_id, webhook_endpoint_id, status_code, response_body, attempt, next_retry_at, delivered_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		delivery.ID,
		delivery.EventID,
		delivery.WebhookEndpointID,
		delivery.StatusCode,
		delivery.ResponseBody,
		delivery.Attempt,
		delivery.NextRetryAt,
		delivery.DeliveredAt,
	)
	return err
}

func (r *EventDeliveryRepository) Update(ctx context.Context, delivery *domain.EventDelivery) error {
	query := `
		UPDATE event_deliveries 
		SET status_code = $2, response_body = $3, attempt = $4, next_retry_at = $5, delivered_at = $6
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		delivery.ID,
		delivery.StatusCode,
		delivery.ResponseBody,
		delivery.Attempt,
		delivery.NextRetryAt,
		delivery.DeliveredAt,
	)
	return err
}

func (r *EventDeliveryRepository) ListByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.EventDelivery, error) {
	query := `
		SELECT id, event_id, webhook_endpoint_id, status_code, response_body, attempt, next_retry_at, delivered_at, created_at
		FROM event_deliveries WHERE event_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var deliveries []*domain.EventDelivery
	for rows.Next() {
		var delivery domain.EventDelivery
		if err := rows.Scan(
			&delivery.ID,
			&delivery.EventID,
			&delivery.WebhookEndpointID,
			&delivery.StatusCode,
			&delivery.ResponseBody,
			&delivery.Attempt,
			&delivery.NextRetryAt,
			&delivery.DeliveredAt,
			&delivery.CreatedAt,
		); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &delivery)
	}
	return deliveries, nil
}

func (r *EventDeliveryRepository) ListByEndpointID(ctx context.Context, endpointID uuid.UUID, status string, limit, offset int) ([]*domain.EventDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, event_id, webhook_endpoint_id, status_code, response_body, attempt, next_retry_at, delivered_at, created_at
		FROM event_deliveries WHERE webhook_endpoint_id = $1
	`
	switch status {
	case domain.DeliveryStatusPending:
		query += ` AND delivered_at IS NULL`
	case domain.DeliveryStatusSucceeded:
		query += ` AND delivered_at IS NOT NULL AND status_code BETWEEN 200 AND 299`
	case domain.DeliveryStatusFailed:
		query += ` AND delivered_at IS NOT NULL AND (status_code < 200 OR status_code > 299)`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, endpointID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var deliveries []*domain.EventDelivery
	for rows.Next() {
		var delivery domain.EventDelivery
		if err := rows.Scan(
			&delivery.ID,
			&delivery.EventID,
			&delivery.WebhookEndpointID,
			&delivery.StatusCode,
			&delivery.ResponseBody,
			&delivery.Attempt,
			&delivery.NextRetryAt,
			&delivery.DeliveredAt,
			&delivery.CreatedAt,
		); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &delivery)
	}
	return deliveries, nil
}

func (r *EventDeliveryRepository) ListPending(ctx context.Context, limit int) ([]*domain.EventDelivery, error) {
	query := `
		SELECT id, event_id, webhook_endpoint_id, status_code, response_body, attempt, next_retry_at, delivered_at, created_at
		FROM event_deliveries 
		WHERE delivered_at IS NULL AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var deliveries []*domain.EventDelivery
	for rows.Next() {
		var delivery domain.EventDelivery
		if err := rows.Scan(
			&delivery.ID,
			&delivery.EventID,
			&delivery.WebhookEndpointID,
			&delivery.StatusCode,
			&delivery.ResponseBody,
			&delivery.Attempt,
			&delivery.NextRetryAt,
			&delivery.DeliveredAt,
			&delivery.CreatedAt,
		); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &delivery)
	}
	return deliveries, nil
}

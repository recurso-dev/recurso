package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// --- Mock WebhookEndpointRepository ---
type mockEndpointRepo struct {
	endpoints map[uuid.UUID]*domain.WebhookEndpoint
}

func newMockEndpointRepo() *mockEndpointRepo {
	return &mockEndpointRepo{endpoints: make(map[uuid.UUID]*domain.WebhookEndpoint)}
}

func (m *mockEndpointRepo) Create(ctx context.Context, e *domain.WebhookEndpoint) error {
	m.endpoints[e.ID] = e
	return nil
}

func (m *mockEndpointRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookEndpoint, error) {
	if e, ok := m.endpoints[id]; ok {
		return e, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockEndpointRepo) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*domain.WebhookEndpoint, error) {
	var out []*domain.WebhookEndpoint
	for _, e := range m.endpoints {
		if e.TenantID == tenantID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *mockEndpointRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if e, ok := m.endpoints[id]; ok && e.TenantID == tenantID {
		delete(m.endpoints, id)
	}
	return nil
}

func (m *mockEndpointRepo) Update(ctx context.Context, e *domain.WebhookEndpoint) error {
	m.endpoints[e.ID] = e
	return nil
}

func (m *mockEndpointRepo) GetByTenantAndEventType(ctx context.Context, tenantID uuid.UUID, eventType string) ([]*domain.WebhookEndpoint, error) {
	var out []*domain.WebhookEndpoint
	for _, e := range m.endpoints {
		if e.TenantID != tenantID || e.Status != "active" {
			continue
		}
		for _, evt := range e.Events {
			if evt == eventType {
				out = append(out, e)
				break
			}
		}
	}
	return out, nil
}

// --- Mock EventRepository ---
type mockEventRepo struct {
	events map[uuid.UUID]*domain.Event
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{events: make(map[uuid.UUID]*domain.Event)}
}

func (m *mockEventRepo) Create(ctx context.Context, e *domain.Event) error {
	m.events[e.ID] = e
	return nil
}

func (m *mockEventRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	if e, ok := m.events[id]; ok {
		return e, nil
	}
	return nil, sql.ErrNoRows
}

func (m *mockEventRepo) ListByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Event, error) {
	var out []*domain.Event
	for _, e := range m.events {
		if e.TenantID == tenantID {
			out = append(out, e)
		}
	}
	return out, nil
}

// --- Mock EventDeliveryRepository ---
type mockDeliveryRepo struct {
	deliveries map[uuid.UUID]*domain.EventDelivery
	created    []*domain.EventDelivery
	updated    []*domain.EventDelivery
	// lastListStatus records the status filter passed to ListByEndpointID.
	lastListStatus string
}

func newMockDeliveryRepo() *mockDeliveryRepo {
	return &mockDeliveryRepo{deliveries: make(map[uuid.UUID]*domain.EventDelivery)}
}

func (m *mockDeliveryRepo) Create(ctx context.Context, d *domain.EventDelivery) error {
	m.deliveries[d.ID] = d
	m.created = append(m.created, d)
	return nil
}

func (m *mockDeliveryRepo) Update(ctx context.Context, d *domain.EventDelivery) error {
	m.deliveries[d.ID] = d
	m.updated = append(m.updated, d)
	return nil
}

func (m *mockDeliveryRepo) ListByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.EventDelivery, error) {
	var out []*domain.EventDelivery
	for _, d := range m.deliveries {
		if d.EventID == eventID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (m *mockDeliveryRepo) ListByEndpointID(ctx context.Context, endpointID uuid.UUID, status string, limit, offset int) ([]*domain.EventDelivery, error) {
	m.lastListStatus = status
	var out []*domain.EventDelivery
	for _, d := range m.deliveries {
		if d.WebhookEndpointID != endpointID {
			continue
		}
		if status != "" && d.DeliveryStatus() != status {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (m *mockDeliveryRepo) ListPending(ctx context.Context, limit int) ([]*domain.EventDelivery, error) {
	var out []*domain.EventDelivery
	for _, d := range m.deliveries {
		if d.DeliveredAt == nil {
			out = append(out, d)
		}
	}
	return out, nil
}

// --- Test fixtures ---

type webhookFixture struct {
	svc          *WebhookService
	endpointRepo *mockEndpointRepo
	eventRepo    *mockEventRepo
	deliveryRepo *mockDeliveryRepo
	tenantID     uuid.UUID
	endpoint     *domain.WebhookEndpoint
	event        *domain.Event
}

func newWebhookFixture(t *testing.T) *webhookFixture {
	t.Helper()
	endpointRepo := newMockEndpointRepo()
	eventRepo := newMockEventRepo()
	deliveryRepo := newMockDeliveryRepo()

	tenantID := uuid.New()
	endpoint := &domain.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: tenantID,
		URL:      "https://example.com/hooks",
		Events:   []string{domain.EventTypeInvoicePaid},
		Status:   "active",
	}
	endpointRepo.endpoints[endpoint.ID] = endpoint

	event := &domain.Event{
		ID:       uuid.New(),
		TenantID: tenantID,
		Type:     domain.EventTypeInvoicePaid,
	}
	eventRepo.events[event.ID] = event

	return &webhookFixture{
		svc:          NewWebhookService(endpointRepo, eventRepo, deliveryRepo),
		endpointRepo: endpointRepo,
		eventRepo:    eventRepo,
		deliveryRepo: deliveryRepo,
		tenantID:     tenantID,
		endpoint:     endpoint,
		event:        event,
	}
}

func (f *webhookFixture) addDelivery(d *domain.EventDelivery) *domain.EventDelivery {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	f.deliveryRepo.deliveries[d.ID] = d
	return d
}

// --- ListEventDeliveries ---

func TestListEventDeliveries_ReturnsDeliveriesWithEndpointURL(t *testing.T) {
	f := newWebhookFixture(t)
	f.addDelivery(&domain.EventDelivery{
		EventID:           f.event.ID,
		WebhookEndpointID: f.endpoint.ID,
		StatusCode:        200,
		Attempt:           1,
	})

	details, err := f.svc.ListEventDeliveries(context.Background(), f.tenantID, f.event.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(details) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(details))
	}
	if details[0].EndpointURL != f.endpoint.URL {
		t.Errorf("expected endpoint URL %q, got %q", f.endpoint.URL, details[0].EndpointURL)
	}
	if details[0].Delivery.EventID != f.event.ID {
		t.Errorf("expected event ID %v, got %v", f.event.ID, details[0].Delivery.EventID)
	}
}

func TestListEventDeliveries_EventNotFound(t *testing.T) {
	f := newWebhookFixture(t)

	_, err := f.svc.ListEventDeliveries(context.Background(), f.tenantID, uuid.New())
	if err != ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}
}

func TestListEventDeliveries_TenantIsolation(t *testing.T) {
	f := newWebhookFixture(t)
	otherTenant := uuid.New()

	_, err := f.svc.ListEventDeliveries(context.Background(), otherTenant, f.event.ID)
	if err != ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound for cross-tenant access, got %v", err)
	}
}

// --- ListEndpointDeliveries ---

func TestListEndpointDeliveries_FiltersByStatus(t *testing.T) {
	f := newWebhookFixture(t)
	now := time.Now()
	// Succeeded delivery
	f.addDelivery(&domain.EventDelivery{
		EventID:           f.event.ID,
		WebhookEndpointID: f.endpoint.ID,
		StatusCode:        200,
		Attempt:           1,
		DeliveredAt:       &now,
	})
	// Pending delivery
	f.addDelivery(&domain.EventDelivery{
		EventID:           f.event.ID,
		WebhookEndpointID: f.endpoint.ID,
		Attempt:           2,
	})

	deliveries, endpoint, err := f.svc.ListEndpointDeliveries(context.Background(), f.tenantID, f.endpoint.ID, domain.DeliveryStatusPending, 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint == nil || endpoint.ID != f.endpoint.ID {
		t.Fatalf("expected endpoint %v to be returned", f.endpoint.ID)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 pending delivery, got %d", len(deliveries))
	}
	if deliveries[0].DeliveryStatus() != domain.DeliveryStatusPending {
		t.Errorf("expected pending delivery, got %q", deliveries[0].DeliveryStatus())
	}
	if f.deliveryRepo.lastListStatus != domain.DeliveryStatusPending {
		t.Errorf("expected status filter to reach repo, got %q", f.deliveryRepo.lastListStatus)
	}
}

func TestListEndpointDeliveries_InvalidStatus(t *testing.T) {
	f := newWebhookFixture(t)

	_, _, err := f.svc.ListEndpointDeliveries(context.Background(), f.tenantID, f.endpoint.ID, "bogus", 50, 0)
	if err != ErrInvalidDeliveryStatus {
		t.Fatalf("expected ErrInvalidDeliveryStatus, got %v", err)
	}
}

func TestListEndpointDeliveries_TenantIsolation(t *testing.T) {
	f := newWebhookFixture(t)
	otherTenant := uuid.New()

	_, _, err := f.svc.ListEndpointDeliveries(context.Background(), otherTenant, f.endpoint.ID, "", 50, 0)
	if err != ErrEndpointNotFound {
		t.Fatalf("expected ErrEndpointNotFound for cross-tenant access, got %v", err)
	}
}

func TestListEndpointDeliveries_EndpointNotFound(t *testing.T) {
	f := newWebhookFixture(t)

	_, _, err := f.svc.ListEndpointDeliveries(context.Background(), f.tenantID, uuid.New(), "", 50, 0)
	if err != ErrEndpointNotFound {
		t.Fatalf("expected ErrEndpointNotFound, got %v", err)
	}
}

// --- RedeliverEvent ---

func TestRedeliverEvent_ResetsExistingDelivery(t *testing.T) {
	f := newWebhookFixture(t)
	now := time.Now()
	retry := now.Add(time.Hour)
	delivery := f.addDelivery(&domain.EventDelivery{
		EventID:           f.event.ID,
		WebhookEndpointID: f.endpoint.ID,
		StatusCode:        500,
		ResponseBody:      "HTTP 500: boom",
		Attempt:           5,
		NextRetryAt:       &retry,
		DeliveredAt:       &now, // terminal failure
	})

	queued, err := f.svc.RedeliverEvent(context.Background(), f.tenantID, f.event.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 1 {
		t.Fatalf("expected 1 delivery queued, got %d", queued)
	}
	if len(f.deliveryRepo.updated) != 1 {
		t.Fatalf("expected 1 update, got %d", len(f.deliveryRepo.updated))
	}
	if len(f.deliveryRepo.created) != 0 {
		t.Fatalf("expected no new rows, got %d", len(f.deliveryRepo.created))
	}
	if delivery.DeliveredAt != nil || delivery.NextRetryAt != nil {
		t.Error("expected delivered_at and next_retry_at to be reset to nil")
	}
	if delivery.Attempt != 0 || delivery.StatusCode != 0 || delivery.ResponseBody != "" {
		t.Errorf("expected attempt/status/response reset, got attempt=%d code=%d body=%q",
			delivery.Attempt, delivery.StatusCode, delivery.ResponseBody)
	}
	if delivery.DeliveryStatus() != domain.DeliveryStatusPending {
		t.Errorf("expected reset delivery to be pending (worker will pick it up), got %q", delivery.DeliveryStatus())
	}
}

func TestRedeliverEvent_CreatesRowForEndpointWithoutOne(t *testing.T) {
	f := newWebhookFixture(t)
	// No existing delivery rows for the event.

	queued, err := f.svc.RedeliverEvent(context.Background(), f.tenantID, f.event.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 1 {
		t.Fatalf("expected 1 delivery queued, got %d", queued)
	}
	if len(f.deliveryRepo.created) != 1 {
		t.Fatalf("expected 1 created delivery, got %d", len(f.deliveryRepo.created))
	}
	fresh := f.deliveryRepo.created[0]
	if fresh.EventID != f.event.ID || fresh.WebhookEndpointID != f.endpoint.ID {
		t.Errorf("fresh delivery targets wrong event/endpoint: %+v", fresh)
	}
	if fresh.DeliveredAt != nil || fresh.NextRetryAt != nil || fresh.Attempt != 0 {
		t.Errorf("fresh delivery should be immediately pending: %+v", fresh)
	}
}

func TestRedeliverEvent_IsIdempotentish(t *testing.T) {
	f := newWebhookFixture(t)

	if _, err := f.svc.RedeliverEvent(context.Background(), f.tenantID, f.event.ID); err != nil {
		t.Fatalf("first redeliver failed: %v", err)
	}
	if _, err := f.svc.RedeliverEvent(context.Background(), f.tenantID, f.event.ID); err != nil {
		t.Fatalf("second redeliver failed: %v", err)
	}

	// Second call must reset the existing row, not spawn a duplicate.
	if len(f.deliveryRepo.created) != 1 {
		t.Fatalf("expected exactly 1 delivery row after two redeliveries, got %d", len(f.deliveryRepo.created))
	}
	if len(f.deliveryRepo.updated) != 1 {
		t.Fatalf("expected second redelivery to update the existing row, got %d updates", len(f.deliveryRepo.updated))
	}
}

func TestRedeliverEvent_SkipsUnsubscribedEndpoints(t *testing.T) {
	f := newWebhookFixture(t)
	// Inactive endpoint with an old delivery row must not be re-queued.
	inactive := &domain.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: f.tenantID,
		URL:      "https://example.com/old",
		Events:   []string{domain.EventTypeInvoicePaid},
		Status:   "inactive",
	}
	f.endpointRepo.endpoints[inactive.ID] = inactive
	now := time.Now()
	stale := f.addDelivery(&domain.EventDelivery{
		EventID:           f.event.ID,
		WebhookEndpointID: inactive.ID,
		StatusCode:        200,
		Attempt:           1,
		DeliveredAt:       &now,
	})

	queued, err := f.svc.RedeliverEvent(context.Background(), f.tenantID, f.event.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 1 {
		t.Fatalf("expected only the active endpoint to be queued, got %d", queued)
	}
	if stale.DeliveredAt == nil {
		t.Error("delivery for inactive endpoint must not be reset")
	}
}

func TestRedeliverEvent_TenantIsolation(t *testing.T) {
	f := newWebhookFixture(t)
	otherTenant := uuid.New()

	if _, err := f.svc.RedeliverEvent(context.Background(), otherTenant, f.event.ID); err != ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound for cross-tenant redelivery, got %v", err)
	}
	if len(f.deliveryRepo.created) != 0 || len(f.deliveryRepo.updated) != 0 {
		t.Error("cross-tenant redelivery must not touch delivery rows")
	}
}

// --- DeleteEndpoint ---

func TestDeleteEndpoint_OwnerSucceeds(t *testing.T) {
	f := newWebhookFixture(t)

	if err := f.svc.DeleteEndpoint(context.Background(), f.tenantID, f.endpoint.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := f.endpointRepo.endpoints[f.endpoint.ID]; ok {
		t.Error("expected endpoint to be deleted")
	}
}

func TestDeleteEndpoint_TenantIsolation(t *testing.T) {
	f := newWebhookFixture(t)
	otherTenant := uuid.New()

	err := f.svc.DeleteEndpoint(context.Background(), otherTenant, f.endpoint.ID)
	if err != ErrEndpointNotFound {
		t.Fatalf("expected ErrEndpointNotFound for cross-tenant delete, got %v", err)
	}
	// The endpoint must survive a cross-tenant delete attempt (ENG-160 IDOR).
	if _, ok := f.endpointRepo.endpoints[f.endpoint.ID]; !ok {
		t.Error("cross-tenant delete must not remove the endpoint")
	}
}

func TestDeleteEndpoint_NotFound(t *testing.T) {
	f := newWebhookFixture(t)

	err := f.svc.DeleteEndpoint(context.Background(), f.tenantID, uuid.New())
	if err != ErrEndpointNotFound {
		t.Fatalf("expected ErrEndpointNotFound, got %v", err)
	}
}

// --- PublishEvent sanity (delivery fan-out feeds the tracking endpoints) ---

func TestPublishEvent_CreatesDeliveryRowsForSubscribedEndpoints(t *testing.T) {
	f := newWebhookFixture(t)

	event, err := f.svc.PublishEvent(context.Background(), PublishEventInput{
		TenantID:   f.tenantID,
		Type:       domain.EventTypeInvoicePaid,
		ObjectType: "invoice",
		ObjectID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.deliveryRepo.created) != 1 {
		t.Fatalf("expected 1 delivery row, got %d", len(f.deliveryRepo.created))
	}
	if f.deliveryRepo.created[0].EventID != event.ID {
		t.Errorf("delivery row references wrong event")
	}
}

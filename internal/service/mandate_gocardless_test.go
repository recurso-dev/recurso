package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

var errMandateNotFound = errors.New("mandate not found")

// fakeMandateRepo is an in-memory MandateRepository covering the lookup and
// update paths the GoCardless webhook handlers use.
type fakeMandateRepo struct {
	byToken map[string]*domain.Mandate
}

func (f *fakeMandateRepo) Create(_ context.Context, m *domain.Mandate) error {
	f.byToken[m.RazorpayTokenID] = m
	return nil
}
func (f *fakeMandateRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Mandate, error) {
	return nil, errMandateNotFound
}
func (f *fakeMandateRepo) List(context.Context, uuid.UUID, int, int) ([]*domain.Mandate, error) {
	return nil, nil
}
func (f *fakeMandateRepo) Update(_ context.Context, m *domain.Mandate) error {
	// Reindex: fulfilment rewrites the token column from BRQ... to MD...
	for tok, existing := range f.byToken {
		if existing.ID == m.ID && tok != m.RazorpayTokenID {
			delete(f.byToken, tok)
		}
	}
	f.byToken[m.RazorpayTokenID] = m
	return nil
}
func (f *fakeMandateRepo) GetByRazorpayTokenID(_ context.Context, tokenID string) (*domain.Mandate, error) {
	if m, ok := f.byToken[tokenID]; ok {
		return m, nil
	}
	return nil, errMandateNotFound
}
func (f *fakeMandateRepo) GetDueForPreNotification(context.Context) ([]*domain.Mandate, error) {
	return nil, nil
}
func (f *fakeMandateRepo) GetReadyForDebit(context.Context) ([]*domain.Mandate, error) {
	return nil, nil
}
func (f *fakeMandateRepo) ClaimDueForDebit(context.Context, time.Duration) ([]*domain.Mandate, error) {
	return nil, nil
}

func newGCMandateFixture() (*MandateService, *fakeMandateRepo, *domain.Mandate) {
	repo := &fakeMandateRepo{byToken: map[string]*domain.Mandate{}}
	m := &domain.Mandate{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		Currency:        "EUR",
		RazorpayTokenID: "BRQ123",
		Status:          domain.MandateStatusCreated,
	}
	repo.byToken[m.RazorpayTokenID] = m
	return NewMandateService(repo, nil, nil, nil), repo, m
}

func TestHandleGoCardlessFulfilmentActivatesAndSwapsToken(t *testing.T) {
	svc, repo, m := newGCMandateFixture()

	if err := svc.HandleGoCardlessFulfilment(context.Background(), "BRQ123", "MD999"); err != nil {
		t.Fatalf("HandleGoCardlessFulfilment: %v", err)
	}
	if m.Status != domain.MandateStatusActive {
		t.Fatalf("status = %s, want active", m.Status)
	}
	if m.ActivatedAt == nil || m.AuthorizedAt == nil {
		t.Fatal("activation timestamps not set")
	}
	// The token column must now hold the MD... id the debit worker sends to
	// GoCardless — debits against the BRQ... id 404.
	if m.RazorpayTokenID != "MD999" {
		t.Fatalf("token = %s, want MD999", m.RazorpayTokenID)
	}
	if _, err := repo.GetByRazorpayTokenID(context.Background(), "MD999"); err != nil {
		t.Fatal("mandate not findable by its GoCardless mandate id")
	}
}

func TestHandleGoCardlessFulfilmentUnknownBRQ(t *testing.T) {
	svc, _, _ := newGCMandateFixture()
	if err := svc.HandleGoCardlessFulfilment(context.Background(), "BRQ_UNKNOWN", "MD1"); err == nil {
		t.Fatal("want error for unknown billing request")
	}
}

func TestHandleGoCardlessMandateEventLifecycle(t *testing.T) {
	svc, repo, m := newGCMandateFixture()
	ctx := context.Background()

	// Activate via a mandates.active event addressed by the MD id.
	m.RazorpayTokenID = "MD1"
	delete(repo.byToken, "BRQ123")
	repo.byToken["MD1"] = m
	if err := svc.HandleGoCardlessMandateEvent(ctx, "MD1", "active"); err != nil {
		t.Fatalf("active: %v", err)
	}
	if m.Status != domain.MandateStatusActive || m.ActivatedAt == nil {
		t.Fatalf("after active: status=%s activated=%v", m.Status, m.ActivatedAt)
	}
	firstActivation := *m.ActivatedAt

	// A repeated activation is idempotent and keeps the original timestamp.
	if err := svc.HandleGoCardlessMandateEvent(ctx, "MD1", "active"); err != nil {
		t.Fatalf("re-active: %v", err)
	}
	if !m.ActivatedAt.Equal(firstActivation) {
		t.Fatal("re-activation moved the activation timestamp")
	}

	// Cancellation revokes so the debit worker stops claiming the row.
	if err := svc.HandleGoCardlessMandateEvent(ctx, "MD1", "cancelled"); err != nil {
		t.Fatalf("cancelled: %v", err)
	}
	if m.Status != domain.MandateStatusRevoked || m.RevokedAt == nil {
		t.Fatalf("after cancelled: status=%s revoked=%v", m.Status, m.RevokedAt)
	}

	// Unknown actions (submitted, created, ...) are deliberate no-ops.
	before := m.Status
	if err := svc.HandleGoCardlessMandateEvent(ctx, "MD1", "submitted"); err != nil {
		t.Fatalf("submitted: %v", err)
	}
	if m.Status != before {
		t.Fatal("no-op action changed status")
	}
}

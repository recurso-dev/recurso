package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// fakeEURetryStore stands in for the repository: it hands out a fixed claim batch
// once and records UpdateDelivery writes so the test can assert the outcome.
type fakeEURetryStore struct {
	claim   []*domain.EUInvoice
	claimed bool
	updated []*domain.EUInvoice
}

func (f *fakeEURetryStore) ClaimFailedEUInvoices(_ context.Context, _, _ time.Time, _ int) ([]*domain.EUInvoice, error) {
	if f.claimed {
		return nil, nil
	}
	f.claimed = true
	return f.claim, nil
}

func (f *fakeEURetryStore) UpdateDelivery(_ context.Context, e *domain.EUInvoice) error {
	// copy so later mutations of the same pointer don't rewrite history
	cp := *e
	f.updated = append(f.updated, &cp)
	return nil
}

// fakeEURetransmitter returns a fixed transport outcome (or error).
type fakeEURetransmitter struct {
	res  *domain.EUInvoiceTransmission
	err  error
	seen []*domain.EUInvoice
}

func (f *fakeEURetransmitter) RetryTransmission(_ context.Context, rec *domain.EUInvoice) (*domain.EUInvoiceTransmission, error) {
	f.seen = append(f.seen, rec)
	return f.res, f.err
}

func failedRec() *domain.EUInvoice {
	return &domain.EUInvoice{
		ID:             uuid.New(),
		InvoiceID:      uuid.New(),
		Syntax:         domain.EUInvoiceSyntaxUBL,
		Status:         domain.EUInvoiceStatusFailed,
		Document:       "<Invoice/>",
		RecipientVATID: "DE123456789",
		RetryCount:     0,
	}
}

// On a successful redrive the record becomes 'sent', records the message id, and
// clears the retry schedule.
func TestEURetryWorker_SuccessMarksSentAndClearsSchedule(t *testing.T) {
	store := &fakeEURetryStore{claim: []*domain.EUInvoice{failedRec()}}
	svc := &fakeEURetransmitter{res: &domain.EUInvoiceTransmission{MessageID: "ap-42", Status: domain.EUInvoiceStatusSent}}
	w := NewEUEInvoiceRetryWorker(store, svc)

	w.processRetries(context.Background())

	if len(store.updated) != 1 {
		t.Fatalf("want 1 update, got %d", len(store.updated))
	}
	got := store.updated[0]
	if got.Status != domain.EUInvoiceStatusSent {
		t.Errorf("status = %q, want sent", got.Status)
	}
	if got.MessageID != "ap-42" {
		t.Errorf("message id = %q, want ap-42", got.MessageID)
	}
	if got.NextRetryAt != nil {
		t.Errorf("next_retry_at should be cleared on success, got %v", got.NextRetryAt)
	}
	if got.ErrorMessage != "" {
		t.Errorf("error should be cleared on success, got %q", got.ErrorMessage)
	}
}

// A failed redrive advances the count and schedules the next attempt on the
// backoff curve, keeping the record 'failed'.
func TestEURetryWorker_FailureReschedulesOnBackoff(t *testing.T) {
	rec := failedRec()
	rec.RetryCount = 1 // next attempt is the 2nd → backoff index 1 (15m)
	store := &fakeEURetryStore{claim: []*domain.EUInvoice{rec}}
	svc := &fakeEURetransmitter{err: errors.New("access point 503")}
	w := NewEUEInvoiceRetryWorker(store, svc)

	before := time.Now().UTC()
	w.processRetries(context.Background())

	got := store.updated[0]
	if got.Status != domain.EUInvoiceStatusFailed {
		t.Errorf("status = %q, want failed", got.Status)
	}
	if got.RetryCount != 2 {
		t.Errorf("retry_count = %d, want 2", got.RetryCount)
	}
	if got.NextRetryAt == nil {
		t.Fatal("next_retry_at should be scheduled after a failed retry")
	}
	// index = retry_count-1 = 1 → 15m
	wantAt := before.Add(service.EUEInvoiceBackoff[1])
	if got.NextRetryAt.Before(wantAt.Add(-time.Minute)) || got.NextRetryAt.After(time.Now().UTC().Add(service.EUEInvoiceBackoff[1]).Add(time.Minute)) {
		t.Errorf("next_retry_at = %v, want ~%v", got.NextRetryAt, wantAt)
	}
}

// Once the retry count reaches the cap the worker stops cycling: it clears the
// schedule (so the row leaves the claim set) without another transmit attempt.
func TestEURetryWorker_MaxRetriesPermanentlyFails(t *testing.T) {
	rec := failedRec()
	rec.RetryCount = service.MaxEUEInvoiceRetries
	store := &fakeEURetryStore{claim: []*domain.EUInvoice{rec}}
	svc := &fakeEURetransmitter{res: &domain.EUInvoiceTransmission{MessageID: "should-not-be-used"}}
	w := NewEUEInvoiceRetryWorker(store, svc)

	w.processRetries(context.Background())

	if len(svc.seen) != 0 {
		t.Errorf("transport should not be called past the cap, got %d calls", len(svc.seen))
	}
	got := store.updated[0]
	if got.NextRetryAt != nil {
		t.Errorf("next_retry_at should be cleared at the cap, got %v", got.NextRetryAt)
	}
	if got.Status != domain.EUInvoiceStatusFailed {
		t.Errorf("status = %q, want failed", got.Status)
	}
}

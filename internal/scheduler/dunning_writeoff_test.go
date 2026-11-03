package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

type uncollectibleStub struct {
	InvoiceRepoForDunning
	marked int
}

func (s *uncollectibleStub) MarkAsUncollectible(_ context.Context, _ uuid.UUID) error {
	s.marked++
	return nil
}

// tenantScopedReader fails on a tenant-less context, like the real repository.
type tenantScopedReader struct{}

func (tenantScopedReader) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	tenantID, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if !ok {
		return nil, errors.New("tenant_id missing from context")
	}
	return &domain.Invoice{ID: id, TenantID: tenantID, SubscriptionID: &id}, nil
}

type recordingWriteOffPoster struct{ posted []*domain.Invoice }

func (p *recordingWriteOffPoster) RecordInvoiceWriteOff(_ context.Context, inv *domain.Invoice) error {
	p.posted = append(p.posted, inv)
	return nil
}

type recordingCanceler struct{ canceled []uuid.UUID }

func (c *recordingCanceler) CancelForNonPayment(_ context.Context, _, subscriptionID uuid.UUID) error {
	c.canceled = append(c.canceled, subscriptionID)
	return nil
}

// The sweep runs on context.Background(), so the write-off's tenant-scoped
// invoice read has to carry the invoice's own tenant or the reversal never posts.
func TestDunning_AutoWriteOffPostsTheLedgerReversal(t *testing.T) {
	repo := &uncollectibleStub{}
	poster := &recordingWriteOffPoster{}
	s := &DunningScheduler{invoiceRepo: repo, config: DefaultDunningConfig()}
	s.SetWriteOffLedger(poster, tenantScopedReader{})
	canceler := &recordingCanceler{}
	s.SetNonPaymentCanceler(canceler)

	tenantID := uuid.New()
	s.processInvoice(context.Background(), domain.OverdueInvoice{
		ID: uuid.New(), TenantID: tenantID, InvoiceNumber: "INV-1",
		DueDate: time.Now().AddDate(0, 0, -31),
	})

	if repo.marked != 1 {
		t.Fatalf("marked uncollectible %d times, want 1", repo.marked)
	}
	if len(poster.posted) != 1 || poster.posted[0].TenantID != tenantID {
		t.Fatalf("posted = %+v, want one write-off for tenant %s", poster.posted, tenantID)
	}
	if len(canceler.canceled) != 1 {
		t.Errorf("canceled %v, want the invoice's subscription", canceler.canceled)
	}
}

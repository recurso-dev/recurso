package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// reverseStubInvoiceRepo stubs the two reads ReverseSettledPayment needs.
type reverseStubInvoiceRepo struct {
	port.InvoiceRepository
	inv            *domain.Invoice
	reverseReturns bool
	reverseCalls   int
}

func (r *reverseStubInvoiceRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Invoice, error) {
	return r.inv, nil
}

func (r *reverseStubInvoiceRepo) ReverseToUnpaid(_ context.Context, _, _ uuid.UUID) (bool, error) {
	r.reverseCalls++
	return r.reverseReturns, nil
}

func newReversalSubService(repo port.InvoiceRepository, ledger *LedgerService) *SubscriptionService {
	return &SubscriptionService{invoiceRepo: repo, ledger: ledger, logger: slog.Default()}
}

// A paid invoice whose ACH debit bounced is reopened AND its settlement cash leg
// is reversed (DR AR / CR Cash, code 19) — exactly once.
func TestReverseSettledPayment_ReopensAndPostsReversal(t *testing.T) {
	ledgerRepo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	ledger := NewLedgerService(nil, ledgerRepo)

	repo := &reverseStubInvoiceRepo{
		inv: &domain.Invoice{
			ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
			InvoiceNumber: "INV-ACH-RET-3", Status: domain.InvoiceStatusPaid, Total: 1000,
		},
		reverseReturns: true,
	}
	svc := newReversalSubService(repo, ledger)

	reversed, err := svc.ReverseSettledPayment(context.Background(), repo.inv.ID)
	if err != nil {
		t.Fatalf("ReverseSettledPayment: %v", err)
	}
	if !reversed {
		t.Fatal("expected reversed=true for the caller that reopened the invoice")
	}
	if repo.reverseCalls != 1 {
		t.Errorf("ReverseToUnpaid called %d times, want 1", repo.reverseCalls)
	}
	if len(ledgerRepo.transactions) != 1 || ledgerRepo.transactions[0].Code != domain.LedgerCodePaymentReversal {
		t.Fatalf("expected exactly one reversal leg (code 19), got %+v", ledgerRepo.transactions)
	}
}

// A redelivered return finds the invoice already reopened (past_due) — it must
// NOT touch the DB again and must NOT re-post the ledger reversal.
func TestReverseSettledPayment_AlreadyReopenedIsNoop(t *testing.T) {
	ledgerRepo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	ledger := NewLedgerService(nil, ledgerRepo)

	repo := &reverseStubInvoiceRepo{
		inv: &domain.Invoice{
			ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
			InvoiceNumber: "INV-ACH-RET-4", Status: domain.InvoiceStatusPastDue, Total: 1000,
		},
	}
	svc := newReversalSubService(repo, ledger)

	reversed, err := svc.ReverseSettledPayment(context.Background(), repo.inv.ID)
	if err != nil {
		t.Fatalf("ReverseSettledPayment: %v", err)
	}
	if reversed {
		t.Error("a non-paid invoice must not report a reversal")
	}
	if repo.reverseCalls != 0 {
		t.Errorf("ReverseToUnpaid must not be called for a non-paid invoice, got %d calls", repo.reverseCalls)
	}
	if len(ledgerRepo.transactions) != 0 {
		t.Errorf("no ledger reversal must post on a no-op, got %d", len(ledgerRepo.transactions))
	}
}

// If a concurrent settler wins the conditional UPDATE (ReverseToUnpaid returns
// false), this caller must not post the ledger reversal — only the winner does.
func TestReverseSettledPayment_LostRacePostsNoLedger(t *testing.T) {
	ledgerRepo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	ledger := NewLedgerService(nil, ledgerRepo)

	repo := &reverseStubInvoiceRepo{
		inv: &domain.Invoice{
			ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
			InvoiceNumber: "INV-ACH-RET-5", Status: domain.InvoiceStatusPaid, Total: 1000,
		},
		reverseReturns: false, // lost the race
	}
	svc := newReversalSubService(repo, ledger)

	reversed, err := svc.ReverseSettledPayment(context.Background(), repo.inv.ID)
	if err != nil {
		t.Fatalf("ReverseSettledPayment: %v", err)
	}
	if reversed {
		t.Error("the caller that lost the reopen race must report reversed=false")
	}
	if len(ledgerRepo.transactions) != 0 {
		t.Errorf("the loser must not post the ledger reversal, got %d legs", len(ledgerRepo.transactions))
	}
}

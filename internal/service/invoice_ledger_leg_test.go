package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// fakeLedgerPoster captures the RecordInvoice call the invoice generators are
// expected to make. It records the invoice total AND the AmountPaid seen at
// call time, so a test can prove the base leg is posted at the GROSS amount and
// BEFORE any wallet/credit reducing leg has touched the invoice.
type fakeLedgerPoster struct {
	calls           int
	sawTotal        int64
	sawAmountPaid   int64
	sawSubscription bool
}

func (f *fakeLedgerPoster) RecordInvoice(ctx context.Context, inv *domain.Invoice) error {
	f.calls++
	f.sawTotal = inv.Total
	f.sawAmountPaid = inv.AmountPaid
	f.sawSubscription = inv.SubscriptionID != nil
	return nil
}

// TestGenerateInvoice_PostsBaseLedgerLeg is the regression guard for the
// renewal-ledger bug: GenerateInvoice (the renewal path's only invoice source)
// must post its base AR→Deferred leg via the ledger poster, at the gross total,
// before the wallet drain relieves AR. Without this, renewal invoices carried
// no Code-1 ledger row and their deferred revenue was never funded.
func TestGenerateInvoice_PostsBaseLedgerLeg(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"), // 18% IGST -> total 118000
	}}
	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})

	poster := &fakeLedgerPoster{}
	svc.LedgerPoster = poster
	// A wallet that partly covers the invoice: the base leg must still post the
	// GROSS total and must be posted before this drain runs.
	svc.WalletDrainer = &fakeInvoiceWalletDrainer{gives: 30000}

	sub := &domain.Subscription{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: custRepo.customer.ID, PlanID: planRepo.plan.ID,
	}
	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if poster.calls != 1 {
		t.Fatalf("RecordInvoice called %d times, want exactly 1", poster.calls)
	}
	if poster.sawTotal != inv.Total {
		t.Fatalf("base leg posted total %d, want the gross %d", poster.sawTotal, inv.Total)
	}
	// Posted before the wallet drain: no AmountPaid yet at call time.
	if poster.sawAmountPaid != 0 {
		t.Fatalf("base leg saw AmountPaid=%d at post time, want 0 (must precede wallet drain)", poster.sawAmountPaid)
	}
	// A subscription invoice must defer revenue (RecordInvoice branches on this).
	if !poster.sawSubscription {
		t.Fatal("base leg posted a non-subscription invoice; renewal invoices must carry SubscriptionID so revenue defers")
	}
	// Sanity: the wallet drain still ran afterward.
	if inv.AmountPaid != 30000 {
		t.Fatalf("AmountPaid = %d, want the 30000 wallet drain applied after the leg", inv.AmountPaid)
	}
}

// TestGenerateInvoice_NoLedgerPoster_NoPanic proves the leg posting is nil-safe:
// an InvoiceService with no wired ledger generates invoices without panicking.
func TestGenerateInvoice_NoLedgerPoster_NoPanic(t *testing.T) {
	invRepo := &mockInvoiceRepoForInvAmt{}
	planRepo := &mockPlanRepoForInvAmt{plan: &domain.Plan{
		ID:     uuid.New(),
		Prices: []domain.Price{{Amount: 100000, Currency: "INR"}},
	}}
	custRepo := &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID:            uuid.New(),
		PlaceOfSupply: domain.StringPtr("KA"),
	}}
	svc := newInvAmtService(invRepo, planRepo, custRepo, &mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{})
	// LedgerPoster deliberately left nil.

	if _, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: custRepo.customer.ID, PlanID: planRepo.plan.ID,
	}); err != nil {
		t.Fatalf("unexpected error with no ledger poster: %v", err)
	}
}

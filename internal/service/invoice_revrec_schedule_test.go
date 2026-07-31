package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/gsp"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeWalletDrainerFull struct{ drain bool }

func (f fakeWalletDrainerFull) DrainForInvoice(_ context.Context, inv *domain.Invoice) (int64, error) {
	if !f.drain {
		return 0, nil
	}
	return inv.Total, nil // fully cover
}

type recordingRevRecScheduler struct{ scheduled []uuid.UUID }

func (r *recordingRevRecScheduler) CreateScheduleForInvoice(_ context.Context, invoice *domain.Invoice, _ *domain.Subscription) error {
	r.scheduled = append(r.scheduled, invoice.ID)
	return nil
}

// TestGenerateInvoice_WalletCoveredCreatesRecognitionSchedule proves an invoice
// fully covered by a wallet drain at generation (so it's Paid without ever going
// through MarkInvoicePaid) still gets a recognition schedule — otherwise the
// Deferred it funded would never be recognized (revenue understated forever).
func TestGenerateInvoice_WalletCoveredCreatesRecognitionSchedule(t *testing.T) {
	planID := uuid.New()
	custID := uuid.New()
	tenantID := uuid.New()

	newSvc := func() (*InvoiceService, *recordingRevRecScheduler) {
		svc := NewInvoiceService(
			&MockInvoiceRepo{},
			&MockPlanRepo{Plan: &domain.Plan{ID: planID, Prices: []domain.Price{{Amount: 100000, Currency: "USD"}}}},
			&MockCustomerRepo{Customer: &domain.Customer{ID: custID}},
			&MockUnbilledChargeRepo{}, &MockSubscriptionRepo{}, gsp.NewMockGSPAdapter(), nil,
		)
		sched := &recordingRevRecScheduler{}
		svc.RevRecScheduler = sched
		return svc, sched
	}
	sub := &domain.Subscription{ID: uuid.New(), CustomerID: custID, PlanID: planID, TenantID: tenantID}

	// Wallet fully covers → invoice Paid → schedule created.
	svc, sched := newSvc()
	svc.WalletDrainer = fakeWalletDrainerFull{drain: true}
	inv, err := svc.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("GenerateInvoice: %v", err)
	}
	if inv.Status != domain.InvoiceStatusPaid {
		t.Fatalf("invoice status = %q, want paid (wallet covered)", inv.Status)
	}
	if len(sched.scheduled) != 1 || sched.scheduled[0] != inv.ID {
		t.Errorf("recognition schedule created %d times, want 1 for the wallet-covered invoice", len(sched.scheduled))
	}

	// Control: no wallet coverage → invoice stays Open → schedule NOT created here
	// (MarkInvoicePaid creates it when the invoice is actually paid).
	svc2, sched2 := newSvc()
	svc2.WalletDrainer = fakeWalletDrainerFull{drain: false}
	inv2, err := svc2.GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("GenerateInvoice(control): %v", err)
	}
	if inv2.Status == domain.InvoiceStatusPaid {
		t.Fatalf("control invoice should not be paid")
	}
	if len(sched2.scheduled) != 0 {
		t.Errorf("an unpaid invoice must not create a schedule at generation; got %d", len(sched2.scheduled))
	}
}

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

var errInvoiceNotFoundTest = errors.New("invoice not found")

// --- fakes ---

type fakeOfflineRepo struct {
	port.OfflinePaymentRepository
	payments []*domain.OfflinePayment
	vas      map[string]*domain.VirtualAccount // by RazorpayVAID
}

func (r *fakeOfflineRepo) CreateOfflinePayment(_ context.Context, p *domain.OfflinePayment) error {
	r.payments = append(r.payments, p)
	return nil
}
func (r *fakeOfflineRepo) GetVirtualAccountByRazorpayID(_ context.Context, id string) (*domain.VirtualAccount, error) {
	return r.vas[id], nil
}
func (r *fakeOfflineRepo) UpdateVirtualAccount(_ context.Context, va *domain.VirtualAccount) error {
	r.vas[va.RazorpayVAID] = va
	return nil
}

type fakeOfflineInvoiceRepo struct {
	port.InvoiceRepository
	invoices map[uuid.UUID]*domain.Invoice
}

func (r *fakeOfflineInvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if _, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID); !ok {
		return nil, errInvoiceNotFoundTest
	}
	inv, ok := r.invoices[id]
	if !ok {
		return nil, errInvoiceNotFoundTest
	}
	return inv, nil
}

type fakeInvoiceMarker struct {
	marked []uuid.UUID
}

func (m *fakeInvoiceMarker) MarkInvoicePaid(_ context.Context, id uuid.UUID) (bool, error) {
	m.marked = append(m.marked, id)
	return true, nil
}

// --- tests ---

// TestRecordOfflinePayment_SettlesOnlyWhenCovered proves the ENG-169 fix: a
// recorded offline payment settles the linked invoice only when the amount
// (plus anything already paid) covers the total. A short payment is recorded
// but must NOT mark the whole invoice paid — previously any amount did, silently
// writing off the balance.
func TestRecordOfflinePayment_SettlesOnlyWhenCovered(t *testing.T) {
	tenant := uuid.New()
	invID := uuid.New()
	cust := uuid.New()

	newSvc := func(total, alreadyPaid int64) (*OfflinePaymentService, *fakeInvoiceMarker) {
		invRepo := &fakeOfflineInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
			invID: {ID: invID, TenantID: tenant, CustomerID: cust, Currency: "INR", Total: total, AmountPaid: alreadyPaid},
		}}
		marker := &fakeInvoiceMarker{}
		return NewOfflinePaymentService(&fakeOfflineRepo{}, nil, invRepo, marker), marker
	}
	record := func(svc *OfflinePaymentService, amount int64, currency string, customer uuid.UUID) error {
		_, err := svc.RecordOfflinePayment(context.Background(), RecordOfflinePaymentInput{
			TenantID: tenant, CustomerID: customer, InvoiceID: &invID, Amount: amount, Currency: currency, PaymentType: "bank_transfer",
		})
		return err
	}

	// Full payment settles the invoice.
	svc, marker := newSvc(10000, 0)
	if err := record(svc, 10000, "INR", cust); err != nil {
		t.Fatalf("full payment: %v", err)
	}
	if len(marker.marked) != 1 {
		t.Fatalf("full payment: invoice marked %d times, want 1", len(marker.marked))
	}

	// Short payment records but does NOT settle (the revenue-leak case).
	svc, marker = newSvc(10000, 0)
	if err := record(svc, 100, "INR", cust); err != nil {
		t.Fatalf("short payment: %v", err)
	}
	if len(marker.marked) != 0 {
		t.Fatalf("short payment settled the invoice (%d marks); a partial amount must not write off the balance", len(marker.marked))
	}

	// A top-up that brings the paid amount up to the total settles it.
	svc, marker = newSvc(10000, 9000)
	if err := record(svc, 1000, "INR", cust); err != nil {
		t.Fatalf("top-up payment: %v", err)
	}
	if len(marker.marked) != 1 {
		t.Fatalf("top-up payment: invoice marked %d times, want 1", len(marker.marked))
	}

	// A payment for a DIFFERENT customer, or in a different currency, is refused
	// (ENG-182) — it must not settle this invoice.
	svc, marker = newSvc(10000, 0)
	if err := record(svc, 10000, "INR", uuid.New()); err == nil {
		t.Error("cross-customer offline payment: expected error, got nil")
	}
	if err := record(svc, 10000, "JPY", cust); err == nil {
		t.Error("currency-mismatch offline payment: expected error, got nil")
	}
	if len(marker.marked) != 0 {
		t.Fatalf("mismatched payment settled the invoice (%d marks)", len(marker.marked))
	}
}

// TestReconcileVirtualAccount_SettlesWhenExpectedReached proves the VA path only
// closes and settles once the accumulated received amount reaches the expected.
func TestReconcileVirtualAccount_SettlesWhenExpectedReached(t *testing.T) {
	tenant := uuid.New()
	invID := uuid.New()
	vaID := "va_test_1"

	repo := &fakeOfflineRepo{vas: map[string]*domain.VirtualAccount{
		vaID: {ID: uuid.New(), TenantID: tenant, InvoiceID: &invID, RazorpayVAID: vaID, AmountExpected: 10000, Status: "active"},
	}}
	marker := &fakeInvoiceMarker{}
	svc := NewOfflinePaymentService(repo, nil, &fakeOfflineInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{}}, marker)

	// First partial deposit: not closed, invoice untouched.
	if err := svc.ReconcileVirtualAccount(context.Background(), vaID, 4000, "pay_1"); err != nil {
		t.Fatalf("first deposit: %v", err)
	}
	if repo.vas[vaID].Status != "active" || len(marker.marked) != 0 {
		t.Fatalf("after partial deposit: status=%s marks=%d, want active/0", repo.vas[vaID].Status, len(marker.marked))
	}

	// Second deposit reaches the expected: closed and invoice settled.
	if err := svc.ReconcileVirtualAccount(context.Background(), vaID, 6000, "pay_2"); err != nil {
		t.Fatalf("second deposit: %v", err)
	}
	if repo.vas[vaID].Status != "closed" {
		t.Fatalf("after full deposit: status=%s, want closed", repo.vas[vaID].Status)
	}
	if len(marker.marked) != 1 || marker.marked[0] != invID {
		t.Fatalf("after full deposit: invoice marks=%v, want [%s]", marker.marked, invID)
	}
}

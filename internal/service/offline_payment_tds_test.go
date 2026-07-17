package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// Update lets the TDS path persist the accumulated deduction on the invoice.
func (r *fakeOfflineInvoiceRepo) Update(ctx context.Context, inv *domain.Invoice) error {
	if _, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID); !ok {
		return errInvoiceNotFoundTest
	}
	r.invoices[inv.ID] = inv
	return nil
}

// A customer that deducts TDS pays net of it: the receipt settles the invoice
// when amount + TDS covers the total, and the deduction is accumulated on the
// invoice so the ledger books it to TDS Receivable at settlement.
func TestRecordOfflinePayment_TDSSettlesInvoice(t *testing.T) {
	tenant := uuid.New()
	invID := uuid.New()
	cust := uuid.New()

	invRepo := &fakeOfflineInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
		invID: {ID: invID, TenantID: tenant, CustomerID: cust, Currency: "INR", Total: 118000},
	}}
	marker := &fakeInvoiceMarker{}
	svc := NewOfflinePaymentService(&fakeOfflineRepo{}, nil, invRepo, marker)

	// ₹1,180.00 invoice, customer deducts 10% TDS on the ₹1,000 taxable value
	// (₹100) and transfers ₹1,080.00.
	p, err := svc.RecordOfflinePayment(context.Background(), RecordOfflinePaymentInput{
		TenantID: tenant, CustomerID: cust, InvoiceID: &invID,
		Amount: 108000, TDSAmount: 10000, Currency: "INR", PaymentType: "bank_transfer",
	})
	if err != nil {
		t.Fatalf("RecordOfflinePayment: %v", err)
	}
	if p.TDSAmount != 10000 {
		t.Errorf("payment TDSAmount = %d, want 10000", p.TDSAmount)
	}
	if got := invRepo.invoices[invID].TDSAmount; got != 10000 {
		t.Errorf("invoice TDSAmount = %d, want 10000", got)
	}
	if len(marker.marked) != 1 {
		t.Errorf("invoice should settle (amount+TDS covers total); marked %d times", len(marker.marked))
	}
}

// A receipt whose amount + TDS still falls short is recorded but must not
// settle the invoice.
func TestRecordOfflinePayment_TDSShortPaymentStaysOpen(t *testing.T) {
	tenant := uuid.New()
	invID := uuid.New()
	cust := uuid.New()

	invRepo := &fakeOfflineInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
		invID: {ID: invID, TenantID: tenant, CustomerID: cust, Currency: "INR", Total: 118000},
	}}
	marker := &fakeInvoiceMarker{}
	svc := NewOfflinePaymentService(&fakeOfflineRepo{}, nil, invRepo, marker)

	if _, err := svc.RecordOfflinePayment(context.Background(), RecordOfflinePaymentInput{
		TenantID: tenant, CustomerID: cust, InvoiceID: &invID,
		Amount: 50000, TDSAmount: 10000, Currency: "INR", PaymentType: "bank_transfer",
	}); err != nil {
		t.Fatalf("RecordOfflinePayment: %v", err)
	}
	if len(marker.marked) != 0 {
		t.Error("short payment must not settle the invoice")
	}
	if got := invRepo.invoices[invID].TDSAmount; got != 10000 {
		t.Errorf("invoice TDSAmount = %d, want 10000 (recorded even when open)", got)
	}
}

func TestRecordOfflinePayment_TDSValidation(t *testing.T) {
	tenant := uuid.New()
	invID := uuid.New()
	cust := uuid.New()

	newSvc := func() *OfflinePaymentService {
		invRepo := &fakeOfflineInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
			invID: {ID: invID, TenantID: tenant, CustomerID: cust, Currency: "INR", Total: 118000, AmountPaid: 100000},
		}}
		return NewOfflinePaymentService(&fakeOfflineRepo{}, nil, invRepo, &fakeInvoiceMarker{})
	}

	// TDS without a linked invoice is meaningless.
	if _, err := newSvc().RecordOfflinePayment(context.Background(), RecordOfflinePaymentInput{
		TenantID: tenant, CustomerID: cust, Amount: 1000, TDSAmount: 100, PaymentType: "cash",
	}); err == nil {
		t.Error("TDS without an invoice must be rejected")
	}

	// TDS above the outstanding balance (118000-100000=18000) must be rejected.
	if _, err := newSvc().RecordOfflinePayment(context.Background(), RecordOfflinePaymentInput{
		TenantID: tenant, CustomerID: cust, InvoiceID: &invID,
		Amount: 1000, TDSAmount: 20000, Currency: "INR", PaymentType: "bank_transfer",
	}); err == nil {
		t.Error("TDS exceeding the outstanding balance must be rejected")
	}

	// Negative TDS must be rejected.
	if _, err := newSvc().RecordOfflinePayment(context.Background(), RecordOfflinePaymentInput{
		TenantID: tenant, CustomerID: cust, InvoiceID: &invID,
		Amount: 1000, TDSAmount: -1, Currency: "INR", PaymentType: "bank_transfer",
	}); err == nil {
		t.Error("negative TDS must be rejected")
	}
}

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// A clawed-back ACH debit reverses the settlement leg: DR Accounts Receivable /
// CR Cash for the exact cash that was collected — the inverse of the Code-3
// payment leg — so the receivable is reinstated and the invoice can be
// re-collected. The amount is net of any account credit applied at settlement.
func TestLedgerRecordPaymentReversal_InvertsCashLeg(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-ACH-RET-1", Total: 1000, CreditApplied: 200,
	}
	if err := svc.RecordPaymentReversal(context.Background(), inv); err != nil {
		t.Fatalf("RecordPaymentReversal: %v", err)
	}

	if len(repo.transactions) != 1 {
		t.Fatalf("expected exactly the reversal leg, got %d", len(repo.transactions))
	}
	rev := repo.transactions[0]
	if rev.Code != domain.LedgerCodePaymentReversal {
		t.Errorf("code = %d, want %d (payment reversal)", rev.Code, domain.LedgerCodePaymentReversal)
	}
	if rev.Amount != 800 { // Total 1000 − CreditApplied 200 = cash actually collected
		t.Errorf("amount = %d, want 800 (cash collected, net of applied credit)", rev.Amount)
	}
	if rev.DebitAccountID != inv.CustomerID {
		t.Error("reversal must DEBIT the customer's AR account (reinstate the receivable)")
	}
	if rev.CreditAccountID == inv.CustomerID {
		t.Error("reversal must CREDIT Cash, not AR")
	}
	if rev.ReferenceID != inv.ID {
		t.Error("reversal must reference the invoice — the (reference_id, code) index makes it idempotent")
	}
}

// An invoice fully covered by account credit collected no gateway cash, so a
// return has no cash leg to reverse — posting one would drive Cash negative.
func TestLedgerRecordPaymentReversal_NoCashNoLeg(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-ACH-RET-2", Total: 500, CreditApplied: 500,
	}
	if err := svc.RecordPaymentReversal(context.Background(), inv); err != nil {
		t.Fatalf("RecordPaymentReversal: %v", err)
	}
	if len(repo.transactions) != 0 {
		t.Fatalf("expected no legs when no cash was collected, got %d", len(repo.transactions))
	}
}

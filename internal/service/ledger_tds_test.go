package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// A paid invoice with TDS deducted must post two legs, both relieving AR:
// DR TDS Receivable for the withheld portion, DR Cash only for the net amount
// actually received — never the gross total.
func TestLedgerRecordPayment_TDSSplitsCashAndReceivable(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-TDS-1", Total: 118000, TDSAmount: 10000,
	}
	if err := svc.RecordPayment(context.Background(), inv); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}

	if len(repo.transactions) != 2 {
		t.Fatalf("expected 2 legs (TDS + cash), got %d", len(repo.transactions))
	}

	tdsLeg, cashLeg := repo.transactions[0], repo.transactions[1]
	if tdsLeg.Code != domain.LedgerCodeTDSReceivable || tdsLeg.Amount != 10000 {
		t.Errorf("TDS leg = code %d amount %d, want code %d amount 10000", tdsLeg.Code, tdsLeg.Amount, domain.LedgerCodeTDSReceivable)
	}
	if cashLeg.Code != 3 || cashLeg.Amount != 108000 {
		t.Errorf("cash leg = code %d amount %d, want code 3 amount 108000 (net of TDS)", cashLeg.Code, cashLeg.Amount)
	}
	if tdsLeg.CreditAccountID != inv.CustomerID || cashLeg.CreditAccountID != inv.CustomerID {
		t.Error("both legs must credit the customer's AR account")
	}
	if tdsLeg.DebitAccountID == cashLeg.DebitAccountID {
		t.Error("TDS leg must debit TDS Receivable, not the Cash account")
	}
}

// An invoice fully settled by TDS plus account credit has no cash leg at all.
func TestLedgerRecordPayment_TDSNoCashLegWhenFullyCovered(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-TDS-2", Total: 10000, TDSAmount: 4000, CreditApplied: 6000,
	}
	if err := svc.RecordPayment(context.Background(), inv); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}
	if len(repo.transactions) != 1 {
		t.Fatalf("expected only the TDS leg, got %d legs", len(repo.transactions))
	}
	if repo.transactions[0].Code != domain.LedgerCodeTDSReceivable || repo.transactions[0].Amount != 4000 {
		t.Errorf("TDS leg = code %d amount %d, want code %d amount 4000",
			repo.transactions[0].Code, repo.transactions[0].Amount, domain.LedgerCodeTDSReceivable)
	}
}

// Negative TDS on the invoice is a caller bug and must be rejected outright.
func TestLedgerRecordPayment_NegativeTDSRejected(t *testing.T) {
	repo := &mockLedgerRepoForLedger{}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(), Total: 1000, TDSAmount: -1}
	if err := svc.RecordPayment(context.Background(), inv); err == nil {
		t.Error("RecordPayment must reject negative TDS, got nil")
	}
	if len(repo.transactions) != 0 {
		t.Errorf("expected 0 persisted transactions, got %d", len(repo.transactions))
	}
}

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// A wallet-drained invoice must NOT re-book the wallet portion as cash when it
// is finally paid. The wallet drain already relieved AR (DR Customer-Credit /
// CR AR) at generation and the money was cashed at top-up; the payment's cash
// leg is only the NET gateway cash. Booking the gross here would double-count
// the wallet amount as cash and drive AR negative.
func TestLedgerRecordPaymentWithSettled_ExcludesWalletPortion(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	// Total 1000, of which 400 was already settled by a prepaid wallet drain.
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-WAL-1", Total: 1000,
	}
	if err := svc.RecordPaymentWithSettled(context.Background(), inv, 400); err != nil {
		t.Fatalf("RecordPaymentWithSettled: %v", err)
	}

	if len(repo.transactions) != 1 {
		t.Fatalf("expected exactly the cash leg, got %d legs", len(repo.transactions))
	}
	cash := repo.transactions[0]
	if cash.Code != 3 || cash.Amount != 600 {
		t.Errorf("cash leg = code %d amount %d, want code 3 amount 600 (net of the 400 wallet)", cash.Code, cash.Amount)
	}
	if cash.CreditAccountID != inv.CustomerID {
		t.Error("cash leg must credit the customer's AR account")
	}
}

// When the wallet fully covers the invoice, the final payment posts NO cash leg
// at all — there is no gateway cash to book.
func TestLedgerRecordPaymentWithSettled_WalletFullyCovers_NoCashLeg(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-WAL-2", Total: 1000,
	}
	if err := svc.RecordPaymentWithSettled(context.Background(), inv, 1000); err != nil {
		t.Fatalf("RecordPaymentWithSettled: %v", err)
	}
	if len(repo.transactions) != 0 {
		t.Fatalf("wallet fully covered the invoice — expected no cash leg, got %d", len(repo.transactions))
	}
}

// Wallet + TDS combined: cash leg is Total − TDS − wallet, TDS leg posts for the
// withheld portion, and AR (credited by every leg plus the earlier wallet drain)
// nets exactly to the invoice total.
func TestLedgerRecordPaymentWithSettled_WalletPlusTDS(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	// Total 118000; wallet already settled 18000; customer withholds 10000 TDS.
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-WAL-3", Total: 118000, TDSAmount: 10000,
	}
	if err := svc.RecordPaymentWithSettled(context.Background(), inv, 18000); err != nil {
		t.Fatalf("RecordPaymentWithSettled: %v", err)
	}

	var arCredited int64
	var cashAmt int64
	for _, tx := range repo.transactions {
		if tx.CreditAccountID == inv.CustomerID {
			arCredited += int64(tx.Amount)
		}
		if tx.Code == 3 {
			cashAmt = int64(tx.Amount)
		}
	}
	if cashAmt != 90000 { // 118000 - 10000 TDS - 18000 wallet
		t.Errorf("cash leg = %d, want 90000 (net of TDS and wallet)", cashAmt)
	}
	// The payment legs relieve AR for everything except the wallet portion, which
	// the earlier wallet-drain leg already relieved: 10000 (TDS) + 90000 (cash)
	// = 100000, and 100000 + 18000 wallet = 118000 = Total. AR nets to zero.
	if arCredited != 100000 {
		t.Errorf("payment legs credited AR %d, want 100000 (18000 was relieved by the wallet drain)", arCredited)
	}
}

// Negative already-settled is a caller bug and must be rejected.
func TestLedgerRecordPaymentWithSettled_NegativeRejected(t *testing.T) {
	repo := &mockLedgerRepoForLedger{}
	svc := NewLedgerService(nil, repo)
	inv := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(), Total: 1000}
	if err := svc.RecordPaymentWithSettled(context.Background(), inv, -1); err == nil {
		t.Error("must reject a negative already-settled amount, got nil")
	}
	if len(repo.transactions) != 0 {
		t.Error("no legs should post when validation fails")
	}
}

// RecordPayment stays byte-compatible: it delegates with alreadySettled=0.
func TestLedgerRecordPayment_DelegatesWithZeroSettled(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-WAL-4", Total: 1000,
	}
	if err := svc.RecordPayment(context.Background(), inv); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}
	if len(repo.transactions) != 1 || repo.transactions[0].Amount != 1000 {
		t.Fatalf("RecordPayment should book the full 1000 cash leg when nothing was pre-settled")
	}
}

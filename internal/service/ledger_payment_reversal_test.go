package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// A clawed-back ACH debit reverses the settlement by inverting the ACTUAL
// latest Code-3 cash leg — same amount, same two accounts swapped — under a
// new ledger code 19 (docs/design-ledger-occurrence.md).
func TestLedgerRecordPaymentReversal_InvertsCashLeg(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-ACH-RET-1", Total: 1000, CreditApplied: 200,
	}
	// Settle first: posts the real cash leg (800 = Total − CreditApplied).
	if err := svc.RecordPaymentWithSettled(context.Background(), inv, 0); err != nil {
		t.Fatalf("RecordPaymentWithSettled: %v", err)
	}
	if len(repo.transactions) != 1 {
		t.Fatalf("expected the settlement cash leg, got %d legs", len(repo.transactions))
	}
	cash := repo.transactions[0]

	if err := svc.RecordPaymentReversal(context.Background(), inv); err != nil {
		t.Fatalf("RecordPaymentReversal: %v", err)
	}
	if len(repo.transactions) != 2 {
		t.Fatalf("expected settlement + reversal, got %d legs", len(repo.transactions))
	}
	rev := repo.transactions[1]
	if rev.Code != domain.LedgerCodePaymentReversal {
		t.Errorf("code = %d, want %d (payment reversal)", rev.Code, domain.LedgerCodePaymentReversal)
	}
	if rev.Amount != cash.Amount {
		t.Errorf("reversal amount = %d, want %d (the actual cash leg)", rev.Amount, cash.Amount)
	}
	// Exact inverse: the cash leg's two accounts, swapped.
	if rev.DebitAccountID != cash.CreditAccountID || rev.CreditAccountID != cash.DebitAccountID {
		t.Error("reversal must swap the cash leg's debit/credit accounts (DR AR / CR Cash)")
	}
	if rev.ReferenceID != inv.ID {
		t.Error("reversal must reference the invoice")
	}
	if rev.Occurrence != 0 {
		t.Errorf("first reversal occurrence = %d, want 0", rev.Occurrence)
	}
}

// A wallet-part-funded settlement posted only the NET cash; the reversal must
// claw back exactly that net figure, never the gross (QA finding C).
func TestLedgerRecordPaymentReversal_WalletFunded_ReversesNetOnly(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-ACH-RET-W", Total: 1000,
	}
	// 400 was pre-settled by a wallet drain — cash leg is 600.
	if err := svc.RecordPaymentWithSettled(context.Background(), inv, 400); err != nil {
		t.Fatalf("RecordPaymentWithSettled: %v", err)
	}
	if err := svc.RecordPaymentReversal(context.Background(), inv); err != nil {
		t.Fatalf("RecordPaymentReversal: %v", err)
	}
	rev := repo.transactions[len(repo.transactions)-1]
	if rev.Code != domain.LedgerCodePaymentReversal || rev.Amount != 600 {
		t.Errorf("reversal = code %d amount %d, want code 19 amount 600 (net cash, not the 1000 gross)", rev.Code, rev.Amount)
	}
}

// An invoice that never collected gateway cash (fully covered by credit and/or
// wallet) has no cash leg — a return has nothing to claw back.
func TestLedgerRecordPaymentReversal_NoCashNoLeg(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-ACH-RET-2", Total: 500, CreditApplied: 500,
	}
	// Settle posts nothing (no cash collected)…
	if err := svc.RecordPaymentWithSettled(context.Background(), inv, 0); err != nil {
		t.Fatalf("RecordPaymentWithSettled: %v", err)
	}
	// …so the reversal must post nothing either.
	if err := svc.RecordPaymentReversal(context.Background(), inv); err != nil {
		t.Fatalf("RecordPaymentReversal: %v", err)
	}
	if len(repo.transactions) != 0 {
		t.Fatalf("expected no legs when no cash was collected, got %d", len(repo.transactions))
	}
}

// THE re-collection fix (QA finding A): settle → reverse → re-settle must post
// a FRESH cash leg at occurrence 1 instead of being swallowed by the dedup, and
// a same-cycle duplicate settle must still dedup. A second full cycle works too.
func TestLedgerPaymentCycle_ResettleAfterReversalPosts(t *testing.T) {
	repo := &mockLedgerRepoForLedger{accountsByCode: map[int]*domain.LedgerAccount{}}
	svc := NewLedgerService(nil, repo)
	ctx := context.Background()

	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), CustomerID: uuid.New(),
		InvoiceNumber: "INV-ACH-CYCLE", Total: 1000,
	}

	countByCode := func(code uint16) int {
		n := 0
		for _, tx := range repo.transactions {
			if tx.Code == code {
				n++
			}
		}
		return n
	}

	// Cycle 1: settle (occ 0) — and a redelivered duplicate dedups.
	if err := svc.RecordPaymentWithSettled(ctx, inv, 0); err != nil {
		t.Fatalf("settle 1: %v", err)
	}
	if err := svc.RecordPaymentWithSettled(ctx, inv, 0); err != nil {
		t.Fatalf("settle 1 (dup): %v", err)
	}
	if got := countByCode(3); got != 1 {
		t.Fatalf("after duplicate settle: %d cash legs, want 1 (same-cycle dedup)", got)
	}

	// Return: reversal occ 0 — duplicate reversal dedups too.
	if err := svc.RecordPaymentReversal(ctx, inv); err != nil {
		t.Fatalf("reverse 1: %v", err)
	}
	if err := svc.RecordPaymentReversal(ctx, inv); err != nil {
		t.Fatalf("reverse 1 (dup): %v", err)
	}
	if got := countByCode(domain.LedgerCodePaymentReversal); got != 1 {
		t.Fatalf("after duplicate reversal: %d reversal legs, want 1", got)
	}

	// Re-collection: the second settle MUST land (occurrence 1). This is the
	// leg the old (reference, code) dedup silently swallowed.
	if err := svc.RecordPaymentWithSettled(ctx, inv, 0); err != nil {
		t.Fatalf("re-settle: %v", err)
	}
	if got := countByCode(3); got != 2 {
		t.Fatalf("after re-settle: %d cash legs, want 2 (the re-collection must post)", got)
	}
	last := repo.transactions[len(repo.transactions)-1]
	if last.Code != 3 || last.Occurrence != 1 {
		t.Errorf("re-settle leg = code %d occ %d, want code 3 occ 1", last.Code, last.Occurrence)
	}

	// Cycle 2: a second return and a third collection also work.
	if err := svc.RecordPaymentReversal(ctx, inv); err != nil {
		t.Fatalf("reverse 2: %v", err)
	}
	if got := countByCode(domain.LedgerCodePaymentReversal); got != 2 {
		t.Fatalf("second reversal: %d reversal legs, want 2", got)
	}
	if err := svc.RecordPaymentWithSettled(ctx, inv, 0); err != nil {
		t.Fatalf("settle 3: %v", err)
	}
	if got := countByCode(3); got != 3 {
		t.Fatalf("third settle: %d cash legs, want 3", got)
	}

	// Net effect: 3 settles − 2 reversals = exactly one settlement's cash.
	var net int64
	for _, tx := range repo.transactions {
		if tx.Code == 3 {
			net += int64(tx.Amount)
		}
		if tx.Code == domain.LedgerCodePaymentReversal {
			net -= int64(tx.Amount)
		}
	}
	if net != 1000 {
		t.Errorf("net cash across the cycles = %d, want 1000", net)
	}
}

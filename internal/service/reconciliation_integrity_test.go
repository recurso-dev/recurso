package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// timeZero is a fixed as-of stamp; the integrity helpers ignore it.
func timeZero() time.Time { return time.Time{} }

// TestTrialBalanceDiscrepancies_Clean: a balanced set of accounts with no
// wrong-sign balances yields no integrity findings.
func TestTrialBalanceDiscrepancies_Clean(t *testing.T) {
	tb := finalizeTrialBalance(uuid.New(), []domain.TrialBalanceLine{
		{Code: domain.AccountCodeCash, Type: domain.AccountTypeAsset, Debits: 10000, Credits: 0},
		{Code: domain.AccountCodeDeferredRevenue, Type: domain.AccountTypeLiability, Debits: 0, Credits: 10000},
	}, timeZero())
	if d := trialBalanceDiscrepancies(tb); len(d) != 0 {
		t.Fatalf("clean books produced %d discrepancies: %+v", len(d), d)
	}
}

// TestTrialBalanceDiscrepancies_Unbalanced flags a books imbalance.
func TestTrialBalanceDiscrepancies_Unbalanced(t *testing.T) {
	tb := finalizeTrialBalance(uuid.New(), []domain.TrialBalanceLine{
		{Code: domain.AccountCodeCash, Type: domain.AccountTypeAsset, Debits: 10000, Credits: 0},
		{Code: domain.AccountCodeRevenue, Type: domain.AccountTypeRevenue, Debits: 0, Credits: 7000},
	}, timeZero())
	d := trialBalanceDiscrepancies(tb)
	if len(d) != 1 || d[0].Type != DiscrepancyLedgerUnbalanced {
		t.Fatalf("want one ledger_unbalanced, got %+v", d)
	}
	if d[0].ExpectedAmount != 10000 || d[0].FoundAmount != 7000 {
		t.Errorf("unbalanced finding = D%d/C%d, want D10000/C7000", d[0].ExpectedAmount, d[0].FoundAmount)
	}
}

// TestTrialBalanceDiscrepancies_AbnormalAccount flags the Deferred net-debit
// class (the ENG-191 posting bug) even when the books happen to balance.
func TestTrialBalanceDiscrepancies_AbnormalAccount(t *testing.T) {
	// Balanced overall (debits 11800 == credits 11800) but Deferred carries a
	// net debit of 1800 -> wrong sign for a liability.
	tb := finalizeTrialBalance(uuid.New(), []domain.TrialBalanceLine{
		{Code: domain.AccountCodeCash, Type: domain.AccountTypeAsset, Debits: 0, Credits: 1800},
		{Code: domain.AccountCodeDeferredRevenue, Type: domain.AccountTypeLiability, Debits: 11800, Credits: 10000},
	}, timeZero())
	d := trialBalanceDiscrepancies(tb)
	var found *ReconciliationDiscrepancy
	for i := range d {
		if d[i].Type == DiscrepancyAbnormalBalance {
			found = &d[i]
		}
	}
	if found == nil {
		t.Fatalf("expected an abnormal_account_balance finding, got %+v", d)
	}
	if found.AccountCode != domain.AccountCodeDeferredRevenue || found.FoundAmount != -1800 {
		t.Errorf("abnormal finding = code %d balance %d, want code %d balance -1800",
			found.AccountCode, found.FoundAmount, domain.AccountCodeDeferredRevenue)
	}
}

// TestReconciliationRun_DeferredBelowScheduled flags the case where Deferred
// Revenue has been drained below the revenue still scheduled to be recognized —
// the class the downgrade-credit over-draw produced, which the abnormal-sign
// check misses because the account stays positive (masked by other
// subscriptions). The books balance and no account is wrong-sign, so ONLY the
// new invariant fires.
func TestReconciliationRun_DeferredBelowScheduled(t *testing.T) {
	repo := &mockReconciliationRepo{
		nonDraft: 1, paid: 1,
		// Balanced, non-abnormal: Cash debit 3000 == Deferred credit 3000. Deferred
		// carries a healthy +3000 credit balance.
		trialBalanceLines: []domain.TrialBalanceLine{
			{Code: domain.AccountCodeCash, Type: domain.AccountTypeAsset, Debits: 3000, Credits: 0},
			{Code: domain.AccountCodeDeferredRevenue, Type: domain.AccountTypeLiability, Debits: 0, Credits: 3000},
		},
		// ...but 5000 of revenue is still scheduled to be recognized. Deferred is
		// 2000 short of what its schedule holds.
		pendingEvents: 5000,
	}
	svc := NewReconciliationService(nil, nil)
	svc.repo = repo

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.TotalDiscrepancies != 1 {
		t.Fatalf("want exactly 1 discrepancy, got %d: %+v", report.TotalDiscrepancies, report.Discrepancies)
	}
	d := report.Discrepancies[0]
	if d.Type != DiscrepancyDeferredBelowScheduled {
		t.Fatalf("want %q, got %q", DiscrepancyDeferredBelowScheduled, d.Type)
	}
	if d.ExpectedAmount != 5000 || d.FoundAmount != 3000 {
		t.Errorf("finding = scheduled %d / deferred %d, want 5000 / 3000", d.ExpectedAmount, d.FoundAmount)
	}
	if d.AccountCode != domain.AccountCodeDeferredRevenue {
		t.Errorf("account code = %d, want %d", d.AccountCode, domain.AccountCodeDeferredRevenue)
	}
}

// TestReconciliationRun_DeferredCoversScheduled: when Deferred is at least the
// scheduled remainder (the healthy invariant), no finding fires — including the
// common case where Deferred EXCEEDS pending because a recorded invoice is not
// yet paid (funded Deferred, no schedule yet).
func TestReconciliationRun_DeferredCoversScheduled(t *testing.T) {
	repo := &mockReconciliationRepo{
		nonDraft: 2, paid: 1,
		trialBalanceLines: []domain.TrialBalanceLine{
			{Code: domain.AccountCodeCash, Type: domain.AccountTypeAsset, Debits: 8000, Credits: 0},
			{Code: domain.AccountCodeDeferredRevenue, Type: domain.AccountTypeLiability, Debits: 0, Credits: 8000},
		},
		pendingEvents: 5000, // <= 8000 Deferred (3000 is an unpaid recorded deferral)
	}
	svc := NewReconciliationService(nil, nil)
	svc.repo = repo

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, d := range report.Discrepancies {
		if d.Type == DiscrepancyDeferredBelowScheduled {
			t.Fatalf("Deferred (8000) covers scheduled (5000) but got a shortfall finding: %+v", d)
		}
	}
}

// TestReconciliationRun_SurfacesIntegrityFindings proves the integrity check is
// wired into Run and its findings are prepended (survive truncation).
func TestReconciliationRun_SurfacesIntegrityFindings(t *testing.T) {
	repo := &mockReconciliationRepo{
		nonDraft: 1, paid: 1,
		trialBalanceLines: []domain.TrialBalanceLine{
			{Code: domain.AccountCodeDeferredRevenue, Type: domain.AccountTypeLiability, Debits: 5000, Credits: 0},
		},
	}
	svc := NewReconciliationService(nil, nil)
	svc.repo = repo

	report, err := svc.Run(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The lone Deferred line (net debit, no offset) is unbalanced, abnormal, AND
	// below its scheduled recognition (0) — three integrity findings, all prepended
	// ahead of any billing drift.
	if report.TotalDiscrepancies < 2 || len(report.Discrepancies) < 2 {
		t.Fatalf("expected integrity discrepancies in report, got total=%d list=%d", report.TotalDiscrepancies, len(report.Discrepancies))
	}
	first := report.Discrepancies[0].Type
	if first != DiscrepancyLedgerUnbalanced && first != DiscrepancyAbnormalBalance && first != DiscrepancyDeferredBelowScheduled {
		t.Errorf("integrity finding not prepended: first = %q", first)
	}
}

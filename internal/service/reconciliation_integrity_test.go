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
	// The lone Deferred line (net debit, no offset) is both unbalanced and
	// abnormal -> two integrity findings, prepended ahead of any billing drift.
	if report.TotalDiscrepancies < 2 || len(report.Discrepancies) < 2 {
		t.Fatalf("expected integrity discrepancies in report, got total=%d list=%d", report.TotalDiscrepancies, len(report.Discrepancies))
	}
	first := report.Discrepancies[0].Type
	if first != DiscrepancyLedgerUnbalanced && first != DiscrepancyAbnormalBalance {
		t.Errorf("integrity finding not prepended: first = %q", first)
	}
}

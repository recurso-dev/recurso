package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestAccountTypeIsDebitNormal pins the normal balance side per account type:
// assets and expenses are debit-normal; liabilities, equity, and revenue are
// credit-normal. The trial balance's abnormal-sign detection depends on this.
func TestAccountTypeIsDebitNormal(t *testing.T) {
	cases := map[domain.AccountType]bool{
		domain.AccountTypeAsset:     true,
		domain.AccountTypeExpense:   true,
		domain.AccountTypeLiability: false,
		domain.AccountTypeRevenue:   false,
		domain.AccountTypeEquity:    false,
	}
	for typ, want := range cases {
		if got := typ.IsDebitNormal(); got != want {
			t.Errorf("%s.IsDebitNormal() = %v, want %v", typ, got, want)
		}
	}
}

// TestNormalBalance_SignByAccountType proves the balance is computed on the
// account's normal side — and that a credit-normal account carrying a net
// debit (the ENG-191 class: Deferred drained by more than was deferred) comes
// out NEGATIVE, which is what flags the bug.
func TestNormalBalance_SignByAccountType(t *testing.T) {
	// Deferred Revenue (liability, credit-normal) drained 11,800 against only
	// 10,000 ever deferred -> a net debit -> negative balance.
	if got := normalBalance(domain.AccountTypeLiability, 11800, 10000); got != -1800 {
		t.Fatalf("liability net-debit balance = %d, want -1800", got)
	}
	// Accounts Receivable (asset, debit-normal): debits over credits is healthy.
	if got := normalBalance(domain.AccountTypeAsset, 10000, 3000); got != 7000 {
		t.Fatalf("asset balance = %d, want 7000", got)
	}
}

// TestFinalizeTrialBalance_FlagsAbnormalAndDetectsImbalance verifies the report
// flags a wrong-sign account and reports the double-entry invariant honestly
// when the per-account totals don't tie out.
func TestFinalizeTrialBalance_FlagsAbnormalAndDetectsImbalance(t *testing.T) {
	tenantID := uuid.New()
	asOf := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	lines := []domain.TrialBalanceLine{
		{AccountID: uuid.New(), Code: domain.AccountCodeCash, Name: "Cash", Type: domain.AccountTypeAsset, Debits: 11800, Credits: 0},
		{AccountID: uuid.New(), Code: domain.AccountCodeDeferredRevenue, Name: "Deferred Revenue", Type: domain.AccountTypeLiability, Debits: 11800, Credits: 10000},
	}

	tb := finalizeTrialBalance(tenantID, lines, asOf)

	if tb.TotalDebits != 23600 || tb.TotalCredits != 10000 {
		t.Fatalf("totals = D%d/C%d, want D23600/C10000", tb.TotalDebits, tb.TotalCredits)
	}
	if tb.Balanced {
		t.Error("Balanced should be false when total debits != total credits")
	}
	if tb.AsOf != asOf {
		t.Errorf("AsOf = %v, want %v", tb.AsOf, asOf)
	}

	deferred := findLine(t, tb, domain.AccountCodeDeferredRevenue)
	if deferred.Balance != -1800 || !deferred.Abnormal {
		t.Errorf("Deferred Revenue balance=%d abnormal=%v, want -1800/true", deferred.Balance, deferred.Abnormal)
	}
	cash := findLine(t, tb, domain.AccountCodeCash)
	if cash.Balance != 11800 || cash.Abnormal {
		t.Errorf("Cash balance=%d abnormal=%v, want 11800/false", cash.Balance, cash.Abnormal)
	}
}

// TestFinalizeTrialBalance_BalancedHappyPath: a proper double entry (Cash debit
// against Deferred credit) ties out and flags nothing.
func TestFinalizeTrialBalance_BalancedHappyPath(t *testing.T) {
	lines := []domain.TrialBalanceLine{
		{AccountID: uuid.New(), Code: domain.AccountCodeCash, Name: "Cash", Type: domain.AccountTypeAsset, Debits: 10000, Credits: 0},
		{AccountID: uuid.New(), Code: domain.AccountCodeDeferredRevenue, Name: "Deferred Revenue", Type: domain.AccountTypeLiability, Debits: 0, Credits: 10000},
	}

	tb := finalizeTrialBalance(uuid.New(), lines, time.Now())

	if !tb.Balanced {
		t.Fatalf("expected Balanced=true, got totals D%d/C%d", tb.TotalDebits, tb.TotalCredits)
	}
	for _, l := range tb.Lines {
		if l.Abnormal {
			t.Errorf("account %d unexpectedly abnormal (balance=%d)", l.Code, l.Balance)
		}
	}
}

func findLine(t *testing.T, tb *domain.TrialBalance, code int) domain.TrialBalanceLine {
	t.Helper()
	for _, l := range tb.Lines {
		if l.Code == code {
			return l
		}
	}
	t.Fatalf("no trial-balance line for account code %d", code)
	return domain.TrialBalanceLine{}
}

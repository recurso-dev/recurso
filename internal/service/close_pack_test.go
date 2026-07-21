package service

import (
	"testing"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

func TestCloseBlockers_CleanPeriod(t *testing.T) {
	tb := &domain.TrialBalance{TotalDebits: 1000, TotalCredits: 1000, Balanced: true}
	recon := &ReconciliationReport{TotalDiscrepancies: 0}

	blockers := closeBlockers(tb, recon)
	if len(blockers) != 0 {
		t.Fatalf("clean period should have no blockers, got %v", blockers)
	}
	// Must be non-nil so JSON serializes [] not null.
	if blockers == nil {
		t.Fatal("blockers must be a non-nil empty slice")
	}
}

func TestCloseBlockers_UnbalancedTrialBalance(t *testing.T) {
	tb := &domain.TrialBalance{TotalDebits: 1000, TotalCredits: 900, Balanced: false}
	recon := &ReconciliationReport{TotalDiscrepancies: 0}

	blockers := closeBlockers(tb, recon)
	if len(blockers) != 1 {
		t.Fatalf("expected 1 blocker, got %d: %v", len(blockers), blockers)
	}
	if want := "trial balance out of balance: debits 1000 != credits 900"; blockers[0] != want {
		t.Fatalf("blocker text = %q, want %q", blockers[0], want)
	}
}

func TestCloseBlockers_ReconciliationDiscrepancies(t *testing.T) {
	tb := &domain.TrialBalance{Balanced: true}

	cases := []struct {
		n    int
		want string
	}{
		{1, "1 reconciliation discrepancy"},
		{3, "3 reconciliation discrepancies"},
	}
	for _, tc := range cases {
		blockers := closeBlockers(tb, &ReconciliationReport{TotalDiscrepancies: tc.n})
		if len(blockers) != 1 || blockers[0] != tc.want {
			t.Fatalf("n=%d: got %v, want [%q]", tc.n, blockers, tc.want)
		}
	}
}

func TestCloseBlockers_BothConditions(t *testing.T) {
	tb := &domain.TrialBalance{TotalDebits: 1000, TotalCredits: 900, Balanced: false}
	recon := &ReconciliationReport{TotalDiscrepancies: 2}

	blockers := closeBlockers(tb, recon)
	if len(blockers) != 2 {
		t.Fatalf("expected 2 blockers, got %d: %v", len(blockers), blockers)
	}
}

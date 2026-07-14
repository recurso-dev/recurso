package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// normalBalance returns an account's balance on its normal side: debits minus
// credits for debit-normal accounts (assets, expenses), credits minus debits
// for credit-normal accounts (liabilities, equity, revenue). A negative result
// means the account carries a balance on the wrong side.
func normalBalance(t domain.AccountType, debits, credits int64) int64 {
	if t.IsDebitNormal() {
		return debits - credits
	}
	return credits - debits
}

// finalizeTrialBalance fills each line's normal-side Balance and Abnormal flag,
// totals debits and credits, and asserts the double-entry invariant. Pure so it
// is unit-testable without a database.
func finalizeTrialBalance(tenantID uuid.UUID, lines []domain.TrialBalanceLine, asOf time.Time) *domain.TrialBalance {
	var totalDebits, totalCredits int64
	for i := range lines {
		l := &lines[i]
		l.Balance = normalBalance(l.Type, l.Debits, l.Credits)
		l.Abnormal = l.Balance < 0
		totalDebits += l.Debits
		totalCredits += l.Credits
	}
	return &domain.TrialBalance{
		TenantID:     tenantID,
		Lines:        lines,
		TotalDebits:  totalDebits,
		TotalCredits: totalCredits,
		Balanced:     totalDebits == totalCredits,
		AsOf:         asOf,
	}
}

// GetTrialBalance returns the tenant's trial balance: every account with its
// posted debit/credit totals, its normal-side balance, an abnormal-sign flag,
// and the overall debits==credits invariant. Read-only — the artifact a
// controller uses to prove the books balance and to catch posting bugs (an
// abnormal Deferred Revenue balance surfaces the ENG-191 class immediately).
func (s *LedgerService) GetTrialBalance(ctx context.Context, tenantID uuid.UUID) (*domain.TrialBalance, error) {
	lines, err := s.pgRepo.GetTrialBalanceLines(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return finalizeTrialBalance(tenantID, lines, time.Now()), nil
}

// GeneralLedger returns every posted transaction for a tenant, flattened with
// account codes and names, oldest first. Read-only — the GL export an auditor
// imports.
func (s *LedgerService) GeneralLedger(ctx context.Context, tenantID uuid.UUID) ([]domain.GeneralLedgerRow, error) {
	return s.pgRepo.GetGeneralLedgerRows(ctx, tenantID)
}

// deferredClosing derives the closing deferred balance from the period movement.
// Pure so it is unit-testable without a database.
func deferredClosing(opening, added, released int64) int64 {
	return opening + added - released
}

// GetDeferredRollforward returns the Deferred Revenue account's movement across
// [start, end): opening balance, deferrals added, amounts released, and the
// derived closing balance. Read-only — the deferred-revenue rollforward an
// auditor ties back to the trial balance.
func (s *LedgerService) GetDeferredRollforward(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*domain.DeferredRollforward, error) {
	opening, added, released, err := s.pgRepo.GetDeferredRollforward(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	return &domain.DeferredRollforward{
		TenantID:    tenantID,
		PeriodStart: start,
		PeriodEnd:   end,
		Opening:     opening,
		Added:       added,
		Released:    released,
		Closing:     deferredClosing(opening, added, released),
	}, nil
}

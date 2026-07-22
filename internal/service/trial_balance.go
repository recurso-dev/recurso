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
// GetTrialBalance returns the trial balance for a tenant, optionally scoped to a
// single legal entity (Multi-Entity Books). A nil entityID consolidates across
// every entity ledger (one line per account, tagged with its entity); a non-nil
// entityID filters to that entity's ledger.
func (s *LedgerService) GetTrialBalance(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) (*domain.TrialBalance, error) {
	lines, err := s.pgRepo.GetTrialBalanceLines(ctx, tenantID, s.entityLedgerFilter(ctx, tenantID, entityID))
	if err != nil {
		return nil, err
	}
	tb := finalizeTrialBalance(tenantID, lines, time.Now())
	tb.ReportingCurrency = s.ReportingCurrency(ctx, tenantID)
	return tb, nil
}

// GetConsolidatedTrialBalance rolls the per-entity accounts up by account code,
// summing debits and credits across every entity ledger into one line per code
// — the tenant-wide consolidated view (Multi-Entity Books Inc 4).
func (s *LedgerService) GetConsolidatedTrialBalance(ctx context.Context, tenantID uuid.UUID) (*domain.TrialBalance, error) {
	lines, err := s.pgRepo.GetTrialBalanceLines(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}
	byCode := map[int]*domain.TrialBalanceLine{}
	order := []int{}
	for _, l := range lines {
		agg, ok := byCode[l.Code]
		if !ok {
			// A code-consolidated line carries no single account/entity.
			agg = &domain.TrialBalanceLine{Code: l.Code, Name: l.Name, Type: l.Type}
			byCode[l.Code] = agg
			order = append(order, l.Code)
		}
		agg.Debits += l.Debits
		agg.Credits += l.Credits
	}
	consolidated := make([]domain.TrialBalanceLine, 0, len(order))
	for _, code := range order {
		consolidated = append(consolidated, *byCode[code])
	}
	tb := finalizeTrialBalance(tenantID, consolidated, time.Now())
	tb.ReportingCurrency = s.ReportingCurrency(ctx, tenantID)
	return tb, nil
}

// entityLedgerFilter maps an optional entity id to a ledger-id filter: nil →
// nil (all entities); otherwise the resolved entity's TigerBeetle ledger id.
func (s *LedgerService) entityLedgerFilter(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) *int {
	if entityID == nil {
		return nil
	}
	lid := int(s.resolveEntity(ctx, tenantID, entityID).LedgerID)
	return &lid
}

// GeneralLedger returns posted transactions for a tenant, flattened with account
// codes and names, oldest first, optionally scoped to one entity's ledger. Read-
// only — the GL export an auditor imports.
func (s *LedgerService) GeneralLedger(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) ([]domain.GeneralLedgerRow, error) {
	return s.pgRepo.GetGeneralLedgerRows(ctx, tenantID, s.entityLedgerFilter(ctx, tenantID, entityID))
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
		TenantID:          tenantID,
		PeriodStart:       start,
		PeriodEnd:         end,
		Opening:           opening,
		Added:             added,
		Released:          released,
		Closing:           deferredClosing(opening, added, released),
		ReportingCurrency: s.ReportingCurrency(ctx, tenantID),
	}, nil
}

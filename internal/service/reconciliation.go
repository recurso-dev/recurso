package service

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/tigerbeetle"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// MaxListedDiscrepancies caps how many discrepancies a report lists so a
// huge drift does not explode the response; TotalDiscrepancies still carries
// the full count.
const MaxListedDiscrepancies = 100

// MaxTBComparedRows is the memory guard for the TigerBeetle comparison pass:
// the diff holds every Postgres ledger transaction and every TigerBeetle
// transfer for the tenant in memory, so tenants above this row count are
// skipped with an explicit TBSkipReason instead of risking an OOM. Moving
// past this bound requires a streaming/batched comparison design.
const MaxTBComparedRows = 100_000

// Discrepancy type constants for ReconciliationDiscrepancy.Type.
const (
	DiscrepancyMissingInvoiceTx      = "missing_invoice_transaction"
	DiscrepancyInvoiceAmountMismatch = "invoice_amount_mismatch"
	DiscrepancyMissingPaymentTx      = "missing_payment_transaction"
	DiscrepancyPaymentAmountMismatch = "payment_amount_mismatch"
	DiscrepancyMissingCreditNoteTx   = "missing_credit_note_transaction"
	// Credit-application (account-credit drawdown) completeness: an invoice with
	// credit_applied > 0 must carry code-7 drawdown legs (DR Customer-Credit /
	// CR AR) summing to it. A dropped leg silently overstates AR and the
	// Customer-Credit liability with the books still balanced.
	DiscrepancyMissingCreditApplicationTx      = "missing_credit_application_transaction"
	DiscrepancyCreditApplicationAmountMismatch = "credit_application_amount_mismatch"
	// Write-off completeness: an invoice marked `uncollectible` must carry its
	// write-off legs (codes 22 deferred + 26 bad-debt + 23 tax, all CR A/R)
	// summing to its total. A dropped leg leaves A/R (and Deferred) overstated —
	// the reconciler otherwise misses it (A/R positive is normal-sign) and the
	// close-pack Deferred identity absorbs it into AwaitingPayment (R-010).
	DiscrepancyMissingWriteOffTx      = "missing_write_off_transaction"
	DiscrepancyWriteOffAmountMismatch = "write_off_amount_mismatch"
	// A taxed invoice's Output-Tax reclassification leg (code 6) is missing or the
	// wrong amount. Posted atomically with the AR leg, so the code-1 check can't
	// catch it — books balance while Revenue is gross and Tax Payable understated.
	DiscrepancyMissingTaxTx         = "missing_tax_transaction"
	DiscrepancyTaxAmountMismatch    = "tax_amount_mismatch"
	DiscrepancyOrphanedTransaction  = "orphaned_transaction"
	DiscrepancyMissingInTigerBeetle = "missing_in_tigerbeetle"
	DiscrepancyMissingInPostgres    = "missing_in_postgres"
	DiscrepancyTBAmountMismatch     = "tb_amount_mismatch"
	// Trial-balance integrity: the double-entry books must always balance and
	// no account may carry a wrong-sign balance (e.g. Deferred Revenue going
	// net-debit — the ENG-191 class). These make the trial balance a standing
	// tripwire, not just a report.
	DiscrepancyLedgerUnbalanced = "ledger_unbalanced"
	DiscrepancyAbnormalBalance  = "abnormal_account_balance"
	// Deferred Revenue must always be at least the revenue still scheduled to be
	// recognized (the sum of pending recognition events) — Deferred funds exactly
	// that future recognition, plus any recorded-but-unpaid invoice deferrals.
	// A Deferred balance BELOW the scheduled remainder means a posting drained
	// Deferred past what its schedule holds (e.g. a downgrade credit debiting the
	// full net when recognition had already run ahead). Unlike the abnormal-sign
	// check, this survives aggregation across subscriptions/entities: the over-draw
	// lowers Deferred without lowering the pending-event total, so the inequality
	// breaks even when other subscriptions' positive Deferred masks a single
	// account going net-debit.
	DiscrepancyDeferredBelowScheduled = "deferred_below_scheduled_revenue"
	// Accrual invariant "Revenue Recognized ≤ the amount to recognize": a
	// schedule whose recognized events sum to MORE than its total recognizable
	// amount has fabricated revenue on the P&L. Must never happen; surfaced per
	// offending schedule.
	DiscrepancyRecognizedExceedsInvoice = "recognized_exceeds_invoice"
	// Customer-Credit liability integrity: the Customer-Credit account balance
	// must equal the sum of outstanding spendable (adjustment-type) credit-note
	// balances. Every spendable credit — manual adjustment or downgrade proration
	// — credits Customer-Credit for its balance; every drawdown (application,
	// expiry, void) debits it and lowers the note's balance in lockstep. A gap
	// means a best-effort drawdown/reversal leg was dropped: the liability is
	// overstated while the books still balance and no sign goes abnormal — the
	// class of drift no count- or threshold-based check catches (R-001/008/009).
	DiscrepancyCustomerCreditMismatch = "customer_credit_liability_mismatch"
)

// ReconciliationRepository is the narrow, read-only view of the ledger store
// needed to reconcile billing records against ledger transactions.
type ReconciliationRepository interface {
	CountReconciliationScope(ctx context.Context, tenantID uuid.UUID) (nonDraft int, paid int, err error)
	GetInvoiceLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	GetPaymentLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	GetCreditNoteLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.CreditNoteLedgerMismatch, int, error)
	GetCreditApplicationLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	GetWriteOffLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	// GetTaxLedgerMismatches feeds the Output-Tax leg completeness check.
	GetTaxLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	GetOrphanLedgerTransactions(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.OrphanLedgerTransaction, int, error)
	// GetTrialBalanceLines feeds the double-entry integrity assertion.
	GetTrialBalanceLines(ctx context.Context, tenantID uuid.UUID, ledgerID *int) ([]domain.TrialBalanceLine, error)
	// SumPendingRecognitionEventsByEntity feeds the deferred-vs-scheduled
	// invariant PER ENTITY (Multi-Entity Books): a tenant-wide aggregate would
	// let one entity's Deferred excess mask another's shortfall (R-015). The
	// primary entity keys as uuid.Nil (schedules use the NULL⇒primary convention).
	SumPendingRecognitionEventsByEntity(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]int64, error)
	// GetPrimaryEntityID canonicalizes the primary entity's Deferred line (which
	// the trial balance resolves to the primary UUID) to the same uuid.Nil key.
	GetPrimaryEntityID(ctx context.Context, tenantID uuid.UUID) (uuid.UUID, error)
	// SumSpendableCreditNoteBalance feeds the Customer-Credit liability invariant:
	// the sum of outstanding balances of adjustment-type (spendable) credit notes.
	SumSpendableCreditNoteBalance(ctx context.Context, tenantID uuid.UUID) (int64, error)
	// SumWalletBalance feeds the same invariant: prepaid wallets post to the SAME
	// Customer-Credit account as credit notes, so their balances count too.
	SumWalletBalance(ctx context.Context, tenantID uuid.UUID) (int64, error)
	// GetRecognitionOverruns feeds the "recognized ≤ recognizable" invariant.
	GetRecognitionOverruns(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.RecognitionOverrun, int, error)

	// TigerBeetle comparison inputs (all read-only).
	GetAccountsByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.LedgerAccount, error)
	CountLedgerTransactionsByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
	GetLedgerTransactionSummaries(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.LedgerTransactionSummary, error)
}

// TBTransferReader is the narrow slice of the TigerBeetle adapter the
// reconciler needs. *tigerbeetle.LedgerClient satisfies it; tests substitute
// a fake.
type TBTransferReader interface {
	Connected() bool
	EnumerateAccountTransfers(ctx context.Context, accountID uuid.UUID, maxTransfers int) ([]tigerbeetle.TransferRecord, error)
}

var _ TBTransferReader = (*tigerbeetle.LedgerClient)(nil)

// ReconciliationDiscrepancy is a single disagreement between billing records
// and the Postgres ledger, or between the Postgres ledger and TigerBeetle.
type ReconciliationDiscrepancy struct {
	Type           string     `json:"type"`
	InvoiceID      *uuid.UUID `json:"invoice_id,omitempty"`
	TransactionID  *uuid.UUID `json:"transaction_id,omitempty"`
	ReferenceID    *uuid.UUID `json:"reference_id,omitempty"`
	AccountCode    int        `json:"account_code,omitempty"` // set for abnormal_account_balance
	ExpectedAmount int64      `json:"expected_amount"`
	FoundAmount    int64      `json:"found_amount"`
}

// ReconciliationReport is the on-demand result of a reconciliation run.
// It is computed, never persisted.
type ReconciliationReport struct {
	TenantID            uuid.UUID                   `json:"tenant_id"`
	StartedAt           time.Time                   `json:"started_at"`
	FinishedAt          time.Time                   `json:"finished_at"`
	InvoicesChecked     int                         `json:"invoices_checked"`
	PaidInvoicesChecked int                         `json:"paid_invoices_checked"`
	TotalDiscrepancies  int                         `json:"total_discrepancies"`
	Discrepancies       []ReconciliationDiscrepancy `json:"discrepancies"`
	Truncated           bool                        `json:"truncated"`
	TBCompared          bool                        `json:"tb_compared"`
	TBSkipReason        string                      `json:"tb_skip_reason,omitempty"`
	TBAccountsChecked   int                         `json:"tb_accounts_checked"`
	TBTransfersChecked  int                         `json:"tb_transfers_checked"`
}

// ReconciliationService answers "does the ledger agree with the billing
// records?" for a tenant. It only reads; fixing drift is a human decision.
type ReconciliationService struct {
	repo      ReconciliationRepository
	tb        TBTransferReader
	maxListed int
	runStore  reconciliationRunStore
}

// reconciliationRunStore persists/reads the run-history summary — the audit
// trail of recorded reconciliations. Narrow so it stays optional (nil-safe).
type reconciliationRunStore interface {
	Create(ctx context.Context, run *domain.ReconciliationRun) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.ReconciliationRun, error)
}

// NewReconciliationService creates a reconciliation service. tbClient may be
// nil when TigerBeetle is not connected; the nil pointer is checked here so
// the stored interface is nil too (never a typed-nil interface).
func NewReconciliationService(repo ReconciliationRepository, tbClient *tigerbeetle.LedgerClient) *ReconciliationService {
	s := &ReconciliationService{repo: repo, maxListed: MaxListedDiscrepancies}
	if tbClient != nil {
		s.tb = tbClient
	}
	return s
}

// SetRunStore wires run-history persistence. Without it, RecordRun/ListRuns are
// no-ops (an empty history) — reconciliation itself is unaffected.
func (s *ReconciliationService) SetRunStore(store reconciliationRunStore) { s.runStore = store }

// RecordRun persists a summary of a completed report as an audit-trail entry.
// actorID may be uuid.Nil (system/unauthenticated) → stored as null. No-op when
// no run store is wired.
func (s *ReconciliationService) RecordRun(ctx context.Context, tenantID, actorID uuid.UUID, report *ReconciliationReport) error {
	if s.runStore == nil || report == nil {
		return nil
	}
	run := &domain.ReconciliationRun{
		TenantID:            tenantID,
		RunAt:               report.FinishedAt,
		InvoicesChecked:     report.InvoicesChecked,
		PaidInvoicesChecked: report.PaidInvoicesChecked,
		TotalDiscrepancies:  report.TotalDiscrepancies,
		TBCompared:          report.TBCompared,
		TBAccountsChecked:   report.TBAccountsChecked,
		TBTransfersChecked:  report.TBTransfersChecked,
	}
	if actorID != uuid.Nil {
		run.RunBy = &actorID
	}
	return s.runStore.Create(ctx, run)
}

// ListRuns returns the tenant's recorded reconciliation runs, newest first.
// Returns an empty slice when no run store is wired.
func (s *ReconciliationService) ListRuns(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.ReconciliationRun, error) {
	if s.runStore == nil {
		return []domain.ReconciliationRun{}, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.runStore.ListByTenant(ctx, tenantID, limit)
}

// Run reconciles a tenant's invoices against its Postgres ledger entries:
//   - every non-draft invoice must have Code-1 postings summing to total
//   - every paid invoice must have Code-3 postings summing to amount_paid
//   - every Code-1/3 posting must reference an existing invoice
//
// When a TigerBeetle client is wired in, a second pass diffs the tenant's
// Postgres ledger transactions (authoritative) against TigerBeetle transfers
// by transaction ID; see compareTigerBeetle. If that pass cannot run, the
// report says so explicitly via TBCompared=false and TBSkipReason instead of
// guessing — a TB failure never fails the whole report.
func (s *ReconciliationService) Run(ctx context.Context, tenantID uuid.UUID) (*ReconciliationReport, error) {
	report := &ReconciliationReport{
		TenantID:      tenantID,
		StartedAt:     time.Now().UTC(),
		Discrepancies: []ReconciliationDiscrepancy{},
	}

	nonDraft, paid, err := s.repo.CountReconciliationScope(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("reconciliation scope for tenant %s: %w", tenantID, err)
	}
	report.InvoicesChecked = nonDraft
	report.PaidInvoicesChecked = paid

	invoiceRows, invoiceTotal, err := s.repo.GetInvoiceLedgerMismatches(ctx, tenantID, s.maxListed)
	if err != nil {
		return nil, fmt.Errorf("invoice ledger mismatches for tenant %s: %w", tenantID, err)
	}
	for _, row := range invoiceRows {
		report.Discrepancies = append(report.Discrepancies,
			invoiceDiscrepancy(row, DiscrepancyMissingInvoiceTx, DiscrepancyInvoiceAmountMismatch))
	}

	paymentRows, paymentTotal, err := s.repo.GetPaymentLedgerMismatches(ctx, tenantID, s.maxListed)
	if err != nil {
		return nil, fmt.Errorf("payment ledger mismatches for tenant %s: %w", tenantID, err)
	}
	for _, row := range paymentRows {
		report.Discrepancies = append(report.Discrepancies,
			invoiceDiscrepancy(row, DiscrepancyMissingPaymentTx, DiscrepancyPaymentAmountMismatch))
	}

	creditNoteRows, creditNoteTotal, err := s.repo.GetCreditNoteLedgerMismatches(ctx, tenantID, s.maxListed)
	if err != nil {
		return nil, fmt.Errorf("credit-note ledger mismatches for tenant %s: %w", tenantID, err)
	}
	for _, row := range creditNoteRows {
		cnID := row.CreditNoteID
		report.Discrepancies = append(report.Discrepancies, ReconciliationDiscrepancy{
			Type:        DiscrepancyMissingCreditNoteTx,
			ReferenceID: &cnID,
		})
	}

	creditAppRows, creditAppTotal, err := s.repo.GetCreditApplicationLedgerMismatches(ctx, tenantID, s.maxListed)
	if err != nil {
		return nil, fmt.Errorf("credit-application ledger mismatches for tenant %s: %w", tenantID, err)
	}
	for _, row := range creditAppRows {
		report.Discrepancies = append(report.Discrepancies,
			invoiceDiscrepancy(row, DiscrepancyMissingCreditApplicationTx, DiscrepancyCreditApplicationAmountMismatch))
	}

	writeOffRows, writeOffTotal, err := s.repo.GetWriteOffLedgerMismatches(ctx, tenantID, s.maxListed)
	if err != nil {
		return nil, fmt.Errorf("write-off ledger mismatches for tenant %s: %w", tenantID, err)
	}
	for _, row := range writeOffRows {
		report.Discrepancies = append(report.Discrepancies,
			invoiceDiscrepancy(row, DiscrepancyMissingWriteOffTx, DiscrepancyWriteOffAmountMismatch))
	}

	taxRows, taxTotal, err := s.repo.GetTaxLedgerMismatches(ctx, tenantID, s.maxListed)
	if err != nil {
		return nil, fmt.Errorf("tax ledger mismatches for tenant %s: %w", tenantID, err)
	}
	for _, row := range taxRows {
		report.Discrepancies = append(report.Discrepancies,
			invoiceDiscrepancy(row, DiscrepancyMissingTaxTx, DiscrepancyTaxAmountMismatch))
	}

	orphanRows, orphanTotal, err := s.repo.GetOrphanLedgerTransactions(ctx, tenantID, s.maxListed)
	if err != nil {
		return nil, fmt.Errorf("orphan ledger transactions for tenant %s: %w", tenantID, err)
	}
	for _, row := range orphanRows {
		txID := row.TransactionID
		refID := row.ReferenceID
		report.Discrepancies = append(report.Discrepancies, ReconciliationDiscrepancy{
			Type:          DiscrepancyOrphanedTransaction,
			TransactionID: &txID,
			ReferenceID:   &refID,
			FoundAmount:   row.Amount,
		})
	}

	// Double-entry integrity: the books must balance and no account may carry a
	// wrong-sign balance. Prepended so these critical findings always survive
	// the maxListed truncation even when billing drift is large.
	tbLines, err := s.repo.GetTrialBalanceLines(ctx, tenantID, nil) // all entities — the tenant total must balance
	if err != nil {
		return nil, fmt.Errorf("trial balance for tenant %s: %w", tenantID, err)
	}
	tb := finalizeTrialBalance(tenantID, tbLines, time.Now().UTC())
	integrity := trialBalanceDiscrepancies(tb)
	if len(integrity) > 0 {
		report.Discrepancies = append(integrity, report.Discrepancies...)
	}

	// Deferred-vs-scheduled invariant, PER ENTITY: each entity's Deferred Revenue
	// balance must be at least the revenue still scheduled to recognize on THAT
	// entity. A tenant-wide aggregate would let one entity's Deferred excess mask
	// another entity's shortfall under Multi-Entity Books (R-015). Prepended (like
	// the other integrity findings) so it survives the maxListed truncation.
	//
	// Both sides key the primary entity as uuid.Nil: the pending map uses the
	// schedule's entity_id (NULL⇒primary → uuid.Nil), and the primary Deferred
	// trial-balance line — which the trial-balance query resolves to the primary
	// entity's UUID — is canonicalized back to uuid.Nil here. Without this
	// normalization the primary entity would key differently on each side and
	// false-positive on every tenant with primary-entity subscriptions.
	deferredShort := 0
	pendingByEntity, err := s.repo.SumPendingRecognitionEventsByEntity(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("pending recognition events for tenant %s: %w", tenantID, err)
	}
	primaryEntityID, err := s.repo.GetPrimaryEntityID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("primary entity for tenant %s: %w", tenantID, err)
	}
	deferredByEntity := make(map[uuid.UUID]int64)
	for _, l := range tb.Lines {
		if l.Code == domain.AccountCodeDeferredRevenue {
			deferredByEntity[canonicalEntityKey(l.EntityID, primaryEntityID)] += l.Balance
		}
	}
	// Compare per entity over the union of keys, deterministically ordered so the
	// report is stable across runs (map iteration order is randomized).
	entityKeys := make(map[uuid.UUID]struct{})
	for k := range pendingByEntity {
		entityKeys[k] = struct{}{}
	}
	for k := range deferredByEntity {
		entityKeys[k] = struct{}{}
	}
	ordered := make([]uuid.UUID, 0, len(entityKeys))
	for k := range entityKeys {
		ordered = append(ordered, k)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })
	var deferredDiscrepancies []ReconciliationDiscrepancy
	for _, k := range ordered {
		pending, deferred := pendingByEntity[k], deferredByEntity[k]
		if pending > deferred {
			deferredDiscrepancies = append(deferredDiscrepancies, ReconciliationDiscrepancy{
				Type:           DiscrepancyDeferredBelowScheduled,
				AccountCode:    domain.AccountCodeDeferredRevenue,
				ExpectedAmount: pending,  // this entity's Deferred must be at least this
				FoundAmount:    deferred, // what this entity's Deferred actually is
			})
		}
	}
	if len(deferredDiscrepancies) > 0 {
		report.Discrepancies = append(deferredDiscrepancies, report.Discrepancies...)
		deferredShort = len(deferredDiscrepancies)
	}

	// Customer-Credit liability invariant: the Customer-Credit balance must equal
	// the outstanding spendable balances that fund it — adjustment-type credit
	// notes AND prepaid wallets (wallets post to the SAME Customer-Credit account,
	// so they count too; omitting them false-positives on any tenant with a
	// wallet). A gap means a drawdown/reversal leg was dropped — liability
	// overstated, books still balanced. Prepended so it survives truncation.
	creditMismatch := 0
	var customerCreditBalance int64
	for _, l := range tb.Lines {
		if l.Code == domain.AccountCodeCustomerCredit {
			customerCreditBalance += l.Balance
		}
	}
	spendableCredit, err := s.repo.SumSpendableCreditNoteBalance(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("spendable credit-note balance for tenant %s: %w", tenantID, err)
	}
	walletBalance, err := s.repo.SumWalletBalance(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("wallet balance for tenant %s: %w", tenantID, err)
	}
	expectedCustomerCredit := spendableCredit + walletBalance
	if customerCreditBalance != expectedCustomerCredit {
		report.Discrepancies = append([]ReconciliationDiscrepancy{{
			Type:           DiscrepancyCustomerCreditMismatch,
			AccountCode:    domain.AccountCodeCustomerCredit,
			ExpectedAmount: expectedCustomerCredit, // credit notes + wallets owed
			FoundAmount:    customerCreditBalance,  // the ledger says this
		}}, report.Discrepancies...)
		creditMismatch = 1
	}

	// Recognized-vs-recognizable invariant: no schedule may recognize MORE than
	// its total recognizable amount. A violation fabricates revenue on the P&L.
	// Prepended (like the other integrity findings) so it survives truncation.
	overruns, overrunTotal, err := s.repo.GetRecognitionOverruns(ctx, tenantID, s.maxListed)
	if err != nil {
		return nil, fmt.Errorf("recognition overruns for tenant %s: %w", tenantID, err)
	}
	overrunDiscrepancies := make([]ReconciliationDiscrepancy, 0, len(overruns))
	for _, o := range overruns {
		schedID, invID := o.ScheduleID, o.InvoiceID
		overrunDiscrepancies = append(overrunDiscrepancies, ReconciliationDiscrepancy{
			Type:           DiscrepancyRecognizedExceedsInvoice,
			InvoiceID:      &invID,
			ReferenceID:    &schedID,
			ExpectedAmount: o.TotalAmount, // the most that may be recognized
			FoundAmount:    o.Recognized,  // what was actually recognized (greater)
		})
	}
	if len(overrunDiscrepancies) > 0 {
		report.Discrepancies = append(overrunDiscrepancies, report.Discrepancies...)
	}

	report.TotalDiscrepancies = invoiceTotal + paymentTotal + creditNoteTotal + creditAppTotal + writeOffTotal + taxTotal + orphanTotal + len(integrity) + deferredShort + creditMismatch + overrunTotal

	if s.tb == nil {
		report.TBSkipReason = "TigerBeetle not connected; nothing to compare"
	} else {
		s.compareTigerBeetle(ctx, tenantID, report)
	}

	if len(report.Discrepancies) > s.maxListed {
		report.Discrepancies = report.Discrepancies[:s.maxListed]
	}
	report.Truncated = report.TotalDiscrepancies > len(report.Discrepancies)

	report.FinishedAt = time.Now().UTC()
	return report, nil
}

// canonicalEntityKey maps a Deferred trial-balance line's entity to the same key
// space the pending-by-entity map uses: the primary entity (whether the line
// carries a nil pointer or the resolved primary UUID) collapses to uuid.Nil,
// matching the NULL⇒primary schedules; every other entity keeps its own UUID.
func canonicalEntityKey(lineEntity *uuid.UUID, primaryEntityID uuid.UUID) uuid.UUID {
	if lineEntity == nil {
		return uuid.Nil
	}
	if *lineEntity == primaryEntityID {
		return uuid.Nil
	}
	return *lineEntity
}

// trialBalanceDiscrepancies asserts double-entry integrity over a computed
// trial balance: one ledger_unbalanced finding if total debits != total
// credits, and one abnormal_account_balance finding per account carrying a
// wrong-sign balance. Pure so it is unit-testable without a database.
func trialBalanceDiscrepancies(tb *domain.TrialBalance) []ReconciliationDiscrepancy {
	var out []ReconciliationDiscrepancy
	if !tb.Balanced {
		out = append(out, ReconciliationDiscrepancy{
			Type:           DiscrepancyLedgerUnbalanced,
			ExpectedAmount: tb.TotalDebits,
			FoundAmount:    tb.TotalCredits,
		})
	}
	for _, l := range tb.Lines {
		if l.Abnormal {
			out = append(out, ReconciliationDiscrepancy{
				Type:        DiscrepancyAbnormalBalance,
				AccountCode: l.Code,
				FoundAmount: l.Balance,
			})
		}
	}
	return out
}

// compareTigerBeetle cross-checks the tenant's Postgres ledger transactions
// (authoritative) against TigerBeetle transfers (the dual-write replica).
// Both writes share the same 128-bit ID — LedgerService mints one txID per
// posting and the TB adapter converts UUID<->Uint128 losslessly — so rows are
// matched by transaction ID and then compared by amount:
//
//   - missing_in_tigerbeetle: PG transaction with no TB transfer of that ID
//   - missing_in_postgres:    TB transfer with no PG transaction of that ID
//   - tb_amount_mismatch:     both exist but the amounts differ
//
// TB transfers are enumerated per tenant ledger account (the only exhaustive
// read path in tigerbeetle-go v0.15.x) and deduped by ID, since each transfer
// touches two accounts. The pass degrades honestly: any error, a disconnected
// client, or a tenant above the MaxTBComparedRows memory guard leaves
// TBCompared=false with the reason in TBSkipReason — never a failed report.
func (s *ReconciliationService) compareTigerBeetle(ctx context.Context, tenantID uuid.UUID, report *ReconciliationReport) {
	if !s.tb.Connected() {
		report.TBSkipReason = "TigerBeetle client is not connected"
		return
	}

	pgCount, err := s.repo.CountLedgerTransactionsByTenant(ctx, tenantID)
	if err != nil {
		report.TBSkipReason = fmt.Sprintf("counting tenant ledger transactions failed: %v", err)
		return
	}
	if pgCount > MaxTBComparedRows {
		report.TBSkipReason = fmt.Sprintf("tenant has %d ledger transactions, above the %d-row in-memory comparison guard; TigerBeetle comparison skipped", pgCount, MaxTBComparedRows)
		return
	}

	// Reconciliation must see EVERY account — never clamp this read (0 = no LIMIT).
	accounts, err := s.repo.GetAccountsByTenant(ctx, tenantID, 0, 0)
	if err != nil {
		report.TBSkipReason = fmt.Sprintf("listing tenant ledger accounts failed: %v", err)
		return
	}
	pgTxs, err := s.repo.GetLedgerTransactionSummaries(ctx, tenantID, MaxTBComparedRows)
	if err != nil {
		report.TBSkipReason = fmt.Sprintf("loading tenant ledger transactions failed: %v", err)
		return
	}

	// Each transfer touches two accounts, so a transfer between two tenant
	// accounts is seen from both sides; dedupe by ID.
	tbByID := make(map[uuid.UUID]tigerbeetle.TransferRecord)
	for _, acc := range accounts {
		transfers, err := s.tb.EnumerateAccountTransfers(ctx, acc.ID, MaxTBComparedRows)
		if err != nil {
			report.TBSkipReason = fmt.Sprintf("enumerating TigerBeetle transfers for account %s failed: %v", acc.ID, err)
			return
		}
		for _, tr := range transfers {
			tbByID[tr.ID] = tr
		}
		if len(tbByID) > MaxTBComparedRows {
			report.TBSkipReason = fmt.Sprintf("TigerBeetle holds more than %d transfers for the tenant, above the in-memory comparison guard; TigerBeetle comparison skipped", MaxTBComparedRows)
			return
		}
	}

	pgByID := make(map[uuid.UUID]int64, len(pgTxs))
	for _, tx := range pgTxs {
		pgByID[tx.TransactionID] = tx.Amount
	}

	var discrepancies []ReconciliationDiscrepancy

	// PG rows in query order (ORDER BY id) keep the output deterministic.
	for _, tx := range pgTxs {
		txID := tx.TransactionID
		tr, ok := tbByID[txID]
		if !ok {
			id := txID
			discrepancies = append(discrepancies, ReconciliationDiscrepancy{
				Type:           DiscrepancyMissingInTigerBeetle,
				TransactionID:  &id,
				ExpectedAmount: tx.Amount,
			})
			continue
		}
		if tx.Amount < 0 || uint64(tx.Amount) != tr.Amount {
			id := txID
			discrepancies = append(discrepancies, ReconciliationDiscrepancy{
				Type:           DiscrepancyTBAmountMismatch,
				TransactionID:  &id,
				ExpectedAmount: tx.Amount,
				FoundAmount:    clampToInt64(tr.Amount),
			})
		}
	}

	var onlyTB []tigerbeetle.TransferRecord
	for id, tr := range tbByID {
		if _, ok := pgByID[id]; !ok {
			onlyTB = append(onlyTB, tr)
		}
	}
	sort.Slice(onlyTB, func(i, j int) bool {
		return bytes.Compare(onlyTB[i].ID[:], onlyTB[j].ID[:]) < 0
	})
	for _, tr := range onlyTB {
		id := tr.ID
		d := ReconciliationDiscrepancy{
			Type:          DiscrepancyMissingInPostgres,
			TransactionID: &id,
			FoundAmount:   clampToInt64(tr.Amount),
		}
		if tr.ReferenceID != uuid.Nil {
			ref := tr.ReferenceID
			d.ReferenceID = &ref
		}
		discrepancies = append(discrepancies, d)
	}

	report.TBCompared = true
	report.TBAccountsChecked = len(accounts)
	report.TBTransfersChecked = len(tbByID)
	report.TotalDiscrepancies += len(discrepancies)

	// Respect the listing cap; TotalDiscrepancies already carries the full
	// count and Run recomputes Truncated afterwards.
	if room := s.maxListed - len(report.Discrepancies); room > 0 {
		if len(discrepancies) > room {
			discrepancies = discrepancies[:room]
		}
		report.Discrepancies = append(report.Discrepancies, discrepancies...)
	}
}

// clampToInt64 converts a TigerBeetle amount to the report's int64 field,
// saturating at MaxInt64 instead of wrapping negative.
func clampToInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}

// invoiceDiscrepancy classifies a mismatch row as "missing" (no postings at
// all) or "amount mismatch" (postings exist but sum incorrectly).
func invoiceDiscrepancy(row db.InvoiceLedgerMismatch, missingType, mismatchType string) ReconciliationDiscrepancy {
	invoiceID := row.InvoiceID
	d := ReconciliationDiscrepancy{
		InvoiceID:      &invoiceID,
		ExpectedAmount: row.Expected,
		FoundAmount:    row.Found,
	}
	if row.TxCount == 0 {
		d.Type = missingType
	} else {
		d.Type = mismatchType
	}
	return d
}

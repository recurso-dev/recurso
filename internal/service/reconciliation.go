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
	DiscrepancyOrphanedTransaction   = "orphaned_transaction"
	DiscrepancyMissingInTigerBeetle  = "missing_in_tigerbeetle"
	DiscrepancyMissingInPostgres     = "missing_in_postgres"
	DiscrepancyTBAmountMismatch      = "tb_amount_mismatch"
	// Trial-balance integrity: the double-entry books must always balance and
	// no account may carry a wrong-sign balance (e.g. Deferred Revenue going
	// net-debit — the ENG-191 class). These make the trial balance a standing
	// tripwire, not just a report.
	DiscrepancyLedgerUnbalanced = "ledger_unbalanced"
	DiscrepancyAbnormalBalance  = "abnormal_account_balance"
)

// ReconciliationRepository is the narrow, read-only view of the ledger store
// needed to reconcile billing records against ledger transactions.
type ReconciliationRepository interface {
	CountReconciliationScope(ctx context.Context, tenantID uuid.UUID) (nonDraft int, paid int, err error)
	GetInvoiceLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	GetPaymentLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	GetOrphanLedgerTransactions(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.OrphanLedgerTransaction, int, error)
	// GetTrialBalanceLines feeds the double-entry integrity assertion.
	GetTrialBalanceLines(ctx context.Context, tenantID uuid.UUID, ledgerID *int) ([]domain.TrialBalanceLine, error)

	// TigerBeetle comparison inputs (all read-only).
	GetAccountsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.LedgerAccount, error)
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
	integrity := trialBalanceDiscrepancies(finalizeTrialBalance(tenantID, tbLines, time.Now().UTC()))
	if len(integrity) > 0 {
		report.Discrepancies = append(integrity, report.Discrepancies...)
	}

	report.TotalDiscrepancies = invoiceTotal + paymentTotal + orphanTotal + len(integrity)

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

	accounts, err := s.repo.GetAccountsByTenant(ctx, tenantID)
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

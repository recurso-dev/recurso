package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/adapter/tigerbeetle"
)

// MaxListedDiscrepancies caps how many discrepancies a report lists so a
// huge drift does not explode the response; TotalDiscrepancies still carries
// the full count.
const MaxListedDiscrepancies = 100

// Discrepancy type constants for ReconciliationDiscrepancy.Type.
const (
	DiscrepancyMissingInvoiceTx      = "missing_invoice_transaction"
	DiscrepancyInvoiceAmountMismatch = "invoice_amount_mismatch"
	DiscrepancyMissingPaymentTx      = "missing_payment_transaction"
	DiscrepancyPaymentAmountMismatch = "payment_amount_mismatch"
	DiscrepancyOrphanedTransaction   = "orphaned_transaction"
)

// ReconciliationRepository is the narrow, read-only view of the ledger store
// needed to reconcile billing records against ledger transactions.
type ReconciliationRepository interface {
	CountReconciliationScope(ctx context.Context, tenantID uuid.UUID) (nonDraft int, paid int, err error)
	GetInvoiceLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	GetPaymentLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.InvoiceLedgerMismatch, int, error)
	GetOrphanLedgerTransactions(ctx context.Context, tenantID uuid.UUID, limit int) ([]db.OrphanLedgerTransaction, int, error)
}

// ReconciliationDiscrepancy is a single disagreement between billing records
// and the Postgres ledger.
type ReconciliationDiscrepancy struct {
	Type           string     `json:"type"`
	InvoiceID      *uuid.UUID `json:"invoice_id,omitempty"`
	TransactionID  *uuid.UUID `json:"transaction_id,omitempty"`
	ReferenceID    *uuid.UUID `json:"reference_id,omitempty"`
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
}

// ReconciliationService answers "does the ledger agree with the billing
// records?" for a tenant. It only reads; fixing drift is a human decision.
type ReconciliationService struct {
	repo      ReconciliationRepository
	tbClient  *tigerbeetle.LedgerClient
	maxListed int
}

// NewReconciliationService creates a reconciliation service. tbClient may be
// nil when TigerBeetle is not connected.
func NewReconciliationService(repo ReconciliationRepository, tbClient *tigerbeetle.LedgerClient) *ReconciliationService {
	return &ReconciliationService{repo: repo, tbClient: tbClient, maxListed: MaxListedDiscrepancies}
}

// Run reconciles a tenant's invoices against its Postgres ledger entries:
//   - every non-draft invoice must have Code-1 postings summing to total
//   - every paid invoice must have Code-3 postings summing to amount_paid
//   - every Code-1/3 posting must reference an existing invoice
//
// TigerBeetle is NOT compared: the TB adapter only exposes per-account
// transfer reads capped at 50 entries (GetAccountTransfers), so transfers
// cannot be enumerated by reference. The report says so explicitly via
// TBCompared/TBSkipReason instead of guessing.
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

	report.TotalDiscrepancies = invoiceTotal + paymentTotal + orphanTotal
	if len(report.Discrepancies) > s.maxListed {
		report.Discrepancies = report.Discrepancies[:s.maxListed]
	}
	report.Truncated = report.TotalDiscrepancies > len(report.Discrepancies)

	report.TBCompared = false
	if s.tbClient == nil {
		report.TBSkipReason = "TigerBeetle not connected; nothing to compare"
	} else {
		report.TBSkipReason = "TigerBeetle adapter exposes only per-account transfer reads capped at 50 entries (GetAccountTransfers); transfers cannot be enumerated by reference, so TB comparison was skipped"
	}

	report.FinishedAt = time.Now().UTC()
	return report, nil
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

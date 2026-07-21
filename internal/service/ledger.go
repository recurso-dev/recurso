package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/tigerbeetle"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// LedgerService orchestrates financial movements.
// Dual-write: always writes to PG (pgRepo), and to TigerBeetle (tbClient) if connected.
type LedgerService struct {
	tbClient *tigerbeetle.LedgerClient
	pgRepo   port.LedgerRepository
}

func NewLedgerService(tbClient *tigerbeetle.LedgerClient, pgRepo port.LedgerRepository) *LedgerService {
	return &LedgerService{tbClient: tbClient, pgRepo: pgRepo}
}

// ledgerAmount converts a money amount to the ledger's unsigned representation.
// Negative amounts (e.g. a credit posted through the wrong path) must never
// wrap into huge uint64 postings.
func ledgerAmount(amount int64) (uint64, error) {
	if amount < 0 {
		return 0, fmt.Errorf("ledger amount must be non-negative, got %d", amount)
	}
	return uint64(amount), nil
}

// SetupTenantAccounts creates the standard chart of accounts for a new tenant.
func (s *LedgerService) SetupTenantAccounts(ctx context.Context, tenantID uuid.UUID) error {
	accounts := domain.TenantChartOfAccounts(tenantID)
	for _, acc := range accounts {
		// Write to PG
		if s.pgRepo != nil {
			if err := s.pgRepo.CreateAccount(ctx, acc); err != nil {
				return err
			}
		}
		// Write to TB
		if s.tbClient != nil {
			acc.UserData128 = domain.UUIDToUint128(tenantID)
			if err := s.tbClient.CreateAccounts(ctx, []*domain.LedgerAccount{acc}); err != nil {
				slog.Warn("TB CreateAccounts failed (non-fatal)", "error", err)
			}
		}
	}
	return nil
}

// CreateCustomerAccounts creates the necessary sub-ledgers for a customer (AR).
func (s *LedgerService) CreateCustomerAccounts(ctx context.Context, customerID uuid.UUID) error {
	account := &domain.LedgerAccount{
		ID:          customerID,
		Name:        "Accounts Receivable",
		Type:        domain.AccountTypeAsset,
		Code:        domain.AccountCodeAR,
		LedgerID:    1,
		UserData128: domain.UUIDToUint128(customerID),
	}

	if s.pgRepo != nil {
		if err := s.pgRepo.CreateAccount(ctx, account); err != nil {
			slog.Warn("PG CreateAccount failed", "error", err)
		}
	}

	if s.tbClient != nil {
		return s.tbClient.CreateAccounts(ctx, []*domain.LedgerAccount{account})
	}
	return nil
}

// ensureCustomerAR guarantees the customer's AR sub-ledger account exists
// before a posting references it. CreateAccount is ON CONFLICT DO NOTHING,
// so this is an idempotent no-op after the first call — and it self-heals
// customers created before AR provisioning was wired into every path.
func (s *LedgerService) ensureCustomerAR(ctx context.Context, tenantID, customerID uuid.UUID) {
	if s.pgRepo == nil {
		return
	}
	_ = s.pgRepo.CreateAccount(ctx, &domain.LedgerAccount{
		ID:       customerID,
		TenantID: tenantID,
		Name:     "Accounts Receivable",
		Type:     domain.AccountTypeAsset,
		Code:     domain.AccountCodeAR,
		LedgerID: 1,
	})
}

// getOrCreateTenantAccount resolves a tenant's account by code, creating it
// when absent. The old behavior fell back to hardcoded placeholder UUIDs,
// which can never satisfy the ledger_transactions FK — every posting for a
// tenant without a provisioned chart of accounts failed. Self-heals tenants
// registered before chart provisioning existed.
func (s *LedgerService) getOrCreateTenantAccount(ctx context.Context, tenantID uuid.UUID, code int, name string, accType domain.AccountType) (uuid.UUID, error) {
	if s.pgRepo == nil {
		return uuid.Nil, fmt.Errorf("ledger repository not configured")
	}
	if acc, err := s.pgRepo.GetAccountByTenantAndCode(ctx, tenantID, code); err == nil && acc != nil {
		return acc.ID, nil
	}
	acc := &domain.LedgerAccount{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     name,
		Type:     accType,
		Code:     code,
		LedgerID: 1,
	}
	if err := s.pgRepo.CreateAccount(ctx, acc); err != nil {
		return uuid.Nil, fmt.Errorf("failed to provision %s account for tenant %s: %w", name, tenantID, err)
	}
	return acc.ID, nil
}

// RecordInvoice posts the invoice amount to the ledger.
// Debit: Customer AR (Asset)
// Credit: Revenue
func (s *LedgerService) RecordInvoice(ctx context.Context, invoice *domain.Invoice) error {
	// Reject invalid amounts up front (the per-leg guards below would otherwise
	// silently skip a negative total). tax must be within [0, total].
	if invoice.Total < 0 || invoice.TaxAmount < 0 || invoice.TaxAmount > invoice.Total {
		return fmt.Errorf("invoice %s: invalid amounts (total=%d tax=%d)", invoice.ID, invoice.Total, invoice.TaxAmount)
	}
	s.ensureCustomerAR(ctx, invoice.TenantID, invoice.CustomerID)

	// A subscription invoice bills for a service delivered over the period, so
	// its revenue is DEFERRED (a liability) and recognized month-by-month by the
	// rev-rec scheduler (which drains Deferred → Recognized). A one-off invoice
	// is earned immediately, so it credits Revenue directly. Crediting Revenue
	// for subscriptions is what double-booked revenue against Recognized (ENG-140).
	var revenueAccountID uuid.UUID
	var err error
	if invoice.SubscriptionID != nil {
		revenueAccountID, err = s.getOrCreateTenantAccount(ctx, invoice.TenantID, domain.AccountCodeDeferredRevenue, "Deferred Revenue", domain.AccountTypeLiability)
	} else {
		revenueAccountID, err = s.getOrCreateTenantAccount(ctx, invoice.TenantID, domain.AccountCodeRevenue, "Revenue", domain.AccountTypeRevenue)
	}
	if err != nil {
		return fmt.Errorf("ledger write failed for invoice %s: %w", invoice.ID, err)
	}

	total, err := ledgerAmount(invoice.Total)
	if err != nil {
		return fmt.Errorf("invoice %s: %w", invoice.ID, err)
	}

	// Code-1: the whole invoice posts AR → Revenue/Deferred at the gross total.
	// The reconciler expects exactly one Code-1 per invoice summing to the total,
	// and uq_ledger_tx_reference_code enforces one row per (reference_id, code) —
	// so GST is handled as a separate reclassification below, not a 2nd Code-1.
	transfers := []*domain.LedgerTransaction{{
		ID: uuid.New(), DebitAccountID: invoice.CustomerID, CreditAccountID: revenueAccountID,
		Amount: total, LedgerID: 1, Code: 1, ReferenceID: invoice.ID,
		Description: "Invoice " + invoice.InvoiceNumber, Timestamp: time.Now(),
	}}

	// Reclassify the collected GST out of revenue into Tax Payable — it's a
	// liability owed to the government, not revenue (ENG-159). Debit
	// Revenue/Deferred, credit Tax Payable, under a distinct code. Net effect:
	// Revenue = taxable value, Tax Payable = GST, AR = gross; trial balance holds.
	if invoice.TaxAmount > 0 {
		taxAccountID, terr := s.getOrCreateTenantAccount(ctx, invoice.TenantID, domain.AccountCodeTaxPayable, "Tax Payable", domain.AccountTypeLiability)
		if terr != nil {
			return fmt.Errorf("ledger write failed for invoice %s: %w", invoice.ID, terr)
		}
		taxAmt, aerr := ledgerAmount(invoice.TaxAmount)
		if aerr != nil {
			return fmt.Errorf("invoice %s: %w", invoice.ID, aerr)
		}
		transfers = append(transfers, &domain.LedgerTransaction{
			ID: uuid.New(), DebitAccountID: revenueAccountID, CreditAccountID: taxAccountID,
			Amount: taxAmt, LedgerID: 1, Code: domain.LedgerCodeOutputTax, ReferenceID: invoice.ID,
			Description: "GST on invoice " + invoice.InvoiceNumber, Timestamp: time.Now(),
		})
	}

	// Post both legs (AR→Revenue and, for GST, Revenue→Tax-Payable) atomically:
	// a failure after the first leg would otherwise commit gross revenue with no
	// tax reclassification. A failed write means the invoice has no ledger entry,
	// so surface it rather than losing it in a log line.
	if s.pgRepo != nil {
		if err := s.pgRepo.CreateTransactions(ctx, transfers); err != nil {
			slog.Error("PG CreateTransactions failed", "error", err)
			return fmt.Errorf("ledger write failed for invoice %s: %w", invoice.ID, err)
		}
	}

	// Write to TB if connected
	if s.tbClient != nil {
		return s.tbClient.CreateTransfers(ctx, transfers)
	}
	return nil
}

// RecordPayment posts a payment to the ledger when an invoice is marked paid,
// with no prior non-cash settlement. See RecordPaymentWithSettled.
func (s *LedgerService) RecordPayment(ctx context.Context, invoice *domain.Invoice) error {
	return s.RecordPaymentWithSettled(ctx, invoice, 0)
}

// RecordPaymentWithSettled posts a payment to the ledger, booking the cash leg
// for the NET cash actually collected: Total − CreditApplied − TDS − alreadySettled.
//
// alreadySettled is the portion already relieved from AR by a non-cash channel
// that is NOT CreditApplied — today that is a prepaid wallet drain applied at
// invoice generation. That drain already credited AR (DR Customer-Credit / CR AR)
// and the money was booked as Cash back at wallet top-up (DR Cash / CR
// Customer-Credit); counting it again in this cash leg would double-book it as
// Cash and over-credit (drive negative) the customer's AR.
//
// Debit: Cash (Asset) — the net cash actually received
// Debit: TDS Receivable (Asset) — any portion the customer withheld at source
// Credit: Customer AR (Asset) — reduced by every leg
func (s *LedgerService) RecordPaymentWithSettled(ctx context.Context, invoice *domain.Invoice, alreadySettled int64) error {
	// Reject invalid amounts up front (a negative total is a caller bug, not a
	// zero-cash settlement) before the collected-amount short-circuit below.
	if invoice.Total < 0 || invoice.CreditApplied < 0 || invoice.TDSAmount < 0 || alreadySettled < 0 {
		return fmt.Errorf("invoice %s: invalid amounts (total=%d credit_applied=%d tds=%d already_settled=%d)",
			invoice.ID, invoice.Total, invoice.CreditApplied, invoice.TDSAmount, alreadySettled)
	}

	s.ensureCustomerAR(ctx, invoice.TenantID, invoice.CustomerID)
	var transfers []*domain.LedgerTransaction

	// TDS leg: the customer withheld this portion and remits it to the
	// government against the seller's PAN. It settles AR but is recovered
	// against income tax, not from the customer — so it lands in TDS
	// Receivable, never in Cash (docs/spec_india_decisive.md P1).
	if invoice.TDSAmount > 0 {
		tdsAmount, err := ledgerAmount(invoice.TDSAmount)
		if err != nil {
			return fmt.Errorf("invoice %s: %w", invoice.ID, err)
		}
		tdsAccountID, err := s.getOrCreateTenantAccount(ctx, invoice.TenantID, domain.AccountCodeTDSReceivable, "TDS Receivable", domain.AccountTypeAsset)
		if err != nil {
			return fmt.Errorf("ledger write failed for TDS on invoice %s: %w", invoice.ID, err)
		}
		transfers = append(transfers, &domain.LedgerTransaction{
			ID:              uuid.New(),
			DebitAccountID:  tdsAccountID,       // TDS Receivable
			CreditAccountID: invoice.CustomerID, // AR
			Amount:          tdsAmount,
			LedgerID:        1,
			Code:            domain.LedgerCodeTDSReceivable,
			ReferenceID:     invoice.ID,
			Description:     "TDS deducted at source on " + invoice.InvoiceNumber,
			Timestamp:       time.Now(),
		})
	}

	// Post the CASH actually collected, not the gross Total. Account credit
	// was already relieved from AR by the credit-application posting (ENG-185),
	// and the TDS portion never arrives as cash — posting either here would
	// over-credit AR (drive it negative) and overstate Cash.
	collected := invoice.Total - invoice.CreditApplied - invoice.TDSAmount - alreadySettled
	if collected > 0 {
		amount, err := ledgerAmount(collected)
		if err != nil {
			return fmt.Errorf("invoice %s: %w", invoice.ID, err)
		}
		cashAccountID, err := s.getOrCreateTenantAccount(ctx, invoice.TenantID, domain.AccountCodeCash, "Cash", domain.AccountTypeAsset)
		if err != nil {
			return fmt.Errorf("ledger write failed for payment on invoice %s: %w", invoice.ID, err)
		}
		transfers = append(transfers, &domain.LedgerTransaction{
			ID:              uuid.New(),
			DebitAccountID:  cashAccountID,      // Cash
			CreditAccountID: invoice.CustomerID, // AR
			Amount:          amount,
			LedgerID:        1,
			Code:            3, // Payment
			ReferenceID:     invoice.ID,
			Description:     "Payment for " + invoice.InvoiceNumber,
			Timestamp:       time.Now(),
		})
	}

	if len(transfers) == 0 {
		// Fully covered by account credit — no legs to post.
		return nil
	}

	// Always write to PG; surface failures so callers can retry/reconcile.
	if s.pgRepo != nil {
		for _, transfer := range transfers {
			if err := s.pgRepo.CreateTransaction(ctx, transfer); err != nil {
				slog.Error("PG CreateTransaction failed for payment", "error", err)
				return fmt.Errorf("ledger write failed for payment on invoice %s: %w", invoice.ID, err)
			}
		}
	}

	// Write to TB if connected
	if s.tbClient != nil {
		return s.tbClient.CreateTransfers(ctx, transfers)
	}
	return nil
}

// RecordRefund posts a refund to the ledger.
// Debit: Refunds (Expense)
// Credit: Cash (Asset)
func (s *LedgerService) RecordRefund(ctx context.Context, tenantID uuid.UUID, creditNoteID uuid.UUID, amount int64, description string) error {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return fmt.Errorf("refund %s: %w", creditNoteID, err)
	}
	txID := uuid.New()

	refundsAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeRefunds, "Refunds", domain.AccountTypeExpense)
	if err != nil {
		return fmt.Errorf("ledger write failed for refund on credit note %s: %w", creditNoteID, err)
	}
	cashAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCash, "Cash", domain.AccountTypeAsset)
	if err != nil {
		return fmt.Errorf("ledger write failed for refund on credit note %s: %w", creditNoteID, err)
	}

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  refundsAccountID,
		CreditAccountID: cashAccountID,
		Amount:          amt,
		LedgerID:        1,
		Code:            4, // Refund
		ReferenceID:     creditNoteID,
		Description:     description,
		Timestamp:       time.Now(),
	}

	// Always write to PG; surface failures so callers can retry/reconcile.
	if s.pgRepo != nil {
		if err := s.pgRepo.CreateTransaction(ctx, transfer); err != nil {
			slog.Error("PG CreateTransaction failed for refund", "error", err)
			return fmt.Errorf("ledger write failed for refund on credit note %s: %w", creditNoteID, err)
		}
	}

	// Write to TB if connected
	if s.tbClient != nil {
		return s.tbClient.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer})
	}
	return nil
}

// RecordDeferredRefundReversal reverses still-deferred revenue when a refund is
// issued against a subscription invoice mid-period (ENG-147).
//
//	Debit:  Deferred Revenue (Liability)  — drain the unearned portion
//	Credit: Refunds (Expense)             — offset the DR Refunds/CR Cash refund
//
// so refunding money the customer never earned is not booked as a P&L expense
// and Deferred is not left overstated. referenceID is the credit note id; the
// distinct code (5) keeps this off the code-4 cash-refund idempotency key, so
// the two postings for one credit note never dedupe against each other.
func (s *LedgerService) RecordDeferredRefundReversal(ctx context.Context, tenantID uuid.UUID, creditNoteID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("deferred reversal %s: %w", creditNoteID, err)
	}
	txID := uuid.New()

	deferredAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeDeferredRevenue, "Deferred Revenue", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	refundsAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeRefunds, "Refunds", domain.AccountTypeExpense)
	if err != nil {
		return uuid.Nil, err
	}

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  deferredAccountID,
		CreditAccountID: refundsAccountID,
		Amount:          amt,
		LedgerID:        1,
		Code:            5, // Deferred-revenue reversal (distinct from code 4 cash refund)
		ReferenceID:     creditNoteID,
		Description:     description,
		Timestamp:       time.Now(),
	}

	if s.pgRepo != nil {
		if err := s.pgRepo.CreateTransaction(ctx, transfer); err != nil {
			slog.Error("PG CreateTransaction failed for deferred reversal", "error", err)
			return uuid.Nil, fmt.Errorf("ledger write failed for deferred reversal on credit note %s: %w", creditNoteID, err)
		}
	}
	if s.tbClient != nil {
		if err := s.tbClient.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer}); err != nil {
			return uuid.Nil, err
		}
	}
	return txID, nil
}

// RecordRefundTaxReversal reverses the GST portion of a refund out of Tax
// Payable (ENG-191b):
//
//	Debit:  Tax Payable (Liability) — we no longer owe the government this GST,
//	        because it was returned to the customer with the refund
//	Credit: Refunds (Expense)       — the tax portion is not our expense; it
//	        offsets the gross DR Refunds booked by RecordRefund
//
// This mirrors the invoice-time DR Revenue/Deferred → CR Tax Payable posting in
// reverse. taxAmount is the tax slice of the refund (proportional to the
// invoice's tax rate — see refundTaxPortion). referenceID is the credit note id;
// the distinct code keeps it idempotent and off the code-4/5 keys.
func (s *LedgerService) RecordRefundTaxReversal(ctx context.Context, tenantID uuid.UUID, creditNoteID uuid.UUID, taxAmount int64, description string) (uuid.UUID, error) {
	amt, err := ledgerAmount(taxAmount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("refund tax reversal %s: %w", creditNoteID, err)
	}
	txID := uuid.New()

	taxAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeTaxPayable, "Tax Payable", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	refundsAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeRefunds, "Refunds", domain.AccountTypeExpense)
	if err != nil {
		return uuid.Nil, err
	}

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  taxAccountID,
		CreditAccountID: refundsAccountID,
		Amount:          amt,
		LedgerID:        1,
		Code:            domain.LedgerCodeRefundTaxReversal,
		ReferenceID:     creditNoteID,
		Description:     description,
		Timestamp:       time.Now(),
	}

	if s.pgRepo != nil {
		if err := s.pgRepo.CreateTransaction(ctx, transfer); err != nil {
			slog.Error("PG CreateTransaction failed for refund tax reversal", "error", err)
			return uuid.Nil, fmt.Errorf("ledger write failed for refund tax reversal on credit note %s: %w", creditNoteID, err)
		}
	}
	if s.tbClient != nil {
		if err := s.tbClient.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer}); err != nil {
			return uuid.Nil, err
		}
	}
	return txID, nil
}

// RecordDowngradeTaxReversal reverses the GST portion of a mid-period downgrade
// credit out of Tax Payable into the customer's account credit (ENG-191c):
//
//	Debit:  Tax Payable (Liability)    — the output GST on the reduced supply is
//	        no longer owed to the government
//	Credit: Customer Credit (Liability) — that tax is credited to the customer
//	        alongside the net (RecordDowngradeCredit posts the net portion), so
//	        the total account credit is the gross the customer originally paid
//
// Pairs with RecordDowngradeCredit (which posts DR Deferred / CR Customer Credit
// for the NET). Splitting the credit this way keeps Deferred draining only the
// net it holds (post-ENG-191) instead of going negative by the tax.
func (s *LedgerService) RecordDowngradeTaxReversal(ctx context.Context, tenantID uuid.UUID, creditNoteID uuid.UUID, taxAmount int64, description string) (uuid.UUID, error) {
	amt, err := ledgerAmount(taxAmount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("downgrade tax reversal %s: %w", creditNoteID, err)
	}
	taxAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeTaxPayable, "Tax Payable", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	creditAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCustomerCredit, "Customer Credit", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	return s.postTransfer(ctx, taxAccountID, creditAccountID, amt, domain.LedgerCodeDowngradeTaxReversal, creditNoteID, description)
}

// RecordDowngradeCredit books the deferred revenue freed by a mid-period plan
// downgrade as an account-credit liability (ENG-154).
//
//	Debit:  Deferred Revenue (Liability) — the over-deferred portion is no
//	        longer revenue we will earn from this subscription
//	Credit: Customer Credit (Liability)  — it is now credit we owe the customer
//
// referenceID is the downgrade credit note id; code 6 keeps it idempotent and
// attributable per (reference_id, code).
func (s *LedgerService) RecordDowngradeCredit(ctx context.Context, tenantID uuid.UUID, creditNoteID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("downgrade credit %s: %w", creditNoteID, err)
	}
	deferredAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeDeferredRevenue, "Deferred Revenue", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	creditAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCustomerCredit, "Customer Credit", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	return s.postTransfer(ctx, deferredAccountID, creditAccountID, amt, 6, creditNoteID, description)
}

// RecordAdjustmentCreditIssued books a manually-issued adjustment credit note as
// an account-credit liability (ENG-154).
//
//	Debit:  Credits & Adjustments (Expense) — the cost of the credit given
//	Credit: Customer Credit (Liability)     — the credit we now owe the customer
//
// This gives the later application (DR Customer-Credit / CR AR) an origin to draw
// down, keeping the ledger balanced regardless of where the credit came from.
// Downgrade credits are booked separately (DR Deferred) and don't pass here.
func (s *LedgerService) RecordAdjustmentCreditIssued(ctx context.Context, tenantID uuid.UUID, creditNoteID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("adjustment credit %s: %w", creditNoteID, err)
	}
	expenseAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCreditsIssued, "Credits & Adjustments", domain.AccountTypeExpense)
	if err != nil {
		return uuid.Nil, err
	}
	creditAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCustomerCredit, "Customer Credit", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	return s.postTransfer(ctx, expenseAccountID, creditAccountID, amt, 8, creditNoteID, description)
}

// RecordCreditApplication books the settlement of an invoice by account credit
// when an adjustment credit note is applied at billing time (ENG-153/154).
//
//	Debit:  Customer Credit (Liability) — draw down the credit we owed
//	Credit: Customer AR (Asset)         — the customer owes that much less
//
// referenceID is the invoice the credit settled; code 7 posts once per invoice.
func (s *LedgerService) RecordCreditApplication(ctx context.Context, tenantID, customerID uuid.UUID, referenceID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("credit application %s: %w", referenceID, err)
	}
	s.ensureCustomerAR(ctx, tenantID, customerID)
	creditAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCustomerCredit, "Customer Credit", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	// AR is a per-customer sub-ledger whose account id is the customer id.
	return s.postTransfer(ctx, creditAccountID, customerID, amt, 7, referenceID, description)
}

// RecordWalletTopUp books money received into a prepaid wallet (B1):
//
//	Debit:  Cash (Asset)                — money in
//	Credit: Customer Credit (Liability) — stored value owed to the customer
//
// referenceID is the wallet transaction. Promotional (unpaid) top-ups must
// use RecordAdjustmentCreditIssued instead — no cash moved.
func (s *LedgerService) RecordWalletTopUp(ctx context.Context, tenantID uuid.UUID, walletTxID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("wallet top-up %s: %w", walletTxID, err)
	}
	cashAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCash, "Cash", domain.AccountTypeAsset)
	if err != nil {
		return uuid.Nil, err
	}
	creditAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCustomerCredit, "Customer Credit", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	return s.postTransfer(ctx, cashAccountID, creditAccountID, amt, domain.LedgerCodeWalletTopUp, walletTxID, description)
}

// RecordWalletDrain books wallet balance settling an invoice (B1):
//
//	Debit:  Customer Credit (Liability) — stored value consumed
//	Credit: Customer AR (Asset)         — the customer owes that much less
//
// referenceID is the settled invoice.
func (s *LedgerService) RecordWalletDrain(ctx context.Context, tenantID, customerID, invoiceID uuid.UUID, amount int64, description string) (uuid.UUID, error) {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("wallet drain %s: %w", invoiceID, err)
	}
	s.ensureCustomerAR(ctx, tenantID, customerID)
	creditAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeCustomerCredit, "Customer Credit", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	// AR is a per-customer sub-ledger whose account id is the customer id.
	return s.postTransfer(ctx, creditAccountID, customerID, amt, domain.LedgerCodeWalletDrain, invoiceID, description)
}

// postTransfer writes a single ledger transfer (PG always; TigerBeetle when
// connected), surfacing PG failures for retry/reconciliation.
func (s *LedgerService) postTransfer(ctx context.Context, debitAccountID, creditAccountID uuid.UUID, amount uint64, code uint16, referenceID uuid.UUID, description string) (uuid.UUID, error) {
	txID := uuid.New()
	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  debitAccountID,
		CreditAccountID: creditAccountID,
		Amount:          amount,
		LedgerID:        1,
		Code:            code,
		ReferenceID:     referenceID,
		Description:     description,
		Timestamp:       time.Now(),
	}
	if s.pgRepo != nil {
		if err := s.pgRepo.CreateTransaction(ctx, transfer); err != nil {
			slog.Error("PG CreateTransaction failed", "code", code, "reference_id", referenceID, "error", err)
			return uuid.Nil, fmt.Errorf("ledger write failed (code %d, ref %s): %w", code, referenceID, err)
		}
	}
	if s.tbClient != nil {
		if err := s.tbClient.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer}); err != nil {
			return uuid.Nil, err
		}
	}
	return txID, nil
}

// ListAccounts returns all ledger accounts for a tenant.
func (s *LedgerService) ListAccounts(ctx context.Context, tenantID uuid.UUID) ([]*domain.LedgerAccount, error) {
	if s.pgRepo != nil {
		return s.pgRepo.GetAccountsByTenant(ctx, tenantID)
	}
	return nil, nil
}

// RecordRecognition moves funds from Deferred Revenue (Liability) to Recognized
// Revenue (Income) for a single recognition event. referenceID is the event's
// id, which makes the posting attributable in reconciliation and idempotent (the
// ENG-142 unique index on (reference_id, code) means a replayed event never
// double-posts).
func (s *LedgerService) RecordRecognition(ctx context.Context, tenantID uuid.UUID, amount int64, referenceID uuid.UUID) (uuid.UUID, error) {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return uuid.Nil, err
	}
	txID := uuid.New()

	deferredAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeDeferredRevenue, "Deferred Revenue", domain.AccountTypeLiability)
	if err != nil {
		return uuid.Nil, err
	}
	recognizedAccountID, err := s.getOrCreateTenantAccount(ctx, tenantID, domain.AccountCodeRecognizedRevenue, "Recognized Revenue", domain.AccountTypeRevenue)
	if err != nil {
		return uuid.Nil, err
	}

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  deferredAccountID,
		CreditAccountID: recognizedAccountID,
		Amount:          amt,
		LedgerID:        1,
		Code:            2, // Revenue Recognition
		ReferenceID:     referenceID,
		Description:     "Revenue recognition",
		Timestamp:       time.Now(),
	}

	// Always write to PG; surface failures so callers can retry/reconcile.
	if s.pgRepo != nil {
		if err := s.pgRepo.CreateTransaction(ctx, transfer); err != nil {
			slog.Error("PG CreateTransaction failed", "error", err)
			return uuid.Nil, fmt.Errorf("ledger write failed for revenue recognition: %w", err)
		}
	}

	// Write to TB if connected
	if s.tbClient != nil {
		if err := s.tbClient.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer}); err != nil {
			return uuid.Nil, err
		}
	}
	return txID, nil
}

// GetEntries fetches ledger entries (transfers) for a given account.
// Prefers PG as source of truth; falls back to TB if PG is unavailable.
func (s *LedgerService) GetEntries(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) ([]*domain.LedgerTransaction, error) {
	// Try PG first
	if s.pgRepo != nil {
		entries, err := s.pgRepo.GetTransactionsByAccount(ctx, tenantID, accountID)
		if err == nil {
			return entries, nil
		}
		slog.Warn("PG GetTransactionsByAccount failed, trying TB", "error", err)
	}

	// Fallback to TB
	if s.tbClient != nil {
		transfers, err := s.tbClient.GetAccountTransfers(ctx, accountID)
		if err != nil {
			return nil, err
		}

		var entries []*domain.LedgerTransaction
		for _, tx := range transfers {
			bi := tx.Amount.BigInt()
			entries = append(entries, &domain.LedgerTransaction{
				ID:              uuid.MustParse(tx.ID.String()),
				DebitAccountID:  uuid.MustParse(tx.DebitAccountID.String()),
				CreditAccountID: uuid.MustParse(tx.CreditAccountID.String()),
				Amount:          (&bi).Uint64(),
				LedgerID:        tx.Ledger,
				Code:            tx.Code,
			})
		}
		return entries, nil
	}

	return nil, nil
}

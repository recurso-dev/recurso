package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/tigerbeetle"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
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

// RecordInvoice posts the invoice amount to the ledger.
// Debit: Customer AR (Asset)
// Credit: Revenue
func (s *LedgerService) RecordInvoice(ctx context.Context, invoice *domain.Invoice) error {
	amount, err := ledgerAmount(invoice.Total)
	if err != nil {
		return fmt.Errorf("invoice %s: %w", invoice.ID, err)
	}
	txID := uuid.New()

	// Look up revenue account dynamically
	var revenueAccountID uuid.UUID
	if s.pgRepo != nil {
		revenueAcct, err := s.pgRepo.GetAccountByTenantAndCode(ctx, invoice.TenantID, domain.AccountCodeRevenue)
		if err == nil && revenueAcct != nil {
			revenueAccountID = revenueAcct.ID
		}
	}
	if revenueAccountID == uuid.Nil {
		revenueAccountID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  invoice.CustomerID, // AR
		CreditAccountID: revenueAccountID,
		Amount:          amount,
		LedgerID:        1,
		Code:            1, // Invoice
		ReferenceID:     invoice.ID,
		Description:     "Invoice " + invoice.InvoiceNumber,
		Timestamp:       time.Now(),
	}

	// Always write to PG; a failed write means the invoice has no ledger
	// entry, so surface it to the caller rather than losing it in a log line.
	if s.pgRepo != nil {
		if err := s.pgRepo.CreateTransaction(ctx, transfer); err != nil {
			slog.Error("PG CreateTransaction failed", "error", err)
			return fmt.Errorf("ledger write failed for invoice %s: %w", invoice.ID, err)
		}
	}

	// Write to TB if connected
	if s.tbClient != nil {
		return s.tbClient.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer})
	}
	return nil
}

// RecordPayment posts a payment to the ledger when an invoice is marked paid.
// Debit: Cash (Asset)
// Credit: Customer AR (Asset) — reduces the receivable
func (s *LedgerService) RecordPayment(ctx context.Context, invoice *domain.Invoice) error {
	amount, err := ledgerAmount(invoice.Total)
	if err != nil {
		return fmt.Errorf("invoice %s: %w", invoice.ID, err)
	}
	txID := uuid.New()

	// Look up cash account dynamically
	var cashAccountID uuid.UUID
	if s.pgRepo != nil {
		cashAcct, err := s.pgRepo.GetAccountByTenantAndCode(ctx, invoice.TenantID, domain.AccountCodeCash)
		if err == nil && cashAcct != nil {
			cashAccountID = cashAcct.ID
		}
	}
	if cashAccountID == uuid.Nil {
		cashAccountID = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	}

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  cashAccountID,      // Cash
		CreditAccountID: invoice.CustomerID, // AR
		Amount:          amount,
		LedgerID:        1,
		Code:            3, // Payment
		ReferenceID:     invoice.ID,
		Description:     "Payment for " + invoice.InvoiceNumber,
		Timestamp:       time.Now(),
	}

	// Always write to PG; surface failures so callers can retry/reconcile.
	if s.pgRepo != nil {
		if err := s.pgRepo.CreateTransaction(ctx, transfer); err != nil {
			slog.Error("PG CreateTransaction failed for payment", "error", err)
			return fmt.Errorf("ledger write failed for payment on invoice %s: %w", invoice.ID, err)
		}
	}

	// Write to TB if connected
	if s.tbClient != nil {
		return s.tbClient.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer})
	}
	return nil
}

// ListAccounts returns all ledger accounts for a tenant.
func (s *LedgerService) ListAccounts(ctx context.Context, tenantID uuid.UUID) ([]*domain.LedgerAccount, error) {
	if s.pgRepo != nil {
		return s.pgRepo.GetAccountsByTenant(ctx, tenantID)
	}
	return nil, nil
}

// RecordRecognition moves funds from Deferred Revenue to Recognized Revenue.
// Debit: Deferred Revenue (Liability)
// Credit: Recognized Revenue (Income)
func (s *LedgerService) RecordRecognition(ctx context.Context, tenantID uuid.UUID, amount int64) (uuid.UUID, error) {
	amt, err := ledgerAmount(amount)
	if err != nil {
		return uuid.Nil, err
	}
	txID := uuid.New()

	var deferredAccountID, recognizedAccountID uuid.UUID
	if s.pgRepo != nil {
		da, err := s.pgRepo.GetAccountByTenantAndCode(ctx, tenantID, domain.AccountCodeDeferredRevenue)
		if err == nil && da != nil {
			deferredAccountID = da.ID
		}
		ra, err := s.pgRepo.GetAccountByTenantAndCode(ctx, tenantID, domain.AccountCodeRecognizedRevenue)
		if err == nil && ra != nil {
			recognizedAccountID = ra.ID
		}
	}
	if deferredAccountID == uuid.Nil {
		deferredAccountID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	}
	if recognizedAccountID == uuid.Nil {
		recognizedAccountID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	}

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  deferredAccountID,
		CreditAccountID: recognizedAccountID,
		Amount:          amt,
		LedgerID:        1,
		Code:            2, // Revenue Recognition
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

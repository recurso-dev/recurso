package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/tigerbeetle"
	"github.com/recur-so/recurso/internal/core/domain"
)

// LedgerService orchestrates financial movements.
type LedgerService struct {
	client *tigerbeetle.LedgerClient
}

func NewLedgerService(client *tigerbeetle.LedgerClient) *LedgerService {
	return &LedgerService{client: client}
}

// CreateCustomerAccounts creates the necessary sub-ledgers for a customer (AR).
func (s *LedgerService) CreateCustomerAccounts(ctx context.Context, customerID uuid.UUID) error {
	accounts := []*domain.LedgerAccount{
		{
			ID:            customerID, // Simple mapping: Customer ID = AR Account ID
			Name:          "Accounts Receivable",
			Type:          domain.AccountTypeAsset,
			Code:          1000,
			LedgerID:      1,
			UserData128:   domain.UUIDToUint128(customerID), // Helper needed
			CreditsPosted: 0,
			DebitsPosted:  0,
		},
	}
	return s.client.CreateAccounts(ctx, accounts)
}

// RecordInvoice posts the invoice amount to the ledger.
// Debit: Customer AR (Asset)
// Credit: Revenue (Equity/Revenue)
func (s *LedgerService) RecordInvoice(ctx context.Context, invoice *domain.Invoice) error {
	txID := uuid.New()
	
	// For simplicity, assumed Revenue Account is fixed ID 1 (created at bootstrap)
	// In reality, each Plan might have a Revenue Account.
	revenueAccountID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  invoice.CustomerID, // AR
		CreditAccountID: revenueAccountID,   // Revenue
		Amount:          uint64(invoice.Total),
		LedgerID:        1,
		Code:            1, // Invoice
		Timestamp:       time.Now(),
	}

	return s.client.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer})
}

// RecordRecognition moves funds from Deferred Revenue to Recognized Revenue.
// Debit: Deferred Revenue (Liability)
// Credit: Recognized Revenue (Income)
func (s *LedgerService) RecordRecognition(ctx context.Context, tenantID uuid.UUID, amount int64) (uuid.UUID, error) {
	txID := uuid.New()

	// In production, these account IDs would be fetched from Tenant settings.
	// For MVP, using fixed IDs.
	deferredAccountID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	recognizedAccountID := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	transfer := &domain.LedgerTransaction{
		ID:              txID,
		DebitAccountID:  deferredAccountID,
		CreditAccountID: recognizedAccountID,
		Amount:          uint64(amount),
		LedgerID:        1,
		Code:            2, // Revenue Recognition
		Timestamp:       time.Now(),
	}

	if err := s.client.CreateTransfers(ctx, []*domain.LedgerTransaction{transfer}); err != nil {
		return uuid.Nil, err
	}
	return txID, nil
}

// GetEntries fetches ledger entries (transfers) for a given account.
func (s *LedgerService) GetEntries(ctx context.Context, accountID uuid.UUID) ([]*domain.LedgerTransaction, error) {
	transfers, err := s.client.GetAccountTransfers(ctx, accountID)
	if err != nil {
		return nil, err
	}

	var entries []*domain.LedgerTransaction
	for _, tx := range transfers {
		bi := tx.Amount.BigInt()
		entries = append(entries, &domain.LedgerTransaction{
			ID:              uuid.MustParse(tx.ID.String()), // This might need better UUID conversion from Uint128
			DebitAccountID:  uuid.MustParse(tx.DebitAccountID.String()),
			CreditAccountID: uuid.MustParse(tx.CreditAccountID.String()),
			Amount:          (&bi).Uint64(),
			LedgerID:        tx.Ledger,
			Code:            tx.Code,
			// Timestamp conversion from uint64 nanoseconds? TB uses nanoseconds from epoch??
			// For MVP, we might just use 'now' or decode properly if library supports it.
			// Actually best to leave timestamp empty if format conversion is hard, UI can fallback.
		})
	}
	return entries, nil
}

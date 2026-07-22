package port

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ErrWalletAlreadyClosed is returned when closing a wallet that is already closed.
var ErrWalletAlreadyClosed = errors.New("wallet already closed")

// WalletCloseResult is the outcome of closing a wallet: the amounts settled
// (paid residue refunded to the customer, promotional residue forfeited) and the
// ids of the closing transactions the caller references from the ledger legs.
type WalletCloseResult struct {
	Refunded    int64
	Forfeited   int64
	RefundTxID  uuid.UUID
	ForfeitTxID uuid.UUID
}

// WalletRepository persists prepaid wallets (Lago-parity B1). Drain and
// TopUp are transactional: the movement row, the residue bookkeeping, and
// the denormalized balance always change together.
type WalletRepository interface {
	Create(ctx context.Context, w *domain.Wallet) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Wallet, error)
	// Close settles and closes a wallet atomically: it zeroes every open residue
	// (splitting into a refund of paid residue and a forfeit of promotional
	// residue), appends the closing transactions, and stamps closed_at. Returns
	// ErrWalletAlreadyClosed if it was already closed.
	Close(ctx context.Context, tenantID, walletID uuid.UUID, now time.Time) (WalletCloseResult, error)
	// GetByCustomerEntityAndCurrency returns (nil, nil) when the customer has
	// no wallet for that entity in the currency (Multi-Entity Books: wallets
	// are entity-scoped).
	GetByCustomerEntityAndCurrency(ctx context.Context, tenantID, customerID, entityID uuid.UUID, currency string) (*domain.Wallet, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Wallet, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.Wallet, error)
	UpdateAutoRecharge(ctx context.Context, tenantID, id uuid.UUID, threshold, amount *int64) error

	// TopUp appends a top_up transaction (with residue) and increases the
	// balance atomically.
	TopUp(ctx context.Context, tx *domain.WalletTransaction) error
	// Drain consumes up to maxAmount from the wallet's open, non-expired
	// residues (oldest expiry first), appends one drain transaction linked
	// to invoiceID, and decreases the balance — all atomically. Returns the
	// amount actually drained (0 when the wallet is empty or expired-only).
	Drain(ctx context.Context, tenantID, walletID uuid.UUID, maxAmount int64, invoiceID uuid.UUID, now time.Time) (int64, error)
	// ExpireOverdue writes off expired residues across all wallets,
	// appending expiry transactions and reducing balances. Returns the
	// number of wallets touched.
	ExpireOverdue(ctx context.Context, now time.Time) (int, error)
	// ListDueForRecharge returns wallets whose auto-recharge is configured
	// and whose balance sits below the threshold.
	ListDueForRecharge(ctx context.Context, limit int) ([]domain.Wallet, error)

	ListTransactions(ctx context.Context, tenantID, walletID uuid.UUID, limit int) ([]domain.WalletTransaction, error)
}

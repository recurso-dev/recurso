package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// LedgerRepository provides persistent storage for ledger accounts and transactions.
// Used as the primary store (PostgreSQL) with optional TigerBeetle dual-write.
type LedgerRepository interface {
	CreateAccount(ctx context.Context, account *domain.LedgerAccount) error
	GetAccountsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.LedgerAccount, error)
	GetAccountByTenantAndCode(ctx context.Context, tenantID uuid.UUID, code int) (*domain.LedgerAccount, error)
	CreateTransaction(ctx context.Context, tx *domain.LedgerTransaction) error
	// CreateTransactions posts several transfers atomically (one DB transaction),
	// so a multi-leg posting can't be left half-committed.
	CreateTransactions(ctx context.Context, txs []*domain.LedgerTransaction) error
	GetTransactionsByAccount(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) ([]*domain.LedgerTransaction, error)
	// GetTrialBalanceLines returns each of the tenant's accounts with its posted
	// debit and credit totals (minor units). Balance/Abnormal are computed by the
	// service, not the repository.
	GetTrialBalanceLines(ctx context.Context, tenantID uuid.UUID) ([]domain.TrialBalanceLine, error)
	// GetGeneralLedgerRows returns every posted transaction for a tenant,
	// flattened with account codes and names, for the read-only GL export.
	GetGeneralLedgerRows(ctx context.Context, tenantID uuid.UUID) ([]domain.GeneralLedgerRow, error)
	// GetDeferredRollforward returns the Deferred Revenue account's opening
	// balance, deferrals added, and amounts released over [start, end).
	GetDeferredRollforward(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (opening, added, released int64, err error)
}

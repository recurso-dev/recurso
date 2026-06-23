package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

// LedgerRepository provides persistent storage for ledger accounts and transactions.
// Used as the primary store (PostgreSQL) with optional TigerBeetle dual-write.
type LedgerRepository interface {
	CreateAccount(ctx context.Context, account *domain.LedgerAccount) error
	GetAccountsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.LedgerAccount, error)
	GetAccountByTenantAndCode(ctx context.Context, tenantID uuid.UUID, code int) (*domain.LedgerAccount, error)
	CreateTransaction(ctx context.Context, tx *domain.LedgerTransaction) error
	GetTransactionsByAccount(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) ([]*domain.LedgerTransaction, error)
}

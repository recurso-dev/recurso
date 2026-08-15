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
	GetAccountsByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.LedgerAccount, error)
	GetAccountByTenantAndCode(ctx context.Context, tenantID uuid.UUID, code int) (*domain.LedgerAccount, error)
	// GetAccountByEntityAndCode resolves a GL account scoped to a legal entity
	// (Multi-Entity Books). Nil when the entity has no such account yet.
	GetAccountByEntityAndCode(ctx context.Context, tenantID, entityID uuid.UUID, code int) (*domain.LedgerAccount, error)
	CreateTransaction(ctx context.Context, tx *domain.LedgerTransaction) error
	// CreateTransactions posts several transfers atomically (one DB transaction),
	// so a multi-leg posting can't be left half-committed.
	CreateTransactions(ctx context.Context, txs []*domain.LedgerTransaction) error
	// CountTransactionsByReferenceAndCode counts posted legs of one code for a
	// reference. The ACH settle/reverse cycle uses the code-19 count as its
	// occurrence counter (docs/design-ledger-occurrence.md).
	CountTransactionsByReferenceAndCode(ctx context.Context, referenceID uuid.UUID, code uint16) (int, error)
	// GetLatestTransactionByReferenceAndCode returns the highest-occurrence leg
	// of one code for a reference (nil when none). The ACH reversal inverts the
	// actual latest cash leg rather than recomputing its amount.
	GetLatestTransactionByReferenceAndCode(ctx context.Context, referenceID uuid.UUID, code uint16) (*domain.LedgerTransaction, error)
	GetTransactionsByAccount(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID, code uint16, limit, offset int) ([]*domain.LedgerTransaction, error)
	// GetTrialBalanceLines returns each of the tenant's accounts with its posted
	// debit and credit totals (minor units). Balance/Abnormal are computed by the
	// service, not the repository.
	// A nil ledgerID includes all of the tenant's entity ledgers (consolidated);
	// a non-nil ledgerID filters to that entity's ledger (Multi-Entity Books).
	GetTrialBalanceLines(ctx context.Context, tenantID uuid.UUID, ledgerID *int) ([]domain.TrialBalanceLine, error)
	// GetGeneralLedgerRows returns every posted transaction for a tenant,
	// flattened with account codes and names, for the read-only GL export.
	// Nil from/to mean unbounded; a non-nil pair filters to [from, to).
	GetGeneralLedgerRows(ctx context.Context, tenantID uuid.UUID, ledgerID *int, from, to *time.Time) ([]domain.GeneralLedgerRow, error)
	// GetJournalEntriesByReference returns every posting referencing one source
	// object (an invoice id), with account codes + names — the per-invoice
	// journal drill. Read-only.
	GetJournalEntriesByReference(ctx context.Context, tenantID, referenceID uuid.UUID) ([]domain.GeneralLedgerRow, error)
	// GetTransactionByID returns one posted transaction (a single journal entry)
	// by its ledger_transactions.id, flattened with account ids + codes + names,
	// tenant-scoped. Returns (nil, nil) when absent. Read-only.
	GetTransactionByID(ctx context.Context, tenantID, txID uuid.UUID) (*domain.GeneralLedgerRow, error)
	// GetDeferredRollforward returns the Deferred Revenue account's opening
	// balance, deferrals added, and amounts released over [start, end).
	GetDeferredRollforward(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (opening, added, released int64, err error)
}

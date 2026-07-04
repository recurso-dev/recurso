package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type LedgerRepository struct {
	db *sql.DB
}

func NewLedgerRepository(db *sql.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) CreateAccount(ctx context.Context, account *domain.LedgerAccount) error {
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ledger_accounts (id, tenant_id, name, type, code, ledger_id, currency, balance, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO NOTHING`,
		account.ID, account.TenantID, account.Name, account.Type, account.Code,
		account.LedgerID, account.Currency, account.Balance, account.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create ledger account: %w", err)
	}
	return nil
}

func (r *LedgerRepository) GetAccountsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.LedgerAccount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, name, type, code, ledger_id, COALESCE(currency, ''), balance, created_at
		 FROM ledger_accounts WHERE tenant_id = $1 ORDER BY code`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var accounts []*domain.LedgerAccount
	for rows.Next() {
		a := &domain.LedgerAccount{}
		if err := rows.Scan(&a.ID, &a.TenantID, &a.Name, &a.Type, &a.Code,
			&a.LedgerID, &a.Currency, &a.Balance, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan ledger account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *LedgerRepository) GetAccountByTenantAndCode(ctx context.Context, tenantID uuid.UUID, code int) (*domain.LedgerAccount, error) {
	a := &domain.LedgerAccount{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, type, code, ledger_id, COALESCE(currency, ''), balance, created_at
		 FROM ledger_accounts WHERE tenant_id = $1 AND code = $2`,
		tenantID, code).Scan(&a.ID, &a.TenantID, &a.Name, &a.Type, &a.Code,
		&a.LedgerID, &a.Currency, &a.Balance, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger account: %w", err)
	}
	return a, nil
}

func (r *LedgerRepository) CreateTransaction(ctx context.Context, tx *domain.LedgerTransaction) error {
	if tx.Timestamp.IsZero() {
		tx.Timestamp = time.Now()
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		tx.ID, tx.DebitAccountID, tx.CreditAccountID, tx.Amount,
		tx.LedgerID, tx.Code, tx.ReferenceID, tx.Description, tx.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to create ledger transaction: %w", err)
	}

	// Update balances: debit increases asset/expense, credit increases liability/revenue/equity
	_, err = r.db.ExecContext(ctx,
		`UPDATE ledger_accounts SET balance = balance + $1 WHERE id = $2`,
		int64(tx.Amount), tx.DebitAccountID)
	if err != nil {
		return fmt.Errorf("failed to update debit account balance: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE ledger_accounts SET balance = balance + $1 WHERE id = $2`,
		int64(tx.Amount), tx.CreditAccountID)
	if err != nil {
		return fmt.Errorf("failed to update credit account balance: %w", err)
	}

	return nil
}

func (r *LedgerRepository) GetTransactionsByAccount(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID) ([]*domain.LedgerTransaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.id, t.debit_account_id, t.credit_account_id, t.amount, t.ledger_id, t.code, COALESCE(t.reference_id, '00000000-0000-0000-0000-000000000000'), COALESCE(t.description, ''), t.created_at
		 FROM ledger_transactions t
		 JOIN ledger_accounts a ON a.id = $2 AND a.tenant_id = $1
		 WHERE t.debit_account_id = $2 OR t.credit_account_id = $2
		 ORDER BY t.created_at DESC LIMIT 100`, tenantID, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger transactions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var txns []*domain.LedgerTransaction
	for rows.Next() {
		tx := &domain.LedgerTransaction{}
		if err := rows.Scan(&tx.ID, &tx.DebitAccountID, &tx.CreditAccountID, &tx.Amount,
			&tx.LedgerID, &tx.Code, &tx.ReferenceID, &tx.Description, &tx.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan ledger transaction: %w", err)
		}
		txns = append(txns, tx)
	}
	return txns, rows.Err()
}

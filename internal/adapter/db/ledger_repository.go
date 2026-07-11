package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
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
		`SELECT id, tenant_id, name, type, code, ledger_id, COALESCE(currency, ''), debits_posted, credits_posted, balance, created_at
		 FROM ledger_accounts WHERE tenant_id = $1 ORDER BY code`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var accounts []*domain.LedgerAccount
	for rows.Next() {
		a := &domain.LedgerAccount{}
		if err := rows.Scan(&a.ID, &a.TenantID, &a.Name, &a.Type, &a.Code,
			&a.LedgerID, &a.Currency, &a.DebitsPosted, &a.CreditsPosted, &a.Balance, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan ledger account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *LedgerRepository) GetAccountByTenantAndCode(ctx context.Context, tenantID uuid.UUID, code int) (*domain.LedgerAccount, error) {
	a := &domain.LedgerAccount{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, type, code, ledger_id, COALESCE(currency, ''), debits_posted, credits_posted, balance, created_at
		 FROM ledger_accounts WHERE tenant_id = $1 AND code = $2`,
		tenantID, code).Scan(&a.ID, &a.TenantID, &a.Name, &a.Type, &a.Code,
		&a.LedgerID, &a.Currency, &a.DebitsPosted, &a.CreditsPosted, &a.Balance, &a.CreatedAt)
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

	// The insert and both balance moves run in one transaction so a partial
	// failure can never leave a posted transaction without its balance effect
	// (or vice-versa), permanently diverging the ledger.
	dbtx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin ledger transaction: %w", err)
	}
	defer func() { _ = dbtx.Rollback() }() // no-op once committed

	// Idempotent insert: a duplicate (reference_id, code) for a real reference
	// (invoice/payment/refund) is a no-op via the partial unique index, so a
	// replayed or concurrently-lost settle never double-posts. Recognition rows
	// (zero reference) are excluded from that index and always insert.
	res, err := dbtx.ExecContext(ctx,
		`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, description, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT DO NOTHING`,
		tx.ID, tx.DebitAccountID, tx.CreditAccountID, tx.Amount,
		tx.LedgerID, tx.Code, tx.ReferenceID, tx.Description, tx.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to create ledger transaction: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if n == 0 {
		// Already posted for this (reference_id, code) — do not re-apply balances.
		return dbtx.Commit()
	}

	// Balances move only when a new transaction row was actually inserted.
	// Track debits_posted / credits_posted per side and derive a SIGNED balance
	// from the account's normal side (ENG-148): a debit-normal account
	// (asset/expense) nets debits − credits; a credit-normal account
	// (liability/equity/revenue) nets credits − debits. The old code did
	// `balance += amount` on BOTH sides, so balance was accumulate-only and a
	// trial balance pulled from it never balanced.
	//
	// SQL semantics: every SET RHS reads the row's pre-update column values, so
	// `debits_posted + $1` in the balance CASE is the new debits total.
	if _, err := dbtx.ExecContext(ctx,
		`UPDATE ledger_accounts
		 SET debits_posted = debits_posted + $1,
		     balance = CASE WHEN lower(type) IN ('1', '5', 'asset', 'expense')
		                    THEN (debits_posted + $1) - credits_posted
		                    ELSE credits_posted - (debits_posted + $1) END,
		     updated_at = NOW()
		 WHERE id = $2`,
		int64(tx.Amount), tx.DebitAccountID); err != nil {
		return fmt.Errorf("failed to update debit account balance: %w", err)
	}
	if _, err := dbtx.ExecContext(ctx,
		`UPDATE ledger_accounts
		 SET credits_posted = credits_posted + $1,
		     balance = CASE WHEN lower(type) IN ('1', '5', 'asset', 'expense')
		                    THEN debits_posted - (credits_posted + $1)
		                    ELSE (credits_posted + $1) - debits_posted END,
		     updated_at = NOW()
		 WHERE id = $2`,
		int64(tx.Amount), tx.CreditAccountID); err != nil {
		return fmt.Errorf("failed to update credit account balance: %w", err)
	}

	return dbtx.Commit()
}

// InvoiceLedgerMismatch describes an invoice whose ledger postings for a
// given transaction code are missing or do not sum to the expected amount.
// Read-only reconciliation result; never written back.
type InvoiceLedgerMismatch struct {
	InvoiceID uuid.UUID // invoice whose ledger postings disagree
	Expected  int64     // amount the invoice says should be posted (total or amount_paid)
	Found     int64     // sum of matching ledger transaction amounts
	TxCount   int       // number of matching ledger transactions (0 = missing entirely)
}

// OrphanLedgerTransaction is a ledger transaction whose reference_id points
// to no existing invoice.
type OrphanLedgerTransaction struct {
	TransactionID uuid.UUID
	Code          uint16
	Amount        int64
	ReferenceID   uuid.UUID
}

// CountReconciliationScope returns how many invoices are subject to
// reconciliation for a tenant: all non-draft invoices, and the paid subset.
func (r *LedgerRepository) CountReconciliationScope(ctx context.Context, tenantID uuid.UUID) (nonDraft int, paid int, err error) {
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FILTER (WHERE status <> 'draft'),
		        COUNT(*) FILTER (WHERE status = 'paid')
		 FROM invoices WHERE tenant_id = $1`, tenantID).Scan(&nonDraft, &paid)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count reconciliation scope: %w", err)
	}
	return nonDraft, paid, nil
}

// GetInvoiceLedgerMismatches returns non-draft invoices whose Code-1
// (invoice) ledger postings are missing or do not sum to the invoice total.
// At most limit rows are returned; the second return value is the total
// number of mismatched invoices regardless of limit.
func (r *LedgerRepository) GetInvoiceLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]InvoiceLedgerMismatch, int, error) {
	const query = `
		SELECT sub.id, sub.expected, sub.found, sub.tx_count, COUNT(*) OVER () AS total
		FROM (
			SELECT i.id, i.total AS expected,
			       COALESCE(SUM(t.amount), 0) AS found,
			       COUNT(t.id) AS tx_count
			FROM invoices i
			LEFT JOIN ledger_transactions t ON t.reference_id = i.id AND t.code = 1
			WHERE i.tenant_id = $1 AND i.status <> 'draft'
			GROUP BY i.id, i.total
		) sub
		WHERE sub.tx_count = 0 OR sub.found <> sub.expected
		ORDER BY sub.id
		LIMIT $2`
	return r.queryInvoiceMismatches(ctx, query, tenantID, limit)
}

// GetPaymentLedgerMismatches returns paid invoices whose Code-3 (payment)
// ledger postings are missing or do not sum to amount_paid. At most limit
// rows are returned; the second return value is the total mismatch count.
func (r *LedgerRepository) GetPaymentLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]InvoiceLedgerMismatch, int, error) {
	const query = `
		SELECT sub.id, sub.expected, sub.found, sub.tx_count, COUNT(*) OVER () AS total
		FROM (
			SELECT i.id, COALESCE(i.amount_paid, 0) AS expected,
			       COALESCE(SUM(t.amount), 0) AS found,
			       COUNT(t.id) AS tx_count
			FROM invoices i
			LEFT JOIN ledger_transactions t ON t.reference_id = i.id AND t.code = 3
			WHERE i.tenant_id = $1 AND i.status = 'paid'
			GROUP BY i.id, i.amount_paid
		) sub
		WHERE sub.tx_count = 0 OR sub.found <> sub.expected
		ORDER BY sub.id
		LIMIT $2`
	return r.queryInvoiceMismatches(ctx, query, tenantID, limit)
}

func (r *LedgerRepository) queryInvoiceMismatches(ctx context.Context, query string, tenantID uuid.UUID, limit int) ([]InvoiceLedgerMismatch, int, error) {
	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query invoice ledger mismatches: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var mismatches []InvoiceLedgerMismatch
	total := 0
	for rows.Next() {
		var m InvoiceLedgerMismatch
		if err := rows.Scan(&m.InvoiceID, &m.Expected, &m.Found, &m.TxCount, &total); err != nil {
			return nil, 0, fmt.Errorf("failed to scan invoice ledger mismatch: %w", err)
		}
		mismatches = append(mismatches, m)
	}
	return mismatches, total, rows.Err()
}

// GetOrphanLedgerTransactions returns Code 1/3 ledger transactions for a
// tenant (scoped via account ownership) whose reference_id matches no
// invoice. At most limit rows are returned; the second return value is the
// total orphan count regardless of limit.
func (r *LedgerRepository) GetOrphanLedgerTransactions(ctx context.Context, tenantID uuid.UUID, limit int) ([]OrphanLedgerTransaction, int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sub.id, sub.code, sub.amount, sub.reference_id, COUNT(*) OVER () AS total
		FROM (
			SELECT t.id, t.code, t.amount, t.reference_id
			FROM ledger_transactions t
			WHERE t.code IN (1, 3)
			  AND t.reference_id IS NOT NULL
			  AND NOT EXISTS (SELECT 1 FROM invoices i WHERE i.id = t.reference_id)
			  AND EXISTS (
				SELECT 1 FROM ledger_accounts la
				WHERE la.tenant_id = $1
				  AND (la.id = t.debit_account_id OR la.id = t.credit_account_id)
			  )
		) sub
		ORDER BY sub.id
		LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query orphan ledger transactions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var orphans []OrphanLedgerTransaction
	total := 0
	for rows.Next() {
		var o OrphanLedgerTransaction
		if err := rows.Scan(&o.TransactionID, &o.Code, &o.Amount, &o.ReferenceID, &total); err != nil {
			return nil, 0, fmt.Errorf("failed to scan orphan ledger transaction: %w", err)
		}
		orphans = append(orphans, o)
	}
	return orphans, total, rows.Err()
}

// LedgerTransactionSummary is the minimal projection of a ledger transaction
// used to diff Postgres against TigerBeetle: the shared transaction ID and
// the posted amount. Read-only reconciliation input; never written back.
type LedgerTransactionSummary struct {
	TransactionID uuid.UUID
	Amount        int64
}

// CountLedgerTransactionsByTenant returns how many ledger transactions touch
// any of the tenant's ledger accounts. Used to bound the in-memory
// TigerBeetle comparison pass before loading rows.
func (r *LedgerRepository) CountLedgerTransactionsByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM ledger_transactions t
		WHERE EXISTS (
			SELECT 1 FROM ledger_accounts la
			WHERE la.tenant_id = $1
			  AND (la.id = t.debit_account_id OR la.id = t.credit_account_id)
		)`, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tenant ledger transactions: %w", err)
	}
	return count, nil
}

// GetLedgerTransactionSummaries returns id+amount for every ledger
// transaction touching one of the tenant's accounts, ordered by id, up to
// limit rows. Read-only; callers guard the row count first via
// CountLedgerTransactionsByTenant.
func (r *LedgerRepository) GetLedgerTransactionSummaries(ctx context.Context, tenantID uuid.UUID, limit int) ([]LedgerTransactionSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.amount
		FROM ledger_transactions t
		WHERE EXISTS (
			SELECT 1 FROM ledger_accounts la
			WHERE la.tenant_id = $1
			  AND (la.id = t.debit_account_id OR la.id = t.credit_account_id)
		)
		ORDER BY t.id
		LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger transaction summaries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var summaries []LedgerTransactionSummary
	for rows.Next() {
		var s LedgerTransactionSummary
		if err := rows.Scan(&s.TransactionID, &s.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan ledger transaction summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
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

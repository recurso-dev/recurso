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
	// accountingVersion is the deployment's active accounting model, stamped on
	// every journal that doesn't carry its own version (ADR-008 increment 2).
	// Zero ⇒ the cash default (AccountingModelV1); set to V2 when accrual is on.
	accountingVersion int
}

func NewLedgerRepository(db *sql.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

// SetAccountingVersion sets the accounting model stamped on journals this
// repository posts. Called at wiring time from the RECURSO_ACCRUAL_RECOGNITION
// flag (V2 when accrual is on); the default is V1 (cash). A transaction that
// carries its own non-zero AccountingVersion overrides this.
func (r *LedgerRepository) SetAccountingVersion(v int) { r.accountingVersion = v }

func (r *LedgerRepository) CreateAccount(ctx context.Context, account *domain.LedgerAccount) error {
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ledger_accounts (id, tenant_id, entity_id, name, type, code, ledger_id, currency, balance, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (id) DO NOTHING`,
		account.ID, account.TenantID, account.EntityID, account.Name, account.Type, account.Code,
		account.LedgerID, account.Currency, account.Balance, account.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create ledger account: %w", err)
	}
	return nil
}

func (r *LedgerRepository) GetAccountsByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.LedgerAccount, error) {
	query := `SELECT id, tenant_id, name, type, code, ledger_id, COALESCE(currency, ''), debits_posted, credits_posted, balance, created_at
		 FROM ledger_accounts WHERE tenant_id = $1 ORDER BY code`
	args := []interface{}{tenantID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET $3"
			args = append(args, offset)
		}
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
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

// GetDeferredRollforward returns the movement of the tenant's Deferred Revenue
// account across [start, end): the opening balance (net credits before start),
// deferrals added (credits) in the period, and amounts released (debits) in the
// period. If the tenant has no Deferred account yet, all zeros. Closing is
// derived by the service (opening + added - released).
func (r *LedgerRepository) GetDeferredRollforward(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (opening, added, released int64, err error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT
		   COALESCE(SUM(CASE WHEN t.created_at < $2 AND t.credit_account_id = a.id THEN t.amount
		                     WHEN t.created_at < $2 AND t.debit_account_id  = a.id THEN -t.amount
		                     ELSE 0 END), 0)::bigint AS opening,
		   COALESCE(SUM(CASE WHEN t.created_at >= $2 AND t.created_at < $3 AND t.credit_account_id = a.id THEN t.amount ELSE 0 END), 0)::bigint AS added,
		   COALESCE(SUM(CASE WHEN t.created_at >= $2 AND t.created_at < $3 AND t.debit_account_id  = a.id THEN t.amount ELSE 0 END), 0)::bigint AS released
		 FROM ledger_accounts a
		 LEFT JOIN ledger_transactions t ON a.id IN (t.debit_account_id, t.credit_account_id)
		 WHERE a.tenant_id = $1 AND a.code = $4`,
		tenantID, start, end, domain.AccountCodeDeferredRevenue)
	if err := row.Scan(&opening, &added, &released); err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, 0, nil
		}
		return 0, 0, 0, fmt.Errorf("failed to query deferred rollforward: %w", err)
	}
	return opening, added, released, nil
}

// SumPendingRecognitionEvents returns the total (net) revenue still scheduled to
// be recognized for a tenant: the sum of every pending recognition event across
// all its active schedules. The reconciler compares this against the Deferred
// Revenue balance — Deferred must always be at least this large, since it funds
// exactly this future recognition (plus any recorded-but-unpaid invoice
// deferrals). A Deferred balance below it means a posting drained Deferred past
// what the schedule holds (e.g. a downgrade credit over-debiting Deferred).
func (r *LedgerRepository) SumPendingRecognitionEvents(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0)::bigint FROM recognition_events
		 WHERE tenant_id = $1 AND status = 'pending'`, tenantID).Scan(&total)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to sum pending recognition events: %w", err)
	}
	return total, nil
}

// GetGeneralLedgerRows returns every posted transaction for a tenant, flattened
// with both account codes and names, ordered oldest first. Tenant-scoped via the
// debit account (both sides of a transfer always belong to the same tenant).
// This is the read-only general-ledger export an auditor imports.
// GetGeneralLedgerRows returns posted transactions, optionally filtered to one
// entity's ledger (Multi-Entity Books). A nil ledgerID returns every posting
// across the tenant's entity ledgers. Each row is tagged with the entity of its
// debit account's ledger.
func (r *LedgerRepository) GetGeneralLedgerRows(ctx context.Context, tenantID uuid.UUID, ledgerID *int) ([]domain.GeneralLedgerRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.id, t.created_at, t.code,
		        da.code, da.name, ca.code, ca.name,
		        t.amount::bigint, t.reference_id, COALESCE(t.description, ''),
		        t.accounting_version,
		        e.id, COALESCE(e.name, '')
		 FROM ledger_transactions t
		 JOIN ledger_accounts da ON da.id = t.debit_account_id
		 JOIN ledger_accounts ca ON ca.id = t.credit_account_id
		 LEFT JOIN entities e ON e.tenant_id = da.tenant_id AND e.tb_ledger_id = da.ledger_id
		 WHERE da.tenant_id = $1 AND ($2::int IS NULL OR da.ledger_id = $2)
		 ORDER BY t.created_at, t.id`, tenantID, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query general ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.GeneralLedgerRow
	for rows.Next() {
		var g domain.GeneralLedgerRow
		var entityID uuid.NullUUID
		if err := rows.Scan(&g.TransactionID, &g.Timestamp, &g.Code,
			&g.DebitAccountCode, &g.DebitAccountName, &g.CreditAccountCode, &g.CreditAccountName,
			&g.Amount, &g.ReferenceID, &g.Description, &g.AccountingVersion, &entityID, &g.EntityName); err != nil {
			return nil, fmt.Errorf("failed to scan general ledger row: %w", err)
		}
		if entityID.Valid {
			g.EntityID = &entityID.UUID
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GetTrialBalanceLines aggregates posted debit and credit totals per account
// for a tenant. Each ledger_transactions row carries one debit account and one
// credit account, so a row contributes its amount to exactly one side of each
// of the two accounts it touches. Accounts with no postings return zeros.
// GetTrialBalanceLines aggregates per account, optionally filtered to one
// entity's ledger (Multi-Entity Books). A nil ledgerID returns every account
// across all the tenant's entity ledgers (consolidated) — the reconciler and
// the default report use this. Each line is tagged with its entity, resolved
// from the account's ledger_id.
func (r *LedgerRepository) GetTrialBalanceLines(ctx context.Context, tenantID uuid.UUID, ledgerID *int) ([]domain.TrialBalanceLine, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT a.id, a.code, a.name, a.type,
		        COALESCE(SUM(CASE WHEN t.debit_account_id = a.id THEN t.amount ELSE 0 END), 0)::bigint  AS debits,
		        COALESCE(SUM(CASE WHEN t.credit_account_id = a.id THEN t.amount ELSE 0 END), 0)::bigint AS credits,
		        e.id, COALESCE(e.name, '')
		 FROM ledger_accounts a
		 LEFT JOIN ledger_transactions t ON a.id IN (t.debit_account_id, t.credit_account_id)
		 LEFT JOIN entities e ON e.tenant_id = a.tenant_id AND e.tb_ledger_id = a.ledger_id
		 WHERE a.tenant_id = $1 AND ($2::int IS NULL OR a.ledger_id = $2)
		 GROUP BY a.id, a.code, a.name, a.type, e.id, e.name
		 ORDER BY a.code`, tenantID, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query trial balance: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var lines []domain.TrialBalanceLine
	for rows.Next() {
		var l domain.TrialBalanceLine
		var entityID uuid.NullUUID
		if err := rows.Scan(&l.AccountID, &l.Code, &l.Name, &l.Type, &l.Debits, &l.Credits, &entityID, &l.EntityName); err != nil {
			return nil, fmt.Errorf("failed to scan trial balance line: %w", err)
		}
		if entityID.Valid {
			l.EntityID = &entityID.UUID
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// GetAccountByEntityAndCode resolves a GL account scoped to a legal entity
// (Multi-Entity Books). Used to keep each entity's chart of accounts separate.
func (r *LedgerRepository) GetAccountByEntityAndCode(ctx context.Context, tenantID, entityID uuid.UUID, code int) (*domain.LedgerAccount, error) {
	a := &domain.LedgerAccount{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, type, code, ledger_id, COALESCE(currency, ''), debits_posted, credits_posted, balance, created_at
		 FROM ledger_accounts WHERE tenant_id = $1 AND entity_id = $2 AND code = $3`,
		tenantID, entityID, code).Scan(&a.ID, &a.TenantID, &a.Name, &a.Type, &a.Code,
		&a.LedgerID, &a.Currency, &a.DebitsPosted, &a.CreditsPosted, &a.Balance, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger account by entity and code: %w", err)
	}
	return a, nil
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
	dbtx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin ledger transaction: %w", err)
	}
	defer func() { _ = dbtx.Rollback() }() // no-op once committed
	if err := r.applyLedgerTx(ctx, dbtx, tx); err != nil {
		return err
	}
	return dbtx.Commit()
}

// CreateTransactions posts several transfers in a SINGLE database transaction so
// a multi-leg posting is all-or-nothing. RecordInvoice, for a GST invoice,
// writes an AR→Revenue leg AND a Revenue→Tax-Payable reclassification; posting
// them in separate transactions risked committing the first and losing the
// second, silently leaving Revenue gross and Tax Payable understated (books
// still "balance" per-leg, so the invariant check wouldn't catch it).
func (r *LedgerRepository) CreateTransactions(ctx context.Context, txs []*domain.LedgerTransaction) error {
	if len(txs) == 0 {
		return nil
	}
	dbtx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin ledger transaction: %w", err)
	}
	defer func() { _ = dbtx.Rollback() }()
	for _, tx := range txs {
		if err := r.applyLedgerTx(ctx, dbtx, tx); err != nil {
			return err
		}
	}
	return dbtx.Commit()
}

// applyLedgerTx inserts one transfer (idempotent on reference_id+code+occurrence)
// and moves both account balances, WITHIN the caller's transaction. Extracted so a
// single post and a multi-leg post share identical semantics; the caller owns commit.
func (r *LedgerRepository) applyLedgerTx(ctx context.Context, dbtx *sql.Tx, tx *domain.LedgerTransaction) error {
	if tx.Timestamp.IsZero() {
		tx.Timestamp = time.Now()
	}

	// Journal-level accounting-model stamp (ADR-008): the transaction's own
	// version wins (a per-posting override); otherwise the deployment's active
	// model; otherwise the cash default. Never insert 0 into the NOT NULL column.
	version := tx.AccountingVersion
	if version == 0 {
		version = r.accountingVersion
	}
	if version == 0 {
		version = domain.AccountingModelV1
	}

	// Idempotent insert: a duplicate (reference_id, code, occurrence) for a real
	// reference (invoice/payment/refund) is a no-op via the partial unique index,
	// so a replayed or concurrently-lost settle never double-posts. Occurrence is
	// 0 everywhere except the ACH settle→reverse cycle, where it counts completed
	// reversals so a re-collected invoice posts fresh legs instead of having them
	// swallowed (docs/design-ledger-occurrence.md). Recognition rows (zero
	// reference) are excluded from the index and always insert.
	res, err := dbtx.ExecContext(ctx,
		`INSERT INTO ledger_transactions (id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, occurrence, description, accounting_version, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT DO NOTHING`,
		tx.ID, tx.DebitAccountID, tx.CreditAccountID, tx.Amount,
		tx.LedgerID, tx.Code, tx.ReferenceID, tx.Occurrence, tx.Description, version, tx.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to create ledger transaction: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if n == 0 {
		// Already posted for this (reference_id, code, occurrence) — do not
		// re-apply balances.
		return nil
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

	return nil
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

// CreditNoteLedgerMismatch is a credit note in a leg-bearing status whose
// issuance ledger leg is missing entirely (its id is referenced by no ledger
// transaction).
type CreditNoteLedgerMismatch struct {
	CreditNoteID uuid.UUID
}

// GetCreditNoteLedgerMismatches finds credit notes whose issuance ledger leg
// was never posted — the credit-note analog of a missing invoice Code-1 leg.
// Only completeness is checked (not amount): a credit note's legs vary by type
// (adjustment code 8, downgrade credit, refund, plus any tax reversal), so any
// ledger transaction referencing the credit note clears it — this catches only
// a WHOLLY forgotten leg. Scoped to leg-bearing statuses: pending_approval and
// rejected have no leg yet (posted on approval, ENG maker-checker) and void is
// excluded, so none of those are false positives.
func (r *LedgerRepository) GetCreditNoteLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]CreditNoteLedgerMismatch, int, error) {
	const query = `
		SELECT sub.id, COUNT(*) OVER () AS total
		FROM (
			SELECT cn.id, COUNT(t.id) AS tx_count
			FROM credit_notes cn
			LEFT JOIN ledger_transactions t ON t.reference_id = cn.id
			WHERE cn.tenant_id = $1
			  AND cn.status IN ('issued', 'used', 'expired')
			GROUP BY cn.id
		) sub
		WHERE sub.tx_count = 0
		ORDER BY sub.id
		LIMIT $2`
	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query credit-note ledger mismatches: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var mismatches []CreditNoteLedgerMismatch
	total := 0
	for rows.Next() {
		var m CreditNoteLedgerMismatch
		if err := rows.Scan(&m.CreditNoteID, &total); err != nil {
			return nil, 0, fmt.Errorf("failed to scan credit-note ledger mismatch: %w", err)
		}
		mismatches = append(mismatches, m)
	}
	return mismatches, total, rows.Err()
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

// CountTransactionsByReferenceAndCode counts posted legs of one code for a
// reference. The settle/reverse posting sites use the invoice's code-19 count
// as the occurrence (cycle) counter — docs/design-ledger-occurrence.md.
func (r *LedgerRepository) CountTransactionsByReferenceAndCode(ctx context.Context, referenceID uuid.UUID, code uint16) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ledger_transactions WHERE reference_id = $1 AND code = $2`,
		referenceID, code).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("failed to count ledger transactions: %w", err)
	}
	return n, nil
}

// GetLatestTransactionByReferenceAndCode returns the highest-occurrence leg of
// one code for a reference, or nil when none exists. The ACH reversal inverts
// the ACTUAL latest cash leg (amount and all) rather than recomputing it, so a
// wallet-part-funded settlement reverses exactly the net cash that was posted.
func (r *LedgerRepository) GetLatestTransactionByReferenceAndCode(ctx context.Context, referenceID uuid.UUID, code uint16) (*domain.LedgerTransaction, error) {
	tx := &domain.LedgerTransaction{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, debit_account_id, credit_account_id, amount, ledger_id, code, reference_id, occurrence, description, created_at
		 FROM ledger_transactions
		 WHERE reference_id = $1 AND code = $2
		 ORDER BY occurrence DESC, created_at DESC
		 LIMIT 1`,
		referenceID, code).Scan(
		&tx.ID, &tx.DebitAccountID, &tx.CreditAccountID, &tx.Amount,
		&tx.LedgerID, &tx.Code, &tx.ReferenceID, &tx.Occurrence, &tx.Description, &tx.Timestamp,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load latest ledger transaction: %w", err)
	}
	return tx, nil
}

// GetPaymentLedgerMismatches returns paid invoices whose payment-side ledger
// postings are missing or do not sum to amount_paid. At most limit rows are
// returned; the second return value is the total mismatch count.
//
// amount_paid is total-less-credit (MarkPaid), and that balance is relieved from
// AR by three cash-equivalent legs, not just cash: Code-3 (cash collected),
// Code-10 (TDS receivable — the buyer withheld it and remits to the government),
// and Code-12 (wallet drain — prepaid balance settled the invoice). Summing all
// three is what makes the check correct: a TDS invoice's cash leg is short by the
// withheld tax, and a wallet-fully-paid invoice has NO cash leg at all — checking
// Code-3 alone raised a false payment_amount_mismatch on the former and a false
// missing_payment_transaction on the latter, even though AR was fully relieved.
//
// Credit application (Code-7) is intentionally excluded: amount_paid already nets
// it out. A fully-credit-settled invoice therefore owes zero cash and needs no
// payment leg, so tx_count=0 is only a genuine "missing payment" when expected>0.
func (r *LedgerRepository) GetPaymentLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]InvoiceLedgerMismatch, int, error) {
	// Payment-shaped legs (3, 10, 12) add; ACH-return reversals (19) SUBTRACT,
	// so a settle→reverse→re-settle cycle nets to exactly one settlement
	// (C − C + C = amount_paid). This also makes the pre-occurrence-fix
	// corruption visible: a paid invoice whose re-settle leg was swallowed nets
	// to 0 ≠ amount_paid and is finally reported (docs/design-ledger-occurrence.md).
	const query = `
		SELECT sub.id, sub.expected, sub.found, sub.tx_count, COUNT(*) OVER () AS total
		FROM (
			SELECT i.id, COALESCE(i.amount_paid, 0) AS expected,
			       COALESCE(SUM(CASE WHEN t.code = 19 THEN -t.amount::bigint ELSE t.amount::bigint END), 0) AS found,
			       COUNT(t.id) AS tx_count
			FROM invoices i
			LEFT JOIN ledger_transactions t ON t.reference_id = i.id AND t.code IN (3, 10, 12, 19)
			WHERE i.tenant_id = $1 AND i.status = 'paid'
			GROUP BY i.id, i.amount_paid
		) sub
		WHERE (sub.tx_count = 0 AND sub.expected > 0) OR sub.found <> sub.expected
		ORDER BY sub.id
		LIMIT $2`
	return r.queryInvoiceMismatches(ctx, query, tenantID, limit)
}

// GetCreditApplicationLedgerMismatches finds invoices whose account-credit
// drawdown legs don't reconcile. When adjustment credit is applied to an
// invoice the repo sets credit_applied and books a code-7 leg
// (DR Customer-Credit / CR AR) for the same amount; that ledger post is
// best-effort. So an invoice with credit_applied > 0 MUST carry code-7 legs
// summing to it — a dropped or wrong-amount drawdown leg silently overstates
// both AR and the Customer-Credit liability while the books still balance and no
// sign goes abnormal, so nothing else catches it. Mirrors the payment check:
// expected = credit_applied, found = sum of code-7 legs referencing the invoice.
func (r *LedgerRepository) GetCreditApplicationLedgerMismatches(ctx context.Context, tenantID uuid.UUID, limit int) ([]InvoiceLedgerMismatch, int, error) {
	const query = `
		SELECT sub.id, sub.expected, sub.found, sub.tx_count, COUNT(*) OVER () AS total
		FROM (
			SELECT i.id, COALESCE(i.credit_applied, 0) AS expected,
			       COALESCE(SUM(t.amount::bigint), 0) AS found,
			       COUNT(t.id) AS tx_count
			FROM invoices i
			LEFT JOIN ledger_transactions t ON t.reference_id = i.id AND t.code = 7
			WHERE i.tenant_id = $1 AND COALESCE(i.credit_applied, 0) > 0
			GROUP BY i.id, i.credit_applied
		) sub
		WHERE (sub.tx_count = 0 AND sub.expected > 0) OR sub.found <> sub.expected
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

// RecognitionOverrun is a revenue schedule whose recognized events sum to MORE
// than the schedule's total recognizable amount — i.e. the system recognized
// revenue it was never owed. The accrual invariant "Revenue Recognized ≤ the
// amount to recognize" must never break; this is the reconciler input for it.
type RecognitionOverrun struct {
	ScheduleID  uuid.UUID
	InvoiceID   uuid.UUID
	TotalAmount int64 // the schedule's recognizable base
	Recognized  int64 // sum of recognized events (> TotalAmount when this row exists)
}

// GetRecognitionOverruns returns schedules where the sum of RECOGNIZED events
// exceeds the schedule's total_amount. A non-empty result means revenue was
// over-recognized (money fabricated on the P&L) — a hard accrual violation.
// Empty in a correct system, so it costs nothing on healthy tenants.
func (r *LedgerRepository) GetRecognitionOverruns(ctx context.Context, tenantID uuid.UUID, limit int) ([]RecognitionOverrun, int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sub.id, sub.invoice_id, sub.total_amount, sub.recognized, COUNT(*) OVER () AS total
		FROM (
			SELECT s.id, s.invoice_id, s.total_amount,
			       COALESCE(SUM(CASE WHEN e.status = 'recognized' THEN e.amount ELSE 0 END), 0)::bigint AS recognized
			FROM revenue_schedules s
			LEFT JOIN recognition_events e ON e.revenue_schedule_id = s.id
			WHERE s.tenant_id = $1
			GROUP BY s.id, s.invoice_id, s.total_amount
			HAVING COALESCE(SUM(CASE WHEN e.status = 'recognized' THEN e.amount ELSE 0 END), 0) > s.total_amount
		) sub
		ORDER BY sub.id
		LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query recognition overruns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var overruns []RecognitionOverrun
	total := 0
	for rows.Next() {
		var o RecognitionOverrun
		if err := rows.Scan(&o.ScheduleID, &o.InvoiceID, &o.TotalAmount, &o.Recognized, &total); err != nil {
			return nil, 0, fmt.Errorf("failed to scan recognition overrun: %w", err)
		}
		overruns = append(overruns, o)
	}
	return overruns, total, rows.Err()
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

// GetTransactionsByAccount pages an account's postings, newest first. code=0
// means all posting codes; a nonzero code filters to that posting type.
func (r *LedgerRepository) GetTransactionsByAccount(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID, code uint16, limit, offset int) ([]*domain.LedgerTransaction, error) {
	if limit <= 0 || limit > 250 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.id, t.debit_account_id, t.credit_account_id, t.amount, t.ledger_id, t.code, COALESCE(t.reference_id, '00000000-0000-0000-0000-000000000000'), COALESCE(t.description, ''), t.created_at
		 FROM ledger_transactions t
		 JOIN ledger_accounts a ON a.id = $2 AND a.tenant_id = $1
		 WHERE (t.debit_account_id = $2 OR t.credit_account_id = $2)
		   AND ($3 = 0 OR t.code = $3)
		 ORDER BY t.created_at DESC LIMIT $4 OFFSET $5`, tenantID, accountID, int(code), limit, offset)
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

package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// WalletRepository is the Postgres implementation of port.WalletRepository.
// Every movement (top-up, drain, expiry) runs in one transaction with the
// wallet row locked FOR UPDATE, so the residue bookkeeping and the
// denormalized balance can never diverge under concurrency.
type WalletRepository struct {
	db *sql.DB
}

func NewWalletRepository(db *sql.DB) port.WalletRepository {
	return &WalletRepository{db: db}
}

const walletColumns = `id, tenant_id, entity_id, customer_id, currency, balance, auto_recharge_threshold, auto_recharge_amount, created_at, updated_at, closed_at`

func scanWallet(row interface{ Scan(...any) error }) (*domain.Wallet, error) {
	var w domain.Wallet
	if err := row.Scan(&w.ID, &w.TenantID, &w.EntityID, &w.CustomerID, &w.Currency, &w.Balance,
		&w.AutoRechargeThreshold, &w.AutoRechargeAmount, &w.CreatedAt, &w.UpdatedAt, &w.ClosedAt); err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) Create(ctx context.Context, w *domain.Wallet) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO wallets (id, tenant_id, entity_id, customer_id, currency, balance, auto_recharge_threshold, auto_recharge_amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		w.ID, w.TenantID, w.EntityID, w.CustomerID, w.Currency, w.Balance,
		w.AutoRechargeThreshold, w.AutoRechargeAmount, w.CreatedAt, w.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert wallet: %w", err)
	}
	return nil
}

func (r *WalletRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Wallet, error) {
	w, err := scanWallet(r.db.QueryRowContext(ctx,
		`SELECT `+walletColumns+` FROM wallets WHERE tenant_id = $1 AND id = $2`, tenantID, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	return w, nil
}

// GetByCustomerEntityAndCurrency resolves the (customer, entity, currency)
// wallet — entity-scoped so a balance is spent only on the owning entity's
// invoices (Multi-Entity Books).
func (r *WalletRepository) GetByCustomerEntityAndCurrency(ctx context.Context, tenantID, customerID, entityID uuid.UUID, currency string) (*domain.Wallet, error) {
	w, err := scanWallet(r.db.QueryRowContext(ctx,
		`SELECT `+walletColumns+` FROM wallets WHERE tenant_id = $1 AND customer_id = $2 AND entity_id = $3 AND UPPER(currency) = UPPER($4) AND closed_at IS NULL`,
		tenantID, customerID, entityID, currency))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet by customer: %w", err)
	}
	return w, nil
}

func (r *WalletRepository) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Wallet, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+walletColumns+` FROM wallets WHERE tenant_id = $1 AND customer_id = $2 ORDER BY currency`,
		tenantID, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallets: %w", err)
	}
	defer func() { _ = rows.Close() }()
	wallets := []domain.Wallet{}
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, *w)
	}
	return wallets, rows.Err()
}

func (r *WalletRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.Wallet, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+walletColumns+` FROM wallets WHERE tenant_id = $1 ORDER BY updated_at DESC LIMIT $2`,
		tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant wallets: %w", err)
	}
	defer func() { _ = rows.Close() }()
	wallets := []domain.Wallet{}
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, *w)
	}
	return wallets, rows.Err()
}

func (r *WalletRepository) UpdateAutoRecharge(ctx context.Context, tenantID, id uuid.UUID, threshold, amount *int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE wallets SET auto_recharge_threshold = $3, auto_recharge_amount = $4, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, threshold, amount)
	if err != nil {
		return fmt.Errorf("failed to update wallet auto-recharge: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *WalletRepository) TopUp(ctx context.Context, wtx *domain.WalletTransaction) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var balance int64
	if err := tx.QueryRowContext(ctx,
		`SELECT balance FROM wallets WHERE tenant_id = $1 AND id = $2 FOR UPDATE`,
		wtx.TenantID, wtx.WalletID).Scan(&balance); err != nil {
		return fmt.Errorf("failed to lock wallet for top-up: %w", err)
	}

	balance += wtx.Amount
	wtx.BalanceAfter = balance
	remaining := wtx.Amount
	wtx.Remaining = &remaining

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions (id, tenant_id, wallet_id, type, source, amount, remaining, balance_after, invoice_id, expires_at, created_at)
		VALUES ($1, $2, $3, 'top_up', $4, $5, $6, $7, $8, $9, $10)`,
		wtx.ID, wtx.TenantID, wtx.WalletID, wtx.Source, wtx.Amount, remaining,
		balance, wtx.InvoiceID, wtx.ExpiresAt, wtx.CreatedAt,
	); err != nil {
		return fmt.Errorf("failed to insert wallet top-up: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallets SET balance = $3, updated_at = NOW() WHERE tenant_id = $1 AND id = $2`,
		wtx.TenantID, wtx.WalletID, balance); err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}
	return tx.Commit()
}

// Close settles and closes a wallet in one transaction: it zeroes every open
// residue, splitting the total into a REFUND of paid residue (returned to the
// customer) and a FORFEIT of promotional residue (non-refundable), appends one
// closing transaction per non-zero portion, and stamps closed_at.
func (r *WalletRepository) Close(ctx context.Context, tenantID, walletID uuid.UUID, now time.Time) (port.WalletCloseResult, error) {
	var res port.WalletCloseResult
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback() }()

	var closedAt sql.NullTime
	if err := tx.QueryRowContext(ctx,
		`SELECT closed_at FROM wallets WHERE tenant_id = $1 AND id = $2 FOR UPDATE`,
		tenantID, walletID).Scan(&closedAt); err != nil {
		return res, fmt.Errorf("failed to lock wallet for close: %w", err)
	}
	if closedAt.Valid {
		return res, port.ErrWalletAlreadyClosed
	}

	// Sum open residues, split by whether the top-up was paid or promotional.
	if err := tx.QueryRowContext(ctx, `
		SELECT
		  COALESCE(SUM(CASE WHEN source = 'promotional' THEN remaining ELSE 0 END), 0)::bigint,
		  COALESCE(SUM(CASE WHEN source <> 'promotional' OR source IS NULL THEN remaining ELSE 0 END), 0)::bigint
		FROM wallet_transactions
		WHERE wallet_id = $1 AND type = 'top_up' AND remaining > 0`,
		walletID).Scan(&res.Forfeited, &res.Refunded); err != nil {
		return res, fmt.Errorf("failed to sum wallet residues: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE wallet_transactions SET remaining = 0 WHERE wallet_id = $1 AND type = 'top_up' AND remaining > 0`,
		walletID); err != nil {
		return res, fmt.Errorf("failed to zero wallet residues: %w", err)
	}

	insertClosing := func(txType domain.WalletTransactionType, amount int64) (uuid.UUID, error) {
		id := uuid.New()
		_, e := tx.ExecContext(ctx, `
			INSERT INTO wallet_transactions (id, tenant_id, wallet_id, type, amount, balance_after, created_at)
			VALUES ($1, $2, $3, $4, $5, 0, $6)`,
			id, tenantID, walletID, txType, amount, now)
		return id, e
	}
	if res.Refunded > 0 {
		if res.RefundTxID, err = insertClosing(domain.WalletTxRefund, res.Refunded); err != nil {
			return res, fmt.Errorf("failed to insert wallet refund: %w", err)
		}
	}
	if res.Forfeited > 0 {
		if res.ForfeitTxID, err = insertClosing(domain.WalletTxForfeit, res.Forfeited); err != nil {
			return res, fmt.Errorf("failed to insert wallet forfeit: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE wallets SET balance = 0, closed_at = $3, updated_at = $3 WHERE tenant_id = $1 AND id = $2`,
		tenantID, walletID, now); err != nil {
		return res, fmt.Errorf("failed to close wallet: %w", err)
	}
	return res, tx.Commit()
}

func (r *WalletRepository) Drain(ctx context.Context, tenantID, walletID uuid.UUID, maxAmount int64, invoiceID uuid.UUID, now time.Time) (int64, error) {
	if maxAmount <= 0 {
		return 0, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var balance int64
	if err := tx.QueryRowContext(ctx,
		`SELECT balance FROM wallets WHERE tenant_id = $1 AND id = $2 FOR UPDATE`,
		tenantID, walletID).Scan(&balance); err != nil {
		return 0, fmt.Errorf("failed to lock wallet for drain: %w", err)
	}
	if balance <= 0 {
		return 0, nil
	}

	// Open, non-expired residues — oldest expiry first (NULLS LAST), then
	// oldest top-up first, so dated promotional credit is spent before it
	// can expire.
	rows, err := tx.QueryContext(ctx, `
		SELECT id, remaining FROM wallet_transactions
		WHERE wallet_id = $1 AND type = 'top_up' AND remaining > 0
		AND (expires_at IS NULL OR expires_at > $2)
		ORDER BY expires_at ASC NULLS LAST, created_at ASC
		FOR UPDATE`,
		walletID, now)
	if err != nil {
		return 0, fmt.Errorf("failed to select wallet residues: %w", err)
	}
	type residue struct {
		id        uuid.UUID
		remaining int64
	}
	var residues []residue
	for rows.Next() {
		var res residue
		if err := rows.Scan(&res.id, &res.remaining); err != nil {
			_ = rows.Close()
			return 0, err
		}
		residues = append(residues, res)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var drained int64
	for _, res := range residues {
		if drained >= maxAmount {
			break
		}
		take := maxAmount - drained
		if take > res.remaining {
			take = res.remaining
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE wallet_transactions SET remaining = remaining - $2 WHERE id = $1`,
			res.id, take); err != nil {
			return 0, fmt.Errorf("failed to consume wallet residue: %w", err)
		}
		drained += take
	}
	if drained == 0 {
		return 0, nil // only expired residue left; the expiry sweep will write it off
	}

	balance -= drained
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions (id, tenant_id, wallet_id, type, amount, balance_after, invoice_id, created_at)
		VALUES ($1, $2, $3, 'drain', $4, $5, $6, $7)`,
		uuid.New(), tenantID, walletID, drained, balance, invoiceID, now,
	); err != nil {
		return 0, fmt.Errorf("failed to insert wallet drain: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallets SET balance = $3, updated_at = NOW() WHERE tenant_id = $1 AND id = $2`,
		tenantID, walletID, balance); err != nil {
		return 0, fmt.Errorf("failed to update wallet balance after drain: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return drained, nil
}

func (r *WalletRepository) ExpireOverdue(ctx context.Context, now time.Time) ([]port.WalletExpiry, error) {
	// One pass per wallet holding expired residue; each pass is its own
	// small transaction so a failure never blocks other wallets.
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT wallet_id, tenant_id FROM wallet_transactions
		WHERE type = 'top_up' AND remaining > 0 AND expires_at IS NOT NULL AND expires_at <= $1`,
		now)
	if err != nil {
		return nil, fmt.Errorf("failed to find expired wallet residue: %w", err)
	}
	type target struct{ walletID, tenantID uuid.UUID }
	var targets []target
	for rows.Next() {
		var tgt target
		if err := rows.Scan(&tgt.walletID, &tgt.tenantID); err != nil {
			_ = rows.Close()
			return nil, err
		}
		targets = append(targets, tgt)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var expiries []port.WalletExpiry
	for _, tgt := range targets {
		exp, err := r.expireWallet(ctx, tgt.tenantID, tgt.walletID, now)
		if err != nil {
			return expiries, err
		}
		if exp != nil {
			expiries = append(expiries, *exp)
		}
	}
	return expiries, nil
}

// expireWallet writes off one wallet's expired residue: the wallet row is
// locked, the expired sum is read and zeroed in the same transaction, and
// the balance + expiry transaction land atomically. Returns the write-off
// details (nil when nothing had actually expired for this wallet).
func (r *WalletRepository) expireWallet(ctx context.Context, tenantID, walletID uuid.UUID, now time.Time) (*port.WalletExpiry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var balance int64
	var entityID uuid.UUID
	if err := tx.QueryRowContext(ctx,
		`SELECT balance, entity_id FROM wallets WHERE tenant_id = $1 AND id = $2 FOR UPDATE`,
		tenantID, walletID).Scan(&balance, &entityID); err != nil {
		return nil, fmt.Errorf("failed to lock wallet for expiry: %w", err)
	}

	var expired int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(remaining), 0) FROM wallet_transactions
		WHERE wallet_id = $1 AND type = 'top_up' AND remaining > 0
		AND expires_at IS NOT NULL AND expires_at <= $2`,
		walletID, now).Scan(&expired); err != nil {
		return nil, fmt.Errorf("failed to sum expired residue: %w", err)
	}
	if expired == 0 {
		return nil, tx.Commit()
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE wallet_transactions SET remaining = 0
		WHERE wallet_id = $1 AND type = 'top_up' AND remaining > 0
		AND expires_at IS NOT NULL AND expires_at <= $2`,
		walletID, now); err != nil {
		return nil, fmt.Errorf("failed to zero expired residue: %w", err)
	}
	expiryTxID, err := r.finishExpiry(ctx, tx, tenantID, walletID, balance, expired, now)
	if err != nil {
		return nil, err
	}
	return &port.WalletExpiry{TenantID: tenantID, WalletID: walletID, EntityID: entityID, Amount: expired, ExpiryTxID: expiryTxID}, nil
}

// finishExpiry writes the expiry transaction + updated balance and commits,
// returning the id of the expiry transaction (referenced by the ledger leg).
func (r *WalletRepository) finishExpiry(ctx context.Context, tx *sql.Tx, tenantID, walletID uuid.UUID, balance, expired int64, now time.Time) (uuid.UUID, error) {
	balance -= expired
	if balance < 0 {
		balance = 0
	}
	expiryTxID := uuid.New()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions (id, tenant_id, wallet_id, type, amount, balance_after, created_at)
		VALUES ($1, $2, $3, 'expiry', $4, $5, $6)`,
		expiryTxID, tenantID, walletID, expired, balance, now,
	); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert expiry transaction: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallets SET balance = $3, updated_at = NOW() WHERE tenant_id = $1 AND id = $2`,
		tenantID, walletID, balance); err != nil {
		return uuid.Nil, fmt.Errorf("failed to update wallet balance after expiry: %w", err)
	}
	return expiryTxID, tx.Commit()
}

func (r *WalletRepository) ListDueForRecharge(ctx context.Context, limit int) ([]domain.Wallet, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+walletColumns+` FROM wallets
		WHERE auto_recharge_threshold IS NOT NULL
		AND auto_recharge_amount IS NOT NULL
		AND auto_recharge_amount > 0
		AND balance < auto_recharge_threshold
		AND closed_at IS NULL
		ORDER BY updated_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallets due for recharge: %w", err)
	}
	defer func() { _ = rows.Close() }()
	wallets := []domain.Wallet{}
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, *w)
	}
	return wallets, rows.Err()
}

func (r *WalletRepository) ListTransactions(ctx context.Context, tenantID, walletID uuid.UUID, limit int) ([]domain.WalletTransaction, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, wallet_id, type, source, amount, remaining, balance_after, invoice_id, expires_at, created_at
		FROM wallet_transactions
		WHERE tenant_id = $1 AND wallet_id = $2
		ORDER BY created_at DESC
		LIMIT $3`,
		tenantID, walletID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallet transactions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	txs := []domain.WalletTransaction{}
	for rows.Next() {
		var t domain.WalletTransaction
		if err := rows.Scan(&t.ID, &t.TenantID, &t.WalletID, &t.Type, &t.Source, &t.Amount,
			&t.Remaining, &t.BalanceAfter, &t.InvoiceID, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

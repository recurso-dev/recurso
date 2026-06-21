package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// TxManager provides a helper for executing functions within a database transaction.
// This ensures atomicity: either all operations succeed or all are rolled back.
type TxManager struct {
	db *sql.DB
}

// NewTxManager creates a new TxManager.
func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

// WithTx executes the given function within a transaction.
// If the function returns an error, the transaction is rolled back.
// If the function panics, the transaction is rolled back and an error is returned.
// If the function returns nil, the transaction is committed.
func (tm *TxManager) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) (txErr error) {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			slog.Error("panic recovered in transaction", "panic", p)
			txErr = fmt.Errorf("transaction panicked: %v", p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetDB returns the underlying *sql.DB for cases where direct access is needed.
func (tm *TxManager) GetDB() *sql.DB {
	return tm.db
}

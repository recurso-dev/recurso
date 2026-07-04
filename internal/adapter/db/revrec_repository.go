package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type RevRecRepository struct {
	db *sql.DB
}

func NewRevRecRepository(db *sql.DB) *RevRecRepository {
	return &RevRecRepository{db: db}
}

func (r *RevRecRepository) CreateSchedule(ctx context.Context, schedule *domain.RevenueSchedule) error {
	query := `
		INSERT INTO revenue_schedules (id, tenant_id, invoice_id, subscription_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		schedule.ID, schedule.TenantID, schedule.InvoiceID, schedule.SubscriptionID, schedule.TotalAmount,
		schedule.Currency, schedule.StartDate, schedule.EndDate, schedule.Status, schedule.CreatedAt, schedule.UpdatedAt,
	)
	return err
}

func (r *RevRecRepository) CreateEvents(ctx context.Context, events []*domain.RecognitionEvent) error {
	if len(events) == 0 {
		return nil
	}

	query := `INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at) VALUES `
	args := make([]interface{}, 0, len(events)*7)
	for i, e := range events {
		if i > 0 {
			query += ", "
		}
		base := i * 7
		query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7)
		args = append(args, e.ID, e.RevenueScheduleID, e.TenantID, e.Amount, e.RecognitionDate, e.Status, e.CreatedAt)
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *RevRecRepository) GetDueEvents(ctx context.Context, date time.Time) ([]*domain.RecognitionEvent, error) {
	query := `
		SELECT id, revenue_schedule_id, tenant_id, amount, recognition_date, status, ledger_tx_id, created_at
		FROM recognition_events
		WHERE recognition_date <= $1 AND status = 'pending'
	`
	log.Printf("RevRec Repository: Querying for events <= %v", date)
	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*domain.RecognitionEvent
	for rows.Next() {
		var e domain.RecognitionEvent
		var ledgerTxID sql.NullString // Use NullString for UUID scan safety

		if err := rows.Scan(&e.ID, &e.RevenueScheduleID, &e.TenantID, &e.Amount, &e.RecognitionDate, &e.Status, &ledgerTxID, &e.CreatedAt); err != nil {
			log.Printf("RevRec Repository: Scan error: %v", err)
			return nil, err
		}

		if ledgerTxID.Valid {
			u := uuid.MustParse(ledgerTxID.String)
			e.LedgerTxID = &u
		}
		events = append(events, &e)
	}
	return events, nil
}

func (r *RevRecRepository) MarkEventRecognized(ctx context.Context, eventID uuid.UUID, ledgerTxID uuid.UUID) error {
	query := `UPDATE recognition_events SET status = 'recognized', ledger_tx_id = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, ledgerTxID, eventID)
	return err
}

func (r *RevRecRepository) MarkEventFailed(ctx context.Context, eventID uuid.UUID, reason string) error {
	query := `UPDATE recognition_events SET status = 'failed' WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, eventID)
	return err
}

func (r *RevRecRepository) GetReport(ctx context.Context, tenantID uuid.UUID, month, year int) (map[string]interface{}, error) {
	var totalRecognized int64
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM recognition_events
		WHERE tenant_id = $1 AND status = 'recognized'
		AND EXTRACT(MONTH FROM recognition_date) = $2
		AND EXTRACT(YEAR FROM recognition_date) = $3
	`
	err := r.db.QueryRowContext(ctx, query, tenantID, month, year).Scan(&totalRecognized)
	if err != nil {
		return nil, err
	}

	var totalDeferred int64
	queryDef := `
		SELECT COALESCE(SUM(amount), 0)
		FROM recognition_events
		WHERE tenant_id = $1 AND status = 'pending'
	`
	err = r.db.QueryRowContext(ctx, queryDef, tenantID).Scan(&totalDeferred)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"recognized_amount": totalRecognized,
		"deferred_amount":   totalDeferred,
		"month":             month,
		"year":              year,
	}, nil
}

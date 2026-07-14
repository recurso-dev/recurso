package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
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

// GetActiveSchedulesBySubscription returns a subscription's active schedules
// (tenant-scoped) for an unwind (ENG-147).
func (r *RevRecRepository) GetActiveSchedulesBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.RevenueSchedule, error) {
	query := `
		SELECT id, tenant_id, invoice_id, subscription_id, total_amount, currency, start_date, end_date, status, created_at, updated_at
		FROM revenue_schedules
		WHERE tenant_id = $1 AND subscription_id = $2 AND status = 'active'
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var schedules []*domain.RevenueSchedule
	for rows.Next() {
		var s domain.RevenueSchedule
		if err := rows.Scan(&s.ID, &s.TenantID, &s.InvoiceID, &s.SubscriptionID, &s.TotalAmount,
			&s.Currency, &s.StartDate, &s.EndDate, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	return schedules, rows.Err()
}

// GetActiveScheduleByInvoice returns the active schedule for an invoice, or nil
// when there is none (one-off invoice, or already fully recognized/canceled).
func (r *RevRecRepository) GetActiveScheduleByInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) (*domain.RevenueSchedule, error) {
	query := `
		SELECT id, tenant_id, invoice_id, subscription_id, total_amount, currency, start_date, end_date, status, created_at, updated_at
		FROM revenue_schedules
		WHERE tenant_id = $1 AND invoice_id = $2 AND status = 'active'
		LIMIT 1
	`
	var s domain.RevenueSchedule
	err := r.db.QueryRowContext(ctx, query, tenantID, invoiceID).Scan(
		&s.ID, &s.TenantID, &s.InvoiceID, &s.SubscriptionID, &s.TotalAmount,
		&s.Currency, &s.StartDate, &s.EndDate, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetPendingEventsBySchedule returns a schedule's not-yet-recognized events,
// latest recognition_date first so an unwind reduces from the tail.
func (r *RevRecRepository) GetPendingEventsBySchedule(ctx context.Context, scheduleID uuid.UUID) ([]*domain.RecognitionEvent, error) {
	query := `
		SELECT id, revenue_schedule_id, tenant_id, amount, recognition_date, status, ledger_tx_id, created_at
		FROM recognition_events
		WHERE revenue_schedule_id = $1 AND status = 'pending'
		ORDER BY recognition_date DESC
	`
	rows, err := r.db.QueryContext(ctx, query, scheduleID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*domain.RecognitionEvent
	for rows.Next() {
		var e domain.RecognitionEvent
		var ledgerTxID sql.NullString
		if err := rows.Scan(&e.ID, &e.RevenueScheduleID, &e.TenantID, &e.Amount, &e.RecognitionDate, &e.Status, &ledgerTxID, &e.CreatedAt); err != nil {
			return nil, err
		}
		if ledgerTxID.Valid {
			u := uuid.MustParse(ledgerTxID.String)
			e.LedgerTxID = &u
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// CancelEvent voids a pending event so the recognition worker never posts it.
// Scoped to status='pending' so a recognized event can't be silently unwound.
func (r *RevRecRepository) CancelEvent(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE recognition_events SET status = 'canceled' WHERE id = $1 AND status = 'pending'`, eventID)
	return err
}

// SetEventAmount reduces a pending event's amount (boundary split on a partial
// refund). Scoped to status='pending'.
func (r *RevRecRepository) SetEventAmount(ctx context.Context, eventID uuid.UUID, amount int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE recognition_events SET amount = $1 WHERE id = $2 AND status = 'pending'`, amount, eventID)
	return err
}

// MarkScheduleCanceled marks a schedule canceled once its deferred is unwound.
func (r *RevRecRepository) MarkScheduleCanceled(ctx context.Context, scheduleID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE revenue_schedules SET status = 'canceled', updated_at = NOW() WHERE id = $1`, scheduleID)
	return err
}

// GetReport builds a deferred-revenue rollforward: revenue recognized in the
// requested month/year, the balance still deferred, the schedule of when that
// balance releases (grouped by recognition month), and its split by currency.
// GetWaterfall returns the tenant's recognition curve, one row per month:
// revenue recognized (status=recognized) and revenue still scheduled
// (status=pending) by the month of recognition_date. Canceled/failed events
// are excluded.
func (r *RevRecRepository) GetWaterfall(ctx context.Context, tenantID uuid.UUID) ([]domain.RevenueWaterfallBucket, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT EXTRACT(YEAR FROM recognition_date)::int  AS y,
		        EXTRACT(MONTH FROM recognition_date)::int AS m,
		        COALESCE(SUM(CASE WHEN status = $2 THEN amount ELSE 0 END), 0)::bigint AS recognized,
		        COALESCE(SUM(CASE WHEN status = $3 THEN amount ELSE 0 END), 0)::bigint AS scheduled
		   FROM recognition_events
		  WHERE tenant_id = $1 AND status IN ($2, $3)
		  GROUP BY y, m
		  ORDER BY y, m`,
		tenantID, domain.RecognitionStatusRecognized, domain.RecognitionStatusPending)
	if err != nil {
		return nil, fmt.Errorf("query revenue waterfall: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var buckets []domain.RevenueWaterfallBucket
	for rows.Next() {
		var b domain.RevenueWaterfallBucket
		if err := rows.Scan(&b.Year, &b.Month, &b.Recognized, &b.Scheduled); err != nil {
			return nil, fmt.Errorf("scan waterfall bucket: %w", err)
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

func (r *RevRecRepository) GetReport(ctx context.Context, tenantID uuid.UUID, month, year int) (*domain.DeferredRevenueReport, error) {
	report := &domain.DeferredRevenueReport{
		Month:      month,
		Year:       year,
		Upcoming:   []domain.DeferredRecognitionBucket{},
		ByCurrency: []domain.DeferredCurrencyBalance{},
	}

	// Recognized in the requested period.
	if err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0)
		   FROM recognition_events
		  WHERE tenant_id = $1 AND status = $2
		    AND EXTRACT(MONTH FROM recognition_date) = $3
		    AND EXTRACT(YEAR  FROM recognition_date) = $4`,
		tenantID, domain.RecognitionStatusRecognized, month, year,
	).Scan(&report.RecognizedAmount); err != nil {
		return nil, fmt.Errorf("recognized total: %w", err)
	}

	// Total balance still deferred (all still-pending recognition).
	if err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0)
		   FROM recognition_events
		  WHERE tenant_id = $1 AND status = $2`,
		tenantID, domain.RecognitionStatusPending,
	).Scan(&report.DeferredBalance); err != nil {
		return nil, fmt.Errorf("deferred balance: %w", err)
	}

	// Release schedule: the deferred balance grouped by the month it recognizes.
	rows, err := r.db.QueryContext(ctx,
		`SELECT EXTRACT(YEAR FROM recognition_date)::int  AS y,
		        EXTRACT(MONTH FROM recognition_date)::int AS m,
		        COALESCE(SUM(amount), 0)
		   FROM recognition_events
		  WHERE tenant_id = $1 AND status = $2
		  GROUP BY y, m
		  ORDER BY y, m`,
		tenantID, domain.RecognitionStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("release schedule: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var b domain.DeferredRecognitionBucket
		if err := rows.Scan(&b.Year, &b.Month, &b.Amount); err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}
		report.Upcoming = append(report.Upcoming, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("release schedule rows: %w", err)
	}

	// Deferred balance split by the originating schedule's currency (honest
	// multi-currency: the flat DeferredBalance sums these).
	curRows, err := r.db.QueryContext(ctx,
		`SELECT rs.currency, COALESCE(SUM(re.amount), 0)
		   FROM recognition_events re
		   JOIN revenue_schedules rs ON rs.id = re.revenue_schedule_id
		  WHERE re.tenant_id = $1 AND re.status = $2
		  GROUP BY rs.currency
		  ORDER BY rs.currency`,
		tenantID, domain.RecognitionStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("currency split: %w", err)
	}
	defer func() { _ = curRows.Close() }()
	for curRows.Next() {
		var c domain.DeferredCurrencyBalance
		if err := curRows.Scan(&c.Currency, &c.Deferred); err != nil {
			return nil, fmt.Errorf("scan currency: %w", err)
		}
		report.ByCurrency = append(report.ByCurrency, c)
	}
	if err := curRows.Err(); err != nil {
		return nil, fmt.Errorf("currency split rows: %w", err)
	}

	return report, nil
}

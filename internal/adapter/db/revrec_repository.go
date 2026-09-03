package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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
	// A zero version means the caller didn't stamp one (legacy call path); default
	// to V1 (cash) so the NOT NULL column always gets a meaningful value.
	version := schedule.AccountingVersion
	if version == 0 {
		version = domain.AccountingModelV1
	}
	query := `
		INSERT INTO revenue_schedules (id, tenant_id, invoice_id, subscription_id, entity_id, total_amount, currency, start_date, end_date, status, accounting_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.ExecContext(ctx, query,
		schedule.ID, schedule.TenantID, schedule.InvoiceID, schedule.SubscriptionID, schedule.EntityID, schedule.TotalAmount,
		schedule.Currency, schedule.StartDate, schedule.EndDate, schedule.Status, version, schedule.CreatedAt, schedule.UpdatedAt,
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

// ClaimDueEvents atomically claims every due pending event (status ->
// 'processing') and returns the claimed rows. The single UPDATE ... RETURNING
// is race-free: concurrent workers serialize on the row locks and the loser
// re-evaluates the WHERE against the already-flipped status, so two runners
// always get disjoint sets (F2 — same idiom as MandateRepository.
// ClaimDueForDebit). Claims older than an hour are requeued first, so a
// worker that crashed mid-claim can't strand its events in 'processing'.
func (r *RevRecRepository) ClaimDueEvents(ctx context.Context, date time.Time) ([]*domain.RecognitionEvent, error) {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE recognition_events SET status = 'pending', claimed_at = NULL
		 WHERE status = 'processing' AND claimed_at < NOW() - INTERVAL '1 hour'`); err != nil {
		return nil, fmt.Errorf("requeue stale recognition claims: %w", err)
	}
	// Join the parent schedule so each claimed event carries its entity_id — the
	// recognition worker posts DR Deferred / CR Recognized to that entity's
	// ledger (Multi-Entity Books). UPDATE ... FROM lets RETURNING project the
	// schedule column; the join is on the PK so it can't fan out the claim.
	query := `
		UPDATE recognition_events re SET status = 'processing', claimed_at = NOW()
		FROM revenue_schedules rs
		WHERE re.revenue_schedule_id = rs.id AND re.recognition_date <= $1 AND re.status = 'pending'
		RETURNING re.id, re.revenue_schedule_id, re.tenant_id, rs.entity_id, re.amount, re.recognition_date, re.status, re.ledger_tx_id, re.created_at
	`
	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var events []*domain.RecognitionEvent
	for rows.Next() {
		var e domain.RecognitionEvent
		var ledgerTxID sql.NullString // Use NullString for UUID scan safety
		var entityID sql.NullString

		if err := rows.Scan(&e.ID, &e.RevenueScheduleID, &e.TenantID, &entityID, &e.Amount, &e.RecognitionDate, &e.Status, &ledgerTxID, &e.CreatedAt); err != nil {
			slog.Error("revrec claim scan failed", "error", err)
			return nil, err
		}

		if entityID.Valid {
			u := uuid.MustParse(entityID.String)
			e.EntityID = &u
		}
		if ledgerTxID.Valid {
			u := uuid.MustParse(ledgerTxID.String)
			e.LedgerTxID = &u
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// MarkEventRecognized / MarkEventFailed only transition events THIS worker
// claimed ('processing'). The guard is what makes F2 safe end-to-end: a
// duplicate posting error in a losing worker can no longer demote an event
// the winner already recognized.
func (r *RevRecRepository) MarkEventRecognized(ctx context.Context, eventID uuid.UUID, ledgerTxID uuid.UUID) error {
	query := `UPDATE recognition_events SET status = 'recognized', ledger_tx_id = $1, claimed_at = NULL
		WHERE id = $2 AND status = 'processing'`
	_, err := r.db.ExecContext(ctx, query, ledgerTxID, eventID)
	return err
}

// The reason is logged by the caller; the table has no failure-reason column.
func (r *RevRecRepository) MarkEventFailed(ctx context.Context, eventID uuid.UUID, _ string) error {
	query := `UPDATE recognition_events SET status = 'failed', claimed_at = NULL
		WHERE id = $1 AND status = 'processing'`
	_, err := r.db.ExecContext(ctx, query, eventID)
	return err
}

// GetActiveSchedulesBySubscription returns a subscription's active schedules
// (tenant-scoped) for an unwind (ENG-147).
func (r *RevRecRepository) GetActiveSchedulesBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.RevenueSchedule, error) {
	query := `
		SELECT id, tenant_id, invoice_id, subscription_id, entity_id, total_amount, currency, start_date, end_date, status, accounting_version, created_at, updated_at
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
		var entityID sql.NullString
		if err := rows.Scan(&s.ID, &s.TenantID, &s.InvoiceID, &s.SubscriptionID, &entityID, &s.TotalAmount,
			&s.Currency, &s.StartDate, &s.EndDate, &s.Status, &s.AccountingVersion, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if entityID.Valid {
			u := uuid.MustParse(entityID.String)
			s.EntityID = &u
		}
		schedules = append(schedules, &s)
	}
	return schedules, rows.Err()
}

// GetActiveScheduleByInvoice returns the active schedule for an invoice, or nil
// when there is none (one-off invoice, or already fully recognized/canceled).
func (r *RevRecRepository) GetActiveScheduleByInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) (*domain.RevenueSchedule, error) {
	query := `
		SELECT id, tenant_id, invoice_id, subscription_id, entity_id, total_amount, currency, start_date, end_date, status, accounting_version, created_at, updated_at
		FROM revenue_schedules
		WHERE tenant_id = $1 AND invoice_id = $2 AND status = 'active'
		LIMIT 1
	`
	var s domain.RevenueSchedule
	var entityID sql.NullString
	err := r.db.QueryRowContext(ctx, query, tenantID, invoiceID).Scan(
		&s.ID, &s.TenantID, &s.InvoiceID, &s.SubscriptionID, &entityID, &s.TotalAmount,
		&s.Currency, &s.StartDate, &s.EndDate, &s.Status, &s.AccountingVersion, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if entityID.Valid {
		u := uuid.MustParse(entityID.String)
		s.EntityID = &u
	}
	return &s, nil
}

// SumRecognizedByInvoice returns the total amount already RECOGNIZED (status
// 'recognized') across the invoice's revenue schedule(s). It is what the accrual
// write-off split (accrual epic #466 / #477) needs to decide how much of an
// uncollectible invoice is bad-debt expense (recognized) vs a Deferred reversal
// (unrecognized). Returns 0 when the invoice has no schedule (e.g. an unpaid
// invoice under the current cash model), which keeps the write-off behavior
// byte-identical to today.
func (r *RevRecRepository) SumRecognizedByInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) (int64, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(e.amount), 0)
		FROM recognition_events e
		JOIN revenue_schedules s ON s.id = e.revenue_schedule_id
		WHERE s.tenant_id = $1 AND s.invoice_id = $2 AND e.status = 'recognized'`,
		tenantID, invoiceID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Int64, nil
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

// GetRecognizedEventsBySubscription returns every RECOGNIZED event across all
// of a subscription's schedules (any schedule status — a fully-recognized or
// canceled schedule still owns its recognized history), newest recognition
// first so a downgrade reversal claws back the most recently recognized
// revenue. Feeds ReverseRecognizedForDowngrade's cap: only revenue that
// genuinely recognized (and hasn't already been reversed) may fund the
// DR Recognized Revenue leg of a downgrade credit.
func (r *RevRecRepository) GetRecognizedEventsBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.RecognitionEvent, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.id, e.revenue_schedule_id, e.tenant_id, e.amount, e.recognition_date, e.status, e.ledger_tx_id, e.created_at
		   FROM recognition_events e
		   JOIN revenue_schedules s ON s.id = e.revenue_schedule_id
		  WHERE e.tenant_id = $1 AND s.subscription_id = $2 AND e.status = 'recognized'
		  ORDER BY e.recognition_date DESC, e.id`,
		tenantID, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("query recognized events for subscription %s: %w", subscriptionID, err)
	}
	defer func() { _ = rows.Close() }()

	var events []*domain.RecognitionEvent
	for rows.Next() {
		var e domain.RecognitionEvent
		var ledgerTxID uuid.NullUUID
		if err := rows.Scan(&e.ID, &e.RevenueScheduleID, &e.TenantID, &e.Amount, &e.RecognitionDate, &e.Status, &ledgerTxID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recognized event: %w", err)
		}
		if ledgerTxID.Valid {
			e.LedgerTxID = &ledgerTxID.UUID
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// MarkEventReversed flips a RECOGNIZED event to reversed (a downgrade credit
// clawed its revenue back). Scoped to status='recognized' so a pending or
// already-reversed event can't be double-counted.
func (r *RevRecRepository) MarkEventReversed(ctx context.Context, eventID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE recognition_events SET status = 'reversed' WHERE id = $1 AND status = 'recognized'`, eventID)
	return err
}

// SplitRecognizedEvent reverses PART of a recognized event (the boundary of a
// partial downgrade reversal): atomically shrink the recognized event by
// reverseAmount and insert a sibling 'reversed' event carrying that amount, so
// per-schedule totals are preserved and the reversed portion can never be
// reversed again. No-op (both statements) when the event is not recognized or
// too small — the CTE reads the row once and both writes key off it.
func (r *RevRecRepository) SplitRecognizedEvent(ctx context.Context, eventID, newEventID uuid.UUID, reverseAmount int64) error {
	_, err := r.db.ExecContext(ctx, `
		WITH orig AS (
			SELECT id, revenue_schedule_id, tenant_id, recognition_date
			  FROM recognition_events
			 WHERE id = $1 AND status = 'recognized' AND amount > $3
		), ins AS (
			INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
			SELECT $2, revenue_schedule_id, tenant_id, $3, recognition_date, 'reversed', NOW() FROM orig
		)
		UPDATE recognition_events e SET amount = e.amount - $3
		  FROM orig WHERE e.id = orig.id`,
		eventID, newEventID, reverseAmount)
	if err != nil {
		return fmt.Errorf("split recognized event %s: %w", eventID, err)
	}
	return nil
}

// AddScheduleDebt records that `amount` of the subscription's UNSCHEDULED
// deferral was consumed by a downgrade credit (ENG-191f). The next schedule(s)
// created for this subscription shrink by this debt so they never recognize
// revenue whose deferral was already credited back.
func (r *RevRecRepository) AddScheduleDebt(ctx context.Context, subscriptionID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE subscriptions SET revrec_schedule_debt = revrec_schedule_debt + $2 WHERE id = $1`,
		subscriptionID, amount)
	if err != nil {
		return fmt.Errorf("add schedule debt for subscription %s: %w", subscriptionID, err)
	}
	return nil
}

// ConsumeScheduleDebt atomically draws up to `max` from the subscription's
// schedule debt and returns the amount consumed (0 when there is no debt).
// The row lock in the CTE makes concurrent schedule creations consume disjoint
// portions.
func (r *RevRecRepository) ConsumeScheduleDebt(ctx context.Context, subscriptionID uuid.UUID, max int64) (int64, error) {
	if max <= 0 {
		return 0, nil
	}
	var consumed int64
	err := r.db.QueryRowContext(ctx, `
		WITH cur AS (
			SELECT id, revrec_schedule_debt AS debt FROM subscriptions
			 WHERE id = $1 AND revrec_schedule_debt > 0
			 FOR UPDATE
		)
		UPDATE subscriptions s
		   SET revrec_schedule_debt = s.revrec_schedule_debt - LEAST(cur.debt, $2)
		  FROM cur WHERE s.id = cur.id
		RETURNING LEAST(cur.debt, $2)`,
		subscriptionID, max).Scan(&consumed)
	if err == sql.ErrNoRows {
		return 0, nil // no debt
	}
	if err != nil {
		return 0, fmt.Errorf("consume schedule debt for subscription %s: %w", subscriptionID, err)
	}
	return consumed, nil
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

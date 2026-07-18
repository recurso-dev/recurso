package worker

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/export"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ExportWorker ships each tenant's general ledger to operator-owned object
// storage daily (Track D5): {prefix}{tenant_id}/general-ledger-{date}.csv,
// same columns as GET /v1/ledger/export. Overwrites are idempotent (same
// key per day), so a re-run after a crash is safe.

// exportTenantLister supplies the tenants to export.
type exportTenantLister interface {
	ListTenants(ctx context.Context) ([]*domain.Tenant, error)
}

// exportLedgerSource supplies the GL rows; *service.LedgerService.
type exportLedgerSource interface {
	GeneralLedger(ctx context.Context, tenantID uuid.UUID) ([]domain.GeneralLedgerRow, error)
}

type ExportWorker struct {
	tenants  exportTenantLister
	ledger   exportLedgerSource
	s3       *export.S3Client
	prefix   string
	interval time.Duration
	ticker   *time.Ticker
	done     chan bool
	stopOnce sync.Once
	now      func() time.Time
}

func NewExportWorker(tenants exportTenantLister, ledger exportLedgerSource, s3 *export.S3Client, prefix string) *ExportWorker {
	return &ExportWorker{
		tenants:  tenants,
		ledger:   ledger,
		s3:       s3,
		prefix:   prefix,
		interval: 24 * time.Hour,
		done:     make(chan bool),
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (w *ExportWorker) Start() {
	w.ticker = time.NewTicker(w.interval)
	go func() {
		for {
			select {
			case <-w.done:
				return
			case <-w.ticker.C:
				if _, err := w.RunOnce(context.Background()); err != nil {
					slog.Error("s3 export sweep failed", "error", err)
				}
			}
		}
	}()
	slog.Info("s3 export worker started (daily)", "bucket", w.s3.Bucket, "prefix", w.prefix)
}

func (w *ExportWorker) Stop() {
	w.stopOnce.Do(func() {
		if w.ticker != nil {
			w.ticker.Stop()
		}
		close(w.done)
		slog.Info("s3 export worker stopped")
	})
}

// RunOnce exports every tenant's GL once; per-tenant failures log and
// continue. Returns the number of tenants exported.
func (w *ExportWorker) RunOnce(ctx context.Context) (int, error) {
	tenants, err := w.tenants.ListTenants(ctx)
	if err != nil {
		return 0, fmt.Errorf("s3 export: list tenants: %w", err)
	}
	date := w.now().Format("2006-01-02")
	exported := 0
	for _, tenant := range tenants {
		if err := w.exportTenant(ctx, tenant.ID, date); err != nil {
			slog.Error("s3 export failed for tenant", "tenant_id", tenant.ID, "error", err)
			continue
		}
		exported++
	}
	if exported > 0 {
		slog.Info("s3 export sweep complete", "tenants", exported, "date", date)
	}
	return exported, nil
}

func (w *ExportWorker) exportTenant(ctx context.Context, tenantID uuid.UUID, date string) error {
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	rows, err := w.ledger.GeneralLedger(tctx, tenantID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil // nothing posted yet; skip the empty file
	}

	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	_ = cw.Write([]string{
		"transaction_id", "timestamp", "code",
		"debit_account_code", "debit_account_name",
		"credit_account_code", "credit_account_name",
		"amount", "reference_id", "description",
	})
	for _, e := range rows {
		_ = cw.Write([]string{
			e.TransactionID.String(),
			e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
			strconv.Itoa(int(e.Code)),
			strconv.Itoa(e.DebitAccountCode), e.DebitAccountName,
			strconv.Itoa(e.CreditAccountCode), e.CreditAccountName,
			fmt.Sprintf("%d", e.Amount),
			e.ReferenceID.String(),
			e.Description,
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}

	key := fmt.Sprintf("%s%s/general-ledger-%s.csv", w.prefix, tenantID, date)
	return w.s3.PutObject(ctx, key, buf.Bytes(), "text/csv")
}

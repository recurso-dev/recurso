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
	GeneralLedger(ctx context.Context, tenantID uuid.UUID, ledgerFilter *uuid.UUID) ([]domain.GeneralLedgerRow, error)
}

// ExportUploader is the object-storage slice the worker needs; *export.S3Client.
// Exported so the per-tenant resolver (wired in main) can return one.
type ExportUploader interface {
	PutObject(ctx context.Context, key string, body []byte, contentType string) error
}

type ExportWorker struct {
	tenants exportTenantLister
	ledger  exportLedgerSource
	s3      *export.S3Client // env/default destination; may be nil when only BYO tenants export
	// s3For resolves a tenant's OWN storage (BYO), returning nil to fall back to
	// s3. Optional.
	s3For    func(ctx context.Context, tenantID uuid.UUID) ExportUploader
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
	bucket := "(per-tenant)"
	if w.s3 != nil {
		bucket = w.s3.Bucket
	}
	slog.Info("s3 export worker started (daily)", "bucket", bucket, "prefix", w.prefix)
}

// SetPerTenantStorage wires a resolver returning a tenant's own (BYO) storage,
// used in preference to the env destination. Returning nil falls back to env; a
// tenant with neither is skipped.
func (w *ExportWorker) SetPerTenantStorage(fn func(ctx context.Context, tenantID uuid.UUID) ExportUploader) {
	w.s3For = fn
}

// uploaderForTenant picks the tenant's own storage (BYO) when available, else
// the env destination (only when it's configured). nil means skip the tenant.
func (w *ExportWorker) uploaderForTenant(ctx context.Context, tenantID uuid.UUID) ExportUploader {
	if w.s3For != nil {
		if u := w.s3For(ctx, tenantID); u != nil {
			return u
		}
	}
	if w.s3 != nil && w.s3.Configured() {
		return w.s3
	}
	return nil
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
		uploaded, err := w.exportTenant(ctx, tenant.ID, date)
		if err != nil {
			slog.Error("s3 export failed for tenant", "tenant_id", tenant.ID, "error", err)
			continue
		}
		if uploaded {
			exported++
		}
	}
	if exported > 0 {
		slog.Info("s3 export sweep complete", "tenants", exported, "date", date)
	}
	return exported, nil
}

// exportTenant returns whether it actually uploaded (false when the tenant has
// no destination or an empty ledger — both non-errors).
func (w *ExportWorker) exportTenant(ctx context.Context, tenantID uuid.UUID, date string) (bool, error) {
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	// Resolve this tenant's destination (their own BYO bucket, else the env
	// bucket). A tenant with neither is skipped.
	uploader := w.uploaderForTenant(tctx, tenantID)
	if uploader == nil {
		return false, nil
	}
	rows, err := w.ledger.GeneralLedger(tctx, tenantID, nil)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil // nothing posted yet; skip the empty file
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
		return false, err
	}

	key := fmt.Sprintf("%s%s/general-ledger-%s.csv", w.prefix, tenantID, date)
	if err := uploader.PutObject(ctx, key, buf.Bytes(), "text/csv"); err != nil {
		return false, err
	}
	return true, nil
}

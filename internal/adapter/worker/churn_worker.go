package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

type ChurnWorker struct {
	churnService *service.ChurnService
	customerRepo port.CustomerRepository
	tenantRepo   *db.TenantRepository
	interval     time.Duration
}

func NewChurnWorker(
	churnService *service.ChurnService,
	customerRepo port.CustomerRepository,
	tenantRepo *db.TenantRepository,
	interval time.Duration,
) *ChurnWorker {
	return &ChurnWorker{
		churnService: churnService,
		customerRepo: customerRepo,
		tenantRepo:   tenantRepo,
		interval:     interval,
	}
}

func (w *ChurnWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("churn worker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.RunAnalysis(ctx)
		}
	}
}

func (w *ChurnWorker) RunAnalysis(ctx context.Context) {
	slog.Info("running churn analysis for all tenants")

	tenants, err := w.tenantRepo.ListTenants(ctx)
	if err != nil {
		slog.Error("failed to list tenants for churn analysis", "error", err)
		return
	}

	for _, tenant := range tenants {
		slog.Info("analyzing churn for tenant", "tenant_name", tenant.Name, "tenant_id", tenant.ID)
		w.AnalyzeTenantCustomers(ctx, tenant.ID)
	}

	slog.Info("churn analysis global scan completed")
}

// AnalyzeTenantCustomers runs analysis for a specific tenant
func (w *ChurnWorker) AnalyzeTenantCustomers(ctx context.Context, tenantID uuid.UUID) {
	// ChurnService reads through tenant-scoped repos and the worker's
	// background context carries no tenant — inject it, or every
	// AnalyzeCustomer call fails with "tenant_id missing" (the tenant-context
	// bug class; this meant churn scoring never ran).
	ctx = context.WithValue(ctx, domain.TenantIDKey, tenantID)
	offset := 0
	limit := 100

	for {
		customers, err := w.customerRepo.List(ctx, tenantID, domain.CustomerFilter{Limit: limit, Offset: offset})
		if err != nil {
			slog.Error("failed to list customers for churn analysis", "error", err)
			break
		}
		if len(customers) == 0 {
			break
		}

		for _, customer := range customers {
			if err := w.churnService.AnalyzeCustomer(ctx, customer.ID); err != nil {
				slog.Error("failed to analyze customer", "customer_id", customer.ID, "error", err)
			}
		}

		offset += limit
	}
}

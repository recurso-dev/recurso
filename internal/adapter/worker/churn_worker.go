package worker

import (
	"context"
	"log"
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

	log.Println("Churn Worker started")

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
	log.Println("Running Churn Analysis for all tenants...")

	tenants, err := w.tenantRepo.ListTenants(ctx)
	if err != nil {
		log.Printf("Failed to list tenants for churn analysis: %v", err)
		return
	}

	for _, tenant := range tenants {
		log.Printf("Analyzing Churn for Tenant: %s (%s)", tenant.Name, tenant.ID)
		w.AnalyzeTenantCustomers(ctx, tenant.ID)
	}

	log.Println("Churn Analysis: Global scan completed.")
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
			log.Printf("Failed to list customers for churn analysis: %v", err)
			break
		}
		if len(customers) == 0 {
			break
		}

		for _, customer := range customers {
			if err := w.churnService.AnalyzeCustomer(ctx, customer.ID); err != nil {
				log.Printf("Failed to analyze customer %s: %v", customer.ID, err)
			}
		}

		offset += limit
	}
}

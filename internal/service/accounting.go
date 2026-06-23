package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/accounting"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type AccountingService struct {
	gateway      port.AccountingGateway
	customerRepo port.CustomerRepository
	invoiceRepo  port.InvoiceRepository
	planRepo     port.PlanRepository
	connRepo     port.AccountingConnectionRepository
}

func NewAccountingService(
	gateway port.AccountingGateway,
	customerRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
	planRepo port.PlanRepository,
) *AccountingService {
	return &AccountingService{
		gateway:      gateway,
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
		planRepo:     planRepo,
	}
}

func (s *AccountingService) SetConnectionRepo(repo port.AccountingConnectionRepository) {
	s.connRepo = repo
}

func (s *AccountingService) SyncCustomer(ctx context.Context, customerID uuid.UUID) error {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return err
	}
	return s.gateway.SyncCustomer(ctx, customer)
}

func (s *AccountingService) SyncInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	invoice, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	return s.gateway.SyncInvoice(ctx, invoice)
}

func (s *AccountingService) SyncProduct(ctx context.Context, planID string) error {
	id, err := uuid.Parse(planID)
	if err != nil {
		return err
	}
	plan, err := s.planRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.gateway.SyncProduct(ctx, plan)
}

// SyncAllForTenant syncs all entities for a given tenant using the appropriate adapter.
func (s *AccountingService) SyncAllForTenant(ctx context.Context, tenantID uuid.UUID) error {
	if s.connRepo == nil {
		return fmt.Errorf("accounting connection repository not configured")
	}

	conns, err := s.connRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	for _, conn := range conns {
		if !conn.IsActive {
			continue
		}

		adapter := s.getAdapterForConnection(conn)
		if adapter == nil {
			continue
		}

		// Sync customers (paginated)
		customerOffset := 0
		customerLimit := 100
		for {
			customers, err := s.customerRepo.List(ctx, tenantID, domain.CustomerFilter{Limit: customerLimit, Offset: customerOffset})
			if err != nil {
				slog.Error("failed to list customers for sync", "error", err)
				break
			}
			if len(customers) == 0 {
				break
			}

			for _, customer := range customers {
				if err := adapter.SyncCustomer(ctx, customer); err != nil {
					s.logSyncResult(ctx, conn, "customer", customer.ID, "create", "error", err.Error())
					continue
				}
				s.logSyncResult(ctx, conn, "customer", customer.ID, "create", "success", "")
			}

			if len(customers) < customerLimit {
				break
			}
			customerOffset += customerLimit
		}

		// Sync invoices
		invoices, err := s.invoiceRepo.List(ctx, tenantID)
		if err != nil {
			slog.Error("failed to list invoices for sync", "error", err)
			continue
		}

		for _, invoice := range invoices {
			if err := adapter.SyncInvoice(ctx, invoice); err != nil {
				s.logSyncResult(ctx, conn, "invoice", invoice.ID, "create", "error", err.Error())
				continue
			}
			s.logSyncResult(ctx, conn, "invoice", invoice.ID, "create", "success", "")
		}

		// Update connection status
		now := time.Now()
		conn.LastSyncAt = &now
		conn.SyncStatus = "synced"
		_ = s.connRepo.Update(ctx, conn)
	}

	return nil
}

func (s *AccountingService) getAdapterForConnection(conn *domain.AccountingConnection) port.AccountingGateway {
	switch conn.Provider {
	case "quickbooks":
		adapter := accounting.NewQuickBooksAdapter(conn.AccessToken, conn.RealmID, false)
		return adapter
	case "xero":
		adapter := accounting.NewXeroAdapter(conn.AccessToken, conn.RealmID)
		return adapter
	case "tally":
		adapter := accounting.NewTallyAdapter("")
		return adapter
	default:
		return s.gateway // Fall back to default (mock)
	}
}

func (s *AccountingService) logSyncResult(ctx context.Context, conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, action, status, errMsg string) {
	if s.connRepo == nil {
		return
	}

	syncLog := &domain.AccountingSyncLog{
		ID:           uuid.New(),
		TenantID:     conn.TenantID,
		ConnectionID: conn.ID,
		EntityType:   entityType,
		EntityID:     entityID,
		Action:       action,
		Status:       status,
		ErrorMessage: errMsg,
		SyncedAt:     time.Now(),
	}

	if err := s.connRepo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("failed to create sync log", "error", err)
	}
}

package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type AccountingConnectionRepository interface {
	Create(ctx context.Context, conn *domain.AccountingConnection) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AccountingConnection, error)
	GetByTenantAndProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.AccountingConnection, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.AccountingConnection, error)
	Update(ctx context.Context, conn *domain.AccountingConnection) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetActiveConnections(ctx context.Context) ([]*domain.AccountingConnection, error)
	CreateSyncLog(ctx context.Context, log *domain.AccountingSyncLog) error
	ListSyncLogs(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.AccountingSyncLog, error)
}

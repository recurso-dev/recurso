package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/importer/chargebee"
)

// ChargebeeImportService is the preview (dry-run) half of the Chargebee → Recurso
// migration. It reuses the same customer/catalog listers and the shared
// import-ref store as the Stripe importer. Commit is a follow-up increment.
type ChargebeeImportService struct {
	customers importCustomers
	catalog   importCatalog
	refs      port.ImportRefRepository
}

func NewChargebeeImportService(customers importCustomers, catalog importCatalog, refs port.ImportRefRepository) *ChargebeeImportService {
	return &ChargebeeImportService{customers: customers, catalog: catalog, refs: refs}
}

// Preview returns the dry-run plan for exp against the tenant's current state.
func (s *ChargebeeImportService) Preview(ctx context.Context, tenantID uuid.UUID, exp *chargebee.Export) (*chargebee.ImportPlan, error) {
	existing := chargebee.Existing{
		CustomerEmails: map[string]bool{},
		PlanCodes:      map[string]bool{},
		ImportedIDs:    map[string]bool{},
	}

	customers, err := s.customers.ListCustomers(ctx, tenantID, domain.CustomerFilter{Limit: importExistingScanLimit})
	if err != nil {
		return nil, err
	}
	for _, c := range customers {
		if c.Email != "" {
			existing.CustomerEmails[strings.ToLower(strings.TrimSpace(c.Email))] = true
		}
	}

	plans, err := s.catalog.ListPlans(ctx, tenantID, domain.PlanFilter{Limit: importExistingScanLimit})
	if err != nil {
		return nil, err
	}
	for _, p := range plans {
		if p.Code != "" {
			existing.PlanCodes[p.Code] = true
		}
	}

	imported, err := s.refs.ListExternalIDs(ctx, tenantID, domain.ImportSourceChargebee)
	if err != nil {
		return nil, err
	}
	existing.ImportedIDs = imported

	return chargebee.BuildPlan(exp, existing), nil
}

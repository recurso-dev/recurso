package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	stripeimport "github.com/recurso-dev/recurso/internal/importer/stripe"
)

// importExistingScanLimit bounds how many existing customers/plans are loaded to
// build the conflict sets. A safety ceiling, not a business limit.
const importExistingScanLimit = 100000

// importCustomers / importCatalog are the narrow slices of CustomerService and
// CatalogService the importer needs. Concrete services satisfy them; tests use
// fakes.
type importCustomers interface {
	ListCustomers(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error)
	CreateCustomer(ctx context.Context, input CreateCustomerInput) (*domain.Customer, error)
}

type importCatalog interface {
	ListPlans(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error)
	CreatePlan(ctx context.Context, input CreatePlanInput) (*domain.Plan, error)
}

// StripeImportService orchestrates a Stripe → Recurso migration: a no-side-effect
// preview (dry run) and an idempotent commit. This increment commits customers
// and plans (both side-effect-free creates); subscriptions — which must be
// imported WITHOUT re-triggering billing — are a separate increment.
type StripeImportService struct {
	customers importCustomers
	catalog   importCatalog
	refs      port.ImportRefRepository
}

func NewStripeImportService(customers importCustomers, catalog importCatalog, refs port.ImportRefRepository) *StripeImportService {
	return &StripeImportService{customers: customers, catalog: catalog, refs: refs}
}

// Preview returns the dry-run plan for exp against the tenant's current state.
func (s *StripeImportService) Preview(ctx context.Context, tenantID uuid.UUID, exp *stripeimport.Export) (*stripeimport.Plan, error) {
	existing, err := s.buildExisting(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return stripeimport.BuildPlan(exp, existing), nil
}

// CommitFailure records one object that could not be created during a commit.
type CommitFailure struct {
	Kind     string `json:"kind"`
	StripeID string `json:"stripe_id"`
	Error    string `json:"error"`
}

// CommitResult is the outcome of a commit: the plan that was executed, the count
// actually created per kind, and any per-object failures (a commit is
// best-effort per object so one bad record doesn't abort the whole import).
type CommitResult struct {
	Plan     *stripeimport.Plan `json:"plan"`
	Created  map[string]int     `json:"created"`
	Failures []CommitFailure    `json:"failures"`
}

// Commit imports exp: it recomputes the plan (so it always acts on the current
// state, never a stale client plan) and creates every "create" customer and
// plan, recording an idempotency ref for each. Re-running is safe — already
// imported ids and existing emails/plan-codes are skipped by the planner.
func (s *StripeImportService) Commit(ctx context.Context, tenantID uuid.UUID, exp *stripeimport.Export) (*CommitResult, error) {
	existing, err := s.buildExisting(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan := stripeimport.BuildPlan(exp, existing)
	result := &CommitResult{Plan: plan, Created: map[string]int{}}

	// Index source objects so a plan item (keyed by Stripe id) maps back to its
	// source for building the create input.
	custByID := map[string]stripeimport.Customer{}
	for _, c := range exp.Customers {
		custByID[c.ID] = c
	}
	priceByID := map[string]stripeimport.Price{}
	for _, p := range exp.Prices {
		priceByID[p.ID] = p
	}
	prodByID := map[string]stripeimport.Product{}
	for _, p := range exp.Products {
		prodByID[p.ID] = p
	}

	for _, item := range plan.Items {
		if item.Action != stripeimport.ActionCreate {
			continue
		}
		switch item.Kind {
		case stripeimport.KindCustomer:
			s.commitCustomer(ctx, tenantID, custByID[item.StripeID], result)
		case stripeimport.KindPlan:
			pr := priceByID[item.StripeID]
			s.commitPlan(ctx, tenantID, prodByID[pr.Product], pr, result)
		default:
			// Subscriptions / payment methods are not committed in this increment.
		}
	}
	return result, nil
}

func (s *StripeImportService) commitCustomer(ctx context.Context, tenantID uuid.UUID, c stripeimport.Customer, result *CommitResult) {
	m := stripeimport.MapCustomer(c)
	created, err := s.customers.CreateCustomer(ctx, CreateCustomerInput{
		TenantID: tenantID,
		Email:    m.Email,
		Name:     m.Name,
		Country:  m.Country,
	})
	if err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(stripeimport.KindCustomer), StripeID: c.ID, Error: err.Error()})
		return
	}
	if err := s.recordRef(ctx, tenantID, domain.ImportKindCustomer, c.ID, created.ID); err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(stripeimport.KindCustomer), StripeID: c.ID, Error: err.Error()})
		return
	}
	result.Created[string(stripeimport.KindCustomer)]++
}

func (s *StripeImportService) commitPlan(ctx context.Context, tenantID uuid.UUID, prod stripeimport.Product, pr stripeimport.Price, result *CommitResult) {
	m, ok := stripeimport.MapPlan(prod, pr)
	if !ok {
		// Should not happen (BuildPlan only marks mappable prices as create), but
		// guard rather than create a malformed plan.
		result.Failures = append(result.Failures, CommitFailure{Kind: string(stripeimport.KindPlan), StripeID: pr.ID, Error: "price is not mappable to a plan"})
		return
	}
	created, err := s.catalog.CreatePlan(ctx, CreatePlanInput{
		TenantID:      tenantID,
		Name:          m.Name,
		Code:          m.Code,
		IntervalUnit:  m.IntervalUnit,
		IntervalCount: m.IntervalCount,
		Amount:        m.Amount,
		Currency:      m.Currency,
	})
	if err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(stripeimport.KindPlan), StripeID: pr.ID, Error: err.Error()})
		return
	}
	if err := s.recordRef(ctx, tenantID, domain.ImportKindPlan, pr.ID, created.ID); err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(stripeimport.KindPlan), StripeID: pr.ID, Error: err.Error()})
		return
	}
	result.Created[string(stripeimport.KindPlan)]++
}

// recordRef persists the idempotency mapping. A duplicate (a concurrent run that
// already recorded it) is treated as success — the object is imported.
func (s *StripeImportService) recordRef(ctx context.Context, tenantID uuid.UUID, kind, externalID string, recursoID uuid.UUID) error {
	err := s.refs.Create(ctx, &domain.ImportExternalRef{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Source:     domain.ImportSourceStripe,
		Kind:       kind,
		ExternalID: externalID,
		RecursoID:  recursoID,
	})
	if errors.Is(err, domain.ErrDuplicateImportRef) {
		return nil
	}
	return err
}

// buildExisting loads the tenant's current customer emails, plan codes, and
// previously-imported Stripe ids so the planner links/skips rather than
// duplicating.
func (s *StripeImportService) buildExisting(ctx context.Context, tenantID uuid.UUID) (stripeimport.Existing, error) {
	existing := stripeimport.Existing{
		CustomerEmails:    map[string]bool{},
		PlanCodes:         map[string]bool{},
		ImportedStripeIDs: map[string]bool{},
	}

	customers, err := s.customers.ListCustomers(ctx, tenantID, domain.CustomerFilter{Limit: importExistingScanLimit})
	if err != nil {
		return existing, err
	}
	for _, c := range customers {
		if c.Email != "" {
			existing.CustomerEmails[strings.ToLower(strings.TrimSpace(c.Email))] = true
		}
	}

	plans, err := s.catalog.ListPlans(ctx, tenantID, domain.PlanFilter{Limit: importExistingScanLimit})
	if err != nil {
		return existing, err
	}
	for _, p := range plans {
		if p.Code != "" {
			existing.PlanCodes[p.Code] = true
		}
	}

	imported, err := s.refs.ListExternalIDs(ctx, tenantID, domain.ImportSourceStripe)
	if err != nil {
		return existing, err
	}
	existing.ImportedStripeIDs = imported
	return existing, nil
}

package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	stripeimport "github.com/recurso-dev/recurso/internal/importer/stripe"
)

// importExistingScanLimit bounds how many existing customers/plans are loaded to
// build the conflict sets. A safety ceiling, not a business limit.
const importExistingScanLimit = 100000

// importCustomers / importCatalog / importSubscriptions are the narrow slices of
// the concrete services + subscription repo the importer needs. Concrete types
// satisfy them; tests use fakes.
type importCustomers interface {
	ListCustomers(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error)
	CreateCustomer(ctx context.Context, input CreateCustomerInput) (*domain.Customer, error)
}

type importCatalog interface {
	ListPlans(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error)
	CreatePlan(ctx context.Context, input CreatePlanInput) (*domain.Plan, error)
}

// importSubscriptions is the direct-insert path used to import a subscription in
// its CURRENT billing state. It deliberately bypasses SubscriptionService.Create
// (which would generate a first invoice + ledger postings) — a migration must
// not re-bill customers Stripe has already billed.
type importSubscriptions interface {
	Create(ctx context.Context, sub *domain.Subscription) error
}

// StripeImportService orchestrates a Stripe → Recurso migration: a no-side-effect
// preview (dry run) and an idempotent commit of customers, plans, and
// subscriptions. Payment methods are not imported — card data can't be migrated
// from a static export (it lives at the gateway).
type StripeImportService struct {
	customers importCustomers
	catalog   importCatalog
	subs      importSubscriptions
	refs      port.ImportRefRepository
}

func NewStripeImportService(customers importCustomers, catalog importCatalog, subs importSubscriptions, refs port.ImportRefRepository) *StripeImportService {
	return &StripeImportService{customers: customers, catalog: catalog, subs: subs, refs: refs}
}

// Preview returns the dry-run plan for exp against the tenant's current state.
func (s *StripeImportService) Preview(ctx context.Context, tenantID uuid.UUID, exp *stripeimport.Export) (*stripeimport.Plan, error) {
	st, err := s.loadState(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return stripeimport.BuildPlan(exp, st.existing), nil
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

// importState is the tenant's current data, loaded once per operation: the sets
// the planner needs plus the id-resolution maps a subscription import needs to
// point at the right Recurso customer/plan.
type importState struct {
	existing      stripeimport.Existing
	custEmailToID map[string]uuid.UUID // lower(email) -> recurso customer id
	planCodeToID  map[string]uuid.UUID // plan code    -> recurso plan id
	custRefID     map[string]uuid.UUID // stripe customer id -> recurso id (prior imports)
	planRefID     map[string]uuid.UUID // stripe price id    -> recurso plan id (prior imports)
}

// Commit imports exp: it recomputes the plan (so it always acts on current
// state, never a stale client plan), creates every "create" customer, plan, and
// subscription, and records an idempotency ref for each. Re-running is safe.
func (s *StripeImportService) Commit(ctx context.Context, tenantID uuid.UUID, exp *stripeimport.Export) (*CommitResult, error) {
	st, err := s.loadState(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan := stripeimport.BuildPlan(exp, st.existing)
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
	subByID := map[string]stripeimport.Subscription{}
	for _, sub := range exp.Subscriptions {
		subByID[sub.ID] = sub
	}

	// Resolution maps (stripe id -> recurso id), seeded from prior imports.
	custRecursoID := cloneIDMap(st.custRefID)
	planRecursoID := cloneIDMap(st.planRefID)

	// Pass 1: create customers and plans, extending the resolution maps.
	for _, item := range plan.Items {
		if item.Action != stripeimport.ActionCreate {
			continue
		}
		switch item.Kind {
		case stripeimport.KindCustomer:
			s.commitCustomer(ctx, tenantID, custByID[item.StripeID], custRecursoID, result)
		case stripeimport.KindPlan:
			pr := priceByID[item.StripeID]
			s.commitPlan(ctx, tenantID, prodByID[pr.Product], pr, planRecursoID, result)
		}
	}

	// Fold in objects that resolve to EXISTING records (linked by email / plan
	// code) so subscriptions on them can be pointed at the right Recurso id.
	for _, c := range exp.Customers {
		if _, ok := custRecursoID[c.ID]; !ok {
			if id, ok2 := st.custEmailToID[strings.ToLower(strings.TrimSpace(c.Email))]; ok2 {
				custRecursoID[c.ID] = id
			}
		}
	}
	for _, pr := range exp.Prices {
		if _, ok := planRecursoID[pr.ID]; !ok {
			if id, ok2 := st.planCodeToID[stripeimport.PlanCode(pr)]; ok2 {
				planRecursoID[pr.ID] = id
			}
		}
	}

	// Pass 2: import subscriptions in their current billing state.
	for _, item := range plan.Items {
		if item.Action != stripeimport.ActionCreate || item.Kind != stripeimport.KindSubscription {
			continue
		}
		s.commitSubscription(ctx, tenantID, subByID[item.StripeID], custRecursoID, planRecursoID, result)
	}

	return result, nil
}

func (s *StripeImportService) commitCustomer(ctx context.Context, tenantID uuid.UUID, c stripeimport.Customer, resolve map[string]uuid.UUID, result *CommitResult) {
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
	resolve[c.ID] = created.ID
	result.Created[string(stripeimport.KindCustomer)]++
}

func (s *StripeImportService) commitPlan(ctx context.Context, tenantID uuid.UUID, prod stripeimport.Product, pr stripeimport.Price, resolve map[string]uuid.UUID, result *CommitResult) {
	m, ok := stripeimport.MapPlan(prod, pr)
	if !ok {
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
	resolve[pr.ID] = created.ID
	result.Created[string(stripeimport.KindPlan)]++
}

// commitSubscription imports a subscription in its current billing state via a
// direct repository insert — NO invoice/charge/ledger side effects. The period
// and billing anchor come straight from Stripe, so Recurso takes over billing at
// the next renewal instead of re-charging the current cycle.
func (s *StripeImportService) commitSubscription(ctx context.Context, tenantID uuid.UUID, sub stripeimport.Subscription, custResolve, planResolve map[string]uuid.UUID, result *CommitResult) {
	fail := func(msg string) {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(stripeimport.KindSubscription), StripeID: sub.ID, Error: msg})
	}

	customerID, ok := custResolve[sub.Customer]
	if !ok {
		fail("could not resolve the subscription's customer")
		return
	}
	priceID := stripeimport.SubscriptionPriceID(sub)
	planID, ok := planResolve[priceID]
	if !ok {
		fail("could not resolve the subscription's plan")
		return
	}
	status, ok := stripeimport.MapSubStatus(sub.Status)
	if !ok {
		fail("Stripe status " + sub.Status + " is not importable")
		return
	}

	now := time.Now().UTC()
	rec := &domain.Subscription{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		CustomerID:           customerID,
		PlanID:               planID,
		Status:               domain.SubscriptionStatus(status),
		CurrentPeriodStart:   unixOr(sub.CurrentPeriodStart, now),
		CurrentPeriodEnd:     unixOr(sub.CurrentPeriodEnd, now),
		BillingAnchor:        unixOr(sub.CurrentPeriodEnd, now),
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		StripeSubscriptionID: sub.ID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if status == "trialing" && sub.TrialEnd > 0 {
		t := time.Unix(sub.TrialEnd, 0).UTC()
		rec.TrialEnd = &t
	}

	if err := s.subs.Create(ctx, rec); err != nil {
		fail(err.Error())
		return
	}
	if err := s.recordRef(ctx, tenantID, domain.ImportKindSubscription, sub.ID, rec.ID); err != nil {
		fail(err.Error())
		return
	}
	result.Created[string(stripeimport.KindSubscription)]++
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

// loadState reads the tenant's current customers, plans, and prior import refs
// once, building both the planner's Existing sets and the id-resolution maps.
func (s *StripeImportService) loadState(ctx context.Context, tenantID uuid.UUID) (*importState, error) {
	st := &importState{
		existing: stripeimport.Existing{
			CustomerEmails:    map[string]bool{},
			PlanCodes:         map[string]bool{},
			ImportedStripeIDs: map[string]bool{},
		},
		custEmailToID: map[string]uuid.UUID{},
		planCodeToID:  map[string]uuid.UUID{},
		custRefID:     map[string]uuid.UUID{},
		planRefID:     map[string]uuid.UUID{},
	}

	customers, err := s.customers.ListCustomers(ctx, tenantID, domain.CustomerFilter{Limit: importExistingScanLimit})
	if err != nil {
		return nil, err
	}
	for _, c := range customers {
		if c.Email != "" {
			key := strings.ToLower(strings.TrimSpace(c.Email))
			st.existing.CustomerEmails[key] = true
			st.custEmailToID[key] = c.ID
		}
	}

	plans, err := s.catalog.ListPlans(ctx, tenantID, domain.PlanFilter{Limit: importExistingScanLimit})
	if err != nil {
		return nil, err
	}
	for _, p := range plans {
		if p.Code != "" {
			st.existing.PlanCodes[p.Code] = true
			st.planCodeToID[p.Code] = p.ID
		}
	}

	refs, err := s.refs.ListRefs(ctx, tenantID, domain.ImportSourceStripe)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		st.existing.ImportedStripeIDs[ref.ExternalID] = true
		switch ref.Kind {
		case domain.ImportKindCustomer:
			st.custRefID[ref.ExternalID] = ref.RecursoID
		case domain.ImportKindPlan:
			st.planRefID[ref.ExternalID] = ref.RecursoID
		}
	}
	return st, nil
}

func cloneIDMap(m map[string]uuid.UUID) map[string]uuid.UUID {
	out := make(map[string]uuid.UUID, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// unixOr converts a Stripe unix-seconds timestamp to UTC time, falling back to
// `fallback` when the timestamp is absent (0) — e.g. a slim export.
func unixOr(sec int64, fallback time.Time) time.Time {
	if sec <= 0 {
		return fallback
	}
	return time.Unix(sec, 0).UTC()
}

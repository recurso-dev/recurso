package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/importer/chargebee"
)

// ChargebeeImportService orchestrates the Chargebee → Recurso migration: a
// no-side-effect preview and an idempotent commit of customers, plans, and
// subscriptions. It reuses the same customer/catalog listers, subscription
// repo, and shared import-ref store as the Stripe importer.
type ChargebeeImportService struct {
	customers importCustomers
	catalog   importCatalog
	subs      importSubscriptions
	refs      port.ImportRefRepository
}

func NewChargebeeImportService(customers importCustomers, catalog importCatalog, subs importSubscriptions, refs port.ImportRefRepository) *ChargebeeImportService {
	return &ChargebeeImportService{customers: customers, catalog: catalog, subs: subs, refs: refs}
}

// Preview returns the dry-run plan for exp against the tenant's current state.
func (s *ChargebeeImportService) Preview(ctx context.Context, tenantID uuid.UUID, exp *chargebee.Export) (*chargebee.ImportPlan, error) {
	st, err := s.loadState(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return chargebee.BuildPlan(exp, st.existing), nil
}

// ChargebeeCommitResult is the outcome of a commit: the plan executed, the count
// created per kind, and any per-object failures.
type ChargebeeCommitResult struct {
	Plan     *chargebee.ImportPlan `json:"plan"`
	Created  map[string]int        `json:"created"`
	Failures []CommitFailure       `json:"failures"`
}

type chargebeeState struct {
	existing      chargebee.Existing
	custEmailToID map[string]uuid.UUID
	planCodeToID  map[string]uuid.UUID
	custRefID     map[string]uuid.UUID
	planRefID     map[string]uuid.UUID
}

// Commit imports exp idempotently: customers, plans, then subscriptions (in
// current billing state, no invoice/charge/ledger side effects). Re-running is
// safe.
func (s *ChargebeeImportService) Commit(ctx context.Context, tenantID uuid.UUID, exp *chargebee.Export) (*ChargebeeCommitResult, error) {
	st, err := s.loadState(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan := chargebee.BuildPlan(exp, st.existing)
	result := &ChargebeeCommitResult{Plan: plan, Created: map[string]int{}}

	custByID := map[string]chargebee.Customer{}
	for _, c := range exp.Customers {
		custByID[c.ID] = c
	}
	planByID := map[string]chargebee.Plan{}
	for _, p := range exp.Plans {
		planByID[p.ID] = p
	}
	subByID := map[string]chargebee.Subscription{}
	for _, sub := range exp.Subscriptions {
		subByID[sub.ID] = sub
	}

	custRecursoID := cloneIDMap(st.custRefID)
	planRecursoID := cloneIDMap(st.planRefID)

	// Pass 1: create customers and plans.
	for _, item := range plan.Items {
		if item.Action != chargebee.ActionCreate {
			continue
		}
		switch item.Kind {
		case chargebee.KindCustomer:
			s.commitCustomer(ctx, tenantID, custByID[item.ChargebeeID], custRecursoID, result)
		case chargebee.KindPlan:
			s.commitPlan(ctx, tenantID, planByID[item.ChargebeeID], planRecursoID, result)
		}
	}

	// Fold in link-existing customers + existing-by-code plans.
	for _, c := range exp.Customers {
		if _, ok := custRecursoID[c.ID]; !ok {
			if id, ok2 := st.custEmailToID[strings.ToLower(strings.TrimSpace(c.Email))]; ok2 {
				custRecursoID[c.ID] = id
			}
		}
	}
	for _, pl := range exp.Plans {
		if _, ok := planRecursoID[pl.ID]; !ok {
			if id, ok2 := st.planCodeToID[chargebee.PlanCode(pl.ID)]; ok2 {
				planRecursoID[pl.ID] = id
			}
		}
	}

	// Pass 2: subscriptions.
	for _, item := range plan.Items {
		if item.Action != chargebee.ActionCreate || item.Kind != chargebee.KindSubscription {
			continue
		}
		s.commitSubscription(ctx, tenantID, subByID[item.ChargebeeID], custRecursoID, planRecursoID, result)
	}

	return result, nil
}

func (s *ChargebeeImportService) commitCustomer(ctx context.Context, tenantID uuid.UUID, c chargebee.Customer, resolve map[string]uuid.UUID, result *ChargebeeCommitResult) {
	m := chargebee.MapCustomer(c)
	created, err := s.customers.CreateCustomer(ctx, CreateCustomerInput{TenantID: tenantID, Email: m.Email, Name: m.Name, Country: m.Country})
	if err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(chargebee.KindCustomer), StripeID: c.ID, Error: err.Error()})
		return
	}
	if err := s.recordRef(ctx, tenantID, domain.ImportKindCustomer, c.ID, created.ID); err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(chargebee.KindCustomer), StripeID: c.ID, Error: err.Error()})
		return
	}
	resolve[c.ID] = created.ID
	result.Created[string(chargebee.KindCustomer)]++
}

func (s *ChargebeeImportService) commitPlan(ctx context.Context, tenantID uuid.UUID, pl chargebee.Plan, resolve map[string]uuid.UUID, result *ChargebeeCommitResult) {
	m, ok := chargebee.MapPlan(pl)
	if !ok {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(chargebee.KindPlan), StripeID: pl.ID, Error: "plan is not mappable"})
		return
	}
	created, err := s.catalog.CreatePlan(ctx, CreatePlanInput{
		TenantID: tenantID, Name: m.Name, Code: m.Code,
		IntervalUnit: m.IntervalUnit, IntervalCount: m.IntervalCount, Amount: m.Amount, Currency: m.Currency,
	})
	if err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(chargebee.KindPlan), StripeID: pl.ID, Error: err.Error()})
		return
	}
	if err := s.recordRef(ctx, tenantID, domain.ImportKindPlan, pl.ID, created.ID); err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(chargebee.KindPlan), StripeID: pl.ID, Error: err.Error()})
		return
	}
	resolve[pl.ID] = created.ID
	result.Created[string(chargebee.KindPlan)]++
}

// commitSubscription imports a subscription in its current billing state via a
// direct repository insert — NO invoice/charge/ledger side effects. Period +
// anchor come from Chargebee (non_renewing => cancels at term end).
func (s *ChargebeeImportService) commitSubscription(ctx context.Context, tenantID uuid.UUID, sub chargebee.Subscription, custResolve, planResolve map[string]uuid.UUID, result *ChargebeeCommitResult) {
	fail := func(msg string) {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(chargebee.KindSubscription), StripeID: sub.ID, Error: msg})
	}
	customerID, ok := custResolve[sub.CustomerID]
	if !ok {
		fail("could not resolve the subscription's customer")
		return
	}
	planID, ok := planResolve[sub.PlanID]
	if !ok {
		fail("could not resolve the subscription's plan")
		return
	}
	status, ok := chargebee.MapSubStatus(sub.Status)
	if !ok {
		fail("Chargebee status " + sub.Status + " is not importable")
		return
	}

	now := time.Now().UTC()
	rec := &domain.Subscription{
		ID:                     uuid.New(),
		TenantID:               tenantID,
		CustomerID:             customerID,
		PlanID:                 planID,
		Status:                 domain.SubscriptionStatus(status),
		CurrentPeriodStart:     unixOr(sub.CurrentTermStart, now),
		CurrentPeriodEnd:       unixOr(sub.CurrentTermEnd, now),
		BillingAnchor:          unixOr(sub.CurrentTermEnd, now),
		CancelAtPeriodEnd:      sub.Status == "non_renewing",
		RazorpaySubscriptionID: "",
		StripeSubscriptionID:   "",
		CreatedAt:              now,
		UpdatedAt:              now,
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
	result.Created[string(chargebee.KindSubscription)]++
}

func (s *ChargebeeImportService) recordRef(ctx context.Context, tenantID uuid.UUID, kind, externalID string, recursoID uuid.UUID) error {
	err := s.refs.Create(ctx, &domain.ImportExternalRef{
		ID: uuid.New(), TenantID: tenantID, Source: domain.ImportSourceChargebee,
		Kind: kind, ExternalID: externalID, RecursoID: recursoID,
	})
	if errors.Is(err, domain.ErrDuplicateImportRef) {
		return nil
	}
	return err
}

func (s *ChargebeeImportService) loadState(ctx context.Context, tenantID uuid.UUID) (*chargebeeState, error) {
	st := &chargebeeState{
		existing: chargebee.Existing{
			CustomerEmails: map[string]bool{},
			PlanCodes:      map[string]bool{},
			ImportedIDs:    map[string]bool{},
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

	refs, err := s.refs.ListRefs(ctx, tenantID, domain.ImportSourceChargebee)
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		st.existing.ImportedIDs[ref.ExternalID] = true
		switch ref.Kind {
		case domain.ImportKindCustomer:
			st.custRefID[ref.ExternalID] = ref.RecursoID
		case domain.ImportKindPlan:
			st.planRefID[ref.ExternalID] = ref.RecursoID
		}
	}
	return st, nil
}

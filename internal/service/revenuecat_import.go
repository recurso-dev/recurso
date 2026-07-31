package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/importer/revenuecat"
)

// RevenueCatImportService orchestrates the RevenueCat → Recurso migration:
// preview (dry run) and idempotent commit of customers, plans, and active
// subscriptions. Subscribers without an email can't become Recurso customers
// (RevenueCat identifies by app_user_id) — those surface as conflicts.
type RevenueCatImportService struct {
	customers importCustomers
	catalog   importCatalog
	subs      importSubscriptions
	refs      port.ImportRefRepository
}

func NewRevenueCatImportService(customers importCustomers, catalog importCatalog, subs importSubscriptions, refs port.ImportRefRepository) *RevenueCatImportService {
	return &RevenueCatImportService{customers: customers, catalog: catalog, subs: subs, refs: refs}
}

func (s *RevenueCatImportService) Preview(ctx context.Context, tenantID uuid.UUID, exp *revenuecat.Export) (*revenuecat.ImportPlan, error) {
	st, err := s.loadState(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return revenuecat.BuildPlan(exp, st.existing), nil
}

// RevenueCatCommitResult mirrors the other importers' commit result.
type RevenueCatCommitResult struct {
	Plan     *revenuecat.ImportPlan `json:"plan"`
	Created  map[string]int         `json:"created"`
	Failures []CommitFailure        `json:"failures"`
}

type revenuecatState struct {
	existing      revenuecat.Existing
	custEmailToID map[string]uuid.UUID
	planCodeToID  map[string]uuid.UUID
	custRefID     map[string]uuid.UUID
	planRefID     map[string]uuid.UUID
}

func (s *RevenueCatImportService) Commit(ctx context.Context, tenantID uuid.UUID, exp *revenuecat.Export) (*RevenueCatCommitResult, error) {
	st, err := s.loadState(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan := revenuecat.BuildPlan(exp, st.existing)
	result := &RevenueCatCommitResult{Plan: plan, Created: map[string]int{}}

	// Decision per source id, from the recomputed plan.
	action := map[string]revenuecat.Action{}
	for _, it := range plan.Items {
		action[it.RevenueCatID] = it.Action
	}

	custRecursoID := cloneIDMap(st.custRefID)
	planRecursoID := cloneIDMap(st.planRefID)

	// Plans.
	for _, pr := range exp.Products {
		if action[pr.ID] != revenuecat.ActionCreate {
			continue
		}
		s.commitPlan(ctx, tenantID, pr, planRecursoID, result)
	}
	for _, pr := range exp.Products {
		if _, ok := planRecursoID[pr.ID]; !ok {
			if id, ok2 := st.planCodeToID[revenuecat.PlanCode(pr.ID)]; ok2 {
				planRecursoID[pr.ID] = id
			}
		}
	}

	// Customers (subscribers).
	for _, sb := range exp.Subscribers {
		if action[sb.AppUserID] == revenuecat.ActionCreate {
			s.commitCustomer(ctx, tenantID, sb, custRecursoID, result)
		}
	}
	for _, sb := range exp.Subscribers {
		if _, ok := custRecursoID[sb.AppUserID]; !ok {
			if id, ok2 := st.custEmailToID[strings.ToLower(strings.TrimSpace(sb.Email))]; ok2 {
				custRecursoID[sb.AppUserID] = id
			}
		}
	}

	// Subscriptions (nested under subscribers).
	for _, sb := range exp.Subscribers {
		for _, sub := range sb.Subscriptions {
			id := sub.SubID(sb.AppUserID)
			if action[id] != revenuecat.ActionCreate {
				continue
			}
			s.commitSubscription(ctx, tenantID, id, sb.AppUserID, sub, custRecursoID, planRecursoID, result)
		}
	}

	return result, nil
}

func (s *RevenueCatImportService) commitPlan(ctx context.Context, tenantID uuid.UUID, pr revenuecat.Product, resolve map[string]uuid.UUID, result *RevenueCatCommitResult) {
	unit, count := pr.PeriodUnit, pr.PeriodCount
	if count <= 0 {
		count = 1
	}
	name := pr.Title
	if name == "" {
		name = pr.ID
	}
	created, err := s.catalog.CreatePlan(ctx, CreatePlanInput{
		TenantID: tenantID, Name: name, Code: revenuecat.PlanCode(pr.ID),
		IntervalUnit: unit, IntervalCount: count, Amount: pr.Price, Currency: strings.ToUpper(strings.TrimSpace(pr.Currency)),
	})
	if err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(revenuecat.KindPlan), StripeID: pr.ID, Error: err.Error()})
		return
	}
	if err := s.recordRef(ctx, tenantID, domain.ImportKindPlan, pr.ID, created.ID); err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(revenuecat.KindPlan), StripeID: pr.ID, Error: err.Error()})
		return
	}
	resolve[pr.ID] = created.ID
	result.Created[string(revenuecat.KindPlan)]++
}

func (s *RevenueCatImportService) commitCustomer(ctx context.Context, tenantID uuid.UUID, sb revenuecat.Subscriber, resolve map[string]uuid.UUID, result *RevenueCatCommitResult) {
	created, err := s.customers.CreateCustomer(ctx, CreateCustomerInput{
		TenantID: tenantID, Email: strings.ToLower(strings.TrimSpace(sb.Email)),
	})
	if err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(revenuecat.KindCustomer), StripeID: sb.AppUserID, Error: err.Error()})
		return
	}
	if err := s.recordRef(ctx, tenantID, domain.ImportKindCustomer, sb.AppUserID, created.ID); err != nil {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(revenuecat.KindCustomer), StripeID: sb.AppUserID, Error: err.Error()})
		return
	}
	resolve[sb.AppUserID] = created.ID
	result.Created[string(revenuecat.KindCustomer)]++
}

func (s *RevenueCatImportService) commitSubscription(ctx context.Context, tenantID uuid.UUID, subID, appUserID string, sub revenuecat.Subscription, custResolve, planResolve map[string]uuid.UUID, result *RevenueCatCommitResult) {
	fail := func(msg string) {
		result.Failures = append(result.Failures, CommitFailure{Kind: string(revenuecat.KindSubscription), StripeID: subID, Error: msg})
	}
	customerID, ok := custResolve[appUserID]
	if !ok {
		fail("could not resolve the subscription's customer")
		return
	}
	planID, ok := planResolve[sub.ProductID]
	if !ok {
		fail("could not resolve the subscription's plan")
		return
	}

	now := time.Now().UTC()
	status := "active"
	if sub.IsTrial {
		status = "trialing"
	}
	rec := &domain.Subscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		CustomerID:         customerID,
		PlanID:             planID,
		Status:             domain.SubscriptionStatus(status),
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   unixOr(sub.ExpiresAt, now),
		BillingAnchor:      unixOr(sub.ExpiresAt, now),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if status == "trialing" && sub.ExpiresAt > 0 {
		t := time.Unix(sub.ExpiresAt, 0).UTC()
		rec.TrialEnd = &t
	}

	// Claim-first idempotency: record the (source, external id) ref BEFORE
	// creating the subscription — see the ChargebeeImportService equivalent for
	// the full rationale. Without it a concurrent/retried commit inserts a second
	// subscription for one RevenueCat entitlement and double-bills at renewal.
	refErr := s.refs.Create(ctx, &domain.ImportExternalRef{
		ID: uuid.New(), TenantID: tenantID, Source: domain.ImportSourceRevenueCat,
		Kind: domain.ImportKindSubscription, ExternalID: subID, RecursoID: rec.ID,
	})
	if errors.Is(refErr, domain.ErrDuplicateImportRef) {
		return // already imported (or a concurrent commit claimed it) — do not double-insert
	}
	if refErr != nil {
		fail(refErr.Error())
		return
	}
	if err := s.subs.Create(ctx, rec); err != nil {
		fail(err.Error())
		return
	}
	result.Created[string(revenuecat.KindSubscription)]++
}

func (s *RevenueCatImportService) recordRef(ctx context.Context, tenantID uuid.UUID, kind, externalID string, recursoID uuid.UUID) error {
	err := s.refs.Create(ctx, &domain.ImportExternalRef{
		ID: uuid.New(), TenantID: tenantID, Source: domain.ImportSourceRevenueCat,
		Kind: kind, ExternalID: externalID, RecursoID: recursoID,
	})
	if errors.Is(err, domain.ErrDuplicateImportRef) {
		return nil
	}
	return err
}

func (s *RevenueCatImportService) loadState(ctx context.Context, tenantID uuid.UUID) (*revenuecatState, error) {
	st := &revenuecatState{
		existing: revenuecat.Existing{
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

	refs, err := s.refs.ListRefs(ctx, tenantID, domain.ImportSourceRevenueCat)
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

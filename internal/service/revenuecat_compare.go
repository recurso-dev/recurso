package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	revenuecat "github.com/recurso-dev/recurso/internal/importer/revenuecat"
)

// The RevenueCat Compare gate — same contract as the Stripe and Chargebee
// ones. RevenueCat's resolution model differs: subscribers are keyed by
// app_user_id (refs) with email as the fallback identity, and only
// subscribers WITH an email were importable in the first place — the gate's
// scope mirrors the committer's exactly, so "ready" means "everything the
// import could carry, arrived intact".

// SetSubscriptionReader wires the read-side subscription lookup used by
// Compare. Nil-safe.
func (s *RevenueCatImportService) SetSubscriptionReader(r compareSubReader) { s.subReader = r }

// Compare diffs the export against the tenant's live data. Read-only.
func (s *RevenueCatImportService) Compare(ctx context.Context, tenantID uuid.UUID, exp *revenuecat.Export) (*CompareReport, error) {
	report := &CompareReport{Source: string(domain.ImportSourceRevenueCat), Issues: []CompareIssue{}, GeneratedAt: time.Now().UTC()}
	issue := func(kind, extID, field, src, rec string) {
		report.Issues = append(report.Issues, CompareIssue{Kind: kind, ExternalID: extID, Field: field, Source: src, Recurso: rec})
	}

	refs, err := s.refs.ListRefs(ctx, tenantID, string(domain.ImportSourceRevenueCat))
	if err != nil {
		return nil, fmt.Errorf("load import refs: %w", err)
	}
	refID := map[string]map[string]uuid.UUID{
		domain.ImportKindCustomer:     {},
		domain.ImportKindPlan:         {},
		domain.ImportKindSubscription: {},
	}
	for _, r := range refs {
		if m, ok := refID[r.Kind]; ok {
			m[r.ExternalID] = r.RecursoID
		}
	}

	customers, err := s.customers.ListCustomers(ctx, tenantID, domain.CustomerFilter{Limit: importExistingScanLimit})
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	custByID := map[uuid.UUID]*domain.Customer{}
	custByEmail := map[string]*domain.Customer{}
	for _, c := range customers {
		custByID[c.ID] = c
		if c.Email != "" {
			custByEmail[strings.ToLower(strings.TrimSpace(c.Email))] = c
		}
	}

	plans, err := s.catalog.ListPlans(ctx, tenantID, domain.PlanFilter{Limit: importExistingScanLimit})
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	planByID := map[uuid.UUID]*domain.Plan{}
	planByCode := map[string]*domain.Plan{}
	for _, p := range plans {
		planByID[p.ID] = p
		planByCode[p.Code] = p
	}

	// ---- products (plans)
	planResolved := map[string]uuid.UUID{}
	for _, pr := range exp.Products {
		report.Plans.Source++
		count := pr.PeriodCount
		if count <= 0 {
			count = 1
		}
		wantCurrency := strings.ToUpper(strings.TrimSpace(pr.Currency))
		var rec *domain.Plan
		if id, ok := refID[domain.ImportKindPlan][pr.ID]; ok {
			rec = planByID[id]
		}
		if rec == nil {
			rec = planByCode[revenuecat.PlanCode(pr.ID)]
		}
		if rec == nil {
			report.Plans.Missing++
			issue("plan", pr.ID, "missing", pr.Title, "")
			continue
		}
		report.Plans.Matched++
		planResolved[pr.ID] = rec.ID
		var price *domain.Price
		for i := range rec.Prices {
			if strings.EqualFold(rec.Prices[i].Currency, wantCurrency) {
				price = &rec.Prices[i]
				break
			}
		}
		if price == nil && len(rec.Prices) == 1 {
			price = &rec.Prices[0]
		}
		if price == nil {
			issue("plan", pr.ID, "price", fmt.Sprintf("%d %s", pr.Price, wantCurrency), "no matching price row")
		} else {
			if price.Amount != pr.Price {
				issue("plan", pr.ID, "amount", fmt.Sprintf("%d", pr.Price), fmt.Sprintf("%d", price.Amount))
			}
			if !strings.EqualFold(price.Currency, wantCurrency) {
				issue("plan", pr.ID, "currency", wantCurrency, price.Currency)
			}
		}
		if string(rec.IntervalUnit) != pr.PeriodUnit || rec.IntervalCount != count {
			issue("plan", pr.ID, "interval",
				fmt.Sprintf("%d %s", count, pr.PeriodUnit),
				fmt.Sprintf("%d %s", rec.IntervalCount, rec.IntervalUnit))
		}
	}

	// ---- subscribers (customers) + their subscriptions
	for _, sb := range exp.Subscribers {
		email := strings.ToLower(strings.TrimSpace(sb.Email))
		if email == "" {
			continue // never importable (no email) — outside the gate's scope
		}
		report.Customers.Source++
		var rec *domain.Customer
		if id, ok := refID[domain.ImportKindCustomer][sb.AppUserID]; ok {
			rec = custByID[id]
		}
		if rec == nil {
			rec = custByEmail[email]
		}
		if rec == nil {
			report.Customers.Missing++
			issue("customer", sb.AppUserID, "missing", email, "")
			continue
		}
		report.Customers.Matched++
		if strings.ToLower(strings.TrimSpace(rec.Email)) != email {
			issue("customer", sb.AppUserID, "email", email, rec.Email)
		}

		for _, sub := range sb.Subscriptions {
			if !sub.IsActive {
				continue // only active entitlements are imported
			}
			report.Subscriptions.Source++
			subID := sub.SubID(sb.AppUserID)
			id, ok := refID[domain.ImportKindSubscription][subID]
			if !ok {
				report.Subscriptions.Missing++
				issue("subscription", subID, "missing", sub.ProductID, "")
				continue
			}
			if s.subReader == nil {
				return nil, fmt.Errorf("subscription reader not wired")
			}
			imported, err := s.subReader.GetByID(ctx, id)
			if err != nil || imported == nil || imported.TenantID != tenantID {
				report.Subscriptions.Missing++
				issue("subscription", subID, "missing", sub.ProductID, "")
				continue
			}
			report.Subscriptions.Matched++

			if imported.CustomerID != rec.ID {
				issue("subscription", subID, "customer_link", sb.AppUserID, imported.CustomerID.String())
			}
			if want, ok := planResolved[sub.ProductID]; ok && imported.PlanID != want {
				issue("subscription", subID, "plan_link", sub.ProductID, imported.PlanID.String())
			}
			wantStatus := "active"
			if sub.IsTrial {
				wantStatus = "trialing"
			}
			if string(imported.Status) != wantStatus {
				issue("subscription", subID, "status", wantStatus, string(imported.Status))
			}
			// Continuity: Recurso renews when the RevenueCat entitlement expires.
			if sub.ExpiresAt > 0 {
				src := time.Unix(sub.ExpiresAt, 0).UTC()
				drift := imported.CurrentPeriodEnd.Sub(src)
				if drift < -periodEndTolerance || drift > periodEndTolerance {
					issue("subscription", subID, "current_period_end",
						src.Format(time.RFC3339), imported.CurrentPeriodEnd.UTC().Format(time.RFC3339))
				}
			}
		}
	}

	report.Ready = len(report.Issues) == 0 &&
		report.Customers.Missing == 0 && report.Plans.Missing == 0 && report.Subscriptions.Missing == 0
	return report, nil
}

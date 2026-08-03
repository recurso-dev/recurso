package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	chargebee "github.com/recurso-dev/recurso/internal/importer/chargebee"
)

// The Chargebee Compare gate — same contract as the Stripe one
// (stripe_compare.go): coverage, money-critical fidelity, and billing
// continuity, diffed read-only against live Recurso data. Shares the
// CompareReport/CompareIssue/CompareCount shapes so the wizard renders both
// identically.

// SetSubscriptionReader wires the read-side subscription lookup used by
// Compare. Nil-safe.
func (s *ChargebeeImportService) SetSubscriptionReader(r compareSubReader) { s.subReader = r }

// Compare diffs the export against the tenant's live data. Read-only.
func (s *ChargebeeImportService) Compare(ctx context.Context, tenantID uuid.UUID, exp *chargebee.Export) (*CompareReport, error) {
	report := &CompareReport{Source: string(domain.ImportSourceChargebee), Issues: []CompareIssue{}, GeneratedAt: time.Now().UTC()}
	issue := func(kind, extID, field, src, rec string) {
		report.Issues = append(report.Issues, CompareIssue{Kind: kind, ExternalID: extID, Field: field, Source: src, Recurso: rec})
	}

	refs, err := s.refs.ListRefs(ctx, tenantID, string(domain.ImportSourceChargebee))
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

	// ---- customers
	custResolved := map[string]uuid.UUID{}
	for _, c := range exp.Customers {
		if c.Deleted {
			continue // the committer leaves deleted customers behind by design
		}
		report.Customers.Source++
		email := strings.ToLower(strings.TrimSpace(c.Email))
		var rec *domain.Customer
		if id, ok := refID[domain.ImportKindCustomer][c.ID]; ok {
			rec = custByID[id]
		}
		if rec == nil && email != "" {
			rec = custByEmail[email]
		}
		if rec == nil {
			report.Customers.Missing++
			issue("customer", c.ID, "missing", email, "")
			continue
		}
		report.Customers.Matched++
		custResolved[c.ID] = rec.ID
		if email != "" && strings.ToLower(strings.TrimSpace(rec.Email)) != email {
			issue("customer", c.ID, "email", email, rec.Email)
		}
	}

	// ---- plans
	planResolved := map[string]uuid.UUID{}
	for _, pl := range exp.Plans {
		if pl.Status == "deleted" {
			continue
		}
		m, ok := chargebee.MapPlan(pl)
		if !ok {
			continue // unsupported period unit — preview reports these
		}
		report.Plans.Source++
		var rec *domain.Plan
		if id, ok := refID[domain.ImportKindPlan][pl.ID]; ok {
			rec = planByID[id]
		}
		if rec == nil {
			rec = planByCode[m.Code]
		}
		if rec == nil {
			report.Plans.Missing++
			issue("plan", pl.ID, "missing", m.Name, "")
			continue
		}
		report.Plans.Matched++
		planResolved[pl.ID] = rec.ID
		var price *domain.Price
		for i := range rec.Prices {
			if strings.EqualFold(rec.Prices[i].Currency, m.Currency) {
				price = &rec.Prices[i]
				break
			}
		}
		if price == nil && len(rec.Prices) == 1 {
			price = &rec.Prices[0]
		}
		if price == nil {
			issue("plan", pl.ID, "price", fmt.Sprintf("%d %s", m.Amount, m.Currency), "no matching price row")
		} else {
			if price.Amount != m.Amount {
				issue("plan", pl.ID, "amount", fmt.Sprintf("%d", m.Amount), fmt.Sprintf("%d", price.Amount))
			}
			if !strings.EqualFold(price.Currency, m.Currency) {
				issue("plan", pl.ID, "currency", m.Currency, price.Currency)
			}
		}
		if string(rec.IntervalUnit) != m.IntervalUnit || rec.IntervalCount != m.IntervalCount {
			issue("plan", pl.ID, "interval",
				fmt.Sprintf("%d %s", m.IntervalCount, m.IntervalUnit),
				fmt.Sprintf("%d %s", rec.IntervalCount, rec.IntervalUnit))
		}
	}

	// ---- subscriptions
	for _, sub := range exp.Subscriptions {
		status, ok := chargebee.MapSubStatus(sub.Status)
		if !ok {
			continue
		}
		report.Subscriptions.Source++
		id, ok := refID[domain.ImportKindSubscription][sub.ID]
		if !ok {
			report.Subscriptions.Missing++
			issue("subscription", sub.ID, "missing", sub.Status, "")
			continue
		}
		if s.subReader == nil {
			return nil, fmt.Errorf("subscription reader not wired")
		}
		rec, err := s.subReader.GetByID(ctx, id)
		if err != nil || rec == nil || rec.TenantID != tenantID {
			report.Subscriptions.Missing++
			issue("subscription", sub.ID, "missing", sub.Status, "")
			continue
		}
		report.Subscriptions.Matched++

		if want, ok := custResolved[sub.CustomerID]; ok && rec.CustomerID != want {
			issue("subscription", sub.ID, "customer_link", sub.CustomerID, rec.CustomerID.String())
		}
		if want, ok := planResolved[sub.PlanID]; ok && rec.PlanID != want {
			issue("subscription", sub.ID, "plan_link", sub.PlanID, rec.PlanID.String())
		}
		if string(rec.Status) != status {
			issue("subscription", sub.ID, "status", status, string(rec.Status))
		}
		// non_renewing imports as CancelAtPeriodEnd=true — mirror the committer.
		if wantCancel := sub.Status == "non_renewing"; rec.CancelAtPeriodEnd != wantCancel {
			issue("subscription", sub.ID, "cancel_at_period_end",
				fmt.Sprintf("%t", wantCancel), fmt.Sprintf("%t", rec.CancelAtPeriodEnd))
		}
		if sub.CurrentTermEnd > 0 {
			src := time.Unix(sub.CurrentTermEnd, 0).UTC()
			drift := rec.CurrentPeriodEnd.Sub(src)
			if drift < -periodEndTolerance || drift > periodEndTolerance {
				issue("subscription", sub.ID, "current_period_end",
					src.Format(time.RFC3339), rec.CurrentPeriodEnd.UTC().Format(time.RFC3339))
			}
		}
	}

	report.Ready = len(report.Issues) == 0 &&
		report.Customers.Missing == 0 && report.Plans.Missing == 0 && report.Subscriptions.Missing == 0
	return report, nil
}

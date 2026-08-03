package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	stripeimport "github.com/recurso-dev/recurso/internal/importer/stripe"
)

// The Compare gate: after Import → Validate, and before Go-Live, prove the
// migration against the source export. It re-reads the same Stripe export a
// commit consumed and answers three questions with receipts, per record:
//
//  1. Coverage — is every importable source record present in Recurso?
//  2. Fidelity — do the money-critical fields match exactly (plan amount,
//     currency, interval; customer identity)?
//  3. Continuity — will billing resume exactly where Stripe left off? A
//     subscription whose CurrentPeriodEnd drifted is a double-billing or
//     billing-gap risk, the classic migration disaster.
//
// Read-only: Compare never writes. Ready == zero issues.

// CompareIssue is one discrepancy, addressed by the source record's external id.
type CompareIssue struct {
	Kind       string `json:"kind"`        // customer | plan | subscription
	ExternalID string `json:"external_id"` // the Stripe id
	Field      string `json:"field"`       // "missing" or the mismatched field
	Source     string `json:"source"`      // value in the export
	Recurso    string `json:"recurso"`     // value in Recurso ("" when missing)
}

// CompareCount is coverage per record kind.
type CompareCount struct {
	Source  int `json:"source"`  // importable records in the export
	Matched int `json:"matched"` // found in Recurso
	Missing int `json:"missing"`
}

// CompareReport is the full gate result.
type CompareReport struct {
	Source        string         `json:"source"` // "stripe"
	Customers     CompareCount   `json:"customers"`
	Plans         CompareCount   `json:"plans"`
	Subscriptions CompareCount   `json:"subscriptions"`
	Issues        []CompareIssue `json:"issues"`
	Ready         bool           `json:"ready"`
	GeneratedAt   time.Time      `json:"generated_at"`
}

// compareSubReader is the read side Compare needs on subscriptions; satisfied
// by *db.SubscriptionRepository.
type compareSubReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
}

// SetSubscriptionReader wires the read-side subscription lookup used by
// Compare. Nil-safe: without it, subscription rows report as missing readers
// rather than silently passing.
func (s *StripeImportService) SetSubscriptionReader(r compareSubReader) { s.subReader = r }

// periodEndTolerance absorbs clock/rounding skew between Stripe's unix seconds
// and the imported timestamp without hiding a real drift (a period is days).
const periodEndTolerance = time.Hour

// Compare diffs the export against the tenant's live data. Read-only.
func (s *StripeImportService) Compare(ctx context.Context, tenantID uuid.UUID, exp *stripeimport.Export) (*CompareReport, error) {
	report := &CompareReport{Source: string(domain.ImportSourceStripe), Issues: []CompareIssue{}, GeneratedAt: time.Now().UTC()}
	issue := func(kind, extID, field, src, rec string) {
		report.Issues = append(report.Issues, CompareIssue{Kind: kind, ExternalID: extID, Field: field, Source: src, Recurso: rec})
	}

	// Resolution maps: import refs first (authoritative), then the same
	// fallbacks the committer used (email for customers, plan code for plans).
	refs, err := s.refs.ListRefs(ctx, tenantID, string(domain.ImportSourceStripe))
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

	// ---- customers: coverage + identity fidelity
	custResolved := map[string]uuid.UUID{} // stripe customer id → recurso id (for sub-link checks)
	for _, c := range exp.Customers {
		if c.Deleted {
			continue
		}
		report.Customers.Source++
		m := stripeimport.MapCustomer(c)
		var rec *domain.Customer
		if id, ok := refID[domain.ImportKindCustomer][c.ID]; ok {
			rec = custByID[id]
		}
		if rec == nil && m.Email != "" {
			rec = custByEmail[m.Email]
		}
		if rec == nil {
			report.Customers.Missing++
			issue("customer", c.ID, "missing", m.Email, "")
			continue
		}
		report.Customers.Matched++
		custResolved[c.ID] = rec.ID
		if m.Email != "" && strings.ToLower(strings.TrimSpace(rec.Email)) != m.Email {
			issue("customer", c.ID, "email", m.Email, rec.Email)
		}
	}

	// ---- plans: coverage + amount/currency/interval fidelity
	prodByID := map[string]stripeimport.Product{}
	for _, p := range exp.Products {
		prodByID[p.ID] = p
	}
	priceResolved := map[string]uuid.UUID{} // stripe price id → recurso plan id
	for _, pr := range exp.Prices {
		m, ok := stripeimport.MapPlan(prodByID[pr.Product], pr)
		if !ok {
			continue // not importable (one-off price) — preview already reports these
		}
		report.Plans.Source++
		var rec *domain.Plan
		if id, ok := refID[domain.ImportKindPlan][pr.ID]; ok {
			rec = planByID[id]
		}
		if rec == nil {
			rec = planByCode[m.Code]
		}
		if rec == nil {
			report.Plans.Missing++
			issue("plan", pr.ID, "missing", m.Name, "")
			continue
		}
		report.Plans.Matched++
		priceResolved[pr.ID] = rec.ID
		// The plan's money lives on its price rows; match by currency first so a
		// multi-currency plan compares against the right row.
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
			issue("plan", pr.ID, "price", fmt.Sprintf("%d %s", m.Amount, m.Currency), "no matching price row")
		} else {
			if price.Amount != m.Amount {
				issue("plan", pr.ID, "amount", fmt.Sprintf("%d", m.Amount), fmt.Sprintf("%d", price.Amount))
			}
			if !strings.EqualFold(price.Currency, m.Currency) {
				issue("plan", pr.ID, "currency", m.Currency, price.Currency)
			}
		}
		if string(rec.IntervalUnit) != m.IntervalUnit || rec.IntervalCount != m.IntervalCount {
			issue("plan", pr.ID, "interval",
				fmt.Sprintf("%d %s", m.IntervalCount, m.IntervalUnit),
				fmt.Sprintf("%d %s", rec.IntervalCount, rec.IntervalUnit))
		}
	}

	// ---- subscriptions: coverage + link fidelity + billing continuity
	for _, sub := range exp.Subscriptions {
		status, ok := stripeimport.MapSubStatus(sub.Status)
		if !ok {
			continue // not importable — out of the gate's scope
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

		if want, ok := custResolved[sub.Customer]; ok && rec.CustomerID != want {
			issue("subscription", sub.ID, "customer_link", sub.Customer, rec.CustomerID.String())
		}
		if priceID := stripeimport.SubscriptionPriceID(sub); priceID != "" {
			if want, ok := priceResolved[priceID]; ok && rec.PlanID != want {
				issue("subscription", sub.ID, "plan_link", priceID, rec.PlanID.String())
			}
		}
		if string(rec.Status) != status {
			issue("subscription", sub.ID, "status", status, string(rec.Status))
		}
		if rec.CancelAtPeriodEnd != sub.CancelAtPeriodEnd {
			issue("subscription", sub.ID, "cancel_at_period_end",
				fmt.Sprintf("%t", sub.CancelAtPeriodEnd), fmt.Sprintf("%t", rec.CancelAtPeriodEnd))
		}
		// The double-billing gate: Recurso must resume billing exactly where
		// Stripe stops. A drifted period end re-bills the current cycle or
		// opens a gap.
		if sub.CurrentPeriodEnd > 0 {
			src := time.Unix(sub.CurrentPeriodEnd, 0).UTC()
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

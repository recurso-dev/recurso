// Package stripeimport is the pure engine that migrates a Stripe account into
// Recurso. It parses a Stripe export (customers, products, prices,
// subscriptions, payment methods), maps each object to its Recurso equivalent,
// and computes a dry-run *Plan* describing exactly what a commit would create,
// link, skip, or refuse — with no side effects. Persisting the plan (with
// idempotency) is a separate concern layered on top of this engine.
//
// Money alignment is deliberate: Stripe `unit_amount` and Recurso amounts are
// both integer minor units of the currency (cents, paise, and no sub-unit for
// zero-decimal currencies like JPY), so amounts copy across directly.
package stripeimport

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// --- Stripe input types (the subset Recurso maps) --------------------------

// Export is the assembled Stripe account dump the importer consumes. It mirrors
// the shape produced by concatenating `stripe` CLI list dumps under one object.
type Export struct {
	Customers      []Customer      `json:"customers"`
	Products       []Product       `json:"products"`
	Prices         []Price         `json:"prices"`
	Subscriptions  []Subscription  `json:"subscriptions"`
	PaymentMethods []PaymentMethod `json:"payment_methods"`
}

type Customer struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Currency    string  `json:"currency"`
	Address     Address `json:"address"`
	Deleted     bool    `json:"deleted"`
}

type Address struct {
	Country string `json:"country"`
}

type Product struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type Price struct {
	ID         string     `json:"id"`
	Product    string     `json:"product"`
	UnitAmount int64      `json:"unit_amount"`
	Currency   string     `json:"currency"`
	Active     bool       `json:"active"`
	Nickname   string     `json:"nickname"`
	Recurring  *Recurring `json:"recurring"`
}

type Recurring struct {
	Interval      string `json:"interval"`       // day | week | month | year
	IntervalCount int    `json:"interval_count"` // defaults to 1 when 0
}

type Subscription struct {
	ID                string            `json:"id"`
	Customer          string            `json:"customer"`
	Status            string            `json:"status"`
	CancelAtPeriodEnd bool              `json:"cancel_at_period_end"`
	Items             SubscriptionItems `json:"items"`
}

// SubscriptionItems mirrors Stripe's nested `items: { data: [...] }`.
type SubscriptionItems struct {
	Data []SubscriptionItem `json:"data"`
}

type SubscriptionItem struct {
	Price Price `json:"price"`
}

type PaymentMethod struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Type     string `json:"type"` // "card", "us_bank_account", ...
	Card     *Card  `json:"card"`
}

type Card struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
}

// Parse decodes a Stripe export from JSON. It accepts the wrapper object shape
// ({"customers": [...], "products": [...], ...}); unknown fields are ignored so
// a raw `stripe` API object with extra keys still parses.
func Parse(data []byte) (*Export, error) {
	var exp Export
	if err := json.Unmarshal(data, &exp); err != nil {
		return nil, fmt.Errorf("invalid Stripe export JSON: %w", err)
	}
	return &exp, nil
}

// --- Existing Recurso state (for conflict/idempotency detection) -----------

// Existing captures the slices of the target tenant the planner needs to avoid
// duplicates. All email keys are lower-cased.
type Existing struct {
	// CustomerEmails is the set of emails already present in the tenant. A Stripe
	// customer whose email matches is LINKED (reused), never duplicated.
	CustomerEmails map[string]bool
	// PlanCodes is the set of plan codes already present, so re-mapping a Stripe
	// price that already produced a plan is a skip rather than a duplicate.
	PlanCodes map[string]bool
	// ImportedStripeIDs is the set of Stripe object IDs already imported in a
	// prior run (from the external-ref table). Any match is skipped — this is the
	// idempotency gate. May be nil/empty on a first run.
	ImportedStripeIDs map[string]bool
}

func (e Existing) hasEmail(email string) bool {
	return e.CustomerEmails != nil && e.CustomerEmails[strings.ToLower(strings.TrimSpace(email))]
}
func (e Existing) hasPlanCode(code string) bool {
	return e.PlanCodes != nil && e.PlanCodes[code]
}
func (e Existing) alreadyImported(id string) bool {
	return e.ImportedStripeIDs != nil && e.ImportedStripeIDs[id]
}

// --- Plan (the dry-run result) ---------------------------------------------

// Action is what a commit would do with one source object.
type Action string

const (
	ActionCreate       Action = "create"                // net-new Recurso object
	ActionLinkExisting Action = "link_existing"         // matched an existing record (by email); reused
	ActionSkipImported Action = "skip_already_imported" // seen in a prior import run (idempotent)
	ActionConflict     Action = "conflict"              // references something that couldn't be resolved
	ActionUnsupported  Action = "unsupported"           // no Recurso equivalent (e.g. one-time price)
)

// Kind is the object category of a planned item.
type Kind string

const (
	KindCustomer      Kind = "customer"
	KindPlan          Kind = "plan"
	KindSubscription  Kind = "subscription"
	KindPaymentMethod Kind = "payment_method"
)

// Item is one source object's planned outcome.
type Item struct {
	Kind     Kind   `json:"kind"`
	StripeID string `json:"stripe_id"`
	Label    string `json:"label"` // human label: email / product name / sub id
	Action   Action `json:"action"`
	Detail   string `json:"detail,omitempty"` // mapping notes or the reason for conflict/unsupported
}

// Plan is the full dry-run report the preview endpoint returns.
type Plan struct {
	Items    []Item         `json:"items"`
	Summary  map[string]int `json:"summary"`  // "customer.create" -> count
	Warnings []string       `json:"warnings"` // account-level notes worth surfacing
}

func (p *Plan) add(it Item) {
	p.Items = append(p.Items, it)
	p.Summary[fmt.Sprintf("%s.%s", it.Kind, it.Action)]++
}

// mappedInterval validates a Stripe recurring interval and returns the Recurso
// interval unit (identical vocabulary) plus a normalized count.
func mappedInterval(r *Recurring) (unit string, count int, ok bool) {
	if r == nil {
		return "", 0, false
	}
	switch r.Interval {
	case "day", "week", "month", "year":
		c := r.IntervalCount
		if c <= 0 {
			c = 1
		}
		return r.Interval, c, true
	default:
		return "", 0, false
	}
}

// planCode derives a stable, human Recurso plan code from a Stripe price. It is
// deterministic so re-importing the same price maps to the same code (which is
// what makes PlanCodes-based skipping work).
func planCode(pr Price) string {
	return "stripe_" + pr.ID
}

// PlanMapping is the set of Recurso plan-create fields derived from a Stripe
// product + price. The committer feeds these straight into plan creation, so
// the mapping lives in exactly one place (here) — not duplicated in the service.
type PlanMapping struct {
	Name          string
	Code          string
	IntervalUnit  string
	IntervalCount int
	Amount        int64  // minor units (copied straight from Stripe unit_amount)
	Currency      string // ISO 3-letter, upper-cased
}

// MapPlan derives Recurso plan-create params from a Stripe product+price. ok is
// false when the price has no Recurso plan equivalent (one-time price or an
// unsupported billing interval) — the same predicate BuildPlan uses to mark the
// item unsupported/conflict.
func MapPlan(prod Product, pr Price) (PlanMapping, bool) {
	unit, count, ok := mappedInterval(pr.Recurring)
	if !ok {
		return PlanMapping{}, false
	}
	name := prod.Name
	if pr.Nickname != "" {
		name = fmt.Sprintf("%s (%s)", prod.Name, pr.Nickname)
	}
	if name == "" {
		name = pr.ID
	}
	return PlanMapping{
		Name:          name,
		Code:          planCode(pr),
		IntervalUnit:  unit,
		IntervalCount: count,
		Amount:        pr.UnitAmount,
		Currency:      strings.ToUpper(strings.TrimSpace(pr.Currency)),
	}, true
}

// CustomerMapping is the set of Recurso customer-create fields derived from a
// Stripe customer.
type CustomerMapping struct {
	Email   string
	Name    string
	Country string
}

// MapCustomer derives Recurso customer-create params from a Stripe customer.
func MapCustomer(c Customer) CustomerMapping {
	return CustomerMapping{
		Email:   strings.ToLower(strings.TrimSpace(c.Email)),
		Name:    c.Name,
		Country: strings.ToUpper(strings.TrimSpace(c.Address.Country)),
	}
}

// BuildPlan computes the dry-run outcome of importing exp into a tenant whose
// current state is described by existing. It has no side effects.
//
// Rules:
//   - Customer: already-imported → skip; email matches an existing customer →
//     link; blank email → conflict (Recurso requires an email); else → create.
//   - Plan: built from a recurring, active price + its product. One-time or
//     non-recurring prices are unsupported; an unknown/invalid interval is a
//     conflict; a price whose derived code already exists → skip.
//   - Subscription: create only when BOTH its customer and its (single) price
//     resolve within this import (or already exist); otherwise conflict.
//     Terminal/incomplete Stripe statuses are skipped.
//   - PaymentMethod: card methods are recorded (reference only, never PAN);
//     non-card types are unsupported.
func BuildPlan(exp *Export, existing Existing) *Plan {
	p := &Plan{Summary: map[string]int{}}

	// --- Customers -------------------------------------------------------
	// Track which Stripe customer IDs will resolve (created or linked), so
	// subscriptions can check their customer reference.
	customerResolves := map[string]bool{}
	for _, c := range exp.Customers {
		label := c.Email
		if label == "" {
			label = c.ID
		}
		switch {
		case existing.alreadyImported(c.ID):
			customerResolves[c.ID] = true
			p.add(Item{Kind: KindCustomer, StripeID: c.ID, Label: label, Action: ActionSkipImported})
		case c.Deleted:
			p.add(Item{Kind: KindCustomer, StripeID: c.ID, Label: label, Action: ActionUnsupported, Detail: "customer is deleted in Stripe"})
		case strings.TrimSpace(c.Email) == "":
			p.add(Item{Kind: KindCustomer, StripeID: c.ID, Label: label, Action: ActionConflict, Detail: "Stripe customer has no email; Recurso requires one"})
		case existing.hasEmail(c.Email):
			customerResolves[c.ID] = true
			p.add(Item{Kind: KindCustomer, StripeID: c.ID, Label: label, Action: ActionLinkExisting, Detail: "matched an existing Recurso customer by email"})
		default:
			customerResolves[c.ID] = true
			p.add(Item{Kind: KindCustomer, StripeID: c.ID, Label: label, Action: ActionCreate})
		}
	}

	// --- Plans (product + recurring price) -------------------------------
	products := map[string]Product{}
	for _, pr := range exp.Products {
		products[pr.ID] = pr
	}
	// Track which Stripe price IDs will resolve to a Recurso plan.
	priceResolves := map[string]bool{}
	for _, pr := range exp.Prices {
		prod := products[pr.Product]
		name := prod.Name
		if pr.Nickname != "" {
			name = fmt.Sprintf("%s (%s)", prod.Name, pr.Nickname)
		}
		if name == "" {
			name = pr.ID
		}
		code := planCode(pr)
		switch {
		case existing.alreadyImported(pr.ID) || existing.hasPlanCode(code):
			priceResolves[pr.ID] = true
			p.add(Item{Kind: KindPlan, StripeID: pr.ID, Label: name, Action: ActionSkipImported})
		case pr.Recurring == nil:
			p.add(Item{Kind: KindPlan, StripeID: pr.ID, Label: name, Action: ActionUnsupported, Detail: "one-time price; Recurso plans are recurring"})
		default:
			if _, _, ok := mappedInterval(pr.Recurring); !ok {
				p.add(Item{Kind: KindPlan, StripeID: pr.ID, Label: name, Action: ActionConflict, Detail: fmt.Sprintf("unsupported billing interval %q", pr.Recurring.Interval)})
				continue
			}
			priceResolves[pr.ID] = true
			p.add(Item{Kind: KindPlan, StripeID: pr.ID, Label: name, Action: ActionCreate,
				Detail: fmt.Sprintf("%s every %d %s", money(pr.UnitAmount, pr.Currency), intervalCount(pr.Recurring), pr.Recurring.Interval)})
		}
	}

	// --- Subscriptions ---------------------------------------------------
	for _, s := range exp.Subscriptions {
		label := s.ID
		if existing.alreadyImported(s.ID) {
			p.add(Item{Kind: KindSubscription, StripeID: s.ID, Label: label, Action: ActionSkipImported})
			continue
		}
		if skippableSubStatus(s.Status) {
			p.add(Item{Kind: KindSubscription, StripeID: s.ID, Label: label, Action: ActionUnsupported, Detail: fmt.Sprintf("Stripe status %q is not migrated", s.Status)})
			continue
		}
		if !customerResolves[s.Customer] {
			p.add(Item{Kind: KindSubscription, StripeID: s.ID, Label: label, Action: ActionConflict, Detail: "subscription's customer is not in this import"})
			continue
		}
		priceID := subscriptionPriceID(s)
		if priceID == "" {
			p.add(Item{Kind: KindSubscription, StripeID: s.ID, Label: label, Action: ActionConflict, Detail: "subscription has no price item"})
			continue
		}
		if !priceResolves[priceID] {
			p.add(Item{Kind: KindSubscription, StripeID: s.ID, Label: label, Action: ActionConflict, Detail: "subscription's plan/price is not in this import"})
			continue
		}
		detail := "status " + s.Status
		if s.CancelAtPeriodEnd {
			detail += "; cancels at period end"
		}
		p.add(Item{Kind: KindSubscription, StripeID: s.ID, Label: label, Action: ActionCreate, Detail: detail})
	}

	// --- Payment methods -------------------------------------------------
	for _, pm := range exp.PaymentMethods {
		if existing.alreadyImported(pm.ID) {
			p.add(Item{Kind: KindPaymentMethod, StripeID: pm.ID, Label: pm.ID, Action: ActionSkipImported})
			continue
		}
		if pm.Type != "card" || pm.Card == nil {
			p.add(Item{Kind: KindPaymentMethod, StripeID: pm.ID, Label: pm.ID, Action: ActionUnsupported, Detail: fmt.Sprintf("payment method type %q not migrated", pm.Type)})
			continue
		}
		if !customerResolves[pm.Customer] {
			p.add(Item{Kind: KindPaymentMethod, StripeID: pm.ID, Label: pm.ID, Action: ActionConflict, Detail: "payment method's customer is not in this import"})
			continue
		}
		label := fmt.Sprintf("%s •••• %s", pm.Card.Brand, pm.Card.Last4)
		p.add(Item{Kind: KindPaymentMethod, StripeID: pm.ID, Label: label, Action: ActionCreate})
	}

	// Account-level warnings, deterministically ordered.
	if conflicts := p.countAction(ActionConflict); conflicts > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d item(s) can't be imported yet — resolve the conflicts below (usually a missing customer or plan in the same export).", conflicts))
	}
	if unsupported := p.countAction(ActionUnsupported); unsupported > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d item(s) have no Recurso equivalent and will be left behind (one-time prices, non-card methods, terminal subscriptions).", unsupported))
	}
	sort.Strings(p.Warnings)
	return p
}

func (p *Plan) countAction(a Action) int {
	n := 0
	for _, it := range p.Items {
		if it.Action == a {
			n++
		}
	}
	return n
}

// CreateCounts returns the number of net-new objects a commit would create,
// keyed by kind — the headline a UI shows ("import 12 customers, 3 plans…").
func (p *Plan) CreateCounts() map[Kind]int {
	out := map[Kind]int{}
	for _, it := range p.Items {
		if it.Action == ActionCreate {
			out[it.Kind]++
		}
	}
	return out
}

func skippableSubStatus(status string) bool {
	switch status {
	case "incomplete", "incomplete_expired", "canceled", "unpaid":
		return true
	default:
		return false
	}
}

func subscriptionPriceID(s Subscription) string {
	if len(s.Items.Data) == 0 {
		return ""
	}
	return s.Items.Data[0].Price.ID
}

func intervalCount(r *Recurring) int {
	if r == nil || r.IntervalCount <= 0 {
		return 1
	}
	return r.IntervalCount
}

// money renders a minor-unit amount for display in plan details. It is
// intentionally simple (2-decimal for the common case) — the authoritative,
// exponent-aware formatting is the frontend's job; this is a hint string.
func money(amount int64, currency string) string {
	cur := strings.ToUpper(currency)
	// Zero-decimal currencies Stripe treats as having no minor unit.
	switch cur {
	case "JPY", "KRW", "VND", "CLP":
		return fmt.Sprintf("%d %s", amount, cur)
	default:
		return fmt.Sprintf("%d.%02d %s", amount/100, abs64(amount%100), cur)
	}
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

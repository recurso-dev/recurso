// Package chargebee is the pure engine that migrates a Chargebee account into
// Recurso: parse an export (customers, plans, subscriptions), map each object,
// and compute a dry-run *Plan* of what a commit would do — with no side effects.
// It mirrors internal/importer/stripe; kept separate (its own small result model)
// so it doesn't couple to, or risk, the merged Stripe importer.
//
// Money alignment: Chargebee prices and Recurso amounts are both integer minor
// units, so they copy across directly.
package chargebee

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// --- Chargebee input types (the subset Recurso maps) -----------------------

// Export is the assembled Chargebee account dump the importer consumes.
type Export struct {
	Customers     []Customer     `json:"customers"`
	Plans         []Plan         `json:"plans"`
	Subscriptions []Subscription `json:"subscriptions"`
}

type Customer struct {
	ID             string         `json:"id"`
	Email          string         `json:"email"`
	FirstName      string         `json:"first_name"`
	LastName       string         `json:"last_name"`
	Company        string         `json:"company"`
	BillingAddress BillingAddress `json:"billing_address"`
	Deleted        bool           `json:"deleted"`
}

type BillingAddress struct {
	Country string `json:"country"`
}

// Name assembles a display name from the Chargebee name parts / company.
func (c Customer) Name() string {
	n := strings.TrimSpace(strings.TrimSpace(c.FirstName) + " " + strings.TrimSpace(c.LastName))
	if n != "" {
		return n
	}
	return c.Company
}

type Plan struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Price        int64  `json:"price"`         // minor units
	Period       int    `json:"period"`        // count; defaults to 1 when 0
	PeriodUnit   string `json:"period_unit"`   // day | week | month | year
	CurrencyCode string `json:"currency_code"` // ISO 3-letter
	Status       string `json:"status"`        // active | archived | deleted
}

type Subscription struct {
	ID               string `json:"id"`
	CustomerID       string `json:"customer_id"`
	PlanID           string `json:"plan_id"`
	Status           string `json:"status"`             // active | in_trial | non_renewing | cancelled | paused | future
	CurrentTermStart int64  `json:"current_term_start"` // unix seconds
	CurrentTermEnd   int64  `json:"current_term_end"`   // unix seconds
	TrialEnd         int64  `json:"trial_end"`          // unix seconds; 0 = none
}

// Parse decodes a Chargebee export from JSON (unknown fields ignored).
func Parse(data []byte) (*Export, error) {
	var exp Export
	if err := json.Unmarshal(data, &exp); err != nil {
		return nil, fmt.Errorf("invalid Chargebee export JSON: %w", err)
	}
	return &exp, nil
}

// --- Existing Recurso state (conflict/idempotency detection) ---------------

type Existing struct {
	CustomerEmails map[string]bool
	PlanCodes      map[string]bool
	ImportedIDs    map[string]bool
}

func (e Existing) hasEmail(email string) bool {
	return e.CustomerEmails != nil && e.CustomerEmails[strings.ToLower(strings.TrimSpace(email))]
}
func (e Existing) hasPlanCode(code string) bool {
	return e.PlanCodes != nil && e.PlanCodes[code]
}
func (e Existing) alreadyImported(id string) bool {
	return e.ImportedIDs != nil && e.ImportedIDs[id]
}

// --- Result model ----------------------------------------------------------

type Action string

const (
	ActionCreate       Action = "create"
	ActionLinkExisting Action = "link_existing"
	ActionSkipImported Action = "skip_already_imported"
	ActionConflict     Action = "conflict"
	ActionUnsupported  Action = "unsupported"
)

type Kind string

const (
	KindCustomer     Kind = "customer"
	KindPlan         Kind = "plan"
	KindSubscription Kind = "subscription"
)

type Item struct {
	Kind        Kind   `json:"kind"`
	ChargebeeID string `json:"chargebee_id"`
	Label       string `json:"label"`
	Action      Action `json:"action"`
	Detail      string `json:"detail,omitempty"`
}

type ImportPlan struct {
	Items    []Item         `json:"items"`
	Summary  map[string]int `json:"summary"`
	Warnings []string       `json:"warnings"`
}

func (p *ImportPlan) add(it Item) {
	p.Items = append(p.Items, it)
	p.Summary[fmt.Sprintf("%s.%s", it.Kind, it.Action)]++
}

func (p *ImportPlan) countAction(a Action) int {
	n := 0
	for _, it := range p.Items {
		if it.Action == a {
			n++
		}
	}
	return n
}

// CreateCounts returns the net-new objects per kind a commit would create.
func (p *ImportPlan) CreateCounts() map[Kind]int {
	out := map[Kind]int{}
	for _, it := range p.Items {
		if it.Action == ActionCreate {
			out[it.Kind]++
		}
	}
	return out
}

// PlanCode is the deterministic Recurso plan code for a Chargebee plan.
func PlanCode(id string) string { return "chargebee_" + id }

// CustomerMapping / PlanMapping are the Recurso create fields derived from a
// Chargebee object — the mapping lives in one place, shared by preview + commit.
type CustomerMapping struct {
	Email   string
	Name    string
	Country string
}

func MapCustomer(c Customer) CustomerMapping {
	return CustomerMapping{
		Email:   strings.ToLower(strings.TrimSpace(c.Email)),
		Name:    c.Name(),
		Country: strings.ToUpper(strings.TrimSpace(c.BillingAddress.Country)),
	}
}

type PlanMapping struct {
	Name          string
	Code          string
	IntervalUnit  string
	IntervalCount int
	Amount        int64
	Currency      string
}

// MapPlan derives Recurso plan-create params from a Chargebee plan. ok=false
// when the plan has no Recurso equivalent (unsupported period unit).
func MapPlan(pl Plan) (PlanMapping, bool) {
	unit, count, ok := mappedInterval(pl.PeriodUnit, pl.Period)
	if !ok {
		return PlanMapping{}, false
	}
	name := pl.Name
	if name == "" {
		name = pl.ID
	}
	return PlanMapping{
		Name:          name,
		Code:          PlanCode(pl.ID),
		IntervalUnit:  unit,
		IntervalCount: count,
		Amount:        pl.Price,
		Currency:      strings.ToUpper(strings.TrimSpace(pl.CurrencyCode)),
	}, true
}

// MapSubStatus maps a Chargebee subscription status to the Recurso status a
// commit imports it as. ok=false for statuses that are not migrated.
func MapSubStatus(status string) (string, bool) { return importableSubStatus(status) }

func mappedInterval(unit string, period int) (string, int, bool) {
	switch unit {
	case "day", "week", "month", "year":
		if period <= 0 {
			period = 1
		}
		return unit, period, true
	default:
		return "", 0, false
	}
}

func importableSubStatus(status string) (recurso string, ok bool) {
	switch status {
	case "active", "non_renewing":
		return "active", true
	case "in_trial":
		return "trialing", true
	case "paused":
		return "paused", true
	default: // cancelled, future, deleted, ...
		return "", false
	}
}

// BuildPlan computes the dry-run outcome of importing exp into a tenant.
func BuildPlan(exp *Export, existing Existing) *ImportPlan {
	p := &ImportPlan{Summary: map[string]int{}}

	customerResolves := map[string]bool{}
	for _, c := range exp.Customers {
		label := c.Email
		if label == "" {
			label = c.ID
		}
		switch {
		case existing.alreadyImported(c.ID):
			customerResolves[c.ID] = true
			p.add(Item{Kind: KindCustomer, ChargebeeID: c.ID, Label: label, Action: ActionSkipImported})
		case c.Deleted:
			p.add(Item{Kind: KindCustomer, ChargebeeID: c.ID, Label: label, Action: ActionUnsupported, Detail: "customer is deleted in Chargebee"})
		case strings.TrimSpace(c.Email) == "":
			p.add(Item{Kind: KindCustomer, ChargebeeID: c.ID, Label: label, Action: ActionConflict, Detail: "Chargebee customer has no email; Recurso requires one"})
		case existing.hasEmail(c.Email):
			customerResolves[c.ID] = true
			p.add(Item{Kind: KindCustomer, ChargebeeID: c.ID, Label: label, Action: ActionLinkExisting, Detail: "matched an existing Recurso customer by email"})
		default:
			customerResolves[c.ID] = true
			p.add(Item{Kind: KindCustomer, ChargebeeID: c.ID, Label: label, Action: ActionCreate})
		}
	}

	planResolves := map[string]bool{}
	for _, pl := range exp.Plans {
		name := pl.Name
		if name == "" {
			name = pl.ID
		}
		code := PlanCode(pl.ID)
		switch {
		case existing.alreadyImported(pl.ID) || existing.hasPlanCode(code):
			planResolves[pl.ID] = true
			p.add(Item{Kind: KindPlan, ChargebeeID: pl.ID, Label: name, Action: ActionSkipImported})
		case pl.Status == "deleted":
			p.add(Item{Kind: KindPlan, ChargebeeID: pl.ID, Label: name, Action: ActionUnsupported, Detail: "plan is deleted in Chargebee"})
		default:
			if _, _, ok := mappedInterval(pl.PeriodUnit, pl.Period); !ok {
				p.add(Item{Kind: KindPlan, ChargebeeID: pl.ID, Label: name, Action: ActionConflict, Detail: fmt.Sprintf("unsupported billing period unit %q", pl.PeriodUnit)})
				continue
			}
			planResolves[pl.ID] = true
			p.add(Item{Kind: KindPlan, ChargebeeID: pl.ID, Label: name, Action: ActionCreate,
				Detail: fmt.Sprintf("%s every %d %s", money(pl.Price, pl.CurrencyCode), intervalCount(pl.Period), pl.PeriodUnit)})
		}
	}

	for _, s := range exp.Subscriptions {
		if existing.alreadyImported(s.ID) {
			p.add(Item{Kind: KindSubscription, ChargebeeID: s.ID, Label: s.ID, Action: ActionSkipImported})
			continue
		}
		if _, ok := importableSubStatus(s.Status); !ok {
			p.add(Item{Kind: KindSubscription, ChargebeeID: s.ID, Label: s.ID, Action: ActionUnsupported, Detail: fmt.Sprintf("Chargebee status %q is not migrated", s.Status)})
			continue
		}
		if !customerResolves[s.CustomerID] {
			p.add(Item{Kind: KindSubscription, ChargebeeID: s.ID, Label: s.ID, Action: ActionConflict, Detail: "subscription's customer is not in this import"})
			continue
		}
		if !planResolves[s.PlanID] {
			p.add(Item{Kind: KindSubscription, ChargebeeID: s.ID, Label: s.ID, Action: ActionConflict, Detail: "subscription's plan is not in this import"})
			continue
		}
		detail := "status " + s.Status
		if s.Status == "non_renewing" {
			detail += "; cancels at term end"
		}
		p.add(Item{Kind: KindSubscription, ChargebeeID: s.ID, Label: s.ID, Action: ActionCreate, Detail: detail})
	}

	if n := p.countAction(ActionConflict); n > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d item(s) can't be imported yet — resolve the conflicts below (usually a missing customer or plan in the same export).", n))
	}
	if n := p.countAction(ActionUnsupported); n > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d item(s) have no Recurso equivalent and will be left behind (deleted/cancelled records, unsupported periods).", n))
	}
	sort.Strings(p.Warnings)
	return p
}

func intervalCount(period int) int {
	if period <= 0 {
		return 1
	}
	return period
}

func money(amount int64, currency string) string {
	cur := strings.ToUpper(currency)
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

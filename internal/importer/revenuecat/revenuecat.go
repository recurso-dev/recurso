// Package revenuecat is the pure engine that migrates a RevenueCat account into
// Recurso: parse an export (subscribers + their subscriptions, and products),
// map each object, and compute a dry-run *Plan* — with no side effects. Mirrors
// internal/importer/{stripe,chargebee}; kept self-contained.
//
// RevenueCat is mobile-subscription infrastructure: subscribers are identified
// by an opaque app_user_id and frequently have NO email. Recurso requires an
// email per customer, so a subscriber without one (its $email attribute) is a
// conflict, not a silent drop — the operator must supply emails to migrate those.
//
// Money alignment: RevenueCat prices and Recurso amounts are both integer minor
// units, so they copy across directly.
package revenuecat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// --- Input types (the subset Recurso maps) ---------------------------------

type Export struct {
	Subscribers []Subscriber `json:"subscribers"`
	Products    []Product    `json:"products"`
}

type Subscriber struct {
	AppUserID     string         `json:"app_user_id"`
	Email         string         `json:"email"` // RevenueCat $email attribute; often absent
	Subscriptions []Subscription `json:"subscriptions"`
}

type Subscription struct {
	ID        string `json:"id"` // optional; synthesized from app_user_id+product if empty
	ProductID string `json:"product_id"`
	Store     string `json:"store"`      // app_store | play_store | stripe | ...
	ExpiresAt int64  `json:"expires_at"` // unix seconds
	IsActive  bool   `json:"is_active"`
	IsTrial   bool   `json:"is_trial"`
}

type Product struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Price       int64  `json:"price"`        // minor units
	Currency    string `json:"currency"`     // ISO 3-letter
	PeriodUnit  string `json:"period_unit"`  // day | week | month | year
	PeriodCount int    `json:"period_count"` // defaults to 1 when 0
}

// SubID returns the subscription's stable id (synthesized when the export omits
// one), used for idempotency + display.
func (s Subscription) SubID(appUserID string) string {
	if s.ID != "" {
		return s.ID
	}
	return appUserID + ":" + s.ProductID
}

// Parse decodes a RevenueCat export from JSON (unknown fields ignored).
func Parse(data []byte) (*Export, error) {
	var exp Export
	if err := json.Unmarshal(data, &exp); err != nil {
		return nil, fmt.Errorf("invalid RevenueCat export JSON: %w", err)
	}
	return &exp, nil
}

// --- Existing Recurso state ------------------------------------------------

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
	Kind         Kind   `json:"kind"`
	RevenueCatID string `json:"revenuecat_id"`
	Label        string `json:"label"`
	Action       Action `json:"action"`
	Detail       string `json:"detail,omitempty"`
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

// CreateCounts returns net-new objects per kind a commit would create.
func (p *ImportPlan) CreateCounts() map[Kind]int {
	out := map[Kind]int{}
	for _, it := range p.Items {
		if it.Action == ActionCreate {
			out[it.Kind]++
		}
	}
	return out
}

// PlanCode is the deterministic Recurso plan code for a RevenueCat product.
func PlanCode(id string) string { return "revenuecat_" + id }

func mappedInterval(unit string, count int) (string, int, bool) {
	switch unit {
	case "day", "week", "month", "year":
		if count <= 0 {
			count = 1
		}
		return unit, count, true
	default:
		return "", 0, false
	}
}

// BuildPlan computes the dry-run outcome of importing exp into a tenant.
func BuildPlan(exp *Export, existing Existing) *ImportPlan {
	p := &ImportPlan{Summary: map[string]int{}}

	// Products → plans.
	planResolves := map[string]bool{}
	for _, pr := range exp.Products {
		name := pr.Title
		if name == "" {
			name = pr.ID
		}
		code := PlanCode(pr.ID)
		switch {
		case existing.alreadyImported(pr.ID) || existing.hasPlanCode(code):
			planResolves[pr.ID] = true
			p.add(Item{Kind: KindPlan, RevenueCatID: pr.ID, Label: name, Action: ActionSkipImported})
		default:
			if _, _, ok := mappedInterval(pr.PeriodUnit, pr.PeriodCount); !ok {
				p.add(Item{Kind: KindPlan, RevenueCatID: pr.ID, Label: name, Action: ActionConflict, Detail: fmt.Sprintf("unsupported billing period unit %q", pr.PeriodUnit)})
				continue
			}
			planResolves[pr.ID] = true
			p.add(Item{Kind: KindPlan, RevenueCatID: pr.ID, Label: name, Action: ActionCreate,
				Detail: fmt.Sprintf("%s every %d %s", money(pr.Price, pr.Currency), intervalCount(pr.PeriodCount), pr.PeriodUnit)})
		}
	}

	// Subscribers → customers, and their subscriptions.
	customerResolves := map[string]bool{}
	for _, sub := range exp.Subscribers {
		label := sub.Email
		if label == "" {
			label = sub.AppUserID
		}
		switch {
		case existing.alreadyImported(sub.AppUserID):
			customerResolves[sub.AppUserID] = true
			p.add(Item{Kind: KindCustomer, RevenueCatID: sub.AppUserID, Label: label, Action: ActionSkipImported})
		case strings.TrimSpace(sub.Email) == "":
			p.add(Item{Kind: KindCustomer, RevenueCatID: sub.AppUserID, Label: label, Action: ActionConflict, Detail: "subscriber has no email (RevenueCat identifies by app_user_id); Recurso requires one"})
		case existing.hasEmail(sub.Email):
			customerResolves[sub.AppUserID] = true
			p.add(Item{Kind: KindCustomer, RevenueCatID: sub.AppUserID, Label: label, Action: ActionLinkExisting, Detail: "matched an existing Recurso customer by email"})
		default:
			customerResolves[sub.AppUserID] = true
			p.add(Item{Kind: KindCustomer, RevenueCatID: sub.AppUserID, Label: label, Action: ActionCreate})
		}

		for _, s := range sub.Subscriptions {
			id := s.SubID(sub.AppUserID)
			if existing.alreadyImported(id) {
				p.add(Item{Kind: KindSubscription, RevenueCatID: id, Label: id, Action: ActionSkipImported})
				continue
			}
			if !s.IsActive {
				p.add(Item{Kind: KindSubscription, RevenueCatID: id, Label: id, Action: ActionUnsupported, Detail: "subscription is not active (expired/lapsed); not migrated"})
				continue
			}
			if !customerResolves[sub.AppUserID] {
				p.add(Item{Kind: KindSubscription, RevenueCatID: id, Label: id, Action: ActionConflict, Detail: "subscriber can't be created (see the customer row)"})
				continue
			}
			if !planResolves[s.ProductID] {
				p.add(Item{Kind: KindSubscription, RevenueCatID: id, Label: id, Action: ActionConflict, Detail: "product is not in this import"})
				continue
			}
			detail := "store " + s.Store
			if s.IsTrial {
				detail += "; in trial"
			}
			p.add(Item{Kind: KindSubscription, RevenueCatID: id, Label: id, Action: ActionCreate, Detail: detail})
		}
	}

	if n := p.countAction(ActionConflict); n > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d item(s) can't be imported yet — usually subscribers with no email (RevenueCat identifies by app_user_id). Add emails to migrate them.", n))
	}
	if n := p.countAction(ActionUnsupported); n > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d item(s) have no Recurso equivalent and will be left behind (inactive/expired subscriptions).", n))
	}
	sort.Strings(p.Warnings)
	return p
}

func intervalCount(count int) int {
	if count <= 0 {
		return 1
	}
	return count
}

func money(amount int64, currency string) string {
	// Exponent-aware hint (covers all zero/three-decimal currencies, not just a
	// hardcoded few); the frontend does the authoritative formatting.
	return domain.FormatMoneyPlain(amount, currency) + " " + strings.ToUpper(strings.TrimSpace(currency))
}

package mcp

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// readToolPaths is the single catalogue of Tier-1 tool → /v1 path template.
// contract_test.go asserts every template exists in cmd/api/openapi.yaml, so a
// renamed route can't silently break a tool. Keep it in sync with the
// registrations below.
var readToolPaths = map[string]string{
	"list_customers":              "/v1/customers",
	"get_customer":                "/v1/customers/{id}",
	"list_subscriptions":          "/v1/subscriptions",
	"get_subscription":            "/v1/subscriptions/{id}",
	"preview_subscription_change": "/v1/subscriptions/{id}/preview-change",
	"get_subscription_usage":      "/v1/subscriptions/{id}/usage",
	"list_invoices":               "/v1/invoices",
	"get_invoice_preview":         "/v1/invoices/{id}/preview",
	"list_plans":                  "/v1/plans",
	"get_plan":                    "/v1/plans/{id}",
	"list_quotes":                 "/v1/quotes",
	"get_quote":                   "/v1/quotes/{id}",
	"list_billable_metrics":       "/v1/billable-metrics",
	"simulate_charges":            "/v1/plans/{id}/simulate-charges",
}

// listInput is the shared pagination/search shape for list_* tools.
type listInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum rows to return (default 100)"`
	Offset int    `json:"offset,omitempty" jsonschema:"rows to skip, for pagination"`
	Q      string `json:"q,omitempty" jsonschema:"optional free-text search"`
}

func (l listInput) query() url.Values {
	q := url.Values{}
	limit := l.Limit
	if limit <= 0 {
		limit = 100 // lists silently truncate without an explicit limit (see CLAUDE.md)
	}
	q.Set("limit", strconv.Itoa(limit))
	if l.Offset > 0 {
		q.Set("offset", strconv.Itoa(l.Offset))
	}
	if l.Q != "" {
		q.Set("q", l.Q)
	}
	return q
}

// idInput fetches one resource by UUID.
type idInput struct {
	ID string `json:"id" jsonschema:"the resource UUID"`
}

type previewChangeInput struct {
	ID     string `json:"id" jsonschema:"the subscription UUID"`
	PlanID string `json:"plan_id" jsonschema:"the plan UUID to preview switching to"`
}

type usageSample struct {
	MetricID string `json:"metric_id" jsonschema:"billable metric UUID"`
	Quantity int64  `json:"quantity" jsonschema:"sample quantity"`
}

type simulateInput struct {
	PlanID         string        `json:"plan_id" jsonschema:"the plan UUID to simulate"`
	SubscriptionID string        `json:"subscription_id,omitempty" jsonschema:"optional subscription UUID; fills usage for metrics without an explicit sample"`
	Currency       string        `json:"currency,omitempty" jsonschema:"optional ISO currency code (defaults to the plan's first price currency)"`
	Charges        any           `json:"charges,omitempty" jsonschema:"optional proposed charge set (array of ChargeInput objects)"`
	Usage          []usageSample `json:"usage,omitempty" jsonschema:"optional sample usage entries"`
}

func (s simulateInput) body() any {
	m := map[string]any{}
	if s.SubscriptionID != "" {
		m["subscription_id"] = s.SubscriptionID
	}
	if s.Currency != "" {
		m["currency"] = s.Currency
	}
	if s.Charges != nil {
		m["charges"] = s.Charges
	}
	if len(s.Usage) > 0 {
		m["usage"] = s.Usage
	}
	return m
}

func requireID(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}
	return nil
}

func seg(id string) string { return url.PathEscape(strings.TrimSpace(id)) }

// registerReadTools wires the 14 Tier-1 read/simulate tools. All are read-only:
// they list or fetch, or (simulate_charges) compute without persisting.
func registerReadTools(s *Server) {
	// Customers
	addReadTool(s, "list_customers", "List customers",
		"List the tenant's customers (paginated). Read-only.",
		func(in listInput) (string, url.Values, error) { return "/v1/customers", in.query(), nil })
	addReadTool(s, "get_customer", "Get customer",
		"Fetch one customer by UUID. Read-only.",
		func(in idInput) (string, url.Values, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, err
			}
			return "/v1/customers/" + seg(in.ID), nil, nil
		})

	// Subscriptions
	addReadTool(s, "list_subscriptions", "List subscriptions",
		"List the tenant's subscriptions (paginated). Read-only.",
		func(in listInput) (string, url.Values, error) { return "/v1/subscriptions", in.query(), nil })
	addReadTool(s, "get_subscription", "Get subscription",
		"Fetch one subscription by UUID. Read-only.",
		func(in idInput) (string, url.Values, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, err
			}
			return "/v1/subscriptions/" + seg(in.ID), nil, nil
		})
	addReadTool(s, "preview_subscription_change", "Preview a plan change",
		"Preview the proration for switching a subscription to another plan — credit for unused time, prorated charge, net, tax, and resulting next-invoice amount. Nothing is charged or persisted. Read-only.",
		func(in previewChangeInput) (string, url.Values, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, err
			}
			if strings.TrimSpace(in.PlanID) == "" {
				return "", nil, errors.New("plan_id is required")
			}
			q := url.Values{}
			q.Set("plan_id", strings.TrimSpace(in.PlanID))
			return "/v1/subscriptions/" + seg(in.ID) + "/preview-change", q, nil
		})
	addReadTool(s, "get_subscription_usage", "Get subscription usage",
		"Fetch the current-period metered usage for a subscription. Read-only.",
		func(in idInput) (string, url.Values, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, err
			}
			return "/v1/subscriptions/" + seg(in.ID) + "/usage", nil, nil
		})

	// Invoices
	addReadTool(s, "list_invoices", "List invoices",
		"List the tenant's invoices (paginated). Read-only.",
		func(in listInput) (string, url.Values, error) { return "/v1/invoices", in.query(), nil })
	addReadTool(s, "get_invoice_preview", "Preview invoice",
		"Fetch the rendered preview of an invoice by UUID. Read-only.",
		func(in idInput) (string, url.Values, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, err
			}
			return "/v1/invoices/" + seg(in.ID) + "/preview", nil, nil
		})

	// Plans / catalog
	addReadTool(s, "list_plans", "List plans",
		"List the tenant's plans (paginated). Read-only.",
		func(in listInput) (string, url.Values, error) { return "/v1/plans", in.query(), nil })
	addReadTool(s, "get_plan", "Get plan",
		"Fetch one plan by UUID, including its charges. Read-only.",
		func(in idInput) (string, url.Values, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, err
			}
			return "/v1/plans/" + seg(in.ID), nil, nil
		})

	// Quotes
	addReadTool(s, "list_quotes", "List quotes",
		"List the tenant's quotes (paginated). Read-only.",
		func(in listInput) (string, url.Values, error) { return "/v1/quotes", in.query(), nil })
	addReadTool(s, "get_quote", "Get quote",
		"Fetch one quote by UUID. Read-only.",
		func(in idInput) (string, url.Values, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, err
			}
			return "/v1/quotes/" + seg(in.ID), nil, nil
		})

	// Billable metrics
	addReadTool(s, "list_billable_metrics", "List billable metrics",
		"List the tenant's billable (usage) metrics. Read-only.",
		func(in listInput) (string, url.Values, error) { return "/v1/billable-metrics", in.query(), nil })

	// Pricing simulator (POST, read-only — nothing is persisted)
	addSimTool(s, "simulate_charges", "Simulate charges",
		"Rate a plan's charges (or a proposed charge set) against sample or existing usage and return the priced lines, subtotal, and a balanced general-ledger preview. Pre-tax; nothing is persisted. Read-only.",
		func(in simulateInput) (string, any, error) {
			if err := requireID(in.PlanID); err != nil {
				return "", nil, err
			}
			return "/v1/plans/" + seg(in.PlanID) + "/simulate-charges", in.body(), nil
		})
}

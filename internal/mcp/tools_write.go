package mcp

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// writeToolPaths catalogues each Tier-2 tool → (method, /v1 path template).
// contract_test.go asserts every path exists in cmd/api/openapi.yaml. Keep it
// in sync with the registrations below.
var writeToolPaths = map[string]string{
	"create_customer":     "/v1/customers",
	"update_customer":     "/v1/customers/{id}",
	"record_usage_event":  "/v1/usage/events",
	"record_usage_batch":  "/v1/usage/events/batch",
	"create_subscription": "/v1/subscriptions",
	"update_subscription": "/v1/subscriptions/{id}",
	"create_quote":        "/v1/quotes",
	"update_quote":        "/v1/quotes/{id}",
	"send_quote":          "/v1/quotes/{id}/send",
}

// idem carries an optional caller-supplied idempotency key. Embedding it keeps
// every write input consistent: pass a stable key to make retries safe.
type idem struct {
	IdempotencyKey string `json:"idempotency_key,omitempty" jsonschema:"optional stable key so a retried call is not applied twice; one is generated if omitted"`
}

type createCustomerInput struct {
	idem
	Email   string `json:"email" jsonschema:"customer email (required)"`
	Name    string `json:"name" jsonschema:"customer name (required)"`
	Phone   string `json:"phone,omitempty" jsonschema:"optional phone"`
	Country string `json:"country,omitempty" jsonschema:"optional ISO 3166-1 alpha-2 country code"`
	TaxID   string `json:"tax_id,omitempty" jsonschema:"optional tax identifier"`
	GSTIN   string `json:"gstin,omitempty" jsonschema:"optional Indian GSTIN (B2B)"`
	TaxType string `json:"tax_type,omitempty" jsonschema:"optional: business or consumer"`
}

func (in createCustomerInput) body() any {
	m := map[string]any{"email": in.Email, "name": in.Name}
	putIf(m, "phone", in.Phone)
	putIf(m, "country", in.Country)
	putIf(m, "tax_id", in.TaxID)
	putIf(m, "gstin", in.GSTIN)
	putIf(m, "tax_type", in.TaxType)
	return m
}

type updateCustomerInput struct {
	idem
	ID    string `json:"id" jsonschema:"customer UUID (required)"`
	Email string `json:"email,omitempty" jsonschema:"new email"`
	Name  string `json:"name,omitempty" jsonschema:"new name"`
	Phone string `json:"phone,omitempty" jsonschema:"new phone"`
}

func (in updateCustomerInput) body() any {
	m := map[string]any{}
	putIf(m, "email", in.Email)
	putIf(m, "name", in.Name)
	putIf(m, "phone", in.Phone)
	return m
}

type recordUsageInput struct {
	idem
	SubscriptionID string            `json:"subscription_id" jsonschema:"subscription UUID (required)"`
	CustomerID     string            `json:"customer_id" jsonschema:"customer UUID; must match the subscription's customer (required)"`
	Dimension      string            `json:"dimension" jsonschema:"metered dimension, e.g. api_calls (required)"`
	Quantity       int64             `json:"quantity" jsonschema:"quantity to record (required)"`
	TransactionID  string            `json:"transaction_id,omitempty" jsonschema:"optional per-event dedup key: (subscription, transaction_id) collapses retries"`
	DynamicAmount  int64             `json:"dynamic_amount,omitempty" jsonschema:"optional minor-unit amount for dynamic charges"`
	Properties     map[string]string `json:"properties,omitempty" jsonschema:"optional free-form attributes (used by unique-count aggregation)"`
}

func (in recordUsageInput) body() any {
	m := map[string]any{
		"subscription_id": in.SubscriptionID,
		"customer_id":     in.CustomerID,
		"dimension":       in.Dimension,
		"quantity":        in.Quantity,
	}
	putIf(m, "transaction_id", in.TransactionID)
	if in.DynamicAmount != 0 {
		m["dynamic_amount"] = in.DynamicAmount
	}
	if len(in.Properties) > 0 {
		m["properties"] = in.Properties
	}
	return m
}

type recordUsageBatchInput struct {
	idem
	Events []recordUsageInput `json:"events" jsonschema:"up to 500 usage events to record in one call (required)"`
}

func (in recordUsageBatchInput) body() any {
	events := make([]any, 0, len(in.Events))
	for _, e := range in.Events {
		events = append(events, e.body())
	}
	return map[string]any{"events": events}
}

type createSubscriptionInput struct {
	idem
	CustomerID   string `json:"customer_id" jsonschema:"customer UUID (required)"`
	PlanID       string `json:"plan_id" jsonschema:"plan UUID (required)"`
	CouponCode   string `json:"coupon_code,omitempty" jsonschema:"optional coupon code"`
	TrialDays    int    `json:"trial_days,omitempty" jsonschema:"optional trial length in days"`
	StartDate    string `json:"start_date,omitempty" jsonschema:"optional RFC3339 start date"`
	PaymentTerms string `json:"payment_terms,omitempty" jsonschema:"optional: net0, net15, net30, net60"`
}

func (in createSubscriptionInput) body() any {
	m := map[string]any{"customer_id": in.CustomerID, "plan_id": in.PlanID}
	putIf(m, "coupon_code", in.CouponCode)
	putIf(m, "start_date", in.StartDate)
	putIf(m, "payment_terms", in.PaymentTerms)
	if in.TrialDays > 0 {
		m["trial_days"] = in.TrialDays
	}
	return m
}

type updateSubscriptionInput struct {
	idem
	ID     string `json:"id" jsonschema:"subscription UUID (required)"`
	PlanID string `json:"plan_id,omitempty" jsonschema:"plan UUID to switch to (prorated)"`
}

func (in updateSubscriptionInput) body() any {
	m := map[string]any{}
	putIf(m, "plan_id", in.PlanID)
	return m
}

type createQuoteInput struct {
	idem
	CustomerID string `json:"customer_id" jsonschema:"customer UUID (required)"`
	LineItems  any    `json:"line_items" jsonschema:"array of line items (required); each has description, quantity, and unit amount in minor units"`
	Currency   string `json:"currency,omitempty" jsonschema:"optional ISO currency code"`
	Notes      string `json:"notes,omitempty" jsonschema:"optional notes"`
	ValidUntil string `json:"valid_until,omitempty" jsonschema:"optional RFC3339 expiry"`
}

func (in createQuoteInput) body() any {
	m := map[string]any{"customer_id": in.CustomerID, "line_items": in.LineItems}
	putIf(m, "currency", in.Currency)
	putIf(m, "notes", in.Notes)
	putIf(m, "valid_until", in.ValidUntil)
	return m
}

type updateQuoteInput struct {
	idem
	ID        string `json:"id" jsonschema:"quote UUID (required)"`
	LineItems any    `json:"line_items,omitempty" jsonschema:"replacement line items"`
	Notes     string `json:"notes,omitempty" jsonschema:"updated notes"`
}

func (in updateQuoteInput) body() any {
	m := map[string]any{}
	if in.LineItems != nil {
		m["line_items"] = in.LineItems
	}
	putIf(m, "notes", in.Notes)
	return m
}

// putIf sets m[k]=v only when v is non-empty, keeping request bodies minimal.
func putIf(m map[string]any, k, v string) {
	if strings.TrimSpace(v) != "" {
		m[k] = v
	}
}

// registerWriteTools wires the Tier-2 idempotent write tools. Each is a curated
// create/update/record operation; none is destructive. They are on by default
// but every call carries an Idempotency-Key.
func registerWriteTools(s *Server) {
	addWriteTool(s, http.MethodPost, "create_customer", "Create customer",
		"Create a customer. Idempotent when you pass a stable idempotency_key.",
		func(in createCustomerInput) (string, any, string, error) {
			if strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Name) == "" {
				return "", nil, "", errors.New("email and name are required")
			}
			return "/v1/customers", in.body(), in.IdempotencyKey, nil
		})

	addWriteTool(s, http.MethodPut, "update_customer", "Update customer",
		"Update a customer's details by UUID.",
		func(in updateCustomerInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			return "/v1/customers/" + seg(in.ID), in.body(), in.IdempotencyKey, nil
		})

	addWriteTool(s, http.MethodPost, "record_usage_event", "Record usage event",
		"Record a metered usage event against a subscription dimension. Pass transaction_id to dedupe retried events.",
		func(in recordUsageInput) (string, any, string, error) {
			if err := validateUsage(in); err != nil {
				return "", nil, "", err
			}
			return "/v1/usage/events", in.body(), in.IdempotencyKey, nil
		})

	addWriteTool(s, http.MethodPost, "record_usage_batch", "Record usage events (batch)",
		"Record up to 500 usage events in one call.",
		func(in recordUsageBatchInput) (string, any, string, error) {
			if len(in.Events) == 0 {
				return "", nil, "", errors.New("events must not be empty")
			}
			if len(in.Events) > 500 {
				return "", nil, "", errors.New("at most 500 events per batch")
			}
			for i, e := range in.Events {
				if err := validateUsage(e); err != nil {
					return "", nil, "", errors.New("event " + strconv.Itoa(i) + ": " + err.Error())
				}
			}
			return "/v1/usage/events/batch", in.body(), in.IdempotencyKey, nil
		})

	addWriteTool(s, http.MethodPost, "create_subscription", "Create subscription",
		"Subscribe a customer to a plan. Idempotent when you pass a stable idempotency_key.",
		func(in createSubscriptionInput) (string, any, string, error) {
			if strings.TrimSpace(in.CustomerID) == "" || strings.TrimSpace(in.PlanID) == "" {
				return "", nil, "", errors.New("customer_id and plan_id are required")
			}
			return "/v1/subscriptions", in.body(), in.IdempotencyKey, nil
		})

	addWriteTool(s, http.MethodPut, "update_subscription", "Update subscription",
		"Update a subscription — e.g. switch plans (prorated) by UUID.",
		func(in updateSubscriptionInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			return "/v1/subscriptions/" + seg(in.ID), in.body(), in.IdempotencyKey, nil
		})

	addWriteTool(s, http.MethodPost, "create_quote", "Create quote",
		"Draft a quote for a customer with one or more line items.",
		func(in createQuoteInput) (string, any, string, error) {
			if strings.TrimSpace(in.CustomerID) == "" {
				return "", nil, "", errors.New("customer_id is required")
			}
			if in.LineItems == nil {
				return "", nil, "", errors.New("line_items is required")
			}
			return "/v1/quotes", in.body(), in.IdempotencyKey, nil
		})

	addWriteTool(s, http.MethodPut, "update_quote", "Update quote",
		"Update a draft quote by UUID.",
		func(in updateQuoteInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			return "/v1/quotes/" + seg(in.ID), in.body(), in.IdempotencyKey, nil
		})

	addWriteTool(s, http.MethodPost, "send_quote", "Send quote",
		"Send a drafted quote to the customer by UUID.",
		func(in idInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			return "/v1/quotes/" + seg(in.ID) + "/send", map[string]any{}, "", nil
		})
}

func validateUsage(in recordUsageInput) error {
	if strings.TrimSpace(in.SubscriptionID) == "" ||
		strings.TrimSpace(in.CustomerID) == "" ||
		strings.TrimSpace(in.Dimension) == "" {
		return errors.New("subscription_id, customer_id and dimension are required")
	}
	return nil
}

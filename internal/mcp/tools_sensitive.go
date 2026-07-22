package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// sensitiveToolPaths catalogues each Tier-3 tool → /v1 path template.
// contract_test.go asserts every path exists in cmd/api/openapi.yaml.
var sensitiveToolPaths = map[string]string{
	"convert_quote_to_invoice": "/v1/quotes/{id}/convert",
	"cancel_subscription":      "/v1/subscriptions/{id}/cancel",
	"create_credit_note":       "/v1/credit-notes",
	"wallet_top_up":            "/v1/wallets/{id}/top-up",
	"add_subscription_charge":  "/v1/subscriptions/{id}/charges",
	"bill_usage_now":           "/v1/subscriptions/{id}/bill-usage",
}

// tier3DisabledMsg is returned when a tenant has not opted in to money-path tools.
const tier3DisabledMsg = "money-path MCP tools are disabled for this tenant — enable them in Settings → MCP (tier3_enabled) before an agent can run this action"

// tier3Enabled asks /v1 whether this caller's tenant has opted in. Fail-closed:
// any error (unreachable, unauthorized, malformed) is treated as NOT enabled, so
// a destructive tool never runs unless the tenant is provably opted in.
func (s *Server) tier3Enabled(ctx context.Context, key string) bool {
	body, apiErr := s.client.Get(ctx, key, "/v1/settings/mcp", nil)
	if apiErr != nil {
		return false
	}
	var env struct {
		Data struct {
			Tier3Enabled bool `json:"tier3_enabled"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &env) != nil {
		return false
	}
	return env.Data.Tier3Enabled
}

// addSensitiveTool registers a Tier-3 money-path tool if Tier3 is enabled in the
// server policy. At call time it ALSO checks the tenant's per-tenant opt-in via
// /v1 and refuses unless enabled. Every call carries an Idempotency-Key.
func addSensitiveTool[In any](s *Server, method, name, title, desc string, build func(In) (path string, body any, idemKey string, err error)) {
	if !Tier3.enabled(s.allow) {
		return
	}
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        name,
		Description: desc,
		Annotations: Tier3.annotations(title),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		key, aerr := s.resolveKey(req)
		if aerr != nil {
			return errResult(aerr.Error()), nil, nil
		}
		if !s.tier3Enabled(ctx, key) {
			return errResult(tier3DisabledMsg), nil, nil
		}
		path, b, idem, err := build(in)
		if err != nil {
			return errResult(err.Error()), nil, nil
		}
		if idem == "" {
			idem = uuid.NewString()
		}
		var body []byte
		if method == http.MethodPut {
			body, aerr = s.client.Put(ctx, key, path, b, idem)
		} else {
			body, aerr = s.client.Post(ctx, key, path, b, idem)
		}
		if aerr != nil {
			return errResult(aerr.Error()), nil, nil
		}
		return jsonResult(body), nil, nil
	})
}

// --- Tier-3 tool inputs ---

type convertQuoteInput struct {
	idem
	ID string `json:"id" jsonschema:"quote UUID to convert to an invoice (required)"`
}

type cancelSubscriptionInput struct {
	idem
	ID          string `json:"id" jsonschema:"subscription UUID to cancel (required)"`
	AtPeriodEnd bool   `json:"at_period_end,omitempty" jsonschema:"cancel at the end of the current period instead of immediately"`
	Reason      string `json:"reason,omitempty" jsonschema:"optional cancellation reason"`
}

func (in cancelSubscriptionInput) body() any {
	m := map[string]any{}
	if in.AtPeriodEnd {
		m["at_period_end"] = true
	}
	putIf(m, "reason", in.Reason)
	return m
}

type createCreditNoteInput struct {
	idem
	CustomerID string `json:"customer_id" jsonschema:"customer UUID (required)"`
	Amount     int64  `json:"amount" jsonschema:"credit amount in minor units (required)"`
	Currency   string `json:"currency,omitempty" jsonschema:"ISO currency code"`
	Reason     string `json:"reason,omitempty" jsonschema:"reason for the credit"`
	InvoiceID  string `json:"invoice_id,omitempty" jsonschema:"optional invoice UUID to apply the credit against"`
}

func (in createCreditNoteInput) body() any {
	m := map[string]any{"customer_id": in.CustomerID, "amount": in.Amount}
	putIf(m, "currency", in.Currency)
	putIf(m, "reason", in.Reason)
	putIf(m, "invoice_id", in.InvoiceID)
	return m
}

type walletTopUpInput struct {
	idem
	ID       string `json:"id" jsonschema:"wallet UUID (required)"`
	Amount   int64  `json:"amount" jsonschema:"top-up amount in minor units (required)"`
	Currency string `json:"currency,omitempty" jsonschema:"ISO currency code"`
}

func (in walletTopUpInput) body() any {
	m := map[string]any{"amount": in.Amount}
	putIf(m, "currency", in.Currency)
	return m
}

type addChargeInput struct {
	idem
	ID          string `json:"id" jsonschema:"subscription UUID (required)"`
	Amount      int64  `json:"amount" jsonschema:"charge amount in minor units (required)"`
	Description string `json:"description,omitempty" jsonschema:"line description"`
}

func (in addChargeInput) body() any {
	m := map[string]any{"amount": in.Amount}
	putIf(m, "description", in.Description)
	return m
}

// registerSensitiveTools wires the Tier-3 money-path tools. They are OFF by
// default (both in the server policy and per-tenant opt-in) and annotated
// destructive; each call is idempotency-keyed.
func registerSensitiveTools(s *Server) {
	addSensitiveTool(s, http.MethodPost, "convert_quote_to_invoice", "Convert quote to invoice",
		"Convert an accepted quote into an invoice. Money-path — creates a billable invoice.",
		func(in convertQuoteInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			return "/v1/quotes/" + seg(in.ID) + "/convert", map[string]any{}, in.IdempotencyKey, nil
		})

	addSensitiveTool(s, http.MethodPost, "cancel_subscription", "Cancel subscription",
		"Cancel a subscription (immediately or at period end). Money-path — stops future billing.",
		func(in cancelSubscriptionInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			return "/v1/subscriptions/" + seg(in.ID) + "/cancel", in.body(), in.IdempotencyKey, nil
		})

	addSensitiveTool(s, http.MethodPost, "create_credit_note", "Create credit note",
		"Issue a credit note / refund to a customer. Money-path — moves money.",
		func(in createCreditNoteInput) (string, any, string, error) {
			if strings.TrimSpace(in.CustomerID) == "" {
				return "", nil, "", errors.New("customer_id is required")
			}
			if in.Amount <= 0 {
				return "", nil, "", errors.New("amount must be positive (minor units)")
			}
			return "/v1/credit-notes", in.body(), in.IdempotencyKey, nil
		})

	addSensitiveTool(s, http.MethodPost, "wallet_top_up", "Top up wallet",
		"Add prepaid balance to a customer wallet. Money-path.",
		func(in walletTopUpInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			if in.Amount <= 0 {
				return "", nil, "", errors.New("amount must be positive (minor units)")
			}
			return "/v1/wallets/" + seg(in.ID) + "/top-up", in.body(), in.IdempotencyKey, nil
		})

	addSensitiveTool(s, http.MethodPost, "add_subscription_charge", "Add one-off charge",
		"Add a one-off charge to a subscription's next invoice. Money-path.",
		func(in addChargeInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			if in.Amount <= 0 {
				return "", nil, "", errors.New("amount must be positive (minor units)")
			}
			return "/v1/subscriptions/" + seg(in.ID) + "/charges", in.body(), in.IdempotencyKey, nil
		})

	addSensitiveTool(s, http.MethodPost, "bill_usage_now", "Bill usage now",
		"Generate an interim invoice for a subscription's accrued usage now. Money-path.",
		func(in idInput) (string, any, string, error) {
			if err := requireID(in.ID); err != nil {
				return "", nil, "", err
			}
			return "/v1/subscriptions/" + seg(in.ID) + "/bill-usage", map[string]any{}, "", nil
		})
}

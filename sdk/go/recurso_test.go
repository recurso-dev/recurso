package recurso

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testServer spins up a mock HTTP server, points a Client at it, and records
// the last request it received so tests can assert on method/path/auth/body.
type testServer struct {
	t      *testing.T
	server *httptest.Server
	client *Client

	// captured request details
	method string
	path   string
	query  string
	auth   string
	accept string
	ctype  string
	body   []byte
}

// newTestServer returns a server that replies with the given status code and
// raw JSON body for every request.
func newTestServer(t *testing.T, status int, respBody string) *testServer {
	t.Helper()
	ts := &testServer{t: t}
	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.method = r.Method
		ts.path = r.URL.Path
		ts.query = r.URL.RawQuery
		ts.auth = r.Header.Get("Authorization")
		ts.accept = r.Header.Get("Accept")
		ts.ctype = r.Header.Get("Content-Type")
		ts.body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, respBody)
	}))
	t.Cleanup(ts.server.Close)
	ts.client = NewClient("test_key", WithBaseURL(ts.server.URL))
	return ts
}

// assertRequest checks the recorded method, path, and bearer auth header.
func (ts *testServer) assertRequest(method, path string) {
	ts.t.Helper()
	if ts.method != method {
		ts.t.Errorf("method = %q, want %q", ts.method, method)
	}
	if ts.path != path {
		ts.t.Errorf("path = %q, want %q", ts.path, path)
	}
	if ts.auth != "Bearer test_key" {
		ts.t.Errorf("Authorization = %q, want %q", ts.auth, "Bearer test_key")
	}
	if ts.accept != "application/json" {
		ts.t.Errorf("Accept = %q, want application/json", ts.accept)
	}
}

// bodyField decodes the recorded request body and returns a top-level field.
func (ts *testServer) bodyMap() map[string]any {
	ts.t.Helper()
	var m map[string]any
	if len(ts.body) == 0 {
		return m
	}
	if err := json.Unmarshal(ts.body, &m); err != nil {
		ts.t.Fatalf("request body is not a JSON object: %v (%s)", err, ts.body)
	}
	return m
}

func TestPlansCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"plan_1","name":"Pro","code":"PRO-USD","interval_unit":"month","interval_count":1,"active":true,"prices":[{"id":"pr_1","currency":"USD","amount":2900,"type":"recurring"}]}`)

	plan, err := ts.client.Plans.Create(context.Background(), &PlanCreateParams{
		Name: "Pro", Code: "PRO-USD", Amount: 2900, Currency: "USD", IntervalUnit: "month", IntervalCount: 1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/plans")
	if ts.ctype != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ts.ctype)
	}
	body := ts.bodyMap()
	if body["code"] != "PRO-USD" {
		t.Errorf("body code = %v, want PRO-USD", body["code"])
	}
	if body["amount"].(float64) != 2900 {
		t.Errorf("body amount = %v, want 2900", body["amount"])
	}
	if plan.ID != "plan_1" || plan.Active != true {
		t.Errorf("plan = %+v", plan)
	}
	if len(plan.Prices) != 1 || plan.Prices[0].Amount != 2900 {
		t.Errorf("prices = %+v", plan.Prices)
	}
}

func TestPlansList(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"plan_1","name":"Pro"},{"id":"plan_2","name":"Team"}]}`)
	plans, err := ts.client.Plans.List(context.Background(), &PlanListParams{Limit: 10, Page: 2, Q: "pro"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/plans")
	if ts.query != "limit=10&page=2&q=pro" {
		t.Errorf("query = %q", ts.query)
	}
	if len(plans) != 2 || plans[1].Name != "Team" {
		t.Errorf("plans = %+v", plans)
	}
}

func TestCustomersCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"cus_1","email":"jane@example.com","name":"Jane"}`)
	cus, err := ts.client.Customers.Create(context.Background(), &CustomerCreateParams{Email: "jane@example.com", Name: "Jane", Country: "US"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/customers")
	if ts.bodyMap()["country"] != "US" {
		t.Errorf("country not sent: %s", ts.body)
	}
	if cus.ID != "cus_1" {
		t.Errorf("cus = %+v", cus)
	}
}

func TestCustomersConsents(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"object":"list","data":[{"id":"con_1","consent_type":"recurring_billing","granted":true}]}`)
	consents, err := ts.client.Customers.Consents(context.Background(), "cus_1")
	if err != nil {
		t.Fatalf("Consents: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/customers/cus_1/consents")
	if len(consents) != 1 || !consents[0].Granted {
		t.Errorf("consents = %+v", consents)
	}
}

func TestSubscriptionsCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"sub_1","customer_id":"cus_1","plan_id":"plan_1","status":"active"}`)
	sub, err := ts.client.Subscriptions.Create(context.Background(), &SubscriptionCreateParams{CustomerID: "cus_1", PlanID: "plan_1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/subscriptions")
	if sub.Status != "active" {
		t.Errorf("sub = %+v", sub)
	}
}

func TestSubscriptionsCancel(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"id":"sub_1","status":"canceled","cancel_at_period_end":true,"message":"ok"}`)
	immediately := false
	res, err := ts.client.Subscriptions.Cancel(context.Background(), "sub_1", &SubscriptionCancelParams{Reason: "too_expensive", Immediately: &immediately})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/subscriptions/sub_1/cancel")
	if ts.bodyMap()["reason"] != "too_expensive" {
		t.Errorf("reason not sent: %s", ts.body)
	}
	if res.Status != "canceled" || !res.CancelAtPeriodEnd {
		t.Errorf("res = %+v", res)
	}
}

func TestSubscriptionsPause(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"sub_1","status":"paused"}}`)
	sub, err := ts.client.Subscriptions.Pause(context.Background(), "sub_1")
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/subscriptions/sub_1/pause")
	if sub.Status != "paused" {
		t.Errorf("sub = %+v", sub)
	}
}

func TestSubscriptionsPreviewChange(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"subscription_id":"sub_1","new_plan_id":"plan_2","total_amount":1500,"is_upgrade":true}`)
	preview, err := ts.client.Subscriptions.PreviewChange(context.Background(), "sub_1", "plan_2")
	if err != nil {
		t.Fatalf("PreviewChange: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/subscriptions/sub_1/preview-change")
	if ts.query != "plan_id=plan_2" {
		t.Errorf("query = %q", ts.query)
	}
	if !preview.IsUpgrade || preview.TotalAmount != 1500 {
		t.Errorf("preview = %+v", preview)
	}
}

func TestSubscriptionsAddAddon(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"addon_1","subscription_id":"sub_1","plan_id":"plan_2","quantity":3}`)
	addon, err := ts.client.Subscriptions.AddAddon(context.Background(), "sub_1", &AddonCreateParams{PlanID: "plan_2", Quantity: 3})
	if err != nil {
		t.Fatalf("AddAddon: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/subscriptions/sub_1/addons")
	if addon.Quantity != 3 {
		t.Errorf("addon = %+v", addon)
	}
}

func TestSubscriptionsRemoveAddon(t *testing.T) {
	ts := newTestServer(t, http.StatusNoContent, ``)
	if err := ts.client.Subscriptions.RemoveAddon(context.Background(), "sub_1", "addon_1"); err != nil {
		t.Fatalf("RemoveAddon: %v", err)
	}
	ts.assertRequest(http.MethodDelete, "/subscriptions/sub_1/addons/addon_1")
}

func TestInvoicesList(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"inv_1","total":11800,"currency":"INR"}]}`)
	invoices, err := ts.client.Invoices.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/invoices")
	if len(invoices) != 1 || invoices[0].Total != 11800 {
		t.Errorf("invoices = %+v", invoices)
	}
}

func TestInvoicesPDFURL(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, ``)
	got := ts.client.Invoices.PDFURL("inv_1")
	want := ts.server.URL + "/invoices/inv_1/pdf"
	if got != want {
		t.Errorf("PDFURL = %q, want %q", got, want)
	}
}

func TestCouponsCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"cpn_1","code":"SAVE20","discount_type":"percent","discount_value":20,"duration":"once"}`)
	coupon, err := ts.client.Coupons.Create(context.Background(), &CouponCreateParams{Code: "SAVE20", DiscountType: "percent", DiscountValue: 20, Duration: "once"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/coupons")
	if coupon.Code != "SAVE20" {
		t.Errorf("coupon = %+v", coupon)
	}
}

func TestUsageRecord(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"status":"recorded","event_id":"evt_1"}`)
	res, err := ts.client.Usage.Record(context.Background(), &UsageRecordParams{SubscriptionID: "sub_1", CustomerID: "cus_1", Dimension: "api_calls", Quantity: 42})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/usage/events")
	if res.EventID != "evt_1" || res.Status != "recorded" {
		t.Errorf("res = %+v", res)
	}
}

func TestUsageQuery(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"dimension":"api_calls","quantity":100}],"granularity":"day"}`)
	res, err := ts.client.Usage.Query(context.Background(), &UsageQueryParams{SubscriptionID: "sub_1", Granularity: "day"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/usage")
	if ts.query != "granularity=day&subscription_id=sub_1" {
		t.Errorf("query = %q", ts.query)
	}
	if len(res.Data) != 1 || res.Data[0].Quantity != 100 {
		t.Errorf("res = %+v", res)
	}
}

func TestCreditNotesCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"cn_1","amount":500,"currency":"USD","status":"issued"}}`)
	cn, err := ts.client.CreditNotes.Create(context.Background(), &CreditNoteCreateParams{CustomerID: "cus_1", Amount: 500, Currency: "USD"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/credit-notes")
	if cn.Amount != 500 || cn.Status != "issued" {
		t.Errorf("cn = %+v", cn)
	}
}

func TestQuotesLifecycle(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"q_1","status":"sent"},"message":"sent"}`)
	quote, err := ts.client.Quotes.Send(context.Background(), "q_1")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/quotes/q_1/send")
	if quote.Status != "sent" {
		t.Errorf("quote = %+v", quote)
	}
}

func TestQuotesCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"q_1","status":"draft","total":10000}}`)
	quote, err := ts.client.Quotes.Create(context.Background(), &QuoteCreateParams{
		CustomerID: "cus_1",
		LineItems:  []LineItem{{Description: "Setup", Quantity: 1, UnitPrice: 10000, Amount: 10000}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/quotes")
	if quote.Total != 10000 {
		t.Errorf("quote = %+v", quote)
	}
}

func TestWebhooksCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"data":{"id":"wh_1","url":"https://x.test/hook","secret":"whsec_abc","events":["invoice.paid"]}}`)
	wh, err := ts.client.Webhooks.Create(context.Background(), &WebhookCreateParams{URL: "https://x.test/hook", Events: []string{"invoice.paid"}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/webhooks")
	if wh.Secret != "whsec_abc" || len(wh.Events) != 1 {
		t.Errorf("wh = %+v", wh)
	}
}

func TestWebhooksDeliveries(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"del_1","status":"succeeded","attempts":1}]}`)
	dels, err := ts.client.Webhooks.Deliveries(context.Background(), "wh_1", &DeliveryListParams{Limit: 5, Status: "succeeded"})
	if err != nil {
		t.Fatalf("Deliveries: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/webhooks/wh_1/deliveries")
	if ts.query != "limit=5&status=succeeded" {
		t.Errorf("query = %q", ts.query)
	}
	if len(dels) != 1 || dels[0].Status != "succeeded" {
		t.Errorf("dels = %+v", dels)
	}
}

func TestEventsRedeliver(t *testing.T) {
	ts := newTestServer(t, http.StatusAccepted, `{"data":{"event_id":"evt_1","deliveries_queued":2}}`)
	res, err := ts.client.Events.Redeliver(context.Background(), "evt_1")
	if err != nil {
		t.Fatalf("Redeliver: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/events/evt_1/redeliver")
	if res.DeliveriesQueued != 2 {
		t.Errorf("res = %+v", res)
	}
}

func TestEventsTypes(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":["invoice.paid","subscription.created"]}`)
	types, err := ts.client.Events.Types(context.Background())
	if err != nil {
		t.Fatalf("Types: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/events/types")
	if len(types) != 2 || types[0] != "invoice.paid" {
		t.Errorf("types = %+v", types)
	}
}

func TestEntitlementsCheck(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"feature_key":"seats","granted":true,"limit_value":25}`)
	check, err := ts.client.Entitlements.Check(context.Background(), "cus_1", "seats")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/entitlements/check")
	if ts.query != "customer_id=cus_1&feature=seats" {
		t.Errorf("query = %q", ts.query)
	}
	if !check.Granted || check.LimitValue == nil || *check.LimitValue != 25 {
		t.Errorf("check = %+v", check)
	}
}

func TestEntitlementsSetForPlan(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"ent_1","feature_key":"seats","kind":"limit","limit_value":10}]}`)
	limit := int64(10)
	ents, err := ts.client.Entitlements.SetForPlan(context.Background(), "plan_1", []EntitlementInput{{FeatureKey: "seats", Kind: "limit", LimitValue: &limit}})
	if err != nil {
		t.Fatalf("SetForPlan: %v", err)
	}
	ts.assertRequest(http.MethodPut, "/plans/plan_1/entitlements")
	// Body should be a bare JSON array.
	var arr []map[string]any
	if err := json.Unmarshal(ts.body, &arr); err != nil {
		t.Fatalf("body is not a JSON array: %v (%s)", err, ts.body)
	}
	if len(arr) != 1 || arr[0]["feature_key"] != "seats" {
		t.Errorf("body = %s", ts.body)
	}
	if len(ents) != 1 || ents[0].LimitValue == nil || *ents[0].LimitValue != 10 {
		t.Errorf("ents = %+v", ents)
	}
}

func TestAnalyticsMRR(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"currency":"USD","mrr":250000,"normalized_mrr":250000,"reporting_currency":"USD","fx":{"source":"live"}}`)
	mrr, err := ts.client.Analytics.MRR(context.Background())
	if err != nil {
		t.Fatalf("MRR: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/analytics/mrr")
	if mrr.MRR != 250000 || mrr.FX.Source != "live" {
		t.Errorf("mrr = %+v", mrr)
	}
}

func TestDeveloperCreateKey(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"key_1","key_value":"rsk_live_secret","key_prefix":"rsk_live","is_active":true}`)
	key, err := ts.client.Developer.CreateKey(context.Background())
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/developer/keys")
	if key.KeyValue != "rsk_live_secret" {
		t.Errorf("key = %+v", key)
	}
}

func TestAccountGet(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"id":"ten_1","name":"Acme","email":"ops@acme.test","base_currency":"USD"}}`)
	acct, err := ts.client.Account.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/account")
	if acct.Name != "Acme" || acct.BaseCurrency != "USD" {
		t.Errorf("acct = %+v", acct)
	}
}

func TestReferralsGenerateCode(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":{"code":"REF-ABC123"}}`)
	code, err := ts.client.Referrals.GenerateCode(context.Background(), "cus_1")
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/referrals/generate-code")
	if ts.bodyMap()["customer_id"] != "cus_1" {
		t.Errorf("customer_id not sent: %s", ts.body)
	}
	if code.Code != "REF-ABC123" {
		t.Errorf("code = %+v", code)
	}
}

func TestGiftsPurchase(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"id":"gift_1","code":"GIFT-XYZ","status":"purchased","duration_months":3}`)
	gift, err := ts.client.Gifts.Purchase(context.Background(), &GiftPurchaseParams{BuyerCustomerID: "cus_1", PlanID: "plan_1", DurationMonths: 3})
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/gifts/purchase")
	if gift.Code != "GIFT-XYZ" || gift.DurationMonths != 3 {
		t.Errorf("gift = %+v", gift)
	}
}

func TestMandatesCreate(t *testing.T) {
	ts := newTestServer(t, http.StatusCreated, `{"mandate":{"id":"mnd_1","status":"created","vpa":"jane@bank"},"auth_url":"https://pay.test/authorize/mnd_1"}`)
	res, err := ts.client.Mandates.Create(context.Background(), &MandateCreateParams{CustomerID: "cus_1", VPA: "jane@bank", MaxAmount: 100000, Frequency: "monthly"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.assertRequest(http.MethodPost, "/mandates")
	if res.Mandate.ID != "mnd_1" || res.AuthURL == "" {
		t.Errorf("res = %+v", res)
	}
}

func TestLedgerAccounts(t *testing.T) {
	ts := newTestServer(t, http.StatusOK, `{"data":[{"id":"acc_1","name":"Accounts Receivable","type":"asset","balance":50000}]}`)
	accounts, err := ts.client.Ledger.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	ts.assertRequest(http.MethodGet, "/ledger/accounts")
	if len(accounts) != 1 || accounts[0].Balance != 50000 {
		t.Errorf("accounts = %+v", accounts)
	}
}

// TestErrorEnvelope verifies that a 4xx response decodes into a populated
// *APIError.
func TestErrorEnvelope(t *testing.T) {
	ts := newTestServer(t, http.StatusNotFound, `{"error":{"code":"NOT_FOUND","message":"plan not found"}}`)
	_, err := ts.client.Plans.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Code != "NOT_FOUND" || apiErr.Message != "plan not found" {
		t.Errorf("apiErr = %+v", apiErr)
	}
	if apiErr.Error() == "" {
		t.Error("Error() returned empty string")
	}
}

// TestErrorNonJSON verifies a non-JSON error body still yields a *APIError.
func TestErrorNonJSON(t *testing.T) {
	ts := newTestServer(t, http.StatusInternalServerError, `upstream failure`)
	_, err := ts.client.Customers.List(context.Background(), nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 500 || apiErr.Message != "upstream failure" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

package stripeimport

import (
	"testing"
)

// sampleExport is a small but representative Stripe account: two customers
// (one net-new, one that already exists in Recurso by email), a product with a
// recurring monthly price and a one-time price, an active subscription, and a
// card payment method.
func sampleExport() *Export {
	return &Export{
		Customers: []Customer{
			{ID: "cus_new", Email: "new@acme.com", Name: "Acme New"},
			{ID: "cus_dupe", Email: "existing@acme.com", Name: "Acme Existing"},
			{ID: "cus_noemail", Email: "", Name: "No Email"},
		},
		Products: []Product{{ID: "prod_1", Name: "Pro", Active: true}},
		Prices: []Price{
			{ID: "price_month", Product: "prod_1", UnitAmount: 4900, Currency: "usd", Active: true, Recurring: &Recurring{Interval: "month", IntervalCount: 1}},
			{ID: "price_once", Product: "prod_1", UnitAmount: 9900, Currency: "usd", Active: true}, // one-time
		},
		Subscriptions: []Subscription{
			{ID: "sub_ok", Customer: "cus_new", Status: "active", Items: SubscriptionItems{Data: []SubscriptionItem{{Price: Price{ID: "price_month"}}}}},
			{ID: "sub_badcust", Customer: "cus_ghost", Status: "active", Items: SubscriptionItems{Data: []SubscriptionItem{{Price: Price{ID: "price_month"}}}}},
			{ID: "sub_canceled", Customer: "cus_new", Status: "canceled", Items: SubscriptionItems{Data: []SubscriptionItem{{Price: Price{ID: "price_month"}}}}},
		},
		PaymentMethods: []PaymentMethod{
			{ID: "pm_card", Customer: "cus_new", Type: "card", Card: &Card{Brand: "visa", Last4: "4242", ExpMonth: 12, ExpYear: 2030}},
			{ID: "pm_bank", Customer: "cus_new", Type: "us_bank_account"},
		},
	}
}

// findItem returns the plan item for a Stripe ID (or a zero Item).
func findItem(p *Plan, stripeID string) Item {
	for _, it := range p.Items {
		if it.StripeID == stripeID {
			return it
		}
	}
	return Item{}
}

func TestBuildPlan_CustomerActions(t *testing.T) {
	existing := Existing{CustomerEmails: map[string]bool{"existing@acme.com": true}}
	p := BuildPlan(sampleExport(), existing)

	if got := findItem(p, "cus_new").Action; got != ActionCreate {
		t.Errorf("cus_new: want create, got %s", got)
	}
	if got := findItem(p, "cus_dupe").Action; got != ActionLinkExisting {
		t.Errorf("cus_dupe (email exists): want link_existing, got %s", got)
	}
	if got := findItem(p, "cus_noemail").Action; got != ActionConflict {
		t.Errorf("cus_noemail: want conflict, got %s", got)
	}
}

func TestBuildPlan_PlanActions(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{})

	month := findItem(p, "price_month")
	if month.Action != ActionCreate {
		t.Errorf("recurring price: want create, got %s", month.Action)
	}
	if month.Detail == "" {
		t.Error("recurring price should carry a price/interval detail")
	}
	if got := findItem(p, "price_once").Action; got != ActionUnsupported {
		t.Errorf("one-time price: want unsupported, got %s", got)
	}
}

func TestBuildPlan_UnsupportedInterval(t *testing.T) {
	exp := &Export{
		Products: []Product{{ID: "prod_1", Name: "Odd"}},
		Prices:   []Price{{ID: "price_odd", Product: "prod_1", UnitAmount: 100, Currency: "usd", Recurring: &Recurring{Interval: "fortnight"}}},
	}
	if got := findItem(BuildPlan(exp, Existing{}), "price_odd").Action; got != ActionConflict {
		t.Errorf("unknown interval: want conflict, got %s", got)
	}
}

func TestBuildPlan_SubscriptionActions(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{})

	if got := findItem(p, "sub_ok").Action; got != ActionCreate {
		t.Errorf("sub_ok: want create, got %s", got)
	}
	if got := findItem(p, "sub_badcust").Action; got != ActionConflict {
		t.Errorf("sub_badcust (missing customer): want conflict, got %s", got)
	}
	if got := findItem(p, "sub_canceled").Action; got != ActionUnsupported {
		t.Errorf("sub_canceled: want unsupported, got %s", got)
	}
}

func TestBuildPlan_SubscriptionConflictWhenPriceUnsupported(t *testing.T) {
	// A subscription on a one-time price can't resolve to a plan.
	exp := &Export{
		Customers:     []Customer{{ID: "cus_1", Email: "a@b.com"}},
		Products:      []Product{{ID: "prod_1", Name: "P"}},
		Prices:        []Price{{ID: "price_once", Product: "prod_1", UnitAmount: 500, Currency: "usd"}},
		Subscriptions: []Subscription{{ID: "sub_x", Customer: "cus_1", Status: "active", Items: SubscriptionItems{Data: []SubscriptionItem{{Price: Price{ID: "price_once"}}}}}},
	}
	if got := findItem(BuildPlan(exp, Existing{}), "sub_x").Action; got != ActionConflict {
		t.Errorf("sub on one-time price: want conflict, got %s", got)
	}
}

func TestBuildPlan_PaymentMethodActions(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{})
	if got := findItem(p, "pm_card").Action; got != ActionCreate {
		t.Errorf("card pm: want create, got %s", got)
	}
	if got := findItem(p, "pm_bank").Action; got != ActionUnsupported {
		t.Errorf("bank pm: want unsupported, got %s", got)
	}
}

func TestBuildPlan_Idempotency(t *testing.T) {
	// Everything already imported → all skipped, nothing created.
	existing := Existing{ImportedStripeIDs: map[string]bool{
		"cus_new": true, "cus_dupe": true, "cus_noemail": true,
		"price_month": true, "price_once": true,
		"sub_ok": true, "sub_badcust": true, "sub_canceled": true,
		"pm_card": true, "pm_bank": true,
	}}
	p := BuildPlan(sampleExport(), existing)
	if counts := p.CreateCounts(); len(counts) != 0 {
		t.Fatalf("re-import should create nothing, got %v", counts)
	}
	for _, it := range p.Items {
		if it.Action != ActionSkipImported {
			t.Errorf("%s: want skip_already_imported on re-run, got %s", it.StripeID, it.Action)
		}
	}
}

func TestBuildPlan_PlanCodeSkipWhenAlreadyPresent(t *testing.T) {
	existing := Existing{PlanCodes: map[string]bool{planCode(Price{ID: "price_month"}): true}}
	if got := findItem(BuildPlan(sampleExport(), existing), "price_month").Action; got != ActionSkipImported {
		t.Errorf("existing plan code: want skip, got %s", got)
	}
}

func TestBuildPlan_SummaryAndWarnings(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{CustomerEmails: map[string]bool{"existing@acme.com": true}})

	if p.Summary["customer.create"] != 1 {
		t.Errorf("want 1 customer.create, got %d", p.Summary["customer.create"])
	}
	if p.Summary["customer.link_existing"] != 1 {
		t.Errorf("want 1 customer.link_existing, got %d", p.Summary["customer.link_existing"])
	}
	if len(p.Warnings) == 0 {
		t.Error("expected warnings (conflicts + unsupported present in the sample)")
	}
	cc := p.CreateCounts()
	if cc[KindCustomer] != 1 || cc[KindPlan] != 1 || cc[KindSubscription] != 1 || cc[KindPaymentMethod] != 1 {
		t.Errorf("unexpected create counts: %v", cc)
	}
}

func TestParse(t *testing.T) {
	good := []byte(`{"customers":[{"id":"cus_1","email":"a@b.com"}],"prices":[{"id":"price_1","unit_amount":1000,"currency":"usd","recurring":{"interval":"month"}}]}`)
	exp, err := Parse(good)
	if err != nil {
		t.Fatalf("Parse valid: %v", err)
	}
	if len(exp.Customers) != 1 || exp.Customers[0].Email != "a@b.com" {
		t.Errorf("parsed customers wrong: %+v", exp.Customers)
	}
	if exp.Prices[0].UnitAmount != 1000 || exp.Prices[0].Recurring == nil {
		t.Errorf("parsed price wrong: %+v", exp.Prices)
	}

	if _, err := Parse([]byte(`{not json`)); err == nil {
		t.Error("Parse should reject malformed JSON")
	}
}

func TestParse_IgnoresUnknownFields(t *testing.T) {
	// A raw Stripe object carries many extra keys; they must not break parsing.
	raw := []byte(`{"customers":[{"id":"cus_1","email":"a@b.com","livemode":true,"balance":0,"metadata":{"k":"v"}}]}`)
	exp, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse with extra fields: %v", err)
	}
	if exp.Customers[0].ID != "cus_1" {
		t.Errorf("want cus_1, got %s", exp.Customers[0].ID)
	}
}

func TestMoney(t *testing.T) {
	cases := map[string][2]string{ // key -> [amount+currency rendered]
		"usd_4900": {money(4900, "usd"), "49.00 USD"},
		"inr_150":  {money(150, "inr"), "1.50 INR"},
		"jpy_500":  {money(500, "jpy"), "500 JPY"},
	}
	for name, c := range cases {
		if c[0] != c[1] {
			t.Errorf("%s: got %q, want %q", name, c[0], c[1])
		}
	}
}

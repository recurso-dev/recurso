package revenuecat

import "testing"

func sampleExport() *Export {
	return &Export{
		Products: []Product{
			{ID: "monthly", Title: "Pro Monthly", Price: 999, Currency: "usd", PeriodUnit: "month", PeriodCount: 1},
			{ID: "weird", Title: "Odd", Currency: "usd", PeriodUnit: "fortnight"},
		},
		Subscribers: []Subscriber{
			{AppUserID: "user_a", Email: "a@acme.com", Subscriptions: []Subscription{
				{ProductID: "monthly", Store: "app_store", IsActive: true},
			}},
			{AppUserID: "user_dupe", Email: "existing@acme.com"},
			{AppUserID: "user_noemail", Email: "", Subscriptions: []Subscription{
				{ProductID: "monthly", Store: "play_store", IsActive: true},
			}},
			{AppUserID: "user_expired", Email: "c@acme.com", Subscriptions: []Subscription{
				{ProductID: "monthly", Store: "app_store", IsActive: false},
			}},
		},
	}
}

func find(p *ImportPlan, id string) Item {
	for _, it := range p.Items {
		if it.RevenueCatID == id {
			return it
		}
	}
	return Item{}
}

func TestBuildPlan_Products(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{})
	if got := find(p, "monthly").Action; got != ActionCreate {
		t.Errorf("monthly: want create, got %s", got)
	}
	if got := find(p, "weird").Action; got != ActionConflict {
		t.Errorf("weird period: want conflict, got %s", got)
	}
}

func TestBuildPlan_Subscribers(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{CustomerEmails: map[string]bool{"existing@acme.com": true}})
	if got := find(p, "user_a").Action; got != ActionCreate {
		t.Errorf("user_a: want create, got %s", got)
	}
	if got := find(p, "user_dupe").Action; got != ActionLinkExisting {
		t.Errorf("user_dupe: want link_existing, got %s", got)
	}
	// The defining RevenueCat wrinkle: no email → conflict, not a silent drop.
	if got := find(p, "user_noemail").Action; got != ActionConflict {
		t.Errorf("no-email subscriber: want conflict, got %s", got)
	}
}

func TestBuildPlan_Subscriptions(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{})
	// Active sub on a resolvable customer + product → create.
	if got := find(p, "user_a:monthly").Action; got != ActionCreate {
		t.Errorf("active sub: want create, got %s", got)
	}
	// Expired/inactive → unsupported.
	if got := find(p, "user_expired:monthly").Action; got != ActionUnsupported {
		t.Errorf("inactive sub: want unsupported, got %s", got)
	}
	// Sub under a no-email subscriber can't resolve its customer → conflict.
	if got := find(p, "user_noemail:monthly").Action; got != ActionConflict {
		t.Errorf("sub under no-email subscriber: want conflict, got %s", got)
	}
}

func TestBuildPlan_Idempotency(t *testing.T) {
	ids := map[string]bool{"monthly": true, "user_a": true, "user_a:monthly": true, "user_dupe": true, "user_expired": true, "user_expired:monthly": true, "user_noemail": true, "user_noemail:monthly": true, "weird": true}
	p := BuildPlan(sampleExport(), Existing{ImportedIDs: ids})
	if len(p.CreateCounts()) != 0 {
		t.Fatalf("re-import should create nothing, got %v", p.CreateCounts())
	}
}

func TestBuildPlan_CreateCountsAndWarnings(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{CustomerEmails: map[string]bool{"existing@acme.com": true}})
	cc := p.CreateCounts()
	// 1 plan (monthly), 2 customers (user_a + user_expired; user_dupe links, user_noemail conflict), 1 sub (user_a:monthly).
	if cc[KindPlan] != 1 || cc[KindCustomer] != 2 || cc[KindSubscription] != 1 {
		t.Errorf("unexpected create counts: %v", cc)
	}
	if len(p.Warnings) == 0 {
		t.Error("expected warnings (no-email conflict + inactive-sub unsupported)")
	}
}

func TestParseAndSubID(t *testing.T) {
	exp, err := Parse([]byte(`{"products":[{"id":"p1","price":100,"currency":"usd","period_unit":"month","extra":1}]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if exp.Products[0].Price != 100 {
		t.Errorf("parsed wrong: %+v", exp.Products)
	}
	if _, err := Parse([]byte(`{bad`)); err == nil {
		t.Error("should reject malformed JSON")
	}
	if got := (Subscription{ProductID: "p1"}).SubID("u1"); got != "u1:p1" {
		t.Errorf("synthesized SubID = %q", got)
	}
	if got := (Subscription{ID: "explicit"}).SubID("u1"); got != "explicit" {
		t.Errorf("explicit SubID = %q", got)
	}
}

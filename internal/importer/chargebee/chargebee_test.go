package chargebee

import "testing"

func sampleExport() *Export {
	return &Export{
		Customers: []Customer{
			{ID: "cb_new", Email: "new@acme.com", FirstName: "New", LastName: "Co"},
			{ID: "cb_dupe", Email: "existing@acme.com"},
			{ID: "cb_noemail", Email: ""},
		},
		Plans: []Plan{
			{ID: "pro", Name: "Pro", Price: 4900, Period: 1, PeriodUnit: "month", CurrencyCode: "usd", Status: "active"},
			{ID: "weird", Name: "Weird", Price: 100, PeriodUnit: "fortnight", Status: "active"},
			{ID: "gone", Name: "Gone", PeriodUnit: "month", Status: "deleted"},
		},
		Subscriptions: []Subscription{
			{ID: "sub_ok", CustomerID: "cb_new", PlanID: "pro", Status: "active"},
			{ID: "sub_trial", CustomerID: "cb_new", PlanID: "pro", Status: "in_trial"},
			{ID: "sub_cancel", CustomerID: "cb_new", PlanID: "pro", Status: "cancelled"},
			{ID: "sub_badcust", CustomerID: "ghost", PlanID: "pro", Status: "active"},
		},
	}
}

func find(p *ImportPlan, id string) Item {
	for _, it := range p.Items {
		if it.ChargebeeID == id {
			return it
		}
	}
	return Item{}
}

func TestBuildPlan_Customers(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{CustomerEmails: map[string]bool{"existing@acme.com": true}})
	if got := find(p, "cb_new").Action; got != ActionCreate {
		t.Errorf("cb_new: want create, got %s", got)
	}
	if got := find(p, "cb_dupe").Action; got != ActionLinkExisting {
		t.Errorf("cb_dupe: want link_existing, got %s", got)
	}
	if got := find(p, "cb_noemail").Action; got != ActionConflict {
		t.Errorf("cb_noemail: want conflict, got %s", got)
	}
}

func TestBuildPlan_Plans(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{})
	if got := find(p, "pro").Action; got != ActionCreate {
		t.Errorf("pro: want create, got %s", got)
	}
	if find(p, "pro").Detail == "" {
		t.Error("pro should carry a price/interval detail")
	}
	if got := find(p, "weird").Action; got != ActionConflict {
		t.Errorf("weird interval: want conflict, got %s", got)
	}
	if got := find(p, "gone").Action; got != ActionUnsupported {
		t.Errorf("deleted plan: want unsupported, got %s", got)
	}
}

func TestBuildPlan_Subscriptions(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{})
	if got := find(p, "sub_ok").Action; got != ActionCreate {
		t.Errorf("sub_ok: want create, got %s", got)
	}
	if got := find(p, "sub_trial").Action; got != ActionCreate {
		t.Errorf("in_trial: want create, got %s", got)
	}
	if got := find(p, "sub_cancel").Action; got != ActionUnsupported {
		t.Errorf("cancelled: want unsupported, got %s", got)
	}
	if got := find(p, "sub_badcust").Action; got != ActionConflict {
		t.Errorf("missing customer: want conflict, got %s", got)
	}
}

func TestBuildPlan_Idempotency(t *testing.T) {
	ids := map[string]bool{}
	for _, id := range []string{"cb_new", "cb_dupe", "cb_noemail", "pro", "weird", "gone", "sub_ok", "sub_trial", "sub_cancel", "sub_badcust"} {
		ids[id] = true
	}
	p := BuildPlan(sampleExport(), Existing{ImportedIDs: ids})
	if len(p.CreateCounts()) != 0 {
		t.Fatalf("re-import should create nothing, got %v", p.CreateCounts())
	}
	for _, it := range p.Items {
		if it.Action != ActionSkipImported {
			t.Errorf("%s: want skip on re-run, got %s", it.ChargebeeID, it.Action)
		}
	}
}

func TestBuildPlan_CreateCountsAndWarnings(t *testing.T) {
	p := BuildPlan(sampleExport(), Existing{CustomerEmails: map[string]bool{"existing@acme.com": true}})
	cc := p.CreateCounts()
	// 1 customer (new), 1 plan (pro), 2 subs (ok + trial).
	if cc[KindCustomer] != 1 || cc[KindPlan] != 1 || cc[KindSubscription] != 2 {
		t.Errorf("unexpected create counts: %v", cc)
	}
	if len(p.Warnings) == 0 {
		t.Error("expected warnings (conflicts + unsupported present)")
	}
}

func TestParse(t *testing.T) {
	good := []byte(`{"customers":[{"id":"c1","email":"a@b.com","some_extra":1}],"plans":[{"id":"p1","price":1000,"period_unit":"month","currency_code":"usd","status":"active"}]}`)
	exp, err := Parse(good)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if exp.Customers[0].Email != "a@b.com" || exp.Plans[0].Price != 1000 {
		t.Errorf("parsed wrong: %+v", exp)
	}
	if _, err := Parse([]byte(`{bad`)); err == nil {
		t.Error("should reject malformed JSON")
	}
}

func TestCustomerName(t *testing.T) {
	if got := (Customer{FirstName: "Ada", LastName: "Lovelace"}).Name(); got != "Ada Lovelace" {
		t.Errorf("name = %q", got)
	}
	if got := (Customer{Company: "Acme Inc"}).Name(); got != "Acme Inc" {
		t.Errorf("company fallback = %q", got)
	}
}

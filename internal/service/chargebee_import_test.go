package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/importer/chargebee"
)

func chargebeeFixture() *chargebee.Export {
	return &chargebee.Export{
		Customers: []chargebee.Customer{
			{ID: "cb_a", Email: "a@acme.com", FirstName: "A"},
			{ID: "cb_b", Email: "b@acme.com", FirstName: "B"},
		},
		Plans: []chargebee.Plan{
			{ID: "pro", Name: "Pro", Price: 4900, Period: 1, PeriodUnit: "month", CurrencyCode: "usd", Status: "active"},
		},
		Subscriptions: []chargebee.Subscription{
			{ID: "sub_a", CustomerID: "cb_a", PlanID: "pro", Status: "active", CurrentTermStart: 1_700_000_000, CurrentTermEnd: 1_702_600_000},
		},
	}
}

func newChargebeeSvc() (*ChargebeeImportService, *fakeImportCustomers, *fakeImportCatalog, *fakeImportSubs, *fakeRefRepo) {
	cust := &fakeImportCustomers{}
	cat := &fakeImportCatalog{}
	subs := &fakeImportSubs{}
	refs := newFakeRefRepo()
	return NewChargebeeImportService(cust, cat, subs, refs), cust, cat, subs, refs
}

func TestChargebeeCommit_CreatesCustomersPlansAndSubscriptions(t *testing.T) {
	svc, cust, cat, subs, refs := newChargebeeSvc()
	res, err := svc.Commit(context.Background(), uuid.New(), chargebeeFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Created["customer"] != 2 || res.Created["plan"] != 1 || res.Created["subscription"] != 1 {
		t.Fatalf("created counts wrong: %v (failures %+v)", res.Created, res.Failures)
	}
	if len(cust.created) != 2 || len(cat.created) != 1 || len(subs.created) != 1 {
		t.Errorf("create calls wrong: customers=%d plans=%d subs=%d", len(cust.created), len(cat.created), len(subs.created))
	}
	if len(refs.refs) != 4 {
		t.Errorf("want 4 idempotency refs, got %d", len(refs.refs))
	}
	sub := subs.created[0]
	if sub.CustomerID == uuid.Nil || sub.PlanID == uuid.Nil {
		t.Error("subscription customer/plan not resolved")
	}
	if sub.Status != domain.SubscriptionStatusActive {
		t.Errorf("status = %s, want active", sub.Status)
	}
	if sub.CurrentPeriodEnd.Unix() != 1_702_600_000 || sub.BillingAnchor.Unix() != 1_702_600_000 {
		t.Errorf("period/anchor not from Chargebee term end")
	}
	// Plan maps money 1:1, deterministic code.
	if cat.created[0].Amount != 4900 || cat.created[0].Code != "chargebee_pro" {
		t.Errorf("plan mapping wrong: %+v", cat.created[0])
	}
}

func TestChargebeeCommit_LinksExistingCustomer(t *testing.T) {
	svc, cust, _, subs, _ := newChargebeeSvc()
	existingID := uuid.New()
	cust.existing = []*domain.Customer{{ID: existingID, Email: "a@acme.com"}}

	res, err := svc.Commit(context.Background(), uuid.New(), chargebeeFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Created["customer"] != 1 { // only cb_b created; cb_a linked
		t.Errorf("want 1 customer created, got %d", res.Created["customer"])
	}
	if len(subs.created) != 1 || subs.created[0].CustomerID != existingID {
		t.Errorf("subscription should resolve to the existing linked customer")
	}
}

func TestChargebeeCommit_IsIdempotent(t *testing.T) {
	svc, _, _, subs, _ := newChargebeeSvc()
	tenant := uuid.New()
	exp := chargebeeFixture()
	if _, err := svc.Commit(context.Background(), tenant, exp); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	res2, err := svc.Commit(context.Background(), tenant, exp)
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if len(res2.Created) != 0 {
		t.Errorf("re-run should create nothing, got %v", res2.Created)
	}
	if len(subs.created) != 1 {
		t.Errorf("subscription created twice across runs: %d", len(subs.created))
	}
}

func TestChargebeePreview_NoSideEffects(t *testing.T) {
	svc, cust, cat, subs, refs := newChargebeeSvc()
	plan, err := svc.Preview(context.Background(), uuid.New(), chargebeeFixture())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if plan.Summary["customer.create"] != 2 || plan.Summary["plan.create"] != 1 || plan.Summary["subscription.create"] != 1 {
		t.Errorf("preview summary wrong: %v", plan.Summary)
	}
	if len(cust.created) != 0 || len(cat.created) != 0 || len(subs.created) != 0 || len(refs.refs) != 0 {
		t.Error("preview must not create anything")
	}
}

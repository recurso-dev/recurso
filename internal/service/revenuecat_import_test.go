package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/importer/revenuecat"
)

func revenuecatFixture() *revenuecat.Export {
	return &revenuecat.Export{
		Products: []revenuecat.Product{
			{ID: "monthly", Title: "Pro Monthly", Price: 999, Currency: "usd", PeriodUnit: "month", PeriodCount: 1},
		},
		Subscribers: []revenuecat.Subscriber{
			{AppUserID: "user_a", Email: "a@acme.com", Subscriptions: []revenuecat.Subscription{
				{ProductID: "monthly", Store: "app_store", IsActive: true, ExpiresAt: 1_702_600_000},
			}},
			{AppUserID: "user_noemail", Email: "", Subscriptions: []revenuecat.Subscription{
				{ProductID: "monthly", Store: "play_store", IsActive: true},
			}},
		},
	}
}

func newRevenueCatSvc() (*RevenueCatImportService, *fakeImportCustomers, *fakeImportCatalog, *fakeImportSubs, *fakeRefRepo) {
	cust := &fakeImportCustomers{}
	cat := &fakeImportCatalog{}
	subs := &fakeImportSubs{}
	refs := newFakeRefRepo()
	return NewRevenueCatImportService(cust, cat, subs, refs), cust, cat, subs, refs
}

func TestRevenueCatCommit_CreatesResolvableAndReportsNoEmail(t *testing.T) {
	svc, cust, cat, subs, refs := newRevenueCatSvc()
	res, err := svc.Commit(context.Background(), uuid.New(), revenuecatFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	// 1 plan, 1 customer (user_a; user_noemail is a conflict, not created),
	// 1 subscription (user_a's active sub).
	if res.Created["plan"] != 1 || res.Created["customer"] != 1 || res.Created["subscription"] != 1 {
		t.Fatalf("created counts wrong: %v (failures %+v)", res.Created, res.Failures)
	}
	if len(cust.created) != 1 || len(cat.created) != 1 || len(subs.created) != 1 {
		t.Errorf("create calls wrong: customers=%d plans=%d subs=%d", len(cust.created), len(cat.created), len(subs.created))
	}
	if len(refs.refs) != 3 {
		t.Errorf("want 3 idempotency refs, got %d", len(refs.refs))
	}
	sub := subs.created[0]
	if sub.CustomerID == uuid.Nil || sub.PlanID == uuid.Nil {
		t.Error("subscription customer/plan not resolved")
	}
	if sub.CurrentPeriodEnd.Unix() != 1_702_600_000 {
		t.Errorf("period end not from RevenueCat expires_at")
	}
	// Plan money 1:1 + deterministic code.
	if cat.created[0].Amount != 999 || cat.created[0].Code != "revenuecat_monthly" {
		t.Errorf("plan mapping wrong: %+v", cat.created[0])
	}
}

func TestRevenueCatCommit_IsIdempotent(t *testing.T) {
	svc, _, _, subs, _ := newRevenueCatSvc()
	tenant := uuid.New()
	exp := revenuecatFixture()
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

func TestRevenueCatPreview_NoSideEffects(t *testing.T) {
	svc, cust, cat, subs, refs := newRevenueCatSvc()
	plan, err := svc.Preview(context.Background(), uuid.New(), revenuecatFixture())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if plan.Summary["plan.create"] != 1 || plan.Summary["customer.create"] != 1 || plan.Summary["customer.conflict"] != 1 {
		t.Errorf("preview summary wrong: %v", plan.Summary)
	}
	if len(cust.created) != 0 || len(cat.created) != 0 || len(subs.created) != 0 || len(refs.refs) != 0 {
		t.Error("preview must not create anything")
	}
}

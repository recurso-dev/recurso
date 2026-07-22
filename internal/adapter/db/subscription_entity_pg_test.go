package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestSubscriptionRepository_EntityRoundTrip_Postgres proves Multi-Entity Books
// Inc 2b: a subscription persists its issuing entity_id and every full-row load
// (routed through scanSubscription) returns it — so its invoices can inherit the
// entity. NULL round-trips as nil (⇒ primary).
func TestSubscriptionRepository_EntityRoundTrip_Postgres(t *testing.T) {
	conn := openProgressiveTestDB(t)
	ctx := context.Background()
	run := uuid.NewString()[:8]

	tenantID := uuid.New()
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "SE-"+run, "se-"+run+"@t.com")
	custID := uuid.New()
	must(t, conn, `INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		custID, tenantID, "c-"+run+"@t.com", uuid.New())
	planID := uuid.New()
	must(t, conn, `INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,'P','p-`+run+`','month',1,TRUE)`,
		planID, tenantID)

	entityRepo := NewEntityRepository(conn)
	second := &domain.Entity{TenantID: tenantID, Name: "ACME UK", InvoicePrefix: "UK"}
	if err := entityRepo.Create(ctx, second); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	repo := NewSubscriptionRepository(conn)
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	// A subscription tagged to the second entity round-trips its entity_id.
	subEntity := &domain.Subscription{
		ID: uuid.New(), TenantID: tenantID, EntityID: &second.ID, CustomerID: custID, PlanID: planID,
		Status: domain.SubscriptionStatusActive, CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(720 * time.Hour),
		BillingAnchor: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.Create(ctx, subEntity); err != nil {
		t.Fatalf("create sub (entity): %v", err)
	}
	got, err := repo.GetByID(tctx, subEntity.ID)
	if err != nil {
		t.Fatalf("get sub: %v", err)
	}
	if got.EntityID == nil || *got.EntityID != second.ID {
		t.Errorf("loaded entity_id = %v, want %s", got.EntityID, second.ID)
	}

	// A subscription with no entity round-trips as nil (⇒ primary).
	subPrimary := &domain.Subscription{
		ID: uuid.New(), TenantID: tenantID, CustomerID: custID, PlanID: planID,
		Status: domain.SubscriptionStatusActive, CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(720 * time.Hour),
		BillingAnchor: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.Create(ctx, subPrimary); err != nil {
		t.Fatalf("create sub (primary): %v", err)
	}
	got2, err := repo.GetByID(tctx, subPrimary.ID)
	if err != nil {
		t.Fatalf("get sub: %v", err)
	}
	if got2.EntityID != nil {
		t.Errorf("nil entity_id should round-trip as nil, got %s", got2.EntityID)
	}
}

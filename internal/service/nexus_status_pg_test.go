package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestNexusStatus_ThresholdCrossing_Postgres covers ENG-16 Phase 2 against
// real SQL: cumulative per-state sales/txn tracking computed from invoices,
// auto-establishment of economic nexus on a crossing, idempotency, that
// declared nexus is never overwritten, and that the seed dataset reports
// uncertified.
func TestNexusStatus_ThresholdCrossing_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed nexus test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations (000075 thresholds): %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Nexus-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	// A TX buyer (threshold $500k, sales-only) and a GA buyer ($100k or 200).
	custRepo := db.NewCustomerRepository(sqlx.NewDb(conn, "postgres"))
	mkCustomer := func(state string) uuid.UUID {
		id := uuid.New()
		name := "Buyer " + state
		c := &domain.Customer{ID: id, TenantID: tenantID, Name: &name,
			Email:          state + "-" + id.String()[:8] + "@example.com",
			BillingAddress: domain.BillingAddress{Line1: "1 Main St", City: "City", State: state, Zip: "00000", Country: "US"},
			CreatedAt:      time.Now(), UpdatedAt: time.Now()}
		if err := custRepo.Create(tenantCtx, c); err != nil {
			t.Fatalf("create %s customer: %v", state, err)
		}
		return id
	}
	txCust, gaCust := mkCustomer("TX"), mkCustomer("GA")

	mkInvoice := func(custID uuid.UUID, subtotal int64, status string) {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx, `
			INSERT INTO invoices (id, tenant_id, customer_id, invoice_number, status, currency,
				subtotal, tax_amount, total, created_at, due_date)
			VALUES ($1, $2, $3, $4, $5, 'USD', $6, 0, $6, NOW(), NOW())`,
			id, tenantID, custID, "INV-NX-"+id.String()[:8], status, subtotal); err != nil {
			t.Fatalf("insert invoice: %v", err)
		}
	}

	// GA: $60k paid + $50k open = $110k (crosses $100k); a void $900k must NOT count.
	mkInvoice(gaCust, 6000000, "paid")
	mkInvoice(gaCust, 5000000, "open")
	mkInvoice(gaCust, 90000000, "void")
	// TX: $110k — well under the $500k threshold.
	mkInvoice(txCust, 11000000, "paid")

	repo := db.NewTaxNexusRepository(conn)
	svc := NewNexusStatusService(repo)
	year := time.Now().UTC().Year()

	// The seed dataset must report uncertified until reviewed.
	certified, err := svc.DatasetCertified(ctx)
	if err != nil {
		t.Fatalf("DatasetCertified: %v", err)
	}
	if certified {
		t.Fatal("seed threshold dataset reports certified — it has not passed review")
	}

	established, err := svc.EvaluateEconomicNexus(ctx, tenantID, year)
	if err != nil {
		t.Fatalf("EvaluateEconomicNexus: %v", err)
	}
	if len(established) != 1 || established[0] != "GA" {
		t.Fatalf("established = %v, want [GA] (void invoices must not count toward TX either)", established)
	}

	// Idempotent: a second evaluation establishes nothing new.
	if again, _ := svc.EvaluateEconomicNexus(ctx, tenantID, year); len(again) != 0 {
		t.Fatalf("second evaluation re-established: %v", again)
	}

	// Status view: GA crossed+economic, TX under threshold with proximity.
	statuses, err := svc.Status(ctx, tenantID, year)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	byState := map[string]domain.NexusStateStatus{}
	for _, st := range statuses {
		byState[st.StateCode] = st
	}
	ga := byState["GA"]
	if ga.NexusType != domain.NexusEconomic || !ga.Crossed || ga.TaxableSales != 11000000 || ga.TxnCount != 2 {
		t.Fatalf("GA status = %+v, want economic/crossed with $110k over 2 txns", ga)
	}
	tx := byState["TX"]
	if tx.NexusType != "" || tx.Crossed || tx.ProximityPct != 22 {
		t.Fatalf("TX status = %+v, want no nexus, not crossed, 22%% proximity ($110k of $500k)", tx)
	}

	// A declared nexus must never be downgraded by evaluation.
	if err := repo.SetStates(ctx, tenantID, []domain.TaxNexus{
		{StateCode: "GA", NexusType: domain.NexusPhysical},
		{StateCode: "WA", NexusType: domain.NexusPhysical},
	}); err != nil {
		t.Fatalf("SetStates: %v", err)
	}
	if _, err := svc.EvaluateEconomicNexus(ctx, tenantID, year); err != nil {
		t.Fatalf("re-evaluate: %v", err)
	}
	list, _ := repo.ListByTenant(ctx, tenantID)
	for _, n := range list {
		if n.StateCode == "GA" && n.NexusType != domain.NexusPhysical {
			t.Fatalf("GA nexus downgraded to %s — declared nexus must win", n.NexusType)
		}
	}
}

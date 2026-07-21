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

// The liability report aggregates per-state gross/taxable/non-taxable sales and
// tax collected from real invoices, annotates nexus, and rolls up totals —
// scoped to US/USD/non-void invoices in the period (Track D · D3).
func TestLiabilityReport_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed liability report test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Liab-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	custRepo := db.NewCustomerRepository(sqlx.NewDb(conn, "postgres"))
	mkCustomer := func(state, country string) uuid.UUID {
		id := uuid.New()
		name := "Buyer " + state
		c := &domain.Customer{ID: id, TenantID: tenantID, Name: &name,
			Email:          state + "-" + id.String()[:8] + "@example.com",
			BillingAddress: domain.BillingAddress{Line1: "1 Main St", City: "City", State: state, Zip: "00000", Country: country},
			CreatedAt:      time.Now(), UpdatedAt: time.Now()}
		if err := custRepo.Create(tenantCtx, c); err != nil {
			t.Fatalf("create %s customer: %v", state, err)
		}
		return id
	}

	year := time.Now().UTC().Year()
	inPeriod := time.Date(year, 6, 15, 0, 0, 0, 0, time.UTC)
	mkInvoice := func(custID uuid.UUID, subtotal, tax int64, taxType, status string, at time.Time) {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx, `
			INSERT INTO invoices (id, tenant_id, customer_id, invoice_number, status, currency,
				subtotal, tax_amount, tax_type, total, created_at, due_date)
			VALUES ($1,$2,$3,$4,$5,'USD',$6,$7,$8,$9,$10,NOW())`,
			id, tenantID, custID, "INV-LB-"+id.String()[:8], status, subtotal, tax, taxType, subtotal+tax, at); err != nil {
			t.Fatalf("insert invoice: %v", err)
		}
	}

	caCust := mkCustomer("CA", "US")
	nyCust := mkCustomer("NY", "US")
	deCust := mkCustomer("BE", "Germany") // non-US: must be excluded

	// CA: taxable ($1000 + $80) + exempt ($500, cert) + non-taxable ($300, no-nexus).
	mkInvoice(caCust, 100000, 8000, "sales_tax", "paid", inPeriod)
	mkInvoice(caCust, 50000, 0, "sales_tax_exempt", "open", inPeriod)
	mkInvoice(caCust, 30000, 0, "no_nexus", "open", inPeriod)
	// CA: a void invoice must NOT count, and an out-of-period one must NOT count.
	mkInvoice(caCust, 90000000, 0, "sales_tax", "void", inPeriod)
	mkInvoice(caCust, 70000, 5600, "sales_tax", "paid", time.Date(year-1, 6, 15, 0, 0, 0, 0, time.UTC))
	// NY: one taxable ($2000 + $170 tax).
	mkInvoice(nyCust, 200000, 17000, "sales_tax", "paid", inPeriod)
	// Non-US buyer: excluded regardless.
	mkInvoice(deCust, 300000, 0, "sales_tax", "paid", inPeriod)

	// Declare physical nexus in CA (NY left undeclared to prove has_nexus=false).
	if err := db.NewTaxNexusRepository(conn).SetStates(ctx, tenantID,
		[]domain.TaxNexus{{TenantID: tenantID, StateCode: "CA", NexusType: domain.NexusPhysical}}); err != nil {
		t.Fatalf("declare CA nexus: %v", err)
	}

	svc := NewNexusStatusService(db.NewTaxNexusRepository(conn))
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	report, err := svc.LiabilityReport(ctx, tenantID, from, to)
	if err != nil {
		t.Fatalf("LiabilityReport: %v", err)
	}

	byState := map[string]domain.USLiabilityStateLine{}
	for _, s := range report.States {
		byState[s.StateCode] = s
	}
	if len(byState) != 2 {
		t.Fatalf("want 2 states (CA, NY), got %d: %+v", len(byState), report.States)
	}

	ca := byState["CA"]
	if ca.GrossSales != 180000 { // 100000 + 50000 + 30000 (void + last-year excluded)
		t.Errorf("CA gross = %d, want 180000", ca.GrossSales)
	}
	if ca.TaxableSales != 100000 {
		t.Errorf("CA taxable = %d, want 100000 (only the taxed invoice)", ca.TaxableSales)
	}
	if ca.ExemptSales != 50000 {
		t.Errorf("CA exempt = %d, want 50000 (the sales_tax_exempt invoice only)", ca.ExemptSales)
	}
	if ca.NonTaxableSales != 30000 {
		t.Errorf("CA non-taxable = %d, want 30000 (no-nexus; exempt excluded)", ca.NonTaxableSales)
	}
	if ca.GrossSales != ca.TaxableSales+ca.ExemptSales+ca.NonTaxableSales {
		t.Errorf("CA buckets must partition gross: %d != %d+%d+%d",
			ca.GrossSales, ca.TaxableSales, ca.ExemptSales, ca.NonTaxableSales)
	}
	if ca.TaxCollected != 8000 {
		t.Errorf("CA tax = %d, want 8000", ca.TaxCollected)
	}
	if ca.InvoiceCount != 3 {
		t.Errorf("CA invoice count = %d, want 3 (void + last-year excluded)", ca.InvoiceCount)
	}
	if !ca.HasNexus || ca.NexusType != domain.NexusPhysical {
		t.Errorf("CA nexus = (%v, %q), want (true, physical)", ca.HasNexus, ca.NexusType)
	}

	ny := byState["NY"]
	if ny.TaxCollected != 17000 || ny.GrossSales != 200000 {
		t.Errorf("NY = gross %d / tax %d, want 200000 / 17000", ny.GrossSales, ny.TaxCollected)
	}
	if ny.HasNexus {
		t.Error("NY has_nexus = true, want false (undeclared, collecting tax = worth attention)")
	}

	// Totals exclude void, out-of-period, and non-US.
	if report.TotalGrossSales != 380000 {
		t.Errorf("total gross = %d, want 380000 (CA 180000 + NY 200000)", report.TotalGrossSales)
	}
	if report.TotalTaxCollected != 25000 {
		t.Errorf("total tax = %d, want 25000 (8000 + 17000)", report.TotalTaxCollected)
	}
	if report.Currency != "USD" {
		t.Errorf("currency = %q, want USD", report.Currency)
	}
}

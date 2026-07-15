package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// commitCheckingGSP records, for each IRN request, whether the invoice was
// ALREADY committed to the DB when the (irreversible) government call was made.
type commitCheckingGSP struct {
	conn         *sql.DB
	calls        int
	sawCommitted bool
}

func (g *commitCheckingGSP) GenerateIRN(ctx context.Context, inv *domain.Invoice) (*port.EInvoiceResponse, error) {
	g.calls++
	_ = g.conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM invoices WHERE id=$1)`, inv.ID).Scan(&g.sawCommitted)
	return &port.EInvoiceResponse{IRN: "IRN-TEST", AckNo: "ACK1", SignedQRCode: "QR", Status: "GENERATED"}, nil
}
func (g *commitCheckingGSP) GenerateIRNFull(ctx context.Context, req *port.EInvoiceRequest) (*port.EInvoiceResponse, error) {
	return &port.EInvoiceResponse{IRN: "IRN-TEST"}, nil
}
func (g *commitCheckingGSP) CancelIRN(ctx context.Context, irn, reason string) error { return nil }
func (g *commitCheckingGSP) GetIRNByDocDetails(ctx context.Context, docType, docNum, docDate string) (*port.EInvoiceResponse, error) {
	return nil, nil
}

// TestProrationEInvoice_RequestedOnlyAfterCommit is the PHASE2 #3 guard: on a
// plan change the government IRN must be requested only AFTER the proration
// invoice is durably committed — otherwise a rolled-back transaction would
// orphan an irreversible IRN registered at the government portal. The GSP mock
// records whether the invoice already existed in the DB when the IRN call landed.
func TestProrationEInvoice_RequestedOnlyAfterCommit(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed e-invoice ordering test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "EInv-"+run, "einv-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	// India business customer with a GSTIN → eligible for e-invoicing (the gspAdapter
	// fallback path, since einvoiceService is left unset here).
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, country, gstin, tax_type, ledger_account_id, created_at, updated_at)
		 VALUES ($1,$2,$3,'Acme India','India','27AAAAA0000A1Z5','business',$4,NOW(),NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	seedPlan := func(name string, amountMinor int64) uuid.UUID {
		planID := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO plans (id, tenant_id, name, code, interval_unit, interval_count, active) VALUES ($1,$2,$3,$4,'month',1,TRUE)`,
			planID, tenantID, name, name+"-"+run); err != nil {
			t.Fatalf("seed plan %s: %v", name, err)
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO prices (id, plan_id, currency, amount, type) VALUES ($1,$2,'INR',$3,'recurring')`,
			uuid.New(), planID, amountMinor); err != nil {
			t.Fatalf("seed price %s: %v", name, err)
		}
		return planID
	}
	currentPlanID := seedPlan("Starter", 100000)
	targetPlanID := seedPlan("Pro", 200000) // upgrade → a proration charge invoice

	subID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO subscriptions (id, tenant_id, customer_id, plan_id, status, current_period_start, current_period_end, billing_anchor, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,'active', NOW()-INTERVAL '15 days', NOW()+INTERVAL '15 days', NOW()-INTERVAL '15 days', NOW(), NOW())`,
		subID, tenantID, customerID, currentPlanID); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	gsp := &commitCheckingGSP{conn: conn}
	svc := NewSubscriptionService(
		db.NewSubscriptionRepository(conn), db.NewInvoiceRepository(conn), db.NewPlanRepository(conn), db.NewCustomerRepository(dbx),
		nil, nil, nil, nil, gsp, db.NewTxManager(conn), nil, nil,
	)
	svc.SetCreditNoteRepo(db.NewCreditNoteRepository(dbx))

	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)
	if _, err := svc.UpdateSubscription(tctx, tenantID, subID, targetPlanID); err != nil {
		t.Fatalf("UpdateSubscription (upgrade): %v", err)
	}

	if gsp.calls != 1 {
		t.Fatalf("GSP GenerateIRN called %d times, want 1", gsp.calls)
	}
	if !gsp.sawCommitted {
		t.Error("IRN was requested BEFORE the invoice was committed — a rollback would orphan a government IRN (PHASE2 #3)")
	}
	// The IRN must be persisted onto the committed invoice.
	var irn string
	if err := conn.QueryRowContext(ctx, `SELECT COALESCE(irn,'') FROM invoices WHERE subscription_id=$1`, subID).Scan(&irn); err != nil {
		t.Fatalf("read invoice irn: %v", err)
	}
	if irn != "IRN-TEST" {
		t.Errorf("persisted irn = %q, want IRN-TEST (post-commit e-invoice must persist the IRN)", irn)
	}
}

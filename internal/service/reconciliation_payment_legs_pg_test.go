package service

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestPaymentReconciliation_TDSAndWalletLegs proves the payment-leg reconciler
// counts every cash-equivalent AR relief (Code-3 cash, Code-10 TDS, Code-12
// wallet drain), not just Code-3. Before the fix a TDS invoice raised a false
// payment_amount_mismatch (cash leg short by the withheld tax) and a wallet-paid
// invoice raised a false missing_payment_transaction (no cash leg at all), even
// though AR was fully relieved. A genuinely short payment must still be flagged.
func TestPaymentReconciliation_TDSAndWalletLegs(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed payment reconciliation test")
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

	tenantID := seedRevRecTenant(t, conn)
	run := uuid.New().String()[:8]
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, name, ledger_account_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'Acme', $4, NOW(), NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))

	// seedPaidInvoice inserts a paid invoice row (amount_paid = total, no credit)
	// and posts its Code-1 invoice leg; the caller posts the payment-side legs.
	seedPaidInvoice := func(num string, total int64) *domain.Invoice {
		inv := &domain.Invoice{
			ID: uuid.New(), TenantID: tenantID, CustomerID: customerID,
			InvoiceNumber: num, Total: total, Currency: "USD",
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, created_at, due_date)
			 VALUES ($1,$2,$3,'USD',$4,$4,$4,'paid',$5,NOW(),NOW())`,
			inv.ID, tenantID, customerID, total, num); err != nil {
			t.Fatalf("seed invoice %s: %v", num, err)
		}
		if err := ledger.RecordInvoice(ctx, inv); err != nil {
			t.Fatalf("RecordInvoice %s: %v", num, err)
		}
		return inv
	}

	// (A) TDS invoice: cash leg is short by the withheld tax; Code-10 makes up
	// the difference. amount_paid = 100000, cash = 85000, TDS = 15000.
	tdsInv := seedPaidInvoice("TDS-"+run, 100000)
	tdsInv.TDSAmount = 15000
	if err := ledger.RecordPaymentWithSettled(ctx, tdsInv, 0); err != nil {
		t.Fatalf("RecordPayment (TDS): %v", err)
	}

	// (B) Wallet-fully-paid invoice: NO cash leg at all, only a Code-12 drain.
	walletInv := seedPaidInvoice("WAL-"+run, 50000)
	if _, err := ledger.RecordWalletDrain(ctx, tenantID, nil, customerID, walletInv.ID, 50000, "wallet settle"); err != nil {
		t.Fatalf("RecordWalletDrain: %v", err)
	}

	// (C) Negative control: a genuinely short cash payment MUST still be flagged.
	// amount_paid = 100000 but only 90000 of cash posted.
	shortInv := seedPaidInvoice("SHORT-"+run, 100000)
	shortInv.Total = 90000 // make RecordPayment post only 90000 of cash
	if err := ledger.RecordPaymentWithSettled(ctx, shortInv, 0); err != nil {
		t.Fatalf("RecordPayment (short): %v", err)
	}

	recon := NewReconciliationService(db.NewLedgerRepository(conn), nil)
	report, err := recon.Run(ctx, tenantID)
	if err != nil {
		t.Fatalf("reconciliation Run: %v", err)
	}

	var falseTDSorWallet int
	shortFlagged := false
	for _, d := range report.Discrepancies {
		if d.Type != DiscrepancyMissingPaymentTx && d.Type != DiscrepancyPaymentAmountMismatch {
			continue
		}
		switch {
		case d.InvoiceID != nil && *d.InvoiceID == shortInv.ID:
			shortFlagged = true
		case d.InvoiceID != nil && (*d.InvoiceID == tdsInv.ID || *d.InvoiceID == walletInv.ID):
			falseTDSorWallet++
			t.Errorf("false payment discrepancy on a fully-relieved invoice: %s %+v", d.Type, d)
		}
	}
	if falseTDSorWallet != 0 {
		t.Fatalf("%d false payment discrepancies on TDS/wallet invoices (want 0)", falseTDSorWallet)
	}
	if !shortFlagged {
		t.Fatal("the genuinely short payment was not flagged — the reconciler is now blind to real shortfalls")
	}
}

package db

import (
	"context"
	"testing"
	"time"
)

// TestMarkPaid_TenantIsolation proves the ENG-169 money-path scoping: MarkPaid
// is bound by tenant_id, so a settle call carrying the wrong tenant no-ops
// (returns false, invoice stays open) and only the owning tenant's call flips
// the invoice to paid. This is the guard that a dropped context-tenant would
// otherwise turn into a cross-tenant settlement.
func TestMarkPaid_TenantIsolation(t *testing.T) {
	dbx := openCancelFlowTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()

	owner, customerID := seedCreditAppTenantCustomer(t, conn)
	attacker := seedCancelFlowTenant(t, conn)
	invID := seedInvoiceRow(t, conn, owner, customerID, 100000)

	repo := NewInvoiceRepository(conn)

	// Wrong tenant: no row matches -> not transitioned, invoice stays open.
	ok, err := repo.MarkPaid(ctx, attacker, invID, time.Now().UTC())
	if err != nil {
		t.Fatalf("cross-tenant MarkPaid error: %v", err)
	}
	if ok {
		t.Fatal("cross-tenant MarkPaid reported a transition; want false")
	}
	var status string
	if err := conn.QueryRowContext(ctx, `SELECT status FROM invoices WHERE id = $1`, invID).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "open" {
		t.Errorf("cross-tenant MarkPaid settled the invoice: status = %q, want open", status)
	}

	// Owning tenant: transitions the invoice to paid.
	ok, err = repo.MarkPaid(ctx, owner, invID, time.Now().UTC())
	if err != nil {
		t.Fatalf("owner MarkPaid error: %v", err)
	}
	if !ok {
		t.Fatal("owner MarkPaid did not transition the invoice")
	}
	if err := conn.QueryRowContext(ctx, `SELECT status FROM invoices WHERE id = $1`, invID).Scan(&status); err != nil {
		t.Fatalf("read status after owner settle: %v", err)
	}
	if status != "paid" {
		t.Errorf("owner MarkPaid did not persist: status = %q, want paid", status)
	}
}

// TestSetGatewayPaymentID_TenantIsolation proves SetGatewayPaymentID is
// tenant-scoped: a cross-tenant call leaves gateway_payment_id untouched.
func TestSetGatewayPaymentID_TenantIsolation(t *testing.T) {
	dbx := openCancelFlowTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()

	owner, customerID := seedCreditAppTenantCustomer(t, conn)
	attacker := seedCancelFlowTenant(t, conn)
	invID := seedInvoiceRow(t, conn, owner, customerID, 100000)

	repo := NewInvoiceRepository(conn)

	if err := repo.SetGatewayPaymentID(ctx, attacker, invID, "pay_attacker"); err != nil {
		t.Fatalf("cross-tenant SetGatewayPaymentID error: %v", err)
	}
	var got *string
	if err := conn.QueryRowContext(ctx, `SELECT gateway_payment_id FROM invoices WHERE id = $1`, invID).Scan(&got); err != nil {
		t.Fatalf("read gateway_payment_id: %v", err)
	}
	if got != nil && *got != "" {
		t.Errorf("cross-tenant SetGatewayPaymentID wrote %q; want untouched", *got)
	}

	if err := repo.SetGatewayPaymentID(ctx, owner, invID, "pay_owner"); err != nil {
		t.Fatalf("owner SetGatewayPaymentID error: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT gateway_payment_id FROM invoices WHERE id = $1`, invID).Scan(&got); err != nil {
		t.Fatalf("read gateway_payment_id after owner set: %v", err)
	}
	if got == nil || *got != "pay_owner" {
		t.Errorf("owner SetGatewayPaymentID did not persist: got %v, want pay_owner", got)
	}
}

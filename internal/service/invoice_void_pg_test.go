package service

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestMarkInvoicePaid_RefusesVoidInvoice_Postgres proves a payment arriving for
// a void invoice leaves it void, with no cash leg and no revenue.
func TestMarkInvoicePaid_RefusesVoidInvoice_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)
	e := newRenewalE2E(t, conn, dbx)
	h := e.h

	subID, _ := e.seedDue("1 minute", false, "")
	e.sweep()
	var invID string
	if err := conn.QueryRowContext(h.ctx, `SELECT id FROM invoices WHERE subscription_id = $1`, subID).Scan(&invID); err != nil {
		t.Fatalf("read invoice: %v", err)
	}
	inv, err := e.invoice.InvoiceRepo.GetByID(h.tctx, uuid.MustParse(invID))
	if err != nil || inv == nil {
		t.Fatalf("load invoice: %v", err)
	}
	voided, err := e.invoice.InvoiceRepo.VoidIfOpen(h.tctx, h.tenantID, inv.ID)
	if err != nil || !voided {
		t.Fatalf("void: voided = %v, err = %v", voided, err)
	}

	ledgerRows := func() (n int) {
		t.Helper()
		if err := conn.QueryRowContext(h.ctx, `SELECT COUNT(*) FROM ledger_transactions WHERE reference_id = $1`, inv.ID).Scan(&n); err != nil {
			t.Fatalf("count ledger rows: %v", err)
		}
		return n
	}
	before := ledgerRows()

	var logged bytes.Buffer
	h.subSvc.logger = slog.New(slog.NewTextHandler(&logged, nil))
	transitioned, err := h.subSvc.MarkInvoicePaid(h.tctx, inv.ID)
	if err != nil {
		t.Fatalf("MarkInvoicePaid: %v", err)
	}
	if transitioned {
		t.Error("transitioned = true, want false")
	}
	if !strings.Contains(logged.String(), "needs refund or review") {
		t.Errorf("log = %q, want the payment flagged for a person", logged.String())
	}
	if got := e.invoices(subID); len(got) != 1 || got[0].status != string(domain.InvoiceStatusVoid) || got[0].paid != 0 {
		t.Errorf("invoices = %+v, want one void unpaid invoice", got)
	}
	if after := ledgerRows(); after != before {
		t.Errorf("ledger rows = %d, want unchanged %d", after, before)
	}

	// The repository refuses too, for a void that lands after the status read.
	if won, err := e.invoice.InvoiceRepo.MarkPaid(h.ctx, h.tenantID, inv.ID, inv.CreatedAt); err != nil || won {
		t.Errorf("MarkPaid won = %v, err = %v, want false", won, err)
	}
}

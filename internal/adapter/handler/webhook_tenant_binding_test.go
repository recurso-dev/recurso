package handler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestInvoiceBelongsToWebhookConn is the S2 cross-tenant guard: on a BYO
// (:connID) webhook route, a handler may act on an invoice ONLY if it belongs
// to the connection's own tenant — otherwise a BYO tenant could settle/refund
// another tenant's invoice by signing a body carrying its id. The legacy env
// route (no conn tenant) is unaffected.
func TestInvoiceBelongsToWebhookConn(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	invA := &domain.Invoice{ID: uuid.New(), TenantID: tenantA}

	// Legacy env route: no conn tenant on the context → always allowed.
	if !invoiceBelongsToWebhookConn(context.Background(), invA) {
		t.Error("env route (no conn tenant) must allow any invoice")
	}

	// BYO route bound to tenant A, invoice of tenant A → allowed.
	ctxA := context.WithValue(context.Background(), webhookConnTenantKey, tenantA)
	if !invoiceBelongsToWebhookConn(ctxA, invA) {
		t.Error("BYO route with matching tenant must allow")
	}

	// BYO route bound to tenant B, invoice of tenant A → REJECTED (the attack).
	ctxB := context.WithValue(context.Background(), webhookConnTenantKey, tenantB)
	if invoiceBelongsToWebhookConn(ctxB, invA) {
		t.Error("BYO route must REJECT another tenant's invoice (cross-tenant settlement/refund)")
	}

	// nil invoice → allowed; the caller's own not-found path handles it.
	if !invoiceBelongsToWebhookConn(ctxB, nil) {
		t.Error("nil invoice must be left to the caller's not-found path")
	}

	// webhookConnTenant reads back what was stored (uuid.Nil when absent).
	if got := webhookConnTenant(ctxA); got != tenantA {
		t.Errorf("webhookConnTenant = %s, want %s", got, tenantA)
	}
	if got := webhookConnTenant(context.Background()); got != uuid.Nil {
		t.Errorf("webhookConnTenant (env) = %s, want Nil", got)
	}
}

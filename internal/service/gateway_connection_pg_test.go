package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
)

// TestConnectKeyRotation_Postgres proves rotating keys on the real table keeps
// what the gateway and saved cards depend on: the connection id and webhook
// secret stay, and a saved card's charger opens the new key.
func TestConnectKeyRotation_Postgres(t *testing.T) {
	conn, dbx := openRenewalE2EDB(t)
	h := newInvariantHarness(t, conn, dbx)
	svc := NewGatewayConnectionService(db.NewGatewayConnectionRepository(conn), testVault(t))
	ctx := context.Background()

	before, err := svc.Connect(ctx, h.tenantID, ConnectInput{
		Provider: "stripe", Mode: "test", PublicKey: "pk_test_a", SecretKey: "sk_test_old", WebhookSecret: "whsec_kept",
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	after, err := svc.Connect(ctx, h.tenantID, ConnectInput{
		Provider: "stripe", Mode: "test", PublicKey: "pk_test_b", SecretKey: "sk_test_new",
	})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if after.ID != before.ID {
		t.Fatalf("connection id changed: %s to %s", before.ID, after.ID)
	}

	// Webhooks and saved cards both resolve the connection by id.
	stored, err := svc.GetByID(ctx, before.ID)
	if err != nil || stored == nil || !stored.Active {
		t.Fatalf("stored = %+v, err = %v, want active", stored, err)
	}
	if stored.PublicKey != "pk_test_b" {
		t.Errorf("public key = %q, want pk_test_b", stored.PublicKey)
	}
	if secret, err := svc.OpenSecret(stored); err != nil || secret != "sk_test_new" {
		t.Errorf("secret = %q, err = %v, want the new key", secret, err)
	}
	if webhook, err := svc.OpenWebhookSecret(stored); err != nil || webhook != "whsec_kept" {
		t.Errorf("webhook secret = %q, err = %v, want it kept", webhook, err)
	}
	if !stored.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("updated_at = %v, want it moved so the gateway cache is rebuilt", stored.UpdatedAt)
	}
	var rows int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM gateway_connections WHERE tenant_id = $1`, h.tenantID).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Errorf("rows = %d, want 1", rows)
	}

	// A different mode is a new connection and the old one stops being active.
	live, err := svc.Connect(ctx, h.tenantID, ConnectInput{
		Provider: "stripe", Mode: "live", PublicKey: "pk_live_a", SecretKey: "sk_live_a",
	})
	if err != nil {
		t.Fatalf("connect live: %v", err)
	}
	if live.ID == before.ID || live.ID == uuid.Nil {
		t.Errorf("live connection id = %s, want a new one", live.ID)
	}
	if old, err := svc.GetByID(ctx, before.ID); err != nil || old == nil || old.Active {
		t.Errorf("old connection = %+v, err = %v, want inactive", old, err)
	}
}

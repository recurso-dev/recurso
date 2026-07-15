package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestDunningCampaignUpdate_TenantIsolation proves the ENG-169 defense-in-depth
// tenant scoping on UpdateCampaign: a campaign carrying another tenant's id
// no-ops (the owner's row is untouched), while the owner's own update persists.
func TestDunningCampaignUpdate_TenantIsolation(t *testing.T) {
	dbx := openCancelFlowTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewDunningCampaignRepository(conn)
	ctx := context.Background()

	owner := seedCancelFlowTenant(t, conn)
	attacker := seedCancelFlowTenant(t, conn)

	id := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO dunning_campaigns (id, tenant_id, name, is_active, trigger_event, created_at, updated_at)
		 VALUES ($1, $2, 'Overdue', TRUE, 'invoice_overdue', NOW(), NOW())`, id, owner); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	// Attacker updates the owner's campaign id under their own tenant -> no match.
	tampered := &domain.DunningCampaign{
		ID: id, TenantID: attacker, Name: "HACKED", IsActive: false,
		TriggerEvent: "payment_failed", UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpdateCampaign(ctx, tampered); err != nil {
		t.Fatalf("cross-tenant UpdateCampaign returned error (expected no-op): %v", err)
	}
	var gotName string
	if err := conn.QueryRowContext(ctx, `SELECT name FROM dunning_campaigns WHERE id = $1`, id).Scan(&gotName); err != nil {
		t.Fatalf("read name: %v", err)
	}
	if gotName != "Overdue" {
		t.Errorf("cross-tenant update mutated campaign: name = %q, want %q", gotName, "Overdue")
	}

	// Owner update persists.
	ownerUpd := &domain.DunningCampaign{
		ID: id, TenantID: owner, Name: "Overdue v2", IsActive: true,
		TriggerEvent: "invoice_overdue", UpdatedAt: time.Now().UTC(),
	}
	if err := repo.UpdateCampaign(ctx, ownerUpd); err != nil {
		t.Fatalf("owner UpdateCampaign: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT name FROM dunning_campaigns WHERE id = $1`, id).Scan(&gotName); err != nil {
		t.Fatalf("read name after owner update: %v", err)
	}
	if gotName != "Overdue v2" {
		t.Errorf("owner update did not persist: name = %q, want %q", gotName, "Overdue v2")
	}
}

// TestWebhookEndpoint_TenantIsolation proves the ENG-169 scoping on the webhook
// endpoint Update and Delete: a cross-tenant call no-ops and the owner's row
// survives/persists, while the owner's own calls succeed.
func TestWebhookEndpoint_TenantIsolation(t *testing.T) {
	dbx := openCancelFlowTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewWebhookEndpointRepository(conn)
	ctx := context.Background()

	owner := seedCancelFlowTenant(t, conn)
	attacker := seedCancelFlowTenant(t, conn)

	id := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO webhook_endpoints (id, tenant_id, url, secret, events, status, created_at, updated_at)
		 VALUES ($1, $2, 'https://owner.example/hook', 'sek', $3, 'active', NOW(), NOW())`,
		id, owner, pq.Array([]string{"invoice.paid"})); err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}

	// Attacker Update under their own tenant -> no match, URL untouched.
	tampered := &domain.WebhookEndpoint{
		ID: id, TenantID: attacker, URL: "https://evil.example/hook",
		Events: []string{"invoice.paid"}, Status: "inactive",
	}
	if err := repo.Update(ctx, tampered); err != nil {
		t.Fatalf("cross-tenant Update returned error (expected no-op): %v", err)
	}
	var gotURL, gotStatus string
	if err := conn.QueryRowContext(ctx, `SELECT url, status FROM webhook_endpoints WHERE id = $1`, id).Scan(&gotURL, &gotStatus); err != nil {
		t.Fatalf("read endpoint: %v", err)
	}
	if gotURL != "https://owner.example/hook" || gotStatus != "active" {
		t.Errorf("cross-tenant Update mutated endpoint: url=%q status=%q", gotURL, gotStatus)
	}

	// Attacker Delete under their own tenant -> no match, row survives.
	if err := repo.Delete(ctx, attacker, id); err != nil {
		t.Fatalf("cross-tenant Delete returned error (expected no-op): %v", err)
	}
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_endpoints WHERE id = $1`, id).Scan(&count); err != nil {
		t.Fatalf("count endpoint: %v", err)
	}
	if count != 1 {
		t.Fatalf("cross-tenant Delete removed endpoint: count = %d, want 1", count)
	}

	// Owner Update persists.
	ownerUpd := &domain.WebhookEndpoint{
		ID: id, TenantID: owner, URL: "https://owner.example/hook2",
		Events: []string{"invoice.paid"}, Status: "inactive",
	}
	if err := repo.Update(ctx, ownerUpd); err != nil {
		t.Fatalf("owner Update: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT url, status FROM webhook_endpoints WHERE id = $1`, id).Scan(&gotURL, &gotStatus); err != nil {
		t.Fatalf("read endpoint after owner update: %v", err)
	}
	if gotURL != "https://owner.example/hook2" || gotStatus != "inactive" {
		t.Errorf("owner Update did not persist: url=%q status=%q", gotURL, gotStatus)
	}

	// Owner Delete removes the row.
	if err := repo.Delete(ctx, owner, id); err != nil {
		t.Fatalf("owner Delete: %v", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_endpoints WHERE id = $1`, id).Scan(&count); err != nil {
		t.Fatalf("count endpoint after owner delete: %v", err)
	}
	if count != 0 {
		t.Errorf("owner Delete did not remove endpoint: count = %d, want 0", count)
	}
}

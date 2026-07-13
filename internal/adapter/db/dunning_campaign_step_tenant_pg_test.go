package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func openDunningStepTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed dunning-step test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func seedDunningTenant(t *testing.T, conn *sql.DB) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "DUN-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenantID
}

// TestDunningCampaignStep_TenantIsolation proves the ENG-165 C2 fix: a
// campaign's steps can only be created, updated, or deleted by the tenant that
// owns the parent campaign. A foreign tenant's writes touch zero rows and
// return sql.ErrNoRows, while the owner's writes succeed.
func TestDunningCampaignStep_TenantIsolation(t *testing.T) {
	conn := openDunningStepTestDB(t)
	repo := NewDunningCampaignRepository(conn)
	ctx := context.Background()

	owner := seedDunningTenant(t, conn)
	attacker := seedDunningTenant(t, conn)

	now := time.Now().UTC()
	campaign := &domain.DunningCampaign{
		ID:           uuid.New(),
		TenantID:     owner,
		Name:         "Recovery",
		IsActive:     true,
		TriggerEvent: "payment_failed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateCampaign(ctx, campaign); err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	// Attacker cannot add a step to the owner's campaign.
	rogue := &domain.DunningCampaignStep{
		ID: uuid.New(), CampaignID: campaign.ID, StepOrder: 9,
		Channel: domain.DunningChannelInApp, IsPaymentWall: true, CreatedAt: now,
	}
	if err := repo.CreateStep(ctx, rogue, attacker); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("attacker CreateStep: want sql.ErrNoRows, got %v", err)
	}

	// Owner adds a legitimate step.
	step := &domain.DunningCampaignStep{
		ID: uuid.New(), CampaignID: campaign.ID, StepOrder: 0,
		Channel: domain.DunningChannelEmail, Subject: "Pay up", Body: "please", CreatedAt: now,
	}
	if err := repo.CreateStep(ctx, step, owner); err != nil {
		t.Fatalf("owner CreateStep: %v", err)
	}

	// Attacker cannot update the owner's step.
	tampered := &domain.DunningCampaignStep{ID: step.ID, StepOrder: 99, Channel: domain.DunningChannelEmail, IsPaymentWall: true}
	if err := repo.UpdateStep(ctx, tampered, attacker); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("attacker UpdateStep: want sql.ErrNoRows, got %v", err)
	}
	var order int
	var wall bool
	if err := conn.QueryRowContext(ctx, `SELECT step_order, is_payment_wall FROM dunning_campaign_steps WHERE id = $1`, step.ID).Scan(&order, &wall); err != nil {
		t.Fatalf("read step: %v", err)
	}
	if order != 0 || wall {
		t.Errorf("attacker update mutated step: order=%d wall=%v, want 0/false", order, wall)
	}

	// Attacker cannot delete the owner's step.
	if err := repo.DeleteStep(ctx, step.ID, attacker); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("attacker DeleteStep: want sql.ErrNoRows, got %v", err)
	}

	// Owner update + delete succeed.
	ownerUpd := &domain.DunningCampaignStep{ID: step.ID, StepOrder: 3, Channel: domain.DunningChannelEmail}
	if err := repo.UpdateStep(ctx, ownerUpd, owner); err != nil {
		t.Fatalf("owner UpdateStep: %v", err)
	}
	if err := repo.DeleteStep(ctx, step.ID, owner); err != nil {
		t.Fatalf("owner DeleteStep: %v", err)
	}
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM dunning_campaign_steps WHERE id = $1`, step.ID).Scan(&count); err != nil {
		t.Fatalf("count step: %v", err)
	}
	if count != 0 {
		t.Errorf("owner delete did not remove the step: count = %d", count)
	}
}

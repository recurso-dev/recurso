package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// SetRegistrations is a full atomic replacement, ListRegistrations reads it back
// with status + optional date (Track D · D4).
func TestTaxRegistrations_RoundTrip_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed registrations test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "RG-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewTaxNexusRepository(conn)
	regDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if err := repo.SetRegistrations(ctx, tenantID, []domain.TaxRegistration{
		{StateCode: "CA", RegistrationNumber: "CA-123", Status: domain.RegistrationRegistered, RegisteredAt: &regDate},
		{StateCode: "ny", RegistrationNumber: "", Status: domain.RegistrationPending},
	}); err != nil {
		t.Fatalf("set registrations: %v", err)
	}

	got, err := repo.ListRegistrations(ctx, tenantID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 registrations, got %d", len(got))
	}
	byState := map[string]domain.TaxRegistration{}
	for _, r := range got {
		byState[r.StateCode] = r
	}
	ca := byState["CA"]
	if ca.RegistrationNumber != "CA-123" || ca.Status != domain.RegistrationRegistered {
		t.Errorf("CA = %+v, want CA-123/registered", ca)
	}
	if ca.RegisteredAt == nil || ca.RegisteredAt.Year() != 2026 {
		t.Errorf("CA registered_at = %v, want 2026-03-01", ca.RegisteredAt)
	}
	ny := byState["NY"] // uppercased on write
	if ny.Status != domain.RegistrationPending || ny.RegisteredAt != nil {
		t.Errorf("NY = %+v, want pending / no date", ny)
	}

	// Full replacement: setting a new list drops the old rows.
	if err := repo.SetRegistrations(ctx, tenantID, []domain.TaxRegistration{
		{StateCode: "TX", RegistrationNumber: "TX-9", Status: domain.RegistrationRegistered},
	}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	after, err := repo.ListRegistrations(ctx, tenantID)
	if err != nil {
		t.Fatalf("list after replace: %v", err)
	}
	if len(after) != 1 || after[0].StateCode != "TX" {
		t.Fatalf("replace should leave only TX, got %+v", after)
	}
}

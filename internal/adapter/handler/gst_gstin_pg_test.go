package handler

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestSellerGSTIN_PrimaryEntityFallback_Postgres proves the QA fix for the
// blank-GSTIN filing bug: the primary entity's GST config lives in the
// entity_id IS NULL (tenant/default) row, so filing GSTR for the primary BY ITS
// CONCRETE ID must fall back to that row — while a non-primary entity with no
// config of its own must still yield "" (never inherit another entity's GSTIN).
func TestSellerGSTIN_PrimaryEntityFallback_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed seller-GSTIN test")
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
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "GSTIN-"+run, "gstin-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	// The tenants trigger auto-created the primary entity; load its id.
	var primaryID uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT id FROM entities WHERE tenant_id=$1 AND is_primary=TRUE`, tenantID).Scan(&primaryID); err != nil {
		t.Fatalf("load primary entity: %v", err)
	}
	var branchID uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, is_primary, tb_ledger_id, invoice_prefix) VALUES ($1,'Branch',FALSE,2,$2) RETURNING id`,
		tenantID, "GB"+uuid.New().String()[:4]).Scan(&branchID); err != nil {
		t.Fatalf("seed branch entity: %v", err)
	}

	gstRepo := db.NewGSTConfigRepository(conn)
	// The settings UI writes the primary's config as the tenant/default row
	// (entity_id IS NULL) — reproduce exactly that.
	primaryGSTIN := "27AAAAA0000A1Z5" // 15 chars (gstin col is VARCHAR(15))
	if err := gstRepo.Upsert(ctx, tenantID, nil, &domain.TenantGSTConfig{
		TenantID: tenantID.String(), GSTIN: primaryGSTIN, StateCode: "27", StateName: "Maharashtra",
	}); err != nil {
		t.Fatalf("upsert default gst config: %v", err)
	}

	h := NewGSTHandler(gstRepo, nil)
	h.SetEntityReader(db.NewEntityRepository(conn))

	gin.SetMode(gin.TestMode)
	c, _ := jsonCtx(http.MethodGet, "/v1/india/gstr1", "")

	// THE FIX: primary by concrete id → the tenant/default GSTIN, not "".
	if got := h.sellerGSTIN(c, tenantID, &primaryID); got != primaryGSTIN {
		t.Errorf("sellerGSTIN(primary concrete id) = %q, want %q (fallback to the NULL/default row)", got, primaryGSTIN)
	}
	// nil (SCOPE_ALL / single-entity) → default GSTIN, unchanged behavior.
	if got := h.sellerGSTIN(c, tenantID, nil); got != primaryGSTIN {
		t.Errorf("sellerGSTIN(nil) = %q, want %q", got, primaryGSTIN)
	}
	// A non-primary entity with NO config must yield "" — an obviously-invalid
	// filing, never a silently-wrong inherited GSTIN.
	if got := h.sellerGSTIN(c, tenantID, &branchID); got != "" {
		t.Errorf("sellerGSTIN(branch, unconfigured) = %q, want empty", got)
	}

	// Once the branch has its own config, it resolves its own GSTIN.
	branchGSTIN := "29BBBBB1111B2Z6"
	if err := gstRepo.Upsert(ctx, tenantID, &branchID, &domain.TenantGSTConfig{
		TenantID: tenantID.String(), GSTIN: branchGSTIN, StateCode: "29", StateName: "Karnataka",
	}); err != nil {
		t.Fatalf("upsert branch gst config: %v", err)
	}
	if got := h.sellerGSTIN(c, tenantID, &branchID); got != branchGSTIN {
		t.Errorf("sellerGSTIN(branch, configured) = %q, want %q", got, branchGSTIN)
	}
}

package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestAuthMiddleware_ModeGate_Postgres proves the mode gate end-to-end: a key's
// livemode must match the server's, so a test key is rejected on a live-money
// server and a live key is rejected on a non-live one.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database.
func TestAuthMiddleware_ModeGate_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed mode-gate test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()

	repo := db.NewTenantRepository(conn)
	ctx := context.Background()
	now := time.Now().UTC()

	tenantID := uuid.New()
	if err := repo.CreateTenant(ctx, &domain.Tenant{
		ID: tenantID, Name: "Gate-" + tenantID.String()[:8],
		Email: tenantID.String()[:8] + "@gate.com", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	testKey := domain.NewAPIKeyValue(false, uuid.New().String())
	liveKey := domain.NewAPIKeyValue(true, uuid.New().String())
	for _, kd := range []struct {
		val  string
		live bool
	}{{testKey, false}, {liveKey, true}} {
		if err := repo.CreateAPIKey(ctx, &domain.APIKey{
			ID: uuid.New(), TenantID: tenantID, KeyValue: kd.val,
			Type: "secret", IsActive: true, Livemode: kd.live, CreatedAt: now,
		}); err != nil {
			t.Fatalf("create api key: %v", err)
		}
	}

	gin.SetMode(gin.TestMode)
	// A fresh middleware per call so the verified-key cache never masks a result.
	statusFor := func(serverLive bool, key string) int {
		mw := AuthMiddleware(repo, serverLive)
		req := httptest.NewRequest(http.MethodGet, "/v1/plans", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		mw(c)
		if c.IsAborted() {
			return w.Code
		}
		return http.StatusOK
	}

	cases := []struct {
		name       string
		serverLive bool
		key        string
		want       int
	}{
		{"test key on non-live server", false, testKey, http.StatusOK},
		{"test key on live server", true, testKey, http.StatusUnauthorized},
		{"live key on live server", true, liveKey, http.StatusOK},
		{"live key on non-live server", false, liveKey, http.StatusUnauthorized},
		{"unknown key", false, "rsk_test_" + uuid.New().String(), http.StatusUnauthorized},
	}
	for _, tc := range cases {
		if got := statusFor(tc.serverLive, tc.key); got != tc.want {
			t.Errorf("%s: status = %d, want %d", tc.name, got, tc.want)
		}
	}
}

package service

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// scriptedLLM returns whatever "SQL" the test scripts — simulating both a
// well-behaved model and a fully adversarial / prompt-injected one.
type scriptedLLM struct{ reply string }

func (f *scriptedLLM) GenerateCompletion(ctx context.Context, system, user string) (string, error) {
	return f.reply, nil
}

// TestGenAIAsk_TenantIsolation_Postgres proves the ENG-137 fix: the tenant
// boundary and table allowlist for LLM-generated SQL are enforced by the
// database (genai schema + genai_readonly role), not by the prompt.
func TestGenAIAsk_TenantIsolation_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed genai test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations (000077 genai): %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	// Two tenants, one customer each.
	mkTenant := func() (uuid.UUID, string) {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
			id, "GenAI-"+id.String()[:8], id.String()[:8]+"@t.com"); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
		email := "cust-" + id.String()[:8] + "@example.com"
		custRepo := db.NewCustomerRepository(sqlx.NewDb(conn, "postgres"))
		name := "C"
		c := &domain.Customer{ID: uuid.New(), TenantID: id, Name: &name, Email: email,
			CreatedAt: time.Now(), UpdatedAt: time.Now()}
		tctx := context.WithValue(ctx, domain.TenantIDKey, id)
		if err := custRepo.Create(tctx, c); err != nil {
			t.Fatalf("seed customer: %v", err)
		}
		return id, email
	}
	tenantA, emailA := mkTenant()
	_, emailB := mkTenant()

	ask := func(reply string) (interface{}, error) {
		svc := NewGenAIService(&scriptedLLM{reply: reply}, conn)
		res, _, err := svc.Ask(ctx, tenantA, "q")
		return res, err
	}

	// (1) A legitimate query sees ONLY tenant A's rows.
	res, err := ask("SELECT email FROM customers")
	if err != nil {
		t.Fatalf("legitimate query: %v", err)
	}
	rows := res.([]map[string]interface{})
	seenA, seenB := false, false
	for _, r := range rows {
		if r["email"] == emailA {
			seenA = true
		}
		if r["email"] == emailB {
			seenB = true
		}
	}
	if !seenA || seenB {
		t.Fatalf("tenant isolation broken: seenA=%v seenB=%v rows=%v", seenA, seenB, rows)
	}

	// (2) Prompt-injected reads of sensitive tables are refused by the guard…
	if _, err := ask("SELECT password_hash FROM public.users"); err == nil {
		t.Fatal("public.users read was not blocked")
	}
	// …and even guard-evading unqualified names can't escape the genai schema
	// (search_path is pinned; the role has no access to public).
	if _, err := ask("SELECT password_hash FROM users"); err == nil ||
		!strings.Contains(err.Error(), "does not exist") && !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("users escape: err=%v, want relation-missing/permission-denied", err)
	}

	// (3) Catalog and GUC tampering are refused.
	if _, err := ask("SELECT tablename FROM pg_catalog.pg_tables"); err == nil {
		t.Fatal("pg_catalog read was not blocked")
	}
	if _, err := ask("SELECT set_config('app.tenant_id', '00000000-0000-0000-0000-000000000000', true)"); err == nil {
		t.Fatal("set_config tampering was not blocked")
	}

	// (4) Multi-statement and non-SELECT are refused.
	if _, err := ask("SELECT 1; DROP TABLE customers"); err == nil {
		t.Fatal("multi-statement was not blocked")
	}
	if _, err := ask("DELETE FROM customers"); err == nil {
		t.Fatal("non-SELECT was not blocked")
	}
}

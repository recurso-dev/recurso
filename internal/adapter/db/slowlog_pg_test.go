package db

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

// The wrapper must be invisible when queries are fast, log exactly the slow
// ones, and keep the pq fast paths (prepared statements, CopyIn, transactions)
// working — those are the paths a driver wrapper most often breaks.
func TestSlowQueryLog_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed slow-log test")
	}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	conn, err := NewConnectionWithSlowLog(dbURL, 30*time.Millisecond, logger)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	if _, err := conn.ExecContext(ctx, "SELECT 1"); err != nil {
		t.Fatalf("fast exec: %v", err)
	}
	var n int
	if err := conn.QueryRowContext(ctx, "SELECT $1::int + 1", 41).Scan(&n); err != nil || n != 42 {
		t.Fatalf("fast query with args: n=%d err=%v", n, err)
	}
	if buf.Len() != 0 {
		t.Fatalf("fast statements must not be logged, got: %s", buf.String())
	}

	if _, err := conn.ExecContext(ctx, "SELECT   pg_sleep(0.08)\n  -- multi-line, collapsed in the log"); err != nil {
		t.Fatalf("slow exec: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "slow query") || !strings.Contains(out, "kind=exec") || !strings.Contains(out, "pg_sleep(0.08) -- multi-line") {
		t.Fatalf("slow statement not logged as expected: %q", out)
	}
	if strings.Contains(out, "\n  ") {
		t.Fatalf("statement text should be whitespace-collapsed: %q", out)
	}

	// Prepared statement + CopyIn inside a transaction, the pq paths that go
	// through driver.Stmt rather than the conn's ExecContext.
	buf.Reset()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.ExecContext(ctx, "CREATE TEMP TABLE slowlog_copy (id int, name text) ON COMMIT DROP"); err != nil {
		t.Fatalf("temp table: %v", err)
	}
	stmt, err := tx.PrepareContext(ctx, `COPY "slowlog_copy" ("id", "name") FROM STDIN`)
	if err != nil {
		t.Fatalf("copyin prepare: %v", err)
	}
	defer func() { _ = stmt.Close() }()
	for i := 0; i < 3; i++ {
		if _, err := stmt.ExecContext(ctx, i, "row"); err != nil {
			t.Fatalf("copyin row: %v", err)
		}
	}
	if _, err := stmt.ExecContext(ctx); err != nil {
		t.Fatalf("copyin flush: %v", err)
	}
	if err := stmt.Close(); err != nil {
		t.Fatalf("copyin close: %v", err)
	}
	ps, err := tx.PrepareContext(ctx, "SELECT count(*) FROM slowlog_copy WHERE name = $1")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer func() { _ = ps.Close() }()
	var count int
	if err := ps.QueryRowContext(ctx, "row").Scan(&count); err != nil || count != 3 {
		t.Fatalf("prepared query: count=%d err=%v", count, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("fast prepared statements must not be logged, got: %s", buf.String())
	}
}

func TestCompactQuery(t *testing.T) {
	got := compactQuery("SELECT\n\t\tid,\n\t\tname\n\tFROM   customers")
	if got != "SELECT id, name FROM customers" {
		t.Fatalf("compactQuery = %q", got)
	}
	long := strings.Repeat("x", slowQueryTextLimit+50)
	if got := compactQuery(long); len([]rune(got)) != slowQueryTextLimit+1 || !strings.HasSuffix(got, "…") {
		t.Fatalf("compactQuery should truncate to %d runes plus ellipsis, got len %d", slowQueryTextLimit, len([]rune(got)))
	}
}

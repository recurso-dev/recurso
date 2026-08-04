package logctx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestContextHandler_StampsIDs proves the whole point: a log emitted with the
// ctx variant carries the request_id / tenant_id / user_id put on the context,
// so a production incident is reconstructable. A tenant stored as uuid.UUID
// (the repo layer's type) is rendered via its String().
func TestContextHandler_StampsIDs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	tenant := uuid.New()
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenant)
	ctx = With(ctx, "req-123", "user-456")

	logger.ErrorContext(ctx, "ledger write failed", "invoice_id", "inv-1")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log not valid json: %v", err)
	}
	if rec["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want req-123", rec["request_id"])
	}
	if rec["tenant_id"] != tenant.String() {
		t.Errorf("tenant_id = %v, want %s", rec["tenant_id"], tenant.String())
	}
	if rec["user_id"] != "user-456" {
		t.Errorf("user_id = %v, want user-456", rec["user_id"])
	}
	if rec["invoice_id"] != "inv-1" {
		t.Errorf("call-site attrs must survive: invoice_id = %v", rec["invoice_id"])
	}
}

// TestContextHandler_NoContextIsClean confirms a plain (non-ctx) background log
// isn't decorated with empty id fields — no noise when there's nothing to add.
func TestContextHandler_NoContextIsClean(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(&buf, nil)))
	logger.Info("startup")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log not valid json: %v", err)
	}
	for _, k := range []string{"request_id", "tenant_id", "user_id"} {
		if _, present := rec[k]; present {
			t.Errorf("%s should be absent when not on the context", k)
		}
	}
}

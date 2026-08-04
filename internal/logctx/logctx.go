// Package logctx makes production logs reconstructable. A ContextHandler wraps
// any slog.Handler and, on every record logged with a *Context method
// (slog.ErrorContext, InfoContext, …), stamps the request_id / tenant_id /
// user_id carried on the context. Middleware puts those IDs on the request
// context; services that log with the ctx variant then get them for free — no
// per-call-site plumbing beyond passing ctx.
package logctx

import (
	"context"
	"log/slog"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ContextHandler decorates a base slog.Handler with context-carried IDs.
type ContextHandler struct{ base slog.Handler }

// NewContextHandler wraps base so context IDs are added to every record.
func NewContextHandler(base slog.Handler) *ContextHandler {
	return &ContextHandler{base: base}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if v, ok := ctx.Value(domain.RequestIDKey).(string); ok && v != "" {
		r.AddAttrs(slog.String("request_id", v))
	}
	if v := tenantString(ctx); v != "" {
		r.AddAttrs(slog.String("tenant_id", v))
	}
	if v, ok := ctx.Value(domain.UserIDKey).(string); ok && v != "" {
		r.AddAttrs(slog.String("user_id", v))
	}
	return h.base.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{base: h.base.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{base: h.base.WithGroup(name)}
}

// tenantString reads the tenant id, tolerating either a stringer (uuid.UUID)
// or a plain string on the context.
func tenantString(ctx context.Context) string {
	v := ctx.Value(domain.TenantIDKey)
	switch t := v.(type) {
	case string:
		return t
	case interface{ String() string }:
		return t.String()
	default:
		return ""
	}
}

// With returns a context carrying the request and user IDs (tenant id is set
// separately via domain.TenantIDKey by the existing handler plumbing). Empty
// values are skipped so a partial identity still tags what it knows.
func With(ctx context.Context, requestID, userID string) context.Context {
	if requestID != "" {
		ctx = context.WithValue(ctx, domain.RequestIDKey, requestID)
	}
	if userID != "" {
		ctx = context.WithValue(ctx, domain.UserIDKey, userID)
	}
	return ctx
}

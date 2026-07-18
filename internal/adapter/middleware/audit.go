package middleware

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// Audit (Lago-parity C2) records every successful config-grade mutation
// (POST/PUT/PATCH/DELETE on an allowlisted resource) into the append-only
// audit_logs table: actor, route template, entity, status, and the request
// payload (truncated). One middleware instead of instrumenting every
// service keeps coverage uniform — a new route under an audited prefix is
// audited from day one.

// auditedResources are the first path segments after /v1 that constitute
// tenant configuration. Money-movement and high-volume ingest endpoints
// (payments, usage events, checkout) are deliberately absent: they have
// their own durable records (invoices, events, ledger) and would flood the
// trail.
var auditedResources = map[string]bool{
	"plans":             true,
	"billable-metrics":  true,
	"coupons":           true,
	"webhooks":          true,
	"wallets":           true,
	"usage-alerts":      true,
	"dunning-campaigns": true,
	"cancel-flows":      true,
	"settings":          true,
	"team":              true,
	"developer":         true,
	"accounting":        true,
	"mandates":          true,
	"quotes":            true,
	"credit-notes":      true,
}

// maxAuditBodyLen caps the captured request payload.
const maxAuditBodyLen = 4096

// Audit returns the capture middleware. repo may not be nil.
func Audit(repo port.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
			c.Next()
			return
		}
		entityType, ok := auditedEntity(c.FullPath())
		if !ok {
			// FullPath is empty until routing; fall back to the raw path's
			// segment for the allowlist decision.
			entityType, ok = auditedEntity(c.Request.URL.Path)
		}
		if !ok {
			c.Next()
			return
		}

		// Capture the body (bounded) and restore it for the handler.
		var body string
		if c.Request.Body != nil {
			raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxAuditBodyLen+1))
			if err == nil {
				rest, _ := io.ReadAll(c.Request.Body)
				c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), bytes.NewReader(rest)))
				if len(raw) > maxAuditBodyLen {
					raw = append(raw[:maxAuditBodyLen], []byte("...")...)
				}
				body = string(raw)
			}
		}

		c.Next()

		status := c.Writer.Status()
		if status < 200 || status >= 300 {
			return // only successful mutations are config changes
		}
		tenantID, ok := c.Get("tenant_id")
		if !ok {
			return
		}
		tid, ok := tenantID.(uuid.UUID)
		if !ok {
			return
		}

		actor := "api_key"
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(uuid.UUID); ok {
				actor = uid.String()
			}
		}

		action := method + " " + c.FullPath()
		entry := &domain.AuditLog{
			ID:          uuid.New(),
			TenantID:    tid,
			Actor:       actor,
			Action:      action,
			EntityType:  entityType,
			EntityID:    c.Param("id"),
			Status:      status,
			RequestBody: body,
			IP:          c.ClientIP(),
			CreatedAt:   time.Now().UTC(),
		}
		// Detached context: the write must survive the request context being
		// canceled after the response, and must never block the response.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := repo.Insert(ctx, entry); err != nil {
				slog.Error("audit log write failed", "action", entry.Action, "error", err)
			}
		}()
	}
}

// auditedEntity extracts the resource segment after /v1 and reports
// whether it is allowlisted.
func auditedEntity(path string) (string, bool) {
	rest, ok := strings.CutPrefix(path, "/v1/")
	if !ok {
		return "", false
	}
	seg := rest
	if i := strings.IndexByte(seg, '/'); i >= 0 {
		seg = seg[:i]
	}
	return seg, auditedResources[seg]
}

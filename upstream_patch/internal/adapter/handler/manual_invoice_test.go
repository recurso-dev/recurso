package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// The manual-invoice endpoint books revenue and AR, so a `member` must be turned
// away before any service call (nil service is therefore safe), and malformed
// requests must be rejected as 400 before reaching the service.
func TestManualInvoice_RoleAndValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewManualInvoiceHandler(nil)

	c, w := jsonCtx(http.MethodPost, "/v1/invoices/manual", `{}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "member")
	h.CreateManualInvoice(c)
	if w.Code != http.StatusForbidden {
		t.Errorf("member: status = %d, want 403", w.Code)
	}

	bad := []struct{ name, body string }{
		{"empty body", `{}`},
		{"no lines", `{"customer_id":"` + uuid.NewString() + `","currency":"USD","line_items":[]}`},
		{"bad customer id", `{"customer_id":"nope","currency":"USD","line_items":[{"description":"d","amount":1}]}`},
		{"bad entity id", `{"customer_id":"` + uuid.NewString() + `","entity_id":"nope","currency":"USD","line_items":[{"description":"d","amount":1}]}`},
		{"bad invoice date", `{"customer_id":"` + uuid.NewString() + `","currency":"USD","invoice_date":"01/08/26","line_items":[{"description":"d","amount":1}]}`},
		{"bad due date", `{"customer_id":"` + uuid.NewString() + `","currency":"USD","due_date":"soon","line_items":[{"description":"d","amount":1}]}`},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			c, w := jsonCtx(http.MethodPost, "/v1/invoices/manual", tc.body)
			c.Set("tenant_id", uuid.New())
			c.Set("user_role", "admin")
			h.CreateManualInvoice(c)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestParseManualDate(t *testing.T) {
	d, err := parseManualDate("invoice_date", "2026-08-01")
	if err != nil || d.Format("2006-01-02T15:04:05Z07:00") != "2026-08-01T00:00:00Z" {
		t.Errorf("date-only = %v err=%v, want 2026-08-01T00:00:00Z", d, err)
	}
	d, err = parseManualDate("due_date", "2026-08-31T10:30:00+05:30")
	if err != nil || d.Format("2006-01-02T15:04:05Z07:00") != "2026-08-31T05:00:00Z" {
		t.Errorf("RFC3339 = %v err=%v, want UTC-normalised", d, err)
	}
	if _, err = parseManualDate("invoice_date", "01/08/26"); err == nil {
		t.Error("dd/mm/yy must be rejected (ambiguous)")
	}
}

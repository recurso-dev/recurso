package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

func voidInvoice() *domain.Invoice {
	return &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), Status: domain.InvoiceStatusVoid, Currency: "USD", Total: 5000,
	}
}

func TestVoidInvoice_CreatesNoPaymentOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("checkout", func(t *testing.T) {
		inv := voidInvoice()
		gw := &orderRecordingGateway{}
		h := NewCheckoutHandler(&checkoutInvoiceRepo{inv: inv}, gw, nil, nil, "")
		r := gin.New()
		r.POST("/checkout/:id/pay", h.InitiatePayment)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/checkout/"+inv.ID.String()+"/pay", nil))
		if w.Code != http.StatusConflict || len(gw.amounts) != 0 {
			t.Errorf("status = %d, orders = %v, want 409 and none: %s", w.Code, gw.amounts, w.Body.String())
		}
	})

	t.Run("payment order", func(t *testing.T) {
		inv := voidInvoice()
		gw := &orderRecordingGateway{}
		h := NewPaymentHandler(gw, &checkoutInvoiceRepo{inv: inv})
		r := gin.New()
		r.POST("/orders", h.CreateOrder)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"invoice_id":"`+inv.ID.String()+`"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict || len(gw.amounts) != 0 {
			t.Errorf("status = %d, orders = %v, want 409 and none: %s", w.Code, gw.amounts, w.Body.String())
		}
	})
}

type voidCheckoutRepo struct {
	checkoutInvoiceRepo
	recorded []string
}

func (r *voidCheckoutRepo) SetGatewayPaymentID(_ context.Context, _, _ uuid.UUID, id string) error {
	r.recorded = append(r.recorded, id)
	return nil
}

type succeededInspector struct{ invoiceID string }

func (i succeededInspector) GetPaymentStatus(context.Context, string) (*port.PaymentStatus, error) {
	return &port.PaymentStatus{Status: "succeeded", InvoiceID: i.invoiceID, PaymentID: "pi_late"}, nil
}

type noTransitionSettler struct{ calls int }

func (s *noTransitionSettler) MarkInvoicePaid(context.Context, uuid.UUID) (bool, error) {
	s.calls++
	return false, nil
}

type genuineRazorpay struct{ invoiceID string }

func (genuineRazorpay) VerifyPayment(context.Context, string, string, string) error { return nil }

func (r genuineRazorpay) GetOrderInvoiceID(context.Context, string) (string, error) {
	return r.invoiceID, nil
}

func TestVoidInvoice_RazorpayVerifyReportsVoid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inv := voidInvoice()
	repo := &voidCheckoutRepo{checkoutInvoiceRepo: checkoutInvoiceRepo{inv: inv}}
	settler := &noTransitionSettler{}
	h := NewCheckoutHandler(repo, &orderRecordingGateway{}, nil, settler, "")
	h.SetRazorpay(genuineRazorpay{inv.ID.String()}, "rzp_test")
	r := gin.New()
	r.POST("/checkout/:id/razorpay/verify", h.RazorpayVerify)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/checkout/"+inv.ID.String()+"/razorpay/verify",
		strings.NewReader(`{"razorpay_order_id":"o","razorpay_payment_id":"pay_late","razorpay_signature":"s"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"void"`) {
		t.Errorf("status = %d, body = %s, want a void status", w.Code, w.Body.String())
	}
	if settler.calls != 1 || len(repo.recorded) != 1 || repo.recorded[0] != "pay_late" {
		t.Errorf("settler calls = %d, recorded = %v, want 1 and [pay_late]", settler.calls, repo.recorded)
	}
}

// A payment already in flight when the invoice was voided still reaches the
// success page: the buyer must not be told it paid the invoice, and the gateway
// payment id must survive so the money can be refunded.
func TestVoidInvoice_SuccessPageReportsVoid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inv := voidInvoice()
	repo := &voidCheckoutRepo{checkoutInvoiceRepo: checkoutInvoiceRepo{inv: inv}}
	settler := &noTransitionSettler{}
	h := NewCheckoutHandler(repo, &orderRecordingGateway{}, succeededInspector{inv.ID.String()}, settler, "")
	r := gin.New()
	r.GET("/checkout/:id/success", h.CheckoutSuccess)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/checkout/"+inv.ID.String()+"/success?payment_intent=pi_late", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"void"`) {
		t.Errorf("status = %d, body = %s, want a void status", w.Code, w.Body.String())
	}
	if settler.calls != 1 {
		t.Errorf("settler calls = %d, want 1 so the payment gets flagged", settler.calls)
	}
	if len(repo.recorded) != 1 || repo.recorded[0] != "pi_late" {
		t.Errorf("recorded payment ids = %v, want [pi_late]", repo.recorded)
	}
}

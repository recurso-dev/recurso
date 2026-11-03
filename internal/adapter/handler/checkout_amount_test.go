package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/core/port/porttest"
)

type checkoutInvoiceRepo struct {
	porttest.UnimplementedInvoiceRepository
	inv *domain.Invoice
}

func (r *checkoutInvoiceRepo) GetByIDPublic(context.Context, uuid.UUID) (*domain.Invoice, error) {
	return r.inv, nil
}

type orderRecordingGateway struct {
	port.PaymentGateway
	amounts []int64
}

func (g *orderRecordingGateway) CreateOrder(_ context.Context, amount int64, currency, _, _ string) (*port.PaymentOrder, error) {
	g.amounts = append(g.amounts, amount)
	return &port.PaymentOrder{ID: "order_1", Amount: amount, Currency: currency, Gateway: "mock"}, nil
}

func TestCheckout_CollectsOnlyTheAmountDue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tc := range []struct {
		name               string
		amountPaid, credit int64
		wantStatus         int
		wantCharged        []int64
		wantDue            int64
	}{
		{"nothing applied charges the total", 0, 0, http.StatusOK, []int64{5000}, 5000},
		{"account credit is not charged again", 0, 2000, http.StatusOK, []int64{3000}, 3000},
		{"a wallet drain is not charged again", 1000, 0, http.StatusOK, []int64{4000}, 4000},
		{"both applied", 1000, 2000, http.StatusOK, []int64{2000}, 2000},
		{"fully covered creates no order", 1000, 4000, http.StatusBadRequest, nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inv := &domain.Invoice{
				ID: uuid.New(), TenantID: uuid.New(), Status: domain.InvoiceStatusOpen, Currency: "USD",
				Total: 5000, AmountPaid: tc.amountPaid, CreditApplied: tc.credit,
			}
			gw := &orderRecordingGateway{}
			h := NewCheckoutHandler(&checkoutInvoiceRepo{inv: inv}, gw, nil, nil, "")
			r := gin.New()
			r.GET("/checkout/:id", h.ShowCheckout)
			r.POST("/checkout/:id/pay", h.InitiatePayment)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/checkout/"+inv.ID.String()+"/pay", nil))
			if w.Code != tc.wantStatus {
				t.Fatalf("pay status = %d, want %d: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if len(gw.amounts) != len(tc.wantCharged) || (len(gw.amounts) == 1 && gw.amounts[0] != tc.wantCharged[0]) {
				t.Errorf("order amounts = %v, want %v", gw.amounts, tc.wantCharged)
			}

			w = httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/checkout/"+inv.ID.String(), nil))
			var shown struct {
				Data struct {
					AmountDue int64 `json:"amount_due"`
				} `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &shown); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if tc.wantStatus == http.StatusOK && shown.Data.AmountDue != tc.wantDue {
				t.Errorf("page amount_due = %d, want %d", shown.Data.AmountDue, tc.wantDue)
			}
		})
	}
}

func TestPaymentOrder_CollectsOnlyTheAmountDue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inv := &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), Status: domain.InvoiceStatusOpen, Currency: "USD",
		Total: 5000, CreditApplied: 2000,
	}
	gw := &orderRecordingGateway{}
	h := NewPaymentHandler(gw, &checkoutInvoiceRepo{inv: inv})
	r := gin.New()
	r.POST("/orders", h.CreateOrder)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"invoice_id":"`+inv.ID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if len(gw.amounts) != 1 || gw.amounts[0] != 3000 {
		t.Errorf("order amounts = %v, want [3000]: %s", gw.amounts, w.Body.String())
	}
}

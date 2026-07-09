package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// paymentInspector is the subset of a gateway that can read back a payment for
// server-side verification. Only the Stripe gateway implements it; when nil (or
// for gateways that don't), CheckoutSuccess falls back to reporting the
// invoice's current status without marking it paid.
type paymentInspector interface {
	GetPaymentStatus(ctx context.Context, orderID string) (*port.PaymentStatus, error)
}

// invoiceSettler marks an invoice paid through the full path (ledger posting,
// payment record) — the same method the payment webhook uses. It is idempotent:
// an already-paid invoice is a no-op, so CheckoutSuccess and the webhook can
// both call it without double-posting.
type invoiceSettler interface {
	MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) error
}

type CheckoutHandler struct {
	invoiceRepo    port.InvoiceRepository
	paymentGateway port.PaymentGateway
	inspector      paymentInspector
	settler        invoiceSettler
	publishableKey string
}

// NewCheckoutHandler wires the checkout. gw creates orders (currency-routed);
// inspector (the Stripe gateway, may be nil) verifies a PaymentIntent
// server-side before settling; settler marks the invoice paid via the ledger
// path; publishableKey is the Stripe publishable key handed to the browser to
// mount the Payment Element.
func NewCheckoutHandler(repo port.InvoiceRepository, gw port.PaymentGateway, inspector paymentInspector, settler invoiceSettler, publishableKey string) *CheckoutHandler {
	return &CheckoutHandler{
		invoiceRepo:    repo,
		paymentGateway: gw,
		inspector:      inspector,
		settler:        settler,
		publishableKey: publishableKey,
	}
}

// ShowCheckout returns invoice data as JSON for the frontend checkout page.
func (h *CheckoutHandler) ShowCheckout(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice ID")
		return
	}

	invoice, err := h.invoiceRepo.GetByIDPublic(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}
	if invoice == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":             invoice.ID,
			"invoice_number": invoice.InvoiceNumber,
			"status":         string(invoice.Status),
			"currency":       invoice.Currency,
			"subtotal":       invoice.Subtotal,
			"tax_amount":     invoice.TaxAmount,
			"total":          invoice.Total,
			"display_amount": fmt.Sprintf("%.2f", float64(invoice.Total)/100.0),
			"due_date":       invoice.DueDate.Format("2006-01-02"),
			"customer_id":    invoice.CustomerID,
		},
	})
}

// InitiatePayment creates a payment order via the gateway and returns the order details.
func (h *CheckoutHandler) InitiatePayment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice ID")
		return
	}

	invoice, err := h.invoiceRepo.GetByIDPublic(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}
	if invoice == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		respondError(c, http.StatusBadRequest, codeInvoiceAlreadyPaid, "invoice already paid")
		return
	}

	order, err := h.paymentGateway.CreateOrder(
		c.Request.Context(),
		invoice.Total,
		invoice.Currency,
		invoice.InvoiceNumber,
		invoice.ID.String(),
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to create payment order")
		return
	}

	// A client_secret means this order is a client-confirmed Stripe
	// PaymentIntent — the browser mounts the Payment Element with it. Without
	// one (e.g. Razorpay), the frontend uses that gateway's own flow.
	gatewayName := "other"
	if order.ClientSecret != "" {
		gatewayName = "stripe"
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"order_id":        order.ID,
			"amount":          order.Amount,
			"currency":        order.Currency,
			"invoice_id":      invoice.ID,
			"invoice_number":  invoice.InvoiceNumber,
			"gateway":         gatewayName,
			"client_secret":   order.ClientSecret,
			"publishable_key": h.publishableKey,
		},
	})
}

// CheckoutSuccess verifies a completed payment and reports the invoice's
// settlement status. It NEVER marks an invoice paid on trust: it only settles
// after confirming, directly with the gateway, that the given PaymentIntent
// (a) actually succeeded and (b) carries this invoice's id in its metadata —
// so a succeeded intent for one invoice can't be replayed to pay another. The
// webhook remains the authoritative backstop; both call the same idempotent
// settler, and an unverifiable or still-processing payment leaves the invoice
// untouched.
//
// The frontend passes the confirmed PaymentIntent id as ?payment_intent=pi_...
// (this is also the query param Stripe appends when it redirects to return_url).
func (h *CheckoutHandler) CheckoutSuccess(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice ID")
		return
	}

	ctx := c.Request.Context()
	invoice, err := h.invoiceRepo.GetByIDPublic(ctx, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}
	if invoice == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}

	// Already settled (e.g. the webhook won the race, or a redelivery) — report
	// it and stop. Idempotent.
	if invoice.Status == domain.InvoiceStatusPaid {
		respondCheckoutStatus(c, invoice, "paid")
		return
	}

	paymentIntentID := c.Query("payment_intent")

	// Without a gateway inspector or a payment id we cannot verify anything, so
	// we do NOT mark the invoice paid — we just report its current status. The
	// gateway webhook remains the settlement path in that case.
	if h.inspector == nil || paymentIntentID == "" {
		respondCheckoutStatus(c, invoice, string(invoice.Status))
		return
	}

	st, err := h.inspector.GetPaymentStatus(ctx, paymentIntentID)
	if err != nil {
		// Couldn't reach/verify the intent — never settle on failure to verify.
		respondCheckoutStatus(c, invoice, "processing")
		return
	}

	// Bind the payment to THIS invoice. A succeeded intent whose metadata points
	// at a different invoice must never settle this one.
	if st.InvoiceID != invoice.ID.String() {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "payment does not belong to this invoice")
		return
	}

	if st.Status != "succeeded" {
		// requires_action / processing (ACH settles over days) — not paid yet.
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status":         "processing",
				"payment_status": st.Status,
				"invoice_id":     invoice.ID,
				"invoice_number": invoice.InvoiceNumber,
			},
		})
		return
	}

	// Verified succeeded and bound to this invoice — settle through the ledger
	// path (idempotent with the webhook).
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, invoice.TenantID)
	if h.settler != nil {
		if err := h.settler.MarkInvoicePaid(tenantCtx, invoice.ID); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "failed to settle invoice")
			return
		}
	}
	if st.PaymentID != "" {
		// Record the gateway payment id refunds are issued against (best-effort;
		// the webhook also sets this).
		_ = h.invoiceRepo.SetGatewayPaymentID(ctx, invoice.ID, st.PaymentID)
	}

	respondCheckoutStatus(c, invoice, "paid")
}

// respondCheckoutStatus is the shared success-shape for the checkout status
// endpoint.
func respondCheckoutStatus(c *gin.Context, invoice *domain.Invoice, status string) {
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":         status,
			"invoice_id":     invoice.ID,
			"invoice_number": invoice.InvoiceNumber,
		},
	})
}

package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
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

// razorpayVerifier verifies a Razorpay checkout payment (HMAC signature) and
// resolves the order's invoice_id for binding. Satisfied by
// *gateway.RazorpayGateway; nil when Razorpay isn't configured, which disables
// the INR checkout-verify endpoint.
type razorpayVerifier interface {
	VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error
	GetOrderInvoiceID(ctx context.Context, orderID string) (string, error)
}

// checkoutCustomerReader loads the invoice's buyer on the public checkout
// route (no tenant in context, hence the Public variant).
type checkoutCustomerReader interface {
	GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
}

// checkoutBuyerSetter attaches buyer name/address to a created order.
// Implemented by *gateway.StripeGateway; India-region accounts require these
// on foreign-currency intents (india-exports rules).
type checkoutBuyerSetter interface {
	SetOrderBuyer(ctx context.Context, orderID, name, line1, city, state, zip, country string) error
}

type CheckoutHandler struct {
	invoiceRepo    port.InvoiceRepository
	paymentGateway port.PaymentGateway
	inspector      paymentInspector
	settler        invoiceSettler
	publishableKey string
	razorpay       razorpayVerifier
	razorpayKeyID  string
	customers      checkoutCustomerReader
	buyerSetter    checkoutBuyerSetter
}

// SetBuyerDetails wires buyer name/address propagation onto Stripe orders.
func (h *CheckoutHandler) SetBuyerDetails(customers checkoutCustomerReader, setter checkoutBuyerSetter) {
	h.customers = customers
	h.buyerSetter = setter
}

// SetRazorpay wires the INR/Razorpay checkout verification path (order created
// via CreateOrder -> Razorpay Checkout.js -> this verify). keyID is the public
// Razorpay key id handed to the browser.
func (h *CheckoutHandler) SetRazorpay(v razorpayVerifier, keyID string) {
	h.razorpay = v
	h.razorpayKeyID = keyID
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

	// The gateway self-identifies on the order so the frontend picks the right
	// flow. The ID-prefix inference remains only as a fallback for gateway
	// implementations that predate the field — it must never be primary, since
	// mock orders share Razorpay's "order_" prefix.
	gatewayName := order.Gateway
	if gatewayName == "" {
		gatewayName = "other"
		switch {
		case order.ClientSecret != "":
			gatewayName = "stripe"
		case strings.HasPrefix(order.ID, "order_"):
			gatewayName = "razorpay"
		}
	}

	// A gateway the frontend would drive with a missing browser-side key is a
	// dead end (silent PaymentIntent churn on Stripe, a throwing Razorpay
	// modal) — fail loudly so the misconfiguration is fixable, not invisible.
	if gatewayName == "stripe" && h.publishableKey == "" {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "checkout is not fully configured (missing Stripe publishable key)")
		return
	}
	if gatewayName == "razorpay" && h.razorpayKeyID == "" {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "checkout is not fully configured (missing Razorpay key id)")
		return
	}

	// India-region Stripe accounts reject foreign-currency confirmation unless
	// the intent carries the buyer's name and address (india-exports rules) —
	// attach them from the invoice's customer. Best-effort: a failure here
	// must not block checkout on accounts that don't require it.
	if gatewayName == "stripe" && h.buyerSetter != nil && h.customers != nil {
		if cust, cerr := h.customers.GetByIDPublic(c.Request.Context(), invoice.CustomerID); cerr == nil && cust != nil && cust.Name != nil && *cust.Name != "" {
			ba := cust.BillingAddress
			_ = h.buyerSetter.SetOrderBuyer(c.Request.Context(), order.ID, *cust.Name, ba.Line1, ba.City, ba.State, ba.Zip, ba.Country)
		}
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
			"razorpay_key_id": h.razorpayKeyID,
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
		// requires_payment_method / canceled mean the attempt failed (declined
		// or abandoned) — report that so the buyer can retry, instead of a
		// "processing" screen that tells them no action is needed. Everything
		// else (processing, requires_action; ACH settles over days) is genuinely
		// in flight.
		checkoutStatus := "processing"
		if st.Status == "requires_payment_method" || st.Status == "canceled" {
			checkoutStatus = "failed"
		}
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status":         checkoutStatus,
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

// RazorpayVerify settles an invoice after a Razorpay Checkout payment. Like the
// Stripe path, it never settles on trust: it verifies the HMAC signature the
// browser returns, then confirms — by fetching the order — that the order's
// notes.invoice_id matches THIS invoice, before marking it paid via the
// idempotent ledger settler. The webhook (payment.captured) remains the
// authoritative backstop.
func (h *CheckoutHandler) RazorpayVerify(c *gin.Context) {
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
	if invoice.Status == domain.InvoiceStatusPaid {
		respondCheckoutStatus(c, invoice, "paid")
		return
	}
	if h.razorpay == nil {
		respondError(c, http.StatusServiceUnavailable, codeInternalError, "razorpay checkout isn't available on this deployment")
		return
	}

	var req struct {
		OrderID   string `json:"razorpay_order_id" binding:"required"`
		PaymentID string `json:"razorpay_payment_id" binding:"required"`
		Signature string `json:"razorpay_signature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "razorpay_order_id, razorpay_payment_id and razorpay_signature are required")
		return
	}

	// 1. Signature — proves this is a genuine Razorpay payment for the order.
	if err := h.razorpay.VerifyPayment(ctx, req.OrderID, req.PaymentID, req.Signature); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "payment verification failed")
		return
	}

	// 2. Bind — the order must have been created for THIS invoice, so a genuine
	// payment for a different invoice can't be replayed here.
	orderInvoiceID, err := h.razorpay.GetOrderInvoiceID(ctx, req.OrderID)
	if err != nil {
		respondError(c, http.StatusBadGateway, codeInternalError, "could not confirm the payment")
		return
	}
	if orderInvoiceID != invoice.ID.String() {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "payment does not belong to this invoice")
		return
	}

	// 3. Settle through the ledger path (idempotent with the webhook).
	tenantCtx := context.WithValue(ctx, domain.TenantIDKey, invoice.TenantID)
	if h.settler != nil {
		if err := h.settler.MarkInvoicePaid(tenantCtx, invoice.ID); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "failed to settle invoice")
			return
		}
	}
	if req.PaymentID != "" {
		_ = h.invoiceRepo.SetGatewayPaymentID(ctx, invoice.ID, req.PaymentID)
	}

	respondCheckoutStatus(c, invoice, "paid")
}

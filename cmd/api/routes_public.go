package main

import (
	"crypto/subtle"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/handler"
	"github.com/recurso-dev/recurso/internal/adapter/httperr"
	"github.com/recurso-dev/recurso/internal/adapter/metrics"
	"github.com/recurso-dev/recurso/internal/service"
	"github.com/redis/go-redis/v9"
)

// publicHandlers carries every handler, middleware, repository and setting the
// unauthenticated route table needs: operational endpoints (/metrics,
// /platform/metrics, /health, /version, the OpenAPI spec), hosted checkout,
// the waitlist, inbound gateway webhooks and the accounting OAuth callback.
// main() wires dependencies and builds this once; the table itself lives here
// so main.go stays a wiring file. cmd/api/openapi_drift_test.go scans this
// file, so a dropped or renamed line fails CI.
type publicHandlers struct {
	accountingHandler  *handler.AccountingHandler
	checkoutHandler    *handler.CheckoutHandler
	database           *sql.DB
	founderToken       string
	gatewayMode        string
	httpMetrics        *metrics.HTTPMetrics
	metricsToken       string
	paymentHandler     *handler.PaymentHandler
	platformChargeRepo *db.CloudChargeRepository
	platformRepo       *db.PlatformRepository
	publicLimit        gin.HandlerFunc
	rdb                *redis.Client
	tbConnected        bool
	waitlistHandler    *handler.WaitlistHandler
	webhookHandler     *handler.WebhookHandler
}

// registerPublicRoutes mounts every route that needs neither a dashboard
// session nor a tenant API key. Brute-forceable endpoints (checkout, payment
// initiation, waitlist, the OAuth callback) carry h.publicLimit; webhooks and
// operational endpoints run on the global limiter only.
func registerPublicRoutes(r *gin.Engine, h *publicHandlers) {
	// Prometheus scrape endpoint. Optionally bearer-gated via METRICS_TOKEN so it
	// can be exposed without leaking internal metrics; open when unset (scrape
	// from a trusted network).
	r.GET("/metrics", func(c *gin.Context) {
		if h.metricsToken != "" && subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte("Bearer "+h.metricsToken)) != 1 {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		h.httpMetrics.WriteProm(c.Writer)
	})

	// Founder-only platform metrics: a cross-tenant funnel snapshot (signups,
	// activation, trials, plan/billing breakdown). This is the ONLY cross-tenant
	// surface, kept deliberately outside tenant auth — gated by FOUNDER_TOKEN and
	// returning 404 (feature off) when unset, so no tenant login can reach it.
	r.GET("/platform/metrics", func(c *gin.Context) {
		if h.founderToken == "" {
			c.Status(http.StatusNotFound)
			return
		}
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte("Bearer "+h.founderToken)) != 1 {
			c.Status(http.StatusUnauthorized)
			return
		}
		pm, err := h.platformRepo.PlatformMetrics(c.Request.Context())
		if err != nil {
			slog.Error("platform metrics query failed", "error", err)
			httperr.Respond(c, http.StatusInternalServerError, httperr.CodeInternalError, "failed to compute platform metrics")
			return
		}
		// Attach the money-free Recurso Cloud charge dry-run for the current
		// month (what each tenant WOULD be charged). Best-effort: a failure here
		// leaves the funnel metrics intact rather than failing the whole view.
		periodStart, _ := service.MonthBounds(time.Now().UTC())
		if charges, err := h.platformChargeRepo.ListPreviewsWithTenant(c.Request.Context(), periodStart); err != nil {
			slog.Warn("platform metrics: cloud charge preview unavailable", "error", err)
		} else {
			pm.CloudCharges = charges
			pm.CloudChargeCurrency = getEnvDefault("REPORTING_CURRENCY", "USD")
			for _, ch := range charges {
				pm.CloudChargeTotalMinor += ch.WouldChargeMinor
			}
		}
		c.JSON(http.StatusOK, pm)
	})

	r.GET("/health", func(c *gin.Context) {
		status := "ok"
		httpStatus := http.StatusOK
		components := gin.H{}

		// Check Postgres. Never return the raw error — /health is public and
		// unauthenticated, and a connection error can leak the host/port (and
		// sometimes credentials) from the DSN. Log it server-side; expose only
		// the component status.
		if err := h.database.Ping(); err != nil { //nolint:noctx // moved verbatim from main.go; switching to PingContext is a behaviour change for a follow-up
			slog.Error("health check: postgres ping failed", "error", err)
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
			components["postgres"] = gin.H{"status": "down"}
		} else {
			components["postgres"] = gin.H{"status": "up"}
		}

		// Check Redis (same info-disclosure stance as Postgres above).
		if h.rdb != nil {
			if err := h.rdb.Ping(c.Request.Context()).Err(); err != nil {
				slog.Error("health check: redis ping failed", "error", err)
				components["redis"] = gin.H{"status": "down"}
				// Redis down is degraded, not critical
				if status == "ok" {
					status = "degraded"
				}
			} else {
				components["redis"] = gin.H{"status": "up"}
			}
		}

		// Check TigerBeetle. Report the ACTUAL boot-time connection state
		// (tbConnected), not ledgerService != nil — the ledger service is always
		// constructed (PG-only mode passes a nil TB client), so the latter would
		// always say "connected" and mask a real TigerBeetle outage.
		if h.tbConnected {
			components["tigerbeetle"] = gin.H{"status": "connected"}
		} else {
			components["tigerbeetle"] = gin.H{"status": "disconnected"}
		}

		c.JSON(httpStatus, gin.H{
			"status":     status,
			"version":    version,
			"components": components,
		})
	})

	// gateway_mode drives the dashboard's "Test mode" chip (see main() for how
	// it is derived from the configured gateway keys).
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": version, "gateway_mode": h.gatewayMode})
	})

	// OpenAPI specification (public): GET /openapi.yaml, GET /openapi.json
	if err := registerOpenAPIRoutes(r); err != nil {
		log.Fatalf("Failed to register OpenAPI routes: %v", err)
	}

	r.GET("/checkout/:id", h.publicLimit, h.checkoutHandler.ShowCheckout)
	r.POST("/checkout/:id/pay", h.publicLimit, h.checkoutHandler.InitiatePayment)
	r.GET("/checkout/:id/success", h.publicLimit, h.checkoutHandler.CheckoutSuccess)
	r.POST("/checkout/:id/razorpay/verify", h.publicLimit, h.checkoutHandler.RazorpayVerify)
	r.POST("/payments/order", h.publicLimit, h.paymentHandler.CreateOrder)
	// Recurso Cloud waitlist (ENG-12): public demand capture from the website.
	r.POST("/waitlist", h.publicLimit, h.waitlistHandler.Join)
	r.POST("/webhooks/razorpay", h.webhookHandler.HandleRazorpay) // Webhooks need higher limits
	r.POST("/webhooks/stripe", h.webhookHandler.HandleStripe)
	// BYO increment 3: per-connection webhook endpoints. Each tenant's gateway
	// account posts to its own URL, so we verify with that connection's own
	// signing secret (looked up by :connID) before trusting the payload.
	r.POST("/webhooks/razorpay/:connID", h.webhookHandler.HandleRazorpay)
	r.POST("/webhooks/stripe/:connID", h.webhookHandler.HandleStripe)
	r.POST("/webhooks/gocardless", h.webhookHandler.HandleGoCardless)
	r.POST("/webhooks/gocardless/:connID", h.webhookHandler.HandleGoCardless)

	// OAuth callbacks arrive as bare browser redirects from the provider (no
	// session cookie, no API key) — authentication is the HMAC-signed state
	// the handler verifies. Locally this was a hard 401; in production it only
	// worked when the operator's dashboard cookie happened to ride along.
	r.GET("/v1/accounting/callback/:provider", h.publicLimit, h.accountingHandler.OAuthCallback)
}

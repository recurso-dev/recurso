package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/handler"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
	"github.com/redis/go-redis/v9"
)

// v1Handlers carries every handler, middleware and repository the /v1 route
// table needs. main() wires dependencies and builds this once; the table itself
// lives here so main.go stays a wiring file and the route surface is readable
// in one place. cmd/api/openapi_drift_test.go asserts the registered set
// matches cmd/api/openapi.yaml, so a dropped or renamed line fails CI.
type v1Handlers struct {
	accountingHandler       *handler.AccountingHandler
	advancedBillingHandler  *handler.AdvancedBillingHandler
	analyticsHandler        *handler.AnalyticsHandler
	auditHandler            *handler.AuditHandler
	auditLogRepo            port.AuditLogRepository
	authHandler             *handler.AuthHandler
	authService             *service.AuthService
	billingHandler          *handler.BillingHandler
	cancelFlowHandler       *handler.CancelFlowHandler
	cancellationHandler     *handler.CancellationHandler
	catalogHandler          *handler.CatalogHandler
	chargebeeImportHandler  *handler.ChargebeeImportHandler
	churnHandler            *handler.ChurnHandler
	closePackHandler        *handler.ClosePackHandler
	collectionsHandler      *handler.CollectionsHandler
	compareReportHandler    *handler.CompareReportHandler
	consentHandler          *handler.ConsentHandler
	couponHandler           *handler.CouponHandler
	creditNoteHandler       *handler.CreditNoteHandler
	crmSyncHandler          *handler.CRMSyncHandler
	customerHandler         *handler.CustomerHandler
	disputeHandler          *handler.DisputeHandler
	dunningCampaignHandler  *handler.DunningCampaignHandler
	dunningHandler          *handler.DunningHandler
	einvoiceHandler         *handler.EInvoiceHandler
	entitlementHandler      *handler.EntitlementHandler
	entityHandler           *handler.EntityHandler
	euConfigHandler         *handler.EUConfigHandler
	euEInvoiceHandler       *handler.EUEInvoiceHandler
	expensiveLimit          gin.HandlerFunc
	gatewayConnHandler      *handler.GatewayConnectionHandler
	giftHandler             *handler.GiftHandler
	gstHandler              *handler.GSTHandler
	idempotencyStore        port.IdempotencyStore
	integrationConnHandler  *handler.IntegrationConnectionHandler
	invoiceBrandingHandler  *handler.InvoiceBrandingHandler
	ledgerHandler           *handler.LedgerHandler
	mandateHandler          *handler.MandateHandler
	mcpSettingsHandler      *handler.MCPSettingsHandler
	meteringHandler         *handler.MeteringHandler
	offlinePaymentHandler   *handler.OfflinePaymentHandler
	orgHandler              *handler.OrganizationHandler
	pdfHandler              *handler.InvoicePDFHandler
	quoteHandler            *handler.QuoteHandler
	rdb                     *redis.Client
	reconciliationHandler   *handler.ReconciliationHandler
	referralHandler         *handler.ReferralHandler
	revenuecatImportHandler *handler.RevenueCatImportHandler
	revrecHandler           *handler.RevRecHandler
	serverLive              bool
	ssoHandler              *handler.SSOHandler
	stripeImportHandler     *handler.StripeImportHandler
	subscriptionHandler     *handler.SubscriptionHandler
	taxNexusHandler         *handler.TaxNexusHandler
	teamHandler             *handler.TeamHandler
	tenantHandler           *handler.TenantHandler
	tenantRepo              *db.TenantRepository
	usTaxConfigHandler      *handler.USTaxConfigHandler
	usageAlertHandler       *handler.UsageAlertHandler
	usageHandler            *handler.UsageHandler
	walletHandler           *handler.WalletHandler
	webhookMgmtHandler      *handler.WebhookManagementHandler
}

// registerV1Routes mounts every API-key / session authenticated /v1 route.
func registerV1Routes(r *gin.Engine, h *v1Handlers) {
	v1 := r.Group("/v1")
	v1.Use(middleware.SessionOrAPIKeyMiddleware(h.tenantRepo, h.authService, h.serverLive))
	v1.Use(middleware.IdempotencyMiddleware(h.idempotencyStore)) // P30: Idempotency
	v1.Use(middleware.Audit(h.auditLogRepo))                     // C2: append-only config audit trail
	{
		v1.POST("/plans", h.catalogHandler.CreatePlan)
		v1.GET("/plans", h.catalogHandler.ListPlans)
		v1.GET("/plans/:id", h.catalogHandler.GetPlan)
		v1.PUT("/plans/:id", h.catalogHandler.UpdatePlan)

		// Entitlement Engine v1
		v1.PUT("/plans/:id/entitlements", h.entitlementHandler.SetPlanEntitlements)
		v1.GET("/plans/:id/entitlements", h.entitlementHandler.GetPlanEntitlements)
		v1.GET("/customers/:id/entitlements", h.entitlementHandler.GetCustomerEntitlements)
		v1.GET("/entitlements/check", h.entitlementHandler.CheckEntitlement)

		v1.POST("/customers", h.customerHandler.CreateCustomer)
		v1.GET("/customers", h.customerHandler.ListCustomers)

		// Migration: dry-run preview (no writes) then idempotent commit.
		v1.POST("/import/stripe/preview", h.expensiveLimit, h.stripeImportHandler.Preview)
		v1.POST("/import/stripe/commit", h.expensiveLimit, h.stripeImportHandler.Commit)
		v1.POST("/import/stripe/compare", h.expensiveLimit, h.stripeImportHandler.Compare)
		// Chargebee migration: dry-run preview then idempotent commit.
		v1.POST("/import/chargebee/preview", h.expensiveLimit, h.chargebeeImportHandler.Preview)
		v1.POST("/import/chargebee/commit", h.expensiveLimit, h.chargebeeImportHandler.Commit)
		v1.POST("/import/chargebee/compare", h.expensiveLimit, h.chargebeeImportHandler.Compare)
		// RevenueCat migration: dry-run preview then idempotent commit.
		v1.POST("/import/revenuecat/preview", h.expensiveLimit, h.revenuecatImportHandler.Preview)
		v1.POST("/import/revenuecat/commit", h.expensiveLimit, h.revenuecatImportHandler.Commit)
		v1.POST("/import/revenuecat/compare", h.expensiveLimit, h.revenuecatImportHandler.Compare)

		v1.GET("/import/compare-reports", h.compareReportHandler.List)
		v1.GET("/import/compare-reports/:id", h.compareReportHandler.Get)
		v1.GET("/import/compare-reports/:id/document", h.expensiveLimit, h.compareReportHandler.Document)
		v1.GET("/customers/:id", h.customerHandler.GetCustomer)
		v1.PUT("/customers/:id", h.customerHandler.UpdateCustomer)
		v1.PUT("/customers/:id/payment-method", h.customerHandler.UpdatePaymentMethod)
		// Ledger-backed credits: a customer's consolidated account-credit statement.
		v1.GET("/customers/:id/credit-statement", h.creditNoteHandler.GetCreditStatement)
		v1.GET("/customers/:id/financial-summary", h.customerHandler.GetFinancialSummary)

		v1.POST("/subscriptions", h.subscriptionHandler.CreateSubscription)
		v1.PUT("/subscriptions/:id", h.subscriptionHandler.UpdateSubscription)
		v1.GET("/subscriptions/:id/preview-change", h.subscriptionHandler.PreviewPlanChange)
		// Multi-product catalog v1: subscription add-ons
		v1.POST("/subscriptions/:id/addons", h.subscriptionHandler.AddAddon)
		v1.GET("/subscriptions/:id/addons", h.subscriptionHandler.ListAddons)
		v1.DELETE("/subscriptions/:id/addons/:addonId", h.subscriptionHandler.RemoveAddon)
		v1.GET("/subscriptions", h.subscriptionHandler.ListSubscriptions)
		v1.GET("/subscriptions/:id", h.subscriptionHandler.GetSubscription)
		v1.GET("/subscriptions/:id/history", h.subscriptionHandler.GetSubscriptionHistory)
		// Financial position (MRR, recurring value, next invoice, outstanding).
		v1.GET("/subscriptions/:id/financial-summary", h.subscriptionHandler.GetSubscriptionFinancialSummary)
		v1.GET("/invoices", h.subscriptionHandler.ListInvoices)
		v1.GET("/invoices/:id", h.subscriptionHandler.GetInvoice)
		v1.GET("/invoices/:id/journal-entries", h.subscriptionHandler.GetInvoiceJournalEntries)
		v1.GET("/invoices/:id/payment-attempts", h.subscriptionHandler.GetInvoicePaymentAttempts)
		v1.GET("/invoices/:id/status-history", h.subscriptionHandler.GetInvoiceStatusHistory)
		v1.GET("/payment-attempts", h.subscriptionHandler.ListPaymentAttempts)
		// A single payment attempt as an addressable object, resolved with its
		// invoice/customer/subscription context. Read-only.
		v1.GET("/payment-attempts/:id", h.subscriptionHandler.GetPaymentAttempt)
		// Invoice PDF is tenant-scoped: it renders the buyer's legal name,
		// address, and GSTIN, so it must never be publicly fetchable by UUID.
		v1.GET("/invoices/:id/pdf", h.expensiveLimit, h.pdfHandler.DownloadPDF)
		v1.GET("/invoices/:id/preview", h.expensiveLimit, h.pdfHandler.PreviewHTML)
		v1.POST("/invoices/:id/send", h.advancedBillingHandler.SendInvoice) // email the invoice + Pay Now link

		// Usage Platform v1
		v1.POST("/usage/events", h.usageHandler.RecordEvent)
		v1.POST("/usage/events/batch", h.usageHandler.RecordEventsBatch)        // <=500 events, per-item results (C1)
		v1.GET("/usage", h.usageHandler.QueryUsage)                             // time-windowed buckets
		v1.GET("/usage/dimensions", h.usageHandler.ListDimensions)              // dimension catalog
		v1.GET("/usage/events", h.usageHandler.ListRecentEvents)                // raw event stream (debugging)
		v1.GET("/subscriptions/:id/usage", h.usageHandler.GetSubscriptionUsage) // current period + lifetime

		// Usage-based billing v1 (spec_usage_billing.md)
		v1.POST("/billable-metrics", h.meteringHandler.CreateMetric)
		v1.GET("/billable-metrics", h.meteringHandler.ListMetrics)
		v1.GET("/billable-metrics/:id", h.meteringHandler.GetMetric)
		v1.GET("/billable-metrics/:id/charges", h.meteringHandler.GetMetricCharges)
		v1.PUT("/billable-metrics/:id", h.meteringHandler.UpdateMetric)
		v1.DELETE("/billable-metrics/:id", h.meteringHandler.DeleteMetric)
		v1.PUT("/plans/:id/charges", h.meteringHandler.SetPlanCharges)
		v1.GET("/plans/:id/charges", h.meteringHandler.GetPlanCharges)
		v1.POST("/plans/:id/simulate-charges", h.meteringHandler.SimulateCharges)   // A1.6 read-only pricing simulator
		v1.GET("/subscriptions/:id/usage-amount", h.meteringHandler.GetUsageAmount) // live pre-invoice preview

		v1.PUT("/subscriptions/:id/commitment", h.subscriptionHandler.SetCommitment) // minimum commitment (B2)

		// Prepaid wallets (Lago-parity B1)
		v1.POST("/wallets", h.walletHandler.Create)
		v1.GET("/wallets", h.walletHandler.List)
		v1.GET("/wallets/:id", h.walletHandler.Get)
		v1.POST("/wallets/:id/top-up", h.walletHandler.TopUp)
		v1.POST("/wallets/:id/close", h.walletHandler.Close)
		v1.GET("/wallets/:id/transactions", h.walletHandler.ListTransactions)
		v1.PUT("/wallets/:id/auto-recharge", h.walletHandler.UpdateAutoRecharge)
		v1.GET("/customers/:id/wallets", h.walletHandler.ListForCustomer)

		// Usage threshold alerts (Lago-parity B3)
		v1.POST("/usage-alerts", h.usageAlertHandler.Create)
		v1.GET("/usage-alerts", h.usageAlertHandler.List)
		v1.PUT("/usage-alerts/:id", h.usageAlertHandler.Update)
		v1.DELETE("/usage-alerts/:id", h.usageAlertHandler.Delete)

		// Append-only audit trail (Lago-parity C2)
		v1.GET("/audit-logs", h.auditHandler.List)

		// Analytics (Cached)
		analytics := v1.Group("/analytics")
		analytics.Use(middleware.CacheMiddleware(h.rdb, 5*time.Minute))
		{
			analytics.GET("/mrr", h.analyticsHandler.GetMRR)
			analytics.GET("/mrr/by-entity", h.analyticsHandler.GetMRRByEntity)
			analytics.GET("/entities-overview", h.analyticsHandler.GetEntitiesOverview)
			analytics.GET("/mrr/waterfall", h.analyticsHandler.GetMRRWaterfall)
			analytics.GET("/invoice-aging", h.analyticsHandler.GetInvoiceAging)
			analytics.GET("/unit-economics", h.analyticsHandler.GetUnitEconomics)
			analytics.GET("/revenue-by-plan", h.analyticsHandler.GetRevenueByPlan)
			analytics.GET("/revenue-by-geography", h.analyticsHandler.GetRevenueByGeography)
			analytics.GET("/usage", h.analyticsHandler.GetUsageStats)
			analytics.GET("/dunning/overview", h.dunningHandler.GetOverview)
			analytics.GET("/dunning/weights", h.dunningHandler.GetWeights)
			analytics.GET("/dunning/history", h.dunningHandler.GetHistory)
			analytics.GET("/dunning/recovered", h.dunningHandler.GetRecovered)
			analytics.GET("/dunning/timing", h.dunningHandler.GetTiming)
			analytics.GET("/collections/funnel", h.collectionsHandler.GetFunnel)
			analytics.GET("/collections/failures", h.collectionsHandler.GetFailures)
		}
		v1.POST("/analytics/ask", h.analyticsHandler.Ask) // P48 GenAI

		// Collections Intelligence — operator worklist of currently-failing
		// invoices. Uncached: operational data that changes on every retry.
		v1.GET("/collections/queue", h.collectionsHandler.GetQueue)
		// Manual controls (Inc 3). Mutate a single invoice's dunning state only.
		v1.POST("/collections/invoices/:id/retry-now", h.collectionsHandler.RetryNow)
		v1.POST("/collections/invoices/:id/pause", h.collectionsHandler.PauseDunning)
		v1.POST("/collections/invoices/:id/mark-uncollectible", h.collectionsHandler.MarkUncollectible)

		v1.POST("/coupons", h.couponHandler.CreateCoupon) // P7
		v1.GET("/coupons", h.couponHandler.ListCoupons)
		v1.GET("/coupons/:id", h.couponHandler.GetCoupon)
		v1.PUT("/coupons/:id", h.couponHandler.UpdateCoupon)

		// Developer / Settings
		v1.GET("/developer/keys", h.tenantHandler.ListKeys)
		v1.POST("/developer/keys", h.tenantHandler.CreateKey)
		v1.DELETE("/developer/keys/:id", h.tenantHandler.RevokeKey)

		// Team management (dashboard users). Reads are open to any authed
		// member; writes are gated to owner/admin inside the handler.
		v1.GET("/users", h.teamHandler.ListUsers)
		v1.POST("/users", h.teamHandler.CreateUser)
		v1.POST("/users/invite", h.teamHandler.InviteUser)
		v1.PATCH("/users/:id", h.teamHandler.UpdateUser)
		v1.DELETE("/users/:id", h.teamHandler.DeleteUser)

		// Account security for the logged-in dashboard user (TOTP MFA + active
		// session management). API-key callers have no user and are rejected.
		v1.POST("/auth/mfa/setup", h.authHandler.MFASetup)
		v1.POST("/auth/mfa/verify", h.authHandler.MFAVerify)
		v1.POST("/auth/mfa/disable", h.authHandler.MFADisable)
		v1.GET("/auth/sessions", h.authHandler.ListSessions)
		v1.DELETE("/auth/sessions/:id", h.authHandler.RevokeSession)
		v1.DELETE("/auth/sessions", h.authHandler.RevokeOtherSessions)

		// SAML SSO connection config (tenant-scoped; writes gated to owner/admin
		// inside the handler). The public SP endpoints live under /auth/saml.
		// BYO gateway connections (increment 4): tenants connect their own
		// Stripe/Razorpay. Writes are owner/admin-gated in the handler.
		v1.GET("/gateway-connections", h.gatewayConnHandler.List)
		v1.POST("/gateway-connections", h.gatewayConnHandler.Connect)
		// BYO integrations (increment 5): per-tenant tax/CRM/storage credentials.
		v1.GET("/integration-connections", h.integrationConnHandler.List)
		v1.POST("/integration-connections", h.integrationConnHandler.Connect)
		v1.DELETE("/integration-connections/:category/:provider", h.integrationConnHandler.Disconnect)
		v1.PUT("/gateway-connections/:provider/webhook-secret", h.gatewayConnHandler.SetWebhookSecret)
		v1.DELETE("/gateway-connections/:provider", h.gatewayConnHandler.Disconnect)

		v1.GET("/sso/connection", h.ssoHandler.GetConnection)
		v1.PUT("/sso/connection", h.ssoHandler.UpsertConnection)
		v1.DELETE("/sso/connection", h.ssoHandler.DeleteConnection)

		// Advanced Billing (P15)
		v1.POST("/subscriptions/:id/charges", h.advancedBillingHandler.AddUnbilledCharge)
		v1.GET("/subscriptions/:id/charges", h.advancedBillingHandler.ListUnbilledCharges)
		v1.POST("/subscriptions/:id/advance", h.advancedBillingHandler.GenerateAdvanceInvoice)
		v1.POST("/subscriptions/:id/bill-usage", h.advancedBillingHandler.BillUsageNow) // A5 interim progressive bill

		// Heavy read-only finance reports get the same 5-minute Redis cache as
		// /analytics/*: they re-aggregate the whole ledger per request and are
		// viewed far more often than their inputs change. Reconciliation stays
		// uncached on purpose — its "Run again" button must actually re-run.
		reportCache := middleware.CacheMiddleware(h.rdb, 5*time.Minute)

		// Ledger (P22)
		v1.GET("/ledger/accounts", h.ledgerHandler.ListAccounts)
		v1.GET("/ledger/entries", h.ledgerHandler.GetEntries)
		// A single posted transaction (journal entry) — addressable, each leg
		// deep-linkable to its account. Read-only.
		v1.GET("/ledger/transactions/:id", h.ledgerHandler.GetTransaction)
		// Provable-ledger auditor outputs (ENG-192): trial balance + GL export
		v1.GET("/ledger/trial-balance", reportCache, h.ledgerHandler.GetTrialBalance)
		v1.GET("/ledger/export", h.expensiveLimit, h.ledgerHandler.ExportGL)
		v1.GET("/ledger/deferred-rollforward", reportCache, h.ledgerHandler.GetDeferredRollforward)

		// Ledger Reconciliation — on-demand drift report for the caller's tenant
		v1.GET("/finance/reconciliation", h.reconciliationHandler.RunReconciliation)
		v1.POST("/finance/reconciliation/runs", h.reconciliationHandler.RecordReconciliation)
		v1.GET("/finance/reconciliation/runs", h.reconciliationHandler.ListReconciliationRuns)
		// A single recorded run with its stored discrepancy rows — the
		// addressable, explainable run object. Read-only.
		v1.GET("/finance/reconciliation/runs/:id", h.reconciliationHandler.GetReconciliationRun)

		// Month-end close pack (B2) — trial balance + reconciliation + deferred
		// rollforward + GL export pointer + a ready-to-close verdict. Uncached
		// like reconciliation: the close verdict must reflect the ledger now.
		v1.GET("/finance/close-pack", h.closePackHandler.GetClosePack)

		// Credit Notes (P23)
		v1.POST("/credit-notes", h.creditNoteHandler.CreateCreditNote)
		v1.GET("/credit-notes", h.creditNoteHandler.ListCreditNotes)
		v1.GET("/credit-notes/:id", h.creditNoteHandler.GetCreditNote)
		v1.GET("/credit-notes/:id/journal-entries", h.creditNoteHandler.GetCreditNoteJournalEntries)
		v1.GET("/credit-notes/:id/pdf", h.expensiveLimit, h.creditNoteHandler.DownloadPDF)
		v1.POST("/credit-notes/:id/approve", h.creditNoteHandler.ApproveCreditNote)
		v1.POST("/credit-notes/:id/reject", h.creditNoteHandler.RejectCreditNote)
		v1.POST("/credit-notes/:id/void", h.creditNoteHandler.VoidCreditNote)

		// Webhooks & Events (P24)
		v1.POST("/webhooks", h.webhookMgmtHandler.CreateEndpoint)
		v1.GET("/webhooks", h.webhookMgmtHandler.ListEndpoints)
		v1.PUT("/webhooks/:id/status", h.webhookMgmtHandler.UpdateEndpointStatus)
		v1.DELETE("/webhooks/:id", h.webhookMgmtHandler.DeleteEndpoint)
		v1.GET("/webhooks/:id/deliveries", h.webhookMgmtHandler.ListEndpointDeliveries)

		// Account (Tenant) Management
		v1.GET("/account", h.tenantHandler.GetAccount)
		v1.PUT("/account", h.tenantHandler.UpdateAccount)
		// Managed-cloud billing/trial status + plan catalog (read-only).
		v1.GET("/billing/status", h.billingHandler.Status)
		v1.GET("/billing/plans", h.billingHandler.Plans)

		// Quotes (P27)
		v1.POST("/quotes", h.quoteHandler.CreateQuote)
		v1.GET("/quotes", h.quoteHandler.ListQuotes)
		v1.GET("/quotes/:id", h.quoteHandler.GetQuote)
		v1.PUT("/quotes/:id", h.quoteHandler.UpdateQuote)
		v1.DELETE("/quotes/:id", h.quoteHandler.DeleteQuote)
		v1.POST("/quotes/:id/send", h.quoteHandler.SendQuote)
		v1.POST("/quotes/:id/accept", h.quoteHandler.AcceptQuote)
		v1.POST("/quotes/:id/decline", h.quoteHandler.DeclineQuote)
		v1.POST("/quotes/:id/convert", h.quoteHandler.ConvertToInvoice)

		// Invoice disputes (Track 2) — admin API only; no dashboard UI yet.
		v1.GET("/disputes", h.disputeHandler.ListDisputes)
		v1.GET("/disputes/:id", h.disputeHandler.GetDispute)
		v1.POST("/disputes/:id/resolve", h.disputeHandler.ResolveDispute)

		v1.GET("/events", h.webhookMgmtHandler.ListEvents)
		v1.GET("/events/types", h.webhookMgmtHandler.GetEventTypes)
		v1.GET("/events/:id/deliveries", h.webhookMgmtHandler.ListEventDeliveries)
		v1.POST("/events/:id/redeliver", h.webhookMgmtHandler.RedeliverEvent)

		// GST Settings (P30)
		v1.GET("/settings/gst", h.gstHandler.GetConfig)
		v1.PUT("/settings/gst", h.gstHandler.UpdateConfig)
		// US sales-tax nexus config
		v1.GET("/settings/tax/nexus", h.taxNexusHandler.GetNexus)
		v1.PUT("/settings/tax/nexus", h.taxNexusHandler.SetNexus)
		v1.GET("/settings/tax/nexus/status", h.taxNexusHandler.GetNexusStatus)
		v1.GET("/settings/tax/liability", h.taxNexusHandler.GetLiabilityReport)
		v1.GET("/settings/tax/registrations", h.taxNexusHandler.GetRegistrations)
		v1.PUT("/settings/tax/registrations", h.taxNexusHandler.SetRegistrations)
		v1.POST("/settings/gst/validate", h.gstHandler.ValidateGSTIN)
		v1.GET("/india/gstr1", h.gstHandler.GetGSTR1)
		v1.GET("/india/gstr3b", h.gstHandler.GetGSTR3B)

		// E-Invoice (P25)
		v1.GET("/invoices/:id/einvoice", h.einvoiceHandler.GetEInvoiceStatus)
		v1.POST("/invoices/:id/einvoice/retry", h.einvoiceHandler.RetryEInvoice)
		v1.POST("/invoices/:id/einvoice/cancel", h.einvoiceHandler.CancelEInvoice)
		v1.GET("/settings/irp", h.einvoiceHandler.GetIRPConfig)
		v1.PUT("/settings/irp", h.einvoiceHandler.UpdateIRPConfig)
		v1.POST("/settings/irp/test", h.einvoiceHandler.TestIRPConnection)
		// EU e-invoicing config (Track C): opt-in + EN 16931 seller identity.
		v1.GET("/settings/eu-einvoice", h.euConfigHandler.GetEUConfig)
		v1.PUT("/settings/eu-einvoice", h.euConfigHandler.UpdateEUConfig)
		v1.POST("/crm/sync", h.crmSyncHandler.SyncNow)
		v1.GET("/settings/tax/us", h.usTaxConfigHandler.GetUSTaxConfig)
		v1.PUT("/settings/tax/us", h.usTaxConfigHandler.UpdateUSTaxConfig)

		v1.GET("/settings/invoice-branding", h.invoiceBrandingHandler.GetBranding)
		v1.PUT("/settings/invoice-branding", h.invoiceBrandingHandler.UpdateBranding)
		// EU e-invoicing per invoice (Track C inc 2): inspect the generated UBL +
		// delivery status, and manually regenerate/re-transmit a failed one.
		v1.GET("/invoices/:id/eu-einvoice", h.euEInvoiceHandler.GetEUEInvoice)
		v1.POST("/invoices/:id/eu-einvoice/retry", h.euEInvoiceHandler.RetryEUEInvoice)
		v1.GET("/settings/mcp", h.mcpSettingsHandler.GetMCPSettings)
		v1.PUT("/settings/mcp", h.mcpSettingsHandler.UpdateMCPSettings)

		// Legal entities (Multi-Entity Books)
		v1.GET("/entities", h.entityHandler.ListEntities)
		v1.POST("/entities", h.entityHandler.CreateEntity)
		v1.GET("/entities/:id", h.entityHandler.GetEntity)
		v1.PUT("/entities/:id", h.entityHandler.UpdateEntity)
		v1.DELETE("/entities/:id", h.entityHandler.DeleteEntity)

		// Consent API (P30 - RBI compliance)
		v1.POST("/consents", h.consentHandler.RecordConsent)
		v1.POST("/consents/revoke", h.consentHandler.RevokeConsent)
		v1.GET("/customers/:id/consents", h.consentHandler.GetCustomerConsents)
		v1.GET("/subscriptions/:id/consent", h.consentHandler.GetSubscriptionConsent)

		// Cancellation API (P30 - easy cancellation)
		// Deterministic financial forecast of a cancellation, before the mutation.
		v1.GET("/subscriptions/:id/cancel-preview", h.cancellationHandler.PreviewCancel)
		v1.POST("/subscriptions/:id/cancel", h.cancellationHandler.CancelSubscription)
		v1.POST("/subscriptions/:id/reactivate", h.cancellationHandler.ReactivateSubscription)
		v1.POST("/subscriptions/:id/pause", h.subscriptionHandler.PauseSubscription)
		v1.POST("/subscriptions/:id/resume", h.subscriptionHandler.ResumeSubscription)
		v1.GET("/cancellation-reasons", h.cancellationHandler.GetCancellationReasons)

		// Referral & Gift API
		v1.GET("/referrals", h.referralHandler.ListReferrals)
		v1.POST("/referrals", h.referralHandler.CreateReferral)
		v1.POST("/referrals/generate-code", h.referralHandler.GenerateCode)
		v1.POST("/referrals/:id/qualify", h.referralHandler.QualifyReferral)
		v1.GET("/gifts", h.giftHandler.ListGifts)

		// Gift API (P43)
		v1.POST("/gifts/purchase", h.giftHandler.PurchaseGift)
		v1.POST("/gifts/redeem", h.giftHandler.RedeemGift)
		v1.POST("/gifts/:id/cancel", h.giftHandler.CancelGift)

		// Phase 2: UPI Mandates
		v1.POST("/mandates", h.mandateHandler.CreateMandate)
		v1.GET("/mandates", h.mandateHandler.ListMandates)
		v1.GET("/mandates/:id", h.mandateHandler.GetMandate)
		v1.POST("/mandates/:id/revoke", h.mandateHandler.RevokeMandate)

		// Phase 2: Offline Payments / Virtual Accounts
		v1.POST("/virtual-accounts", h.offlinePaymentHandler.CreateVirtualAccount)
		v1.GET("/virtual-accounts", h.offlinePaymentHandler.ListVirtualAccounts)
		v1.POST("/payments/offline", h.offlinePaymentHandler.RecordOfflinePayment)
		v1.GET("/payments/offline", h.offlinePaymentHandler.ListOfflinePayments)

		// Revenue Recognition Report
		v1.GET("/finance/revrec/report", reportCache, h.revrecHandler.GetReport)
		v1.GET("/finance/revrec/waterfall", reportCache, h.revrecHandler.GetWaterfall)

		// Phase 2: Organizations (Multi-Entity)
		v1.GET("/organizations", h.orgHandler.ListOrganizations)
		v1.POST("/organizations", h.orgHandler.CreateOrganization)
		v1.GET("/organizations/:id", h.orgHandler.GetOrganization)
		v1.PUT("/organizations/:id", h.orgHandler.UpdateOrganization)
		v1.DELETE("/organizations/:id", h.orgHandler.DeleteOrganization)
		v1.POST("/organizations/:id/tenants", h.orgHandler.AddTenant)
		v1.GET("/organizations/:id/tenants", h.orgHandler.ListTenants)
		v1.DELETE("/organizations/:id/tenants/:tenant_id", h.orgHandler.RemoveTenant)
		v1.GET("/organizations/:id/analytics/mrr", h.orgHandler.GetConsolidatedMRR)

		// Phase 2: Accounting / ERP Integrations
		v1.GET("/accounting/connections", h.accountingHandler.ListConnections)
		v1.POST("/accounting/connect/:provider", h.accountingHandler.InitiateOAuth)
		v1.POST("/accounting/connect-token/:provider", h.accountingHandler.ConnectTokenBased)
		// (moved to a public route below — the callback is a browser redirect
		// from the provider carrying no session; the HMAC-signed state is its
		// authentication)
		v1.DELETE("/accounting/connections/:id", h.accountingHandler.Disconnect)
		v1.POST("/accounting/sync", h.accountingHandler.TriggerSync)
		v1.GET("/accounting/sync/status", h.accountingHandler.SyncStatus)

		// Phase 2: Churn Scoring
		v1.GET("/customers/:id/churn", h.churnHandler.GetCustomerChurn)
		v1.GET("/churn/high-risk", h.churnHandler.GetHighRiskCustomers)
		v1.GET("/churn/alerts", h.churnHandler.GetAlerts)
		v1.POST("/churn/alerts/:id/ack", h.churnHandler.AcknowledgeAlert)

		// Cancel Flows (Retention Interventions)
		v1.GET("/cancel-flows", h.cancelFlowHandler.ListFlows)
		v1.POST("/cancel-flows", h.cancelFlowHandler.CreateFlow)
		v1.GET("/cancel-flows/:id", h.cancelFlowHandler.GetFlow)
		v1.PUT("/cancel-flows/:id", h.cancelFlowHandler.UpdateFlow)
		v1.POST("/cancel-flows/:id/steps", h.cancelFlowHandler.CreateStep)
		v1.PUT("/cancel-flows/steps/:id", h.cancelFlowHandler.UpdateStep)
		v1.DELETE("/cancel-flows/steps/:id", h.cancelFlowHandler.DeleteStep)
		v1.POST("/cancel-flows/sessions/start", h.cancelFlowHandler.StartSession)
		v1.POST("/cancel-flows/sessions/:id/submit", h.cancelFlowHandler.SubmitStep)
		v1.GET("/cancel-flows/sessions/:id", h.cancelFlowHandler.GetSession)
		v1.GET("/cancel-flows/stats", h.cancelFlowHandler.GetStats)

		// Dunning Campaigns (Multi-Channel)
		v1.GET("/dunning-campaigns", h.dunningCampaignHandler.ListCampaigns)
		v1.POST("/dunning-campaigns", h.dunningCampaignHandler.CreateCampaign)
		v1.GET("/dunning-campaigns/:id", h.dunningCampaignHandler.GetCampaign)
		v1.PUT("/dunning-campaigns/:id", h.dunningCampaignHandler.UpdateCampaign)
		v1.POST("/dunning-campaigns/:id/steps", h.dunningCampaignHandler.CreateStep)
		v1.PUT("/dunning-campaigns/steps/:id", h.dunningCampaignHandler.UpdateStep)
		v1.DELETE("/dunning-campaigns/steps/:id", h.dunningCampaignHandler.DeleteStep)
		v1.GET("/invoices/:id/payment-wall", h.dunningCampaignHandler.GetPaymentWallStatus)
	}
}

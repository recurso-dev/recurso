package main

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/adapter/accounting"
	"github.com/recurso-dev/recurso/internal/adapter/ai"
	"github.com/recurso-dev/recurso/internal/adapter/alerting"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/adapter/fx"
	"github.com/recurso-dev/recurso/internal/adapter/gateway"
	"github.com/recurso-dev/recurso/internal/adapter/gsp"
	"github.com/recurso-dev/recurso/internal/adapter/handler"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/adapter/notification"
	redisAdapter "github.com/recurso-dev/recurso/internal/adapter/redis"
	"github.com/recurso-dev/recurso/internal/adapter/sms"
	"github.com/recurso-dev/recurso/internal/adapter/taxprovider"
	"github.com/recurso-dev/recurso/internal/adapter/telemetry"
	"github.com/recurso-dev/recurso/internal/adapter/tigerbeetle"
	"github.com/recurso-dev/recurso/internal/adapter/vatprovider"
	"github.com/recurso-dev/recurso/internal/adapter/vault"
	"github.com/recurso-dev/recurso/internal/adapter/worker"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/residency"
	"github.com/recurso-dev/recurso/internal/scheduler"
	"github.com/recurso-dev/recurso/internal/service"
	"github.com/redis/go-redis/v9"
)

// version is stamped at build time via:
//
//	go build -ldflags "-X main.version=v0.1.0"
var version = "dev"

// On Cloud Run every revision gets a K_REVISION env var (e.g.
// "recurso-api-00044-abc"). When the build didn't stamp a version — as with the
// managed Dockerfile Cloud Build trigger, which leaves it "dev" — fall back to
// that so /version and /health report the actual deployed revision instead of a
// generic "dev". A real ldflags-stamped version (release builds) still wins.
func init() {
	if version == "dev" {
		if rev := os.Getenv("K_REVISION"); rev != "" {
			version = rev
		}
	}
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvBool reads a boolean feature flag, accepting the usual truthy spellings
// ("true", "1", "yes", "on"). Anything else (including unset) yields fallback.
func getEnvBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func main() {
	// Structured JSON logging for the whole process: the workers and schedulers
	// log via slog, so make the default handler emit JSON to stdout for
	// machine-parseable, queryable logs in any deployment.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 1. Initialize DB
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		if os.Getenv("APP_ENV") == "development" {
			dbURL = "postgres://user:password@localhost:5432/recurso?sslmode=disable"
			log.Println("Warning: DATABASE_URL not set, using development default")
		} else {
			log.Fatal("DATABASE_URL environment variable is required")
		}
	}

	database, err := db.NewConnection(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() { _ = database.Close() }()

	// 2. Run Migrations
	if err := db.RunMigrations(dbURL); err != nil {
		log.Printf("Migration warning: %v", err)
	}

	// 3. Initialize Repositories
	dbx := sqlx.NewDb(database, "postgres")

	planRepo := db.NewPlanRepository(database)
	customerRepo := db.NewCustomerRepository(dbx)
	subscriptionRepo := db.NewSubscriptionRepository(database)
	invoiceRepo := db.NewInvoiceRepository(database)
	entitlementRepo := db.NewEntitlementRepository(database) // Entitlement Engine v1
	usageRepo := db.NewUsageRepository(database)
	couponRepo := db.NewCouponRepository(database)                       // P7
	tenantRepo := db.NewTenantRepository(database)                       // P8
	unbilledChargeRepo := db.NewUnbilledChargeRepository(database)       // P15
	subscriptionAddonRepo := db.NewSubscriptionAddonRepository(database) // Multi-product catalog v1
	webhookEndpointRepo := db.NewWebhookEndpointRepository(database)     // P24
	eventRepo := db.NewEventRepository(database)                         // P24
	eventDeliveryRepo := db.NewEventDeliveryRepository(database)         // P24
	magicLinkRepo := db.NewMagicLinkRepository(database)                 // P25
	portalSessionRepo := db.NewPortalSessionRepository(database)         // P25
	quoteRepo := db.NewQuoteRepository(database)                         // P27
	disputeRepo := db.NewDisputeRepository(database)                     // Track 2: invoice disputes

	// Create sqlx wrapper for CreditNoteRepository
	creditNoteRepo := db.NewCreditNoteRepository(dbx) // P23

	// Notifications (P29)
	notifier := notification.NewConsoleNotifier()
	if host := os.Getenv("SMTP_HOST"); host != "" {
		notifier = notification.NewSMTPNotifier(
			host,
			os.Getenv("SMTP_PORT"),
			os.Getenv("SMTP_USERNAME"),
			os.Getenv("SMTP_PASSWORD"),
			os.Getenv("SMTP_FROM"),
		)
		log.Println("Using SMTP Notifier")
	} else {
		log.Println("Using Console Notifier (Mock)")
	}

	var emailSender port.EmailSender = email.NewConsoleSender()
	if host := os.Getenv("SMTP_HOST"); host != "" {
		smtpPort, _ := strconv.Atoi(getEnvDefault("SMTP_PORT", "587"))
		emailSender = email.NewSMTPSender(email.SMTPConfig{
			Host:     host,
			Port:     smtpPort,
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     getEnvDefault("SMTP_FROM", "noreply@localhost"),
			FromName: getEnvDefault("SMTP_FROM_NAME", "Recurso"),
			UseTLS:   os.Getenv("SMTP_USE_TLS") == "true",
		})
		log.Println("Using SMTP Email Sender")
	} else {
		log.Println("Using Console Email Sender (emails are logged, not sent — set SMTP_HOST for real delivery)")
	}
	baseURL := getEnvDefault("BASE_URL", "http://localhost:8080")
	notificationService := service.NewNotificationService(emailSender, baseURL)
	// notificationService is wired to subscriptionService, webhookHandler, schedulers, and cancellationHandler below

	// Ledger (P5) — dual-write: PG (always) + TigerBeetle (optional)
	ledgerRepo := db.NewLedgerRepository(database)
	var ledgerService *service.LedgerService
	tbAddr := getEnvDefault("TIGERBEETLE_ADDRESS", "127.0.0.1:3001")
	var tbClientForRecon *tigerbeetle.LedgerClient
	ledgerClient, err := tigerbeetle.NewLedgerClient(0, []string{tbAddr})
	if err == nil {
		ledgerService = service.NewLedgerService(ledgerClient, ledgerRepo)
		tbClientForRecon = ledgerClient
		defer ledgerClient.Close()
	} else {
		slog.Warn("TigerBeetle not connected — ledger PG-only mode", "error", err)
		ledgerService = service.NewLedgerService(nil, ledgerRepo)
	}

	// Ledger reconciliation: on-demand drift detection between billing
	// records (invoices) and the Postgres ledger.
	reconciliationService := service.NewReconciliationService(ledgerRepo, tbClientForRecon)

	// 5. Initialize Gateways
	var razorpayGateway port.PaymentGateway
	if keyID := os.Getenv("RAZORPAY_KEY_ID"); keyID != "" {
		razorpayGateway = gateway.NewRazorpayGateway(keyID, os.Getenv("RAZORPAY_KEY_SECRET"))
		log.Println("Using Real Razorpay Gateway")
	} else {
		razorpayGateway = gateway.NewMockGateway()
		log.Println("Using Razorpay Gateway (Mock — set RAZORPAY_KEY_ID for real payments)")
	}

	var stripeGateway port.PaymentGateway
	if key := os.Getenv("STRIPE_SECRET_KEY"); key != "" {
		stripeGateway = gateway.NewStripeGateway(key, os.Getenv("STRIPE_WEBHOOK_SECRET"))
		log.Println("Using Real Stripe Gateway")
	} else {
		stripeGateway = gateway.NewMockGateway()
		log.Println("Using Stripe Gateway (Mock)")
	}

	// Smart Router routes based on Currency (INR -> Razorpay, USD -> Stripe)
	paymentGateway := gateway.NewSmartRouter(razorpayGateway, stripeGateway)

	// Card Vault — uses Stripe if key exists, else Mock for dev
	var cardVault port.CardVault
	if key := os.Getenv("STRIPE_SECRET_KEY"); key != "" {
		cardVault = vault.NewStripeVault(key)
		log.Println("Using Stripe Card Vault")
	} else {
		cardVault = vault.NewMockVault()
		log.Println("Using Mock Card Vault")
	}
	_ = cardVault // Available for SmartRouter or payment handlers

	// P25: IRP & GST Config Repositories
	irpConfigRepo := db.NewIRPConfigRepository(database)
	gstConfigRepo := db.NewGSTConfigRepository(database)
	taxNexusRepo := db.NewTaxNexusRepository(database)

	// P25: GSP Adapter — use NIC if private key is available, else mock
	var gspAdapter port.GSPAdapter
	if nicKeyPath := os.Getenv("NIC_PRIVATE_KEY_PATH"); nicKeyPath != "" {
		nicKeyPEM, err := os.ReadFile(nicKeyPath)
		if err != nil {
			log.Printf("Warning: Failed to read NIC private key from %s: %v. Falling back to mock.", nicKeyPath, err)
			gspAdapter = gsp.NewMockGSPAdapter()
		} else {
			nicEnv := os.Getenv("NIC_ENVIRONMENT")
			if nicEnv == "" {
				nicEnv = "sandbox"
			}
			nicAdapter, err := gsp.NewNICAdapter(nicEnv, nicKeyPEM, irpConfigRepo)
			if err != nil {
				log.Printf("Warning: Failed to create NIC adapter: %v. Falling back to mock.", err)
				gspAdapter = gsp.NewMockGSPAdapter()
			} else {
				gspAdapter = nicAdapter
				log.Printf("Using NIC GSP Adapter (environment: %s)", nicEnv)
			}
		}
	} else {
		gspAdapter = gsp.NewMockGSPAdapter() // P25 Mock GSP
		log.Println("Using Mock GSP Adapter (NIC_PRIVATE_KEY_PATH not set)")
	}

	// FX Provider — OXR if key set (with static rates as fallback when the
	// live fetch fails), else static rates. OXR caches rates in-memory for 1h.
	var fxProvider port.ExchangeRateProvider
	var fxFallback port.ExchangeRateProvider
	if oxrKey := os.Getenv("OPENEXCHANGERATES_APP_ID"); oxrKey != "" {
		fxProvider = fx.NewOpenExchangeRatesProvider(oxrKey)
		fxFallback = fx.NewStaticRatesProvider()
		log.Println("Using OpenExchangeRates FX provider (static rates fallback)")
	} else {
		fxProvider = fx.NewStaticRatesProvider()
		log.Println("Using Static FX rates provider")
	}
	// Default reporting currency for FX-normalized analytics (MRR). Tenants
	// with a base_currency set report in that currency instead.
	reportingCurrency := getEnvDefault("REPORTING_CURRENCY", "USD")

	// Tax Resolver — per-tenant GST config decides the seller jurisdiction
	// (India + state) when present; env company defaults otherwise. Dispatches
	// to GST/VAT/SalesTax engines per invoice.
	companyCountry := getEnvDefault("COMPANY_COUNTRY", "IN")
	companyState := getEnvDefault("COMPANY_STATE", "TN")
	taxResolver := service.NewTaxResolver(gstConfigRepo, companyCountry, companyState)
	// US sales-tax nexus gating (opt-in): once a tenant declares nexus states,
	// US tax is collected only there; a tenant with none is unaffected.
	taxResolver = taxResolver.WithNexusRepo(taxNexusRepo)
	// US sales tax — TaxJar when a key is set (the resolver caches rates
	// in-memory for 24h per state+zip); otherwise the US engine stays an
	// honest 0% stub (invoices marked sales_tax_stub).
	if taxjarKey := os.Getenv("TAXJAR_API_KEY"); taxjarKey != "" && !residency.SelfHosted() {
		taxResolver = taxResolver.WithSalesTaxProvider(taxprovider.NewTaxJarProvider(taxjarKey, os.Getenv("TAXJAR_API_URL")))
		log.Println("US sales tax: TaxJar provider enabled")
	} else if residency.SelfHosted() {
		log.Println("US sales tax: 0% stub (external tax API disabled by RESIDENCY_MODE=self_hosted)")
	} else {
		log.Println("US sales tax: 0% stub (TAXJAR_API_KEY not set)")
	}
	// EU VAT-number validation — VIES when enabled. With it wired, intra-EU
	// cross-border B2B reverse charge is only granted when the buyer's VAT
	// number validates; a VIES outage degrades to the presence-based behaviour
	// (never fails an invoice). Enabled by VIES_ENABLED=true, or implicitly
	// when VIES_API_URL is set (handy for pointing tests at a stub server).
	viesURL := os.Getenv("VIES_API_URL")
	if getEnvBool("VIES_ENABLED", false) || viesURL != "" {
		taxResolver = taxResolver.WithVATValidator(vatprovider.NewVIESValidator(viesURL))
		log.Println("EU VAT validation: VIES enabled")
	} else {
		log.Println("EU VAT validation: disabled (reverse charge is presence-based)")
	}

	// 4. Initialize Core Services (Invoice)
	invoiceService := service.NewInvoiceService(invoiceRepo, planRepo, customerRepo, unbilledChargeRepo, subscriptionRepo, gspAdapter, taxResolver) // P15, P25

	// Usage-based billing v1 (spec_usage_billing.md): billable metrics,
	// plan charges, and rating of usage into metered invoice lines.
	billableMetricRepo := db.NewBillableMetricRepository(database)
	chargeRepo := db.NewChargeRepository(database)
	usageRatingRepo := db.NewUsageRatingRepository(database)
	invoiceService.ChargeRepo = chargeRepo
	invoiceService.UsageRepo = usageRepo
	invoiceService.RatingRepo = usageRatingRepo

	catalogService := service.NewCatalogService(planRepo)
	entitlementService := service.NewEntitlementService(entitlementRepo, planRepo, customerRepo, subscriptionRepo) // Entitlement Engine v1
	usageService := service.NewUsageService(usageRepo, subscriptionRepo, entitlementService)                       // Usage Platform v1
	meteringService := service.NewMeteringService(billableMetricRepo, chargeRepo, planRepo, subscriptionRepo, usageRepo)
	customerService := service.NewCustomerService(customerRepo)
	tenantService := service.NewTenantService(tenantRepo) // P8 Service

	// Admin-dashboard auth: real user accounts + opaque sessions layered on top
	// of the existing tenant API-key auth (both resolve to the same tenant_id).
	userRepo := db.NewUserRepository(database)
	sessionRepo := db.NewSessionRepository(database)
	passwordResetRepo := db.NewPasswordResetRepository(database)
	mfaBackupRepo := db.NewMFABackupCodeRepository(database)
	mfaLoginTokenRepo := db.NewMFALoginTokenRepository(database)
	sessionTTLHours, _ := strconv.Atoi(getEnvDefault("SESSION_TTL_HOURS", "168")) // default 7 days
	if sessionTTLHours <= 0 {
		sessionTTLHours = 168
	}
	authService := service.NewAuthService(userRepo, sessionRepo, tenantService, time.Duration(sessionTTLHours)*time.Hour)
	// Phase 2 auth: password reset + TOTP MFA. The reset link points at the
	// admin dashboard (DASHBOARD_URL), falling back to the API base URL for dev.
	dashboardURL := getEnvDefault("DASHBOARD_URL", baseURL)
	authService.ConfigurePasswordReset(passwordResetRepo, notificationService, dashboardURL)
	authService.ConfigureMFA(mfaBackupRepo, mfaLoginTokenRepo)
	creditNoteService := service.NewCreditNoteService(creditNoteRepo, customerRepo, invoiceRepo, paymentGateway) // P23 + refunds
	creditNoteService.SetLedgerService(ledgerService)
	txManager := db.NewTxManager(database)

	// Revenue Recognition (P5)
	revrecRepo := db.NewRevRecRepository(database)
	revrecService := service.NewRevRecService(revrecRepo, ledgerService, subscriptionRepo)
	// Unwind deferred revenue when a refund is issued (ENG-147).
	creditNoteService.SetRevRecService(revrecService)

	subscriptionService := service.NewSubscriptionService(
		subscriptionRepo,
		invoiceRepo,
		planRepo,
		customerRepo,
		couponRepo,
		notifier,
		ledgerService,
		paymentGateway,
		gspAdapter,
		txManager,
		revrecService,
		taxResolver,
	)

	// P25: E-Invoice Service
	einvoiceService := service.NewEInvoiceService(gspAdapter, invoiceRepo, customerRepo, irpConfigRepo, gstConfigRepo)
	invoiceService.EInvoiceService = einvoiceService
	subscriptionService.SetEInvoiceService(einvoiceService)
	subscriptionService.SetNotificationService(notificationService)
	subscriptionService.SetFinalUsageInvoicer(invoiceService) // metered final invoice on immediate cancel
	// Persist downgrade proration credits as spendable adjustment credit notes (ENG-150).
	subscriptionService.SetCreditNoteRepo(creditNoteRepo)
	// Apply account credit to proration-upgrade & trial-conversion charge invoices (ENG-154).
	subscriptionService.SetCreditApplier(creditNoteService)

	// Multi-product catalog v1: enable subscription add-ons on the service
	// (add/remove/list) and on the recurring invoice path (extra taxed lines).
	subscriptionService.SetAddonRepository(subscriptionAddonRepo)
	invoiceService.AddonRepo = subscriptionAddonRepo
	// Apply adjustment credit-note balances to generated invoices (ENG-153) and
	// book the settlement in the ledger (ENG-154). creditNoteService wraps the
	// repo draw-down with the DR Customer-Credit / CR AR posting.
	invoiceService.CreditApplier = creditNoteService

	// Anonymous instance telemetry — strictly opt-in (TELEMETRY_OPTIN=true).
	// Disabled (the default) means telemetryClient is nil: zero network calls,
	// zero rows written; all hooks below are nil-safe no-ops. docs/telemetry.md
	// documents every payload.
	telemetryClient := telemetry.NewFromEnv(database, version)
	if telemetryClient != nil {
		telemetryClient.Start(context.Background())
		defer telemetryClient.Stop()
		log.Println("Anonymous telemetry enabled (TELEMETRY_OPTIN=true) — see docs/telemetry.md for exactly what is sent")
	}
	catalogService.SetTelemetry(telemetryClient)
	customerService.SetTelemetry(telemetryClient)
	subscriptionService.SetTelemetry(telemetryClient)
	invoiceService.Telemetry = telemetryClient

	// Phase 2: Mandate Repository
	mandateRepo := db.NewMandateRepository(database)

	// Phase 2: Offline Payment Repository
	offlinePaymentRepo := db.NewOfflinePaymentRepository(database)

	// Phase 2: Organization Repository
	orgRepo := db.NewOrganizationRepository(database)

	// Phase 2: Accounting Connection Repository
	acctConnRepo := db.NewAccountingConnectionRepository(database)

	// Accounting entity mappings (internal ID -> provider ID per connection)
	acctMappingRepo := db.NewAccountingMappingRepository(database)

	// AI Service (P45)
	dunningRepo := db.NewDunningRepository(database)
	retryService := service.NewSmartRetryService(dunningRepo)
	if strategy := os.Getenv("DUNNING_STRATEGY"); strategy != "" {
		retryService.SetStrategy(service.BanditStrategy(strategy))
		slog.Info("Dunning strategy set", "strategy", strategy)
	}
	churnService := service.NewChurnService(customerRepo, invoiceRepo)
	churnService.SetSubscriptionRepo(subscriptionRepo)
	churnService.SetPlanRepo(planRepo)
	churnService.SetDB(database)

	// Cancel Flows
	cancelFlowRepo := db.NewCancelFlowRepository(database)
	cancelFlowService := service.NewCancelFlowService(cancelFlowRepo, subscriptionService, notificationService)

	// Dunning Campaigns
	dunningCampaignRepo := db.NewDunningCampaignRepository(database)
	var smsSender port.SMSSender
	if twilioSID := os.Getenv("TWILIO_ACCOUNT_SID"); twilioSID != "" {
		smsSender = sms.NewTwilioSMSSender(twilioSID, os.Getenv("TWILIO_AUTH_TOKEN"), os.Getenv("TWILIO_FROM_NUMBER"))
		log.Println("Using Twilio SMS Sender")
	} else {
		smsSender = sms.NewConsoleSMSSender()
		log.Println("Using Console SMS Sender (Mock)")
	}
	dunningCampaignService := service.NewDunningCampaignService(dunningCampaignRepo, invoiceRepo, customerRepo, notificationService, smsSender)

	// Dunning Recovery Attribution — provable recovered revenue
	recoveredPaymentRepo := db.NewRecoveredPaymentRepository(database)
	dunningRecoveryService := service.NewDunningRecoveryService(recoveredPaymentRepo, os.Getenv("DUNNING_STRATEGY"))
	dunningRecoveryService.SetCampaignLookup(dunningCampaignRepo)
	dunningRecoveryService.SetFX(fxProvider, fxFallback, reportingCurrency)
	dunningRecoveryService.SetTenantLookup(tenantRepo)
	subscriptionService.SetRecoveryRecorder(dunningRecoveryService)

	// Analytics
	analyticsService := service.NewAnalyticsService(subscriptionRepo, invoiceRepo, planRepo, usageRepo)
	analyticsService.SetFX(fxProvider, fxFallback, reportingCurrency)
	analyticsService.SetTenantLookup(tenantRepo)
	mrrSnapshotRepo := db.NewMRRSnapshotRepository(database)
	analyticsService.SetSnapshotStore(mrrSnapshotRepo)
	if agingStore, ok := invoiceRepo.(service.InvoiceAgingStore); ok {
		analyticsService.SetInvoiceAgingStore(agingStore)
	}
	analyticsService.SetCustomerLookup(customerRepo)

	// GenAI (P48)
	openAIKey := os.Getenv("OPENAI_API_KEY")
	var llmProvider port.LLMProvider
	if openAIKey != "" {
		llmProvider = ai.NewOpenAIProvider(openAIKey)
	} else {
		log.Println("Warning: OPENAI_API_KEY not set. GenAI analytics will be unavailable.")
	}
	genaiService := service.NewGenAIService(llmProvider, database)

	// Advanced Billing (P15)
	advancedBillingService := service.NewAdvancedBillingService(unbilledChargeRepo, subscriptionRepo)

	// Webhooks & Events (P24)
	webhookService := service.NewWebhookService(webhookEndpointRepo, eventRepo, eventDeliveryRepo)

	// Accounting (P41). Syncs run through real QuickBooks/Xero adapters
	// built from per-connection OAuth tokens; the mock gateway is only the
	// fallback for unknown providers. OAuth client credentials are needed
	// to complete the connect flow and to refresh expired tokens.
	oauthConfigs := map[string]*accounting.OAuthConfig{
		"quickbooks": {
			ClientID:     getEnvDefault("QBO_CLIENT_ID", ""),
			ClientSecret: getEnvDefault("QBO_CLIENT_SECRET", ""),
			TokenURL:     "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer",
		},
		"xero": {
			ClientID:     getEnvDefault("XERO_CLIENT_ID", ""),
			ClientSecret: getEnvDefault("XERO_CLIENT_SECRET", ""),
			TokenURL:     "https://identity.xero.com/connect/token",
		},
	}
	if residency.SelfHosted() {
		// Residency guarantee: no accounting-SaaS egress. Blank the OAuth
		// configs so the connect flow can't start; getAdapterForConnection
		// additionally refuses QuickBooks/Xero syncs for existing connections.
		oauthConfigs["quickbooks"].ClientID, oauthConfigs["quickbooks"].ClientSecret = "", ""
		oauthConfigs["xero"].ClientID, oauthConfigs["xero"].ClientSecret = "", ""
		log.Println("Accounting sync (QuickBooks/Xero) disabled by RESIDENCY_MODE=self_hosted; Tally file export remains available")
	}
	qboConfigured := oauthConfigs["quickbooks"].ClientID != ""
	xeroConfigured := oauthConfigs["xero"].ClientID != ""
	switch {
	case qboConfigured && xeroConfigured:
		log.Println("Accounting sync configured for QuickBooks and Xero")
	case qboConfigured:
		log.Println("Accounting sync configured for QuickBooks (set XERO_CLIENT_ID/XERO_CLIENT_SECRET to enable Xero)")
	case xeroConfigured:
		log.Println("Accounting sync configured for Xero (set QBO_CLIENT_ID/QBO_CLIENT_SECRET to enable QuickBooks)")
	case residency.SelfHosted():
		// Logged above; avoid the misleading "set client id to enable" hint.
	default:
		log.Println("Accounting sync running in MOCK mode — set QBO_CLIENT_ID/QBO_CLIENT_SECRET or XERO_CLIENT_ID/XERO_CLIENT_SECRET to enable real providers")
	}
	accountingGateway := accounting.NewMockAccountingAdapter()
	accountingService := service.NewAccountingService(accountingGateway, customerRepo, invoiceRepo, planRepo)
	accountingService.SetConnectionRepo(acctConnRepo)
	accountingService.SetMappingRepo(acctMappingRepo)
	accountingService.SetSubscriptionRepo(subscriptionRepo) // resolve plan ItemRefs on invoice lines
	accountingService.SetOAuthConfigs(oauthConfigs)

	// Phase 2: Mandate Service
	mandateService := service.NewMandateService(mandateRepo, paymentGateway, customerRepo, invoiceRepo)
	// Apply account credit against off-session mandate debits (ENG-153) and book
	// the settlement in the ledger (ENG-154) via creditNoteService.
	mandateService.SetCreditApplier(creditNoteService)
	// Charge the subscription's real recurring amount (plan price + tax) on each
	// mandate cycle instead of the authorization ceiling (ENG-165).
	mandateService.SetBillingResolver(subscriptionRepo, planRepo, taxResolver)
	// Lago-parity A2: mandate-debit invoices carry rated usage lines and the
	// subscription period advances with each cycle.
	mandateService.SetInvoiceService(invoiceService)

	// Phase 2: Offline Payment Service
	offlinePaymentService := service.NewOfflinePaymentService(offlinePaymentRepo, paymentGateway, invoiceRepo, subscriptionService)

	// Phase 2: Organization Service
	orgService := service.NewOrganizationService(orgRepo, subscriptionRepo, planRepo)
	orgService.SetFX(fxProvider, fxFallback, reportingCurrency)

	// Referral (P42)
	referralRepo := db.NewReferralRepository(dbx)
	referralService := service.NewReferralService(referralRepo, customerRepo)
	referralHandler := handler.NewReferralHandler(referralService)

	// Gift (P43)
	giftRepo := db.NewGiftRepository(dbx)
	giftService := service.NewGiftService(giftRepo, subscriptionRepo, invoiceService, planRepo, notificationService)
	giftHandler := handler.NewGiftHandler(giftService)

	// 6. Initialize Workers
	retryWorker := worker.NewRetryWorker(invoiceRepo, retryService, paymentGateway, notifier)
	// ENG-5 Phase 2: charge the customer's saved card off-session on retries.
	// The mock gateway doesn't implement it, so this stays disabled (interactive
	// fallback) until real Stripe keys are set.
	retryStripeCharger, _ := stripeGateway.(interface {
		ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error)
	})
	retryWorker.SetSavedMethodCharging(retryStripeCharger, customerRepo)
	retryWorker.SetDunningCampaignService(dunningCampaignService)
	retryWorker.SetRecoveryRecorder(dunningRecoveryService)
	// Successful retries settle through the same ledger-posting MarkInvoicePaid
	// as checkout and the payment webhooks (idempotent across all three).
	retryWorker.SetSettler(subscriptionService)
	webhookWorker := worker.NewWebhookWorker(eventDeliveryRepo, webhookEndpointRepo, eventRepo)
	churnWorker := worker.NewChurnWorker(churnService, customerRepo, tenantRepo, 24*time.Hour)
	revrecWorker := worker.NewRevRecWorker(revrecService, 24*time.Hour)

	// P25: E-Invoice Retry Worker
	einvoiceWorker := worker.NewEInvoiceRetryWorker(invoiceRepo, einvoiceService)

	// Dunning Campaign Worker
	dunningCampaignWorker := worker.NewDunningCampaignWorker(dunningCampaignService)

	// Phase 2: Accounting Sync Worker (daily). Token refresh happens inside
	// the accounting service per connection.
	acctSyncWorker := worker.NewAccountingSyncWorker(acctConnRepo, accountingService, 24*time.Hour)

	// Start Workers in Background. They all select on ctx.Done(), so a single
	// cancellable context tied to the shutdown signal (cancelWorkers below) lets
	// them stop their tick loops and drain in-flight work on SIGINT/SIGTERM
	// instead of being killed mid-operation.
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	// workersWG lets shutdown block until every worker's Start loop has
	// returned after workerCtx is cancelled, so main() doesn't exit (killing
	// the process and any in-flight tick) before workers finish draining.
	var workersWG sync.WaitGroup
	startWorker := func(start func(context.Context)) {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			start(workerCtx)
		}()
	}
	startWorker(retryWorker.Start)
	startWorker(webhookWorker.Start)
	startWorker(churnWorker.Start)
	startWorker(revrecWorker.Start)
	startWorker(einvoiceWorker.Start)
	startWorker(dunningCampaignWorker.Start)
	startWorker(acctSyncWorker.Start)

	// Distributed Locking & Redis. A working Redis makes the scheduler lock and
	// the idempotency store real across instances; without it the app falls back
	// to a no-op locker + per-instance in-memory store, which is only safe on a
	// single instance (see ENG-161). REQUIRE_REDIS lets a multi-instance
	// deployment refuse to start rather than silently run the unsafe fallback.
	requireRedis := strings.EqualFold(os.Getenv("REQUIRE_REDIS"), "true")
	var locker port.Locker
	var idempotencyStore port.IdempotencyStore
	var rdb *redis.Client

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opt, parseErr := redis.ParseURL(redisURL)
		if parseErr != nil {
			if requireRedis {
				log.Fatalf("REQUIRE_REDIS is set but REDIS_URL is invalid: %v", parseErr)
			}
			slog.Error("failed to parse REDIS_URL, falling back to in-memory", "error", parseErr)
		} else {
			client := redis.NewClient(opt)
			// redis.NewClient is lazy — PING so a dead/misconfigured Redis fails
			// loudly here instead of silently at the first Obtain/Get under load.
			pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			pingErr := client.Ping(pingCtx).Err()
			cancel()
			if pingErr != nil {
				_ = client.Close()
				if requireRedis {
					log.Fatalf("REQUIRE_REDIS is set but Redis is unreachable at REDIS_URL: %v", pingErr)
				}
				slog.Error("Redis unreachable, falling back to in-memory", "error", pingErr)
			} else {
				rdb = client
				locker = redisAdapter.NewRedisLocker(rdb)
				idempotencyStore = redisAdapter.NewRedisIdempotencyStore(rdb, 24*time.Hour)
				log.Println("Using Redis for Locker and Idempotency")
			}
		}
	} else if requireRedis {
		log.Fatal("REQUIRE_REDIS is set but REDIS_URL is empty; refusing to start with a no-op locker")
	}
	if locker == nil {
		locker = memory.NewNoOpLocker()
		idempotencyStore = memory.NewInMemoryIdempotencyStore(24 * time.Hour)
		log.Println("⚠️  Using In-Memory Locker and Idempotency (Redis not configured) — safe on a single instance only; set REDIS_URL for multi-instance")
	}

	// Pre-charge Scheduler (P30 - RBI compliance: 24hr notifications)
	preChargeScheduler := scheduler.NewPreChargeScheduler(
		subscriptionRepo.(*db.SubscriptionRepository),
		notificationService,
		locker,
		baseURL,
	)
	preChargeScheduler.Start()
	defer preChargeScheduler.Stop()

	// Dunning Scheduler (P30 - payment retry and escalation)
	dunningScheduler := scheduler.NewDunningScheduler(
		invoiceRepo.(*db.InvoiceRepository),
		notificationService,
		locker,
		scheduler.DefaultDunningConfig(),
		baseURL,
	)
	dunningScheduler.Start()
	defer dunningScheduler.Stop()

	// Trial Scheduler - sends trial-ending reminders and converts expired
	// trials to active (generating the first invoice, which flows into dunning).
	trialScheduler := scheduler.NewTrialScheduler(
		subscriptionRepo.(*db.SubscriptionRepository),
		subscriptionService,
		notificationService,
		locker,
		baseURL,
	)
	trialScheduler.Start()
	defer trialScheduler.Stop()

	// Card Expiry Scheduler - notifies customers ~30 days before card expires
	cardExpiryScheduler := scheduler.NewCardExpiringScheduler(
		customerRepo,
		notificationService,
		locker,
		baseURL,
	)
	cardExpiryScheduler.Start()
	defer cardExpiryScheduler.Stop()

	// Phase 2: Mandate Debit Scheduler (hourly)
	mandateDebitScheduler := scheduler.NewMandateDebitScheduler(mandateRepo, mandateService, locker)
	mandateDebitScheduler.Start()
	defer mandateDebitScheduler.Stop()

	// Billing Cycle Scheduler (Lago-parity A1): unattended renewal of
	// locally-billed subscriptions — invoice (flat + metered), anchor-
	// preserving period advance, best-effort saved-method payment.
	// BILLING_CYCLE_INTERVAL=0 disables; default 5m.
	renewalService := service.NewRenewalService(subscriptionRepo.(*db.SubscriptionRepository), planRepo, invoiceService)
	renewalCharger, _ := stripeGateway.(interface {
		ChargeSavedPaymentMethod(ctx context.Context, stripeCustomerID, paymentMethodID string, amount int64, currency, invoiceID, idempotencyKey string) (*port.PaymentResult, error)
	})
	renewalService.SetSavedMethodCharging(renewalCharger, customerRepo, subscriptionService)
	var billingCycleScheduler *scheduler.BillingCycleScheduler
	billingCycleInterval := 5 * time.Minute
	if raw := os.Getenv("BILLING_CYCLE_INTERVAL"); raw != "" {
		if raw == "0" {
			billingCycleInterval = 0
		} else if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			billingCycleInterval = d
		} else {
			log.Printf("Invalid BILLING_CYCLE_INTERVAL %q; using default 5m", raw)
		}
	}
	if billingCycleInterval > 0 {
		billingCycleScheduler = scheduler.NewBillingCycleScheduler(renewalService, locker, billingCycleInterval)
		billingCycleScheduler.Start()
		defer billingCycleScheduler.Stop()
	} else {
		log.Println("Billing cycle scheduler disabled (BILLING_CYCLE_INTERVAL=0)")
	}

	// US economic-nexus evaluation (daily): auto-establish nexus when a
	// state threshold is crossed (ENG-16 Phase 2).
	nexusStatusService := service.NewNexusStatusService(taxNexusRepo)
	nexusScheduler := scheduler.NewNexusScheduler(tenantRepo, nexusStatusService, locker)
	nexusScheduler.Start()
	defer nexusScheduler.Stop()

	// Ledger Reconciliation Scheduler (daily) — warns when ledger disagrees with billing records
	reconciliationScheduler := scheduler.NewReconciliationScheduler(tenantRepo, reconciliationService, locker)
	reconciliationScheduler.Start()
	defer reconciliationScheduler.Stop()

	// MRR Snapshot Scheduler (daily) — captures per-subscription MRR history so
	// the MRR waterfall (new/expansion/contraction/churned) has movement to diff.
	mrrSnapshotScheduler := scheduler.NewMRRSnapshotScheduler(tenantRepo, analyticsService, locker)
	mrrSnapshotScheduler.Start()
	defer mrrSnapshotScheduler.Stop()

	// Operational alerting (solo-operator safety net) — POSTs to
	// ALERT_WEBHOOK_URL on component state transitions; no-op when unset.
	// See docs/incident-runbook.md.
	alerter := alerting.NewFromEnv()
	if _, isNoop := alerter.(alerting.NoopAlerter); isNoop {
		log.Println("Alerting disabled (set ALERT_WEBHOOK_URL to enable health alerts)")
	} else {
		log.Println("Alerting enabled via ALERT_WEBHOOK_URL")
	}
	healthChecks := []scheduler.ComponentCheck{
		{
			Name:     "postgres",
			Severity: alerting.SeverityCritical, // system of record — money movement at risk
			Check:    func(ctx context.Context) error { return database.PingContext(ctx) },
		},
	}
	if rdb != nil { // mirror /health: redis only reported when configured
		redisClient := rdb
		healthChecks = append(healthChecks, scheduler.ComponentCheck{
			Name:     "redis",
			Severity: alerting.SeverityWarning, // optional: locking/rate-limit degrade
			Check:    func(ctx context.Context) error { return redisClient.Ping(ctx).Err() },
		})
	}
	// TigerBeetle's client has no liveness probe, so mirror /health exactly:
	// boot-time connection state. Disconnected at boot fires one warning on
	// the first evaluation, then stays silent (PG-only ledger mode is safe).
	tbConnected := tbClientForRecon != nil
	healthChecks = append(healthChecks, scheduler.ComponentCheck{
		Name:     "tigerbeetle",
		Severity: alerting.SeverityWarning, // optional accelerator; ledger is authoritative in PG
		Check: func(ctx context.Context) error {
			if !tbConnected {
				return errors.New("not connected at startup (ledger running PG-only)")
			}
			return nil
		},
	})
	healthAlertScheduler := scheduler.NewHealthAlertScheduler(alerter, healthChecks, 0) // 0 = 60s default
	healthAlertScheduler.Start()
	defer healthAlertScheduler.Stop()

	// Graceful shutdown: on SIGINT/SIGTERM stop schedulers, then drain the
	// HTTP server (srv.Shutdown below) so in-flight requests complete.
	// Stop() blocks until an in-flight tick finishes, so signal all
	// schedulers concurrently — done sequentially, one slow tick would keep
	// every scheduler behind it running (and starting new jobs) meanwhile.
	shutdownSchedulers := func() {
		stops := []func(){
			preChargeScheduler.Stop,
			dunningScheduler.Stop,
			trialScheduler.Stop,
			cardExpiryScheduler.Stop,
			mandateDebitScheduler.Stop,
			nexusScheduler.Stop,
			reconciliationScheduler.Stop,
			mrrSnapshotScheduler.Stop,
			healthAlertScheduler.Stop,
		}
		if billingCycleScheduler != nil {
			stops = append(stops, billingCycleScheduler.Stop)
		}
		var wg sync.WaitGroup
		for _, stop := range stops {
			wg.Add(1)
			go func(stop func()) {
				defer wg.Done()
				stop()
			}(stop)
		}
		wg.Wait()
	}

	// 7. Initialize Handlers
	catalogHandler := handler.NewCatalogHandler(catalogService)
	entitlementHandler := handler.NewEntitlementHandler(entitlementService) // Entitlement Engine v1
	customerHandler := handler.NewCustomerHandler(customerService, subscriptionRepo)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
	// Only the real Stripe gateway can verify a PaymentIntent server-side (the
	// mock can't), so type-assert for the inspector; a nil inspector makes
	// CheckoutSuccess report status only. subscriptionService is the ledger-path
	// settler shared with the webhook.
	checkoutInspector, _ := stripeGateway.(interface {
		GetPaymentStatus(ctx context.Context, orderID string) (*port.PaymentStatus, error)
	})
	checkoutHandler := handler.NewCheckoutHandler(invoiceRepo, paymentGateway, checkoutInspector, subscriptionService, os.Getenv("STRIPE_PUBLISHABLE_KEY"))
	// INR/Razorpay checkout verification (ENG-4 parity). The mock gateway lacks
	// GetOrderInvoiceID, so this stays disabled until real Razorpay keys are set.
	razorpayVerifier, _ := razorpayGateway.(interface {
		VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error
		GetOrderInvoiceID(ctx context.Context, orderID string) (string, error)
	})
	checkoutHandler.SetRazorpay(razorpayVerifier, os.Getenv("RAZORPAY_KEY_ID"))
	// Buyer name/address on Stripe intents — required by India-region accounts
	// for foreign-currency (export) charges; harmless elsewhere.
	checkoutBuyer, _ := stripeGateway.(interface {
		SetOrderBuyer(ctx context.Context, orderID, name, line1, city, state, zip, country string) error
	})
	checkoutHandler.SetBuyerDetails(customerRepo, checkoutBuyer)
	usageHandler := handler.NewUsageHandler(usageService)
	meteringHandler := handler.NewMeteringHandler(meteringService) // Usage-based billing v1
	// Phase 48: Unified Portal API Handler
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService, genaiService)
	couponHandler := handler.NewCouponHandler(couponRepo)    // P7
	tenantHandler := handler.NewTenantHandler(tenantService) // P8 Handler
	// Dashboard auth handlers. Cookies are marked Secure everywhere except
	// development so they still work over plain http://localhost.
	secureCookie := os.Getenv("APP_ENV") != "development"
	authHandler := handler.NewAuthHandler(authService, secureCookie)
	teamHandler := handler.NewTeamHandler(authService)

	// Phase 3 auth: native OAuth social login (Google + GitHub). A provider is
	// only enabled when BOTH its client id and secret are set; the registry
	// omits unconfigured providers (their endpoints 404 and /providers reports
	// them disabled). OAUTH_REDIRECT_BASE_URL is the API's public base used to
	// build each provider's redirect URL (defaults to BASE_URL).
	oauthRedirectBase := getEnvDefault("OAUTH_REDIRECT_BASE_URL", baseURL)
	oauthRegistry := service.NewOAuthRegistry(service.OAuthConfig{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectBaseURL:    oauthRedirectBase,
	})
	oauthIdentityRepo := db.NewOAuthIdentityRepository(database)
	authService.ConfigureOAuth(oauthIdentityRepo)
	// The state cookie is signed with OAUTH_STATE_SECRET; if unset a random
	// per-boot secret is used (in-flight logins started before a restart will
	// then fail state validation and safely redirect to the login error page).
	oauthStateSecret := []byte(os.Getenv("OAUTH_STATE_SECRET"))
	if len(oauthStateSecret) == 0 {
		oauthStateSecret = make([]byte, 32)
		if _, err := cryptorand.Read(oauthStateSecret); err != nil {
			log.Fatalf("failed to generate OAuth state secret: %v", err)
		}
		log.Println("OAUTH_STATE_SECRET not set — using an ephemeral per-boot secret")
	}
	oauthHandler := handler.NewOAuthHandler(authService, oauthRegistry, dashboardURL, oauthStateSecret, secureCookie)
	for _, st := range oauthRegistry.Statuses() {
		if st.Enabled {
			log.Printf("OAuth provider enabled: %s", st.Name)
		}
	}

	// Phase 3 auth: SAML SSO foundation (crewjam/saml). One IdP connection per
	// tenant; SP endpoints are per-tenant and feature-flagged (404 unless the
	// tenant's connection is enabled). The SP signing key/cert come from
	// SAML_SP_KEY / SAML_SP_CERT (PEM); if unset an ephemeral self-signed pair is
	// generated at boot (fine for bringing the SP up — a stable env pair is
	// recommended before certifying against a real IdP).
	ssoConnectionRepo := db.NewSSOConnectionRepository(database)
	ssoReplayStore := db.NewSSOAssertionReplayRepository(database)
	spKey, spCert, err := service.LoadOrGenerateSPKeyPair(os.Getenv("SAML_SP_KEY"), os.Getenv("SAML_SP_CERT"))
	if err != nil {
		log.Fatalf("failed to load SAML SP key/cert: %v", err)
	}
	if os.Getenv("SAML_SP_KEY") == "" || os.Getenv("SAML_SP_CERT") == "" {
		log.Println("SAML_SP_KEY/SAML_SP_CERT not set — generated an ephemeral self-signed SP certificate at boot")
	}
	ssoService := service.NewSSOService(ssoConnectionRepo, userRepo, ssoReplayStore, spKey, spCert, oauthRedirectBase)
	ssoHandler := handler.NewSSOHandler(ssoService, authService, dashboardURL, secureCookie)
	advancedBillingHandler := handler.NewAdvancedBillingHandler(advancedBillingService, invoiceService) // P15
	ledgerHandler := handler.NewLedgerHandler(ledgerService)                                            // P22
	reconciliationHandler := handler.NewReconciliationHandler(reconciliationService)                    // Ledger reconciliation
	creditNoteHandler := handler.NewCreditNoteHandler(creditNoteService)                                // P23
	webhookMgmtHandler := handler.NewWebhookManagementHandler(webhookService)                           // P24

	// Portal (P25)
	// PORTAL_URL is where the customer-facing portal SPA is served; magic
	// link emails point there. Defaults to the API base URL for dev.
	portalBaseURL := getEnvDefault("PORTAL_URL", baseURL)
	portalService := service.NewPortalService(customerRepo, invoiceRepo, magicLinkRepo, portalSessionRepo, disputeRepo, giftService, emailSender, portalBaseURL)
	portalAPIHandler := handler.NewPortalAPIHandler(portalService)
	// ENG-5: wire the Stripe SetupIntent card-update flow. The mock gateway
	// doesn't implement these methods, so the endpoints stay disabled until real
	// Stripe keys are set. customerRepo (concrete) provides the PM persistence.
	portalStripeSetup, _ := stripeGateway.(interface {
		EnsureStripeCustomer(ctx context.Context, existingID, email, name string) (string, error)
		CreateSetupIntent(ctx context.Context, stripeCustomerID string, metadata map[string]string) (string, error)
		FinalizeSetupIntent(ctx context.Context, setupIntentID string) (*port.SavedCard, error)
	})
	portalAPIHandler.SetPaymentMethodSetup(customerRepo, portalStripeSetup, os.Getenv("STRIPE_PUBLISHABLE_KEY"))
	// ENG-5 Phase 3a: portal UPI-mandate re-authorization. Gated on real
	// Razorpay keys — the mock gateway's AuthURL would strand customers on a
	// fake authorization page.
	if os.Getenv("RAZORPAY_KEY_ID") != "" {
		portalAPIHandler.SetMandateReauth(customerRepo, mandateService, invoiceRepo)
	}

	// Invoice disputes (Track 2): admin-facing API; portal-facing raise/list
	// lives on the portal handler above.
	disputeService := service.NewDisputeService(disputeRepo)
	disputeHandler := handler.NewDisputeHandler(disputeService)

	// Quotes (P27)
	quoteService := service.NewQuoteService(quoteRepo, invoiceRepo)
	quoteHandler := handler.NewQuoteHandler(quoteService)

	// GST & PDF (P30)
	pdfService := service.NewInvoicePDFService(
		getEnvDefault("PDF_COMPANY_NAME", "Your Company Name"),
		getEnvDefault("PDF_COMPANY_ADDRESS", "123 Business Street, City, State - 000000"),
		getEnvDefault("PDF_COMPANY_GSTIN", ""),
		getEnvDefault("PDF_COMPANY_PAN", ""),
		getEnvDefault("PDF_COMPANY_STATE", ""),
		getEnvDefault("PDF_BANK_DETAILS", "Bank: HDFC Bank\nAccount: 00000000000000\nIFSC: HDFC0000000"),
		getEnvDefault("PDF_COMPANY_COUNTRY", companyCountry),
		getEnvDefault("PDF_COMPANY_TAX_ID", ""),
	)
	pdfHandler := handler.NewInvoicePDFHandler(pdfService, invoiceRepo, customerRepo)
	// The concrete invoice repository implements the GSTR-1 read side; assert to
	// the narrow source interface so the export service stays db-agnostic.
	var gstrService *service.GSTRService
	if src, ok := invoiceRepo.(service.GSTR1Source); ok {
		gstrService = service.NewGSTRService(src)
	}
	gstHandler := handler.NewGSTHandler(gstConfigRepo, gstrService)
	taxNexusHandler := handler.NewTaxNexusHandler(taxNexusRepo)
	taxNexusHandler.SetStatusService(nexusStatusService)
	einvoiceHandler := handler.NewEInvoiceHandler(einvoiceService, irpConfigRepo)

	// Consent Service & Handler (P30 - RBI compliance)
	consentRepo := db.NewConsentRepository(database)
	consentService := service.NewConsentService(consentRepo)
	consentHandler := handler.NewConsentHandler(consentService)

	// Cancellation Handler (P30 - easy cancellation)
	cancellationHandler := handler.NewCancellationHandler(subscriptionService, consentService, notificationService)

	// Dunning Analytics
	dunningAnalyticsSvc := service.NewDunningAnalyticsService(dunningRepo)
	dunningHandler := handler.NewDunningHandler(dunningAnalyticsSvc, dunningRecoveryService)

	// Phase 2: New Handlers
	mandateHandler := handler.NewMandateHandler(mandateService)
	offlinePaymentHandler := handler.NewOfflinePaymentHandler(offlinePaymentService)
	orgHandler := handler.NewOrganizationHandler(orgService)
	accountingHandler := handler.NewAccountingHandler(acctConnRepo, accountingService, oauthStateSecret)
	churnHandler := handler.NewChurnHandler(churnService, database)

	// Payment Handlers
	paymentHandler := handler.NewPaymentHandler(paymentGateway, invoiceRepo)
	webhookHandler := handler.NewWebhookHandler(subscriptionService, paymentGateway, retryService, invoiceRepo, subscriptionRepo, customerRepo, notificationService, os.Getenv("STRIPE_WEBHOOK_SECRET"))
	webhookHandler.SetMandateService(mandateService)
	webhookHandler.SetOfflinePaymentService(offlinePaymentService)
	webhookHandler.SetDunningCampaignService(dunningCampaignService)
	webhookHandler.SetCreditNoteService(creditNoteService)                          // consume gateway refund events (refund.processed/failed, charge.refunded)
	webhookHandler.SetInboundWebhookDedup(db.NewInboundWebhookRepository(database)) // skip redelivered gateway webhooks (ENG-162)

	// Revenue Recognition Handler
	revrecHandler := handler.NewRevRecHandler(revrecService)

	// Cancel Flow & Dunning Campaign Handlers
	cancelFlowHandler := handler.NewCancelFlowHandler(cancelFlowService)
	dunningCampaignHandler := handler.NewDunningCampaignHandler(dunningCampaignService)

	// 8. Setup Router
	r := gin.Default()

	// Client IP must not be spoofable. gin.Default() trusts ALL proxies
	// (0.0.0.0/0), so it reads a client-supplied X-Forwarded-For — letting
	// anyone reset the per-IP rate limiter (500/min global, 20/min on public
	// auth endpoints) by sending a random XFF, defeating login/forgot-password/
	// register brute-force protection. Trust only real proxy CIDRs: loopback +
	// RFC-1918 private ranges by default (matches the nginx-in-front docker
	// deployment and is unspoofable by public clients), overridable via
	// TRUSTED_PROXIES (comma-separated CIDRs) for a different ingress. Set
	// TRUSTED_PROXIES to a single space (or configure your LB's egress CIDR) if
	// the app sits behind a public-IP load balancer.
	trustedProxies := []string{"127.0.0.0/8", "::1/128", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	if tp := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); tp != "" {
		trustedProxies = trustedProxies[:0]
		for _, p := range strings.Split(tp, ",") {
			if s := strings.TrimSpace(p); s != "" {
				trustedProxies = append(trustedProxies, s)
			}
		}
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES %v: %v", trustedProxies, err)
	}

	// Global Middleware (Phase 47)
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecureMiddleware())
	// Rate limit (per key/IP): RATE_LIMIT_PER_MINUTE, default 500.
	rateLimit, _ := strconv.Atoi(getEnvDefault("RATE_LIMIT_PER_MINUTE", "500"))
	if rateLimit <= 0 {
		rateLimit = 500
	}
	r.Use(middleware.RateLimitMiddleware(rdb, rateLimit, time.Minute))

	// CORS Middleware — comma-separated allowlist. Multiple origins matter:
	// the dashboard and the marketing site (whose waitlist form POSTs here)
	// are different origins. The matching request Origin is echoed back —
	// never "*", since credentials are allowed.
	corsEnv := os.Getenv("CORS_ORIGIN")
	if corsEnv == "" {
		corsEnv = "http://localhost:5173,http://localhost:5174" // Vite dev defaults
	}
	allowedOrigins := map[string]bool{}
	for _, o := range strings.Split(corsEnv, ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowedOrigins[o] = true
		}
	}
	r.Use(func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" && allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Idempotency-Key, X-Portal-Session, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.LoadHTMLGlob("internal/adapter/templates/*.html")

	r.GET("/health", func(c *gin.Context) {
		status := "ok"
		httpStatus := http.StatusOK
		components := gin.H{}

		// Check Postgres. Never return the raw error — /health is public and
		// unauthenticated, and a connection error can leak the host/port (and
		// sometimes credentials) from the DSN. Log it server-side; expose only
		// the component status.
		if err := database.Ping(); err != nil {
			slog.Error("health check: postgres ping failed", "error", err)
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
			components["postgres"] = gin.H{"status": "down"}
		} else {
			components["postgres"] = gin.H{"status": "up"}
		}

		// Check Redis (same info-disclosure stance as Postgres above).
		if rdb != nil {
			if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
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
		if tbConnected {
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

	// gateway_mode drives the dashboard's "Test mode" chip: "test" when any
	// configured gateway key is a test key, "live" when keys are live-only,
	// "none" when no real gateway is configured (mock).
	gatewayMode := "none"
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	razorpayKey := os.Getenv("RAZORPAY_KEY_ID")
	if stripeKey != "" || razorpayKey != "" {
		if strings.HasPrefix(stripeKey, "sk_test") || strings.HasPrefix(razorpayKey, "rzp_test") {
			gatewayMode = "test"
		} else {
			gatewayMode = "live"
		}
	}
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": version, "gateway_mode": gatewayMode})
	})

	// OpenAPI specification (public): GET /openapi.yaml, GET /openapi.json
	if err := registerOpenAPIRoutes(r); err != nil {
		log.Fatalf("Failed to register OpenAPI routes: %v", err)
	}

	// Public Routes — stricter rate limit (20 req/min per IP)
	publicLimit := middleware.RateLimitMiddleware(rdb, 20, time.Minute)

	r.GET("/checkout/:id", publicLimit, checkoutHandler.ShowCheckout)
	r.POST("/checkout/:id/pay", publicLimit, checkoutHandler.InitiatePayment)
	r.GET("/checkout/:id/success", publicLimit, checkoutHandler.CheckoutSuccess)
	r.POST("/checkout/:id/razorpay/verify", publicLimit, checkoutHandler.RazorpayVerify)
	r.POST("/payments/order", publicLimit, paymentHandler.CreateOrder)
	// Recurso Cloud waitlist (ENG-12): public demand capture from the website.
	r.POST("/waitlist", publicLimit, handler.NewWaitlistHandler(db.NewWaitlistRepository(database)).Join)
	r.POST("/webhooks/razorpay", webhookHandler.HandleRazorpay) // Webhooks need higher limits
	r.POST("/webhooks/stripe", webhookHandler.HandleStripe)
	// Dashboard auth (public): register creates tenant + owner user + session;
	// login/logout/me operate purely on the recurso_session cookie.
	r.POST("/auth/register", publicLimit, authHandler.Register)
	r.POST("/auth/login", publicLimit, authHandler.Login)
	r.POST("/auth/login/mfa", publicLimit, authHandler.LoginMFA)
	r.POST("/auth/logout", publicLimit, authHandler.Logout)
	r.GET("/auth/me", publicLimit, authHandler.Me)
	// Password reset (public): forgot-password always answers generically; the
	// reset itself consumes a single-use emailed token.
	r.POST("/auth/forgot-password", publicLimit, authHandler.ForgotPassword)
	r.POST("/auth/reset-password", publicLimit, authHandler.ResetPassword)

	// OAuth social login (public). /providers reflects which providers are
	// configured; /start issues the CSRF-state + PKCE cookie and redirects to
	// the provider; /callback validates, find-or-creates a user, opens a session
	// and redirects to the dashboard. Disabled/unknown providers 404.
	r.GET("/auth/oauth/providers", publicLimit, oauthHandler.Providers)
	r.GET("/auth/oauth/:provider/start", publicLimit, oauthHandler.Start)
	r.GET("/auth/oauth/:provider/callback", publicLimit, oauthHandler.Callback)

	// SAML SSO SP endpoints (public, per-tenant by UUID). metadata renders the
	// SP descriptor; login 302s to the IdP when enabled; acs consumes the
	// SAMLResponse, maps to an existing tenant user (no JIT), opens a session.
	r.GET("/auth/saml/:tenantID/metadata", publicLimit, ssoHandler.Metadata)
	r.GET("/auth/saml/:tenantID/login", publicLimit, ssoHandler.Login)
	r.POST("/auth/saml/:tenantID/acs", publicLimit, ssoHandler.ACS)

	// Customer Portal Auth (P25)
	r.POST("/portal/auth/request", publicLimit, portalAPIHandler.RequestMagicLink)
	r.GET("/portal/auth/verify", publicLimit, portalAPIHandler.VerifyMagicLink)

	// Protected Customer Portal Routes
	portal := r.Group("/portal/api")
	portal.Use(middleware.PortalAuthMiddleware(portalService))
	{
		portal.GET("/profile", portalAPIHandler.GetProfile)
		portal.GET("/invoices", portalAPIHandler.GetInvoices)
		// Customer-scoped invoice PDF (ownership-checked in the handler) so the
		// portal's Download-PDF button has a public, token-authed endpoint (ENG-152).
		portal.GET("/invoices/:id/pdf", pdfHandler.PortalDownloadPDF)
		portal.PUT("/payment-method", portalAPIHandler.UpdatePaymentMethod)
		portal.POST("/payment-method/setup-intent", portalAPIHandler.StartPaymentMethodSetup)
		portal.POST("/payment-method/confirm", portalAPIHandler.ConfirmPaymentMethod)
		portal.POST("/payment-method/mandate", portalAPIHandler.StartMandateReauth)
		portal.GET("/disputes", portalAPIHandler.GetDisputes)
		portal.POST("/invoices/:id/dispute", portalAPIHandler.RaiseDispute)
		portal.POST("/redeem", portalAPIHandler.RedeemGift)
		portal.POST("/logout", portalAPIHandler.Logout)
	}

	// Protected Routes (dashboard session cookie OR tenant API key — both
	// resolve to the same tenant_id, so every handler below is unchanged).
	// serverLive gates API keys by mode: live keys require a live-gateway
	// server, test keys require a non-live one.
	serverLive := gatewayMode == "live"
	v1 := r.Group("/v1")
	v1.Use(middleware.SessionOrAPIKeyMiddleware(tenantRepo, authService, serverLive))
	v1.Use(middleware.IdempotencyMiddleware(idempotencyStore)) // P30: Idempotency
	{
		v1.POST("/plans", catalogHandler.CreatePlan)
		v1.GET("/plans", catalogHandler.ListPlans)

		// Entitlement Engine v1
		v1.PUT("/plans/:id/entitlements", entitlementHandler.SetPlanEntitlements)
		v1.GET("/plans/:id/entitlements", entitlementHandler.GetPlanEntitlements)
		v1.GET("/customers/:id/entitlements", entitlementHandler.GetCustomerEntitlements)
		v1.GET("/entitlements/check", entitlementHandler.CheckEntitlement)

		v1.POST("/customers", customerHandler.CreateCustomer)
		v1.GET("/customers", customerHandler.ListCustomers)
		v1.PUT("/customers/:id/payment-method", customerHandler.UpdatePaymentMethod)

		v1.POST("/subscriptions", subscriptionHandler.CreateSubscription)
		v1.PUT("/subscriptions/:id", subscriptionHandler.UpdateSubscription)
		v1.GET("/subscriptions/:id/preview-change", subscriptionHandler.PreviewPlanChange)
		// Multi-product catalog v1: subscription add-ons
		v1.POST("/subscriptions/:id/addons", subscriptionHandler.AddAddon)
		v1.GET("/subscriptions/:id/addons", subscriptionHandler.ListAddons)
		v1.DELETE("/subscriptions/:id/addons/:addonId", subscriptionHandler.RemoveAddon)
		v1.GET("/subscriptions", subscriptionHandler.ListSubscriptions)
		v1.GET("/invoices", subscriptionHandler.ListInvoices)
		// Invoice PDF is tenant-scoped: it renders the buyer's legal name,
		// address, and GSTIN, so it must never be publicly fetchable by UUID.
		v1.GET("/invoices/:id/pdf", pdfHandler.DownloadPDF)
		v1.GET("/invoices/:id/preview", pdfHandler.PreviewHTML)

		// Usage Platform v1
		v1.POST("/usage/events", usageHandler.RecordEvent)
		v1.GET("/usage", usageHandler.QueryUsage)                             // time-windowed buckets
		v1.GET("/usage/dimensions", usageHandler.ListDimensions)              // dimension catalog
		v1.GET("/subscriptions/:id/usage", usageHandler.GetSubscriptionUsage) // current period + lifetime

		// Usage-based billing v1 (spec_usage_billing.md)
		v1.POST("/billable-metrics", meteringHandler.CreateMetric)
		v1.GET("/billable-metrics", meteringHandler.ListMetrics)
		v1.GET("/billable-metrics/:id", meteringHandler.GetMetric)
		v1.PUT("/billable-metrics/:id", meteringHandler.UpdateMetric)
		v1.DELETE("/billable-metrics/:id", meteringHandler.DeleteMetric)
		v1.PUT("/plans/:id/charges", meteringHandler.SetPlanCharges)
		v1.GET("/plans/:id/charges", meteringHandler.GetPlanCharges)
		v1.GET("/subscriptions/:id/usage-amount", meteringHandler.GetUsageAmount) // live pre-invoice preview

		// Analytics (Cached)
		analytics := v1.Group("/analytics")
		analytics.Use(middleware.CacheMiddleware(rdb, 5*time.Minute))
		{
			analytics.GET("/mrr", analyticsHandler.GetMRR)
			analytics.GET("/mrr/waterfall", analyticsHandler.GetMRRWaterfall)
			analytics.GET("/invoice-aging", analyticsHandler.GetInvoiceAging)
			analytics.GET("/unit-economics", analyticsHandler.GetUnitEconomics)
			analytics.GET("/revenue-by-plan", analyticsHandler.GetRevenueByPlan)
			analytics.GET("/revenue-by-geography", analyticsHandler.GetRevenueByGeography)
			analytics.GET("/usage", analyticsHandler.GetUsageStats)
			analytics.GET("/dunning/overview", dunningHandler.GetOverview)
			analytics.GET("/dunning/weights", dunningHandler.GetWeights)
			analytics.GET("/dunning/history", dunningHandler.GetHistory)
			analytics.GET("/dunning/recovered", dunningHandler.GetRecovered)
		}
		v1.POST("/analytics/ask", analyticsHandler.Ask) // P48 GenAI

		v1.POST("/coupons", couponHandler.CreateCoupon) // P7
		v1.GET("/coupons", couponHandler.ListCoupons)

		// Developer / Settings
		v1.GET("/developer/keys", tenantHandler.ListKeys)
		v1.POST("/developer/keys", tenantHandler.CreateKey)

		// Team management (dashboard users). Reads are open to any authed
		// member; writes are gated to owner/admin inside the handler.
		v1.GET("/users", teamHandler.ListUsers)
		v1.POST("/users", teamHandler.CreateUser)
		v1.POST("/users/invite", teamHandler.InviteUser)
		v1.PATCH("/users/:id", teamHandler.UpdateUser)
		v1.DELETE("/users/:id", teamHandler.DeleteUser)

		// Account security for the logged-in dashboard user (TOTP MFA + active
		// session management). API-key callers have no user and are rejected.
		v1.POST("/auth/mfa/setup", authHandler.MFASetup)
		v1.POST("/auth/mfa/verify", authHandler.MFAVerify)
		v1.POST("/auth/mfa/disable", authHandler.MFADisable)
		v1.GET("/auth/sessions", authHandler.ListSessions)
		v1.DELETE("/auth/sessions/:id", authHandler.RevokeSession)
		v1.DELETE("/auth/sessions", authHandler.RevokeOtherSessions)

		// SAML SSO connection config (tenant-scoped; writes gated to owner/admin
		// inside the handler). The public SP endpoints live under /auth/saml.
		v1.GET("/sso/connection", ssoHandler.GetConnection)
		v1.PUT("/sso/connection", ssoHandler.UpsertConnection)
		v1.DELETE("/sso/connection", ssoHandler.DeleteConnection)

		// Advanced Billing (P15)
		v1.POST("/subscriptions/:id/charges", advancedBillingHandler.AddUnbilledCharge)
		v1.GET("/subscriptions/:id/charges", advancedBillingHandler.ListUnbilledCharges)
		v1.POST("/subscriptions/:id/advance", advancedBillingHandler.GenerateAdvanceInvoice)

		// Ledger (P22)
		v1.GET("/ledger/accounts", ledgerHandler.ListAccounts)
		v1.GET("/ledger/entries", ledgerHandler.GetEntries)
		// Provable-ledger auditor outputs (ENG-192): trial balance + GL export
		v1.GET("/ledger/trial-balance", ledgerHandler.GetTrialBalance)
		v1.GET("/ledger/export", ledgerHandler.ExportGL)
		v1.GET("/ledger/deferred-rollforward", ledgerHandler.GetDeferredRollforward)

		// Ledger Reconciliation — on-demand drift report for the caller's tenant
		v1.GET("/finance/reconciliation", reconciliationHandler.RunReconciliation)

		// Credit Notes (P23)
		v1.POST("/credit-notes", creditNoteHandler.CreateCreditNote)
		v1.GET("/credit-notes", creditNoteHandler.ListCreditNotes)

		// Webhooks & Events (P24)
		v1.POST("/webhooks", webhookMgmtHandler.CreateEndpoint)
		v1.GET("/webhooks", webhookMgmtHandler.ListEndpoints)
		v1.DELETE("/webhooks/:id", webhookMgmtHandler.DeleteEndpoint)
		v1.GET("/webhooks/:id/deliveries", webhookMgmtHandler.ListEndpointDeliveries)

		// Account (Tenant) Management
		v1.GET("/account", tenantHandler.GetAccount)
		v1.PUT("/account", tenantHandler.UpdateAccount)

		// Quotes (P27)
		v1.POST("/quotes", quoteHandler.CreateQuote)
		v1.GET("/quotes", quoteHandler.ListQuotes)
		v1.GET("/quotes/:id", quoteHandler.GetQuote)
		v1.PUT("/quotes/:id", quoteHandler.UpdateQuote)
		v1.DELETE("/quotes/:id", quoteHandler.DeleteQuote)
		v1.POST("/quotes/:id/send", quoteHandler.SendQuote)
		v1.POST("/quotes/:id/accept", quoteHandler.AcceptQuote)
		v1.POST("/quotes/:id/decline", quoteHandler.DeclineQuote)
		v1.POST("/quotes/:id/convert", quoteHandler.ConvertToInvoice)

		// Invoice disputes (Track 2) — admin API only; no dashboard UI yet.
		v1.GET("/disputes", disputeHandler.ListDisputes)
		v1.POST("/disputes/:id/resolve", disputeHandler.ResolveDispute)

		v1.GET("/events", webhookMgmtHandler.ListEvents)
		v1.GET("/events/types", webhookMgmtHandler.GetEventTypes)
		v1.GET("/events/:id/deliveries", webhookMgmtHandler.ListEventDeliveries)
		v1.POST("/events/:id/redeliver", webhookMgmtHandler.RedeliverEvent)

		// GST Settings (P30)
		v1.GET("/settings/gst", gstHandler.GetConfig)
		v1.PUT("/settings/gst", gstHandler.UpdateConfig)
		// US sales-tax nexus config
		v1.GET("/settings/tax/nexus", taxNexusHandler.GetNexus)
		v1.PUT("/settings/tax/nexus", taxNexusHandler.SetNexus)
		v1.GET("/settings/tax/nexus/status", taxNexusHandler.GetNexusStatus)
		v1.POST("/settings/gst/validate", gstHandler.ValidateGSTIN)
		v1.GET("/india/gstr1", gstHandler.GetGSTR1)
		v1.GET("/india/gstr3b", gstHandler.GetGSTR3B)

		// E-Invoice (P25)
		v1.GET("/invoices/:id/einvoice", einvoiceHandler.GetEInvoiceStatus)
		v1.POST("/invoices/:id/einvoice/retry", einvoiceHandler.RetryEInvoice)
		v1.POST("/invoices/:id/einvoice/cancel", einvoiceHandler.CancelEInvoice)
		v1.GET("/settings/irp", einvoiceHandler.GetIRPConfig)
		v1.PUT("/settings/irp", einvoiceHandler.UpdateIRPConfig)
		v1.POST("/settings/irp/test", einvoiceHandler.TestIRPConnection)

		// Consent API (P30 - RBI compliance)
		v1.POST("/consents", consentHandler.RecordConsent)
		v1.POST("/consents/revoke", consentHandler.RevokeConsent)
		v1.GET("/customers/:id/consents", consentHandler.GetCustomerConsents)
		v1.GET("/subscriptions/:id/consent", consentHandler.GetSubscriptionConsent)

		// Cancellation API (P30 - easy cancellation)
		v1.POST("/subscriptions/:id/cancel", cancellationHandler.CancelSubscription)
		v1.POST("/subscriptions/:id/reactivate", cancellationHandler.ReactivateSubscription)
		v1.POST("/subscriptions/:id/pause", subscriptionHandler.PauseSubscription)
		v1.POST("/subscriptions/:id/resume", subscriptionHandler.ResumeSubscription)
		v1.GET("/cancellation-reasons", cancellationHandler.GetCancellationReasons)

		// Referral & Gift API
		v1.GET("/referrals", referralHandler.ListReferrals)
		v1.POST("/referrals", referralHandler.CreateReferral)
		v1.POST("/referrals/generate-code", referralHandler.GenerateCode)
		v1.POST("/referrals/:id/qualify", referralHandler.QualifyReferral)
		v1.GET("/gifts", giftHandler.ListGifts)

		// Gift API (P43)
		v1.POST("/gifts/purchase", giftHandler.PurchaseGift)
		v1.POST("/gifts/redeem", giftHandler.RedeemGift)

		// Phase 2: UPI Mandates
		v1.POST("/mandates", mandateHandler.CreateMandate)
		v1.GET("/mandates", mandateHandler.ListMandates)
		v1.GET("/mandates/:id", mandateHandler.GetMandate)
		v1.POST("/mandates/:id/revoke", mandateHandler.RevokeMandate)

		// Phase 2: Offline Payments / Virtual Accounts
		v1.POST("/virtual-accounts", offlinePaymentHandler.CreateVirtualAccount)
		v1.GET("/virtual-accounts", offlinePaymentHandler.ListVirtualAccounts)
		v1.POST("/payments/offline", offlinePaymentHandler.RecordOfflinePayment)
		v1.GET("/payments/offline", offlinePaymentHandler.ListOfflinePayments)

		// Revenue Recognition Report
		v1.GET("/finance/revrec/report", revrecHandler.GetReport)
		v1.GET("/finance/revrec/waterfall", revrecHandler.GetWaterfall)

		// Phase 2: Organizations (Multi-Entity)
		v1.GET("/organizations", orgHandler.ListOrganizations)
		v1.POST("/organizations", orgHandler.CreateOrganization)
		v1.GET("/organizations/:id", orgHandler.GetOrganization)
		v1.PUT("/organizations/:id", orgHandler.UpdateOrganization)
		v1.DELETE("/organizations/:id", orgHandler.DeleteOrganization)
		v1.POST("/organizations/:id/tenants", orgHandler.AddTenant)
		v1.GET("/organizations/:id/tenants", orgHandler.ListTenants)
		v1.DELETE("/organizations/:id/tenants/:tenant_id", orgHandler.RemoveTenant)
		v1.GET("/organizations/:id/analytics/mrr", orgHandler.GetConsolidatedMRR)

		// Phase 2: Accounting / ERP Integrations
		v1.GET("/accounting/connections", accountingHandler.ListConnections)
		v1.POST("/accounting/connect/:provider", accountingHandler.InitiateOAuth)
		v1.GET("/accounting/callback/:provider", accountingHandler.OAuthCallback)
		v1.DELETE("/accounting/connections/:id", accountingHandler.Disconnect)
		v1.POST("/accounting/sync", accountingHandler.TriggerSync)
		v1.GET("/accounting/sync/status", accountingHandler.SyncStatus)

		// Phase 2: Churn Scoring
		v1.GET("/customers/:id/churn", churnHandler.GetCustomerChurn)
		v1.GET("/churn/high-risk", churnHandler.GetHighRiskCustomers)
		v1.GET("/churn/alerts", churnHandler.GetAlerts)
		v1.POST("/churn/alerts/:id/ack", churnHandler.AcknowledgeAlert)

		// Cancel Flows (Retention Interventions)
		v1.GET("/cancel-flows", cancelFlowHandler.ListFlows)
		v1.POST("/cancel-flows", cancelFlowHandler.CreateFlow)
		v1.GET("/cancel-flows/:id", cancelFlowHandler.GetFlow)
		v1.PUT("/cancel-flows/:id", cancelFlowHandler.UpdateFlow)
		v1.POST("/cancel-flows/:id/steps", cancelFlowHandler.CreateStep)
		v1.PUT("/cancel-flows/steps/:id", cancelFlowHandler.UpdateStep)
		v1.DELETE("/cancel-flows/steps/:id", cancelFlowHandler.DeleteStep)
		v1.POST("/cancel-flows/sessions/start", cancelFlowHandler.StartSession)
		v1.POST("/cancel-flows/sessions/:id/submit", cancelFlowHandler.SubmitStep)
		v1.GET("/cancel-flows/sessions/:id", cancelFlowHandler.GetSession)
		v1.GET("/cancel-flows/stats", cancelFlowHandler.GetStats)

		// Dunning Campaigns (Multi-Channel)
		v1.GET("/dunning-campaigns", dunningCampaignHandler.ListCampaigns)
		v1.POST("/dunning-campaigns", dunningCampaignHandler.CreateCampaign)
		v1.GET("/dunning-campaigns/:id", dunningCampaignHandler.GetCampaign)
		v1.PUT("/dunning-campaigns/:id", dunningCampaignHandler.UpdateCampaign)
		v1.POST("/dunning-campaigns/:id/steps", dunningCampaignHandler.CreateStep)
		v1.PUT("/dunning-campaigns/steps/:id", dunningCampaignHandler.UpdateStep)
		v1.DELETE("/dunning-campaigns/steps/:id", dunningCampaignHandler.DeleteStep)
		v1.GET("/invoices/:id/payment-wall", dunningCampaignHandler.GetPaymentWallStatus)
	}

	// 9. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverAddr := fmt.Sprintf(":%s", port)
	srv := &http.Server{Addr: serverAddr, Handler: r}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down gracefully...")
		shutdownSchedulers()
		cancelWorkers() // stop the background worker tick loops and drain in-flight work

		// Wait for the worker goroutines to actually return, bounded so a stuck
		// worker can't hang shutdown forever. Each worker's in-flight I/O is
		// already time-bounded (e.g. the webhook client's 10s HTTP timeout), so
		// this drain typically completes well inside the budget.
		workersDone := make(chan struct{})
		go func() {
			workersWG.Wait()
			close(workersDone)
		}()
		select {
		case <-workersDone:
			log.Println("Background workers drained.")
		case <-time.After(15 * time.Second):
			log.Println("Timed out waiting for background workers to drain; exiting anyway.")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting Recurso API on %s", serverAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("Server stopped")
}

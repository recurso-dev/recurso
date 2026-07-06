package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/swapnull-in/recur-so/internal/adapter/accounting"
	"github.com/swapnull-in/recur-so/internal/adapter/ai"
	"github.com/swapnull-in/recur-so/internal/adapter/alerting"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/adapter/email"
	"github.com/swapnull-in/recur-so/internal/adapter/fx"
	"github.com/swapnull-in/recur-so/internal/adapter/gateway"
	"github.com/swapnull-in/recur-so/internal/adapter/gsp"
	"github.com/swapnull-in/recur-so/internal/adapter/handler"
	"github.com/swapnull-in/recur-so/internal/adapter/memory"
	"github.com/swapnull-in/recur-so/internal/adapter/middleware"
	"github.com/swapnull-in/recur-so/internal/adapter/notification"
	redisAdapter "github.com/swapnull-in/recur-so/internal/adapter/redis"
	"github.com/swapnull-in/recur-so/internal/adapter/sms"
	"github.com/swapnull-in/recur-so/internal/adapter/tigerbeetle"
	"github.com/swapnull-in/recur-so/internal/adapter/vault"
	"github.com/swapnull-in/recur-so/internal/adapter/worker"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/scheduler"
	"github.com/swapnull-in/recur-so/internal/service"
)

// version is stamped at build time via:
//
//	go build -ldflags "-X main.version=v0.1.0"
var version = "dev"

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
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
	couponRepo := db.NewCouponRepository(database)                   // P7
	tenantRepo := db.NewTenantRepository(database)                   // P8
	unbilledChargeRepo := db.NewUnbilledChargeRepository(database)   // P15
	webhookEndpointRepo := db.NewWebhookEndpointRepository(database) // P24
	eventRepo := db.NewEventRepository(database)                     // P24
	eventDeliveryRepo := db.NewEventDeliveryRepository(database)     // P24
	magicLinkRepo := db.NewMagicLinkRepository(database)             // P25
	portalSessionRepo := db.NewPortalSessionRepository(database)     // P25
	quoteRepo := db.NewQuoteRepository(database)                     // P27

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

	// 4. Initialize Core Services (Invoice)
	invoiceService := service.NewInvoiceService(invoiceRepo, planRepo, customerRepo, unbilledChargeRepo, subscriptionRepo, gspAdapter, taxResolver) // P15, P25

	catalogService := service.NewCatalogService(planRepo)
	entitlementService := service.NewEntitlementService(entitlementRepo, planRepo, customerRepo, subscriptionRepo) // Entitlement Engine v1
	customerService := service.NewCustomerService(customerRepo)
	tenantService := service.NewTenantService(tenantRepo)                                                        // P8 Service
	creditNoteService := service.NewCreditNoteService(creditNoteRepo, customerRepo, invoiceRepo, paymentGateway) // P23 + refunds
	creditNoteService.SetLedgerService(ledgerService)
	txManager := db.NewTxManager(database)

	// Revenue Recognition (P5)
	revrecRepo := db.NewRevRecRepository(database)
	revrecService := service.NewRevRecService(revrecRepo, ledgerService, subscriptionRepo)

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
	subscriptionService.SetRecoveryRecorder(dunningRecoveryService)

	// Analytics
	analyticsService := service.NewAnalyticsService(subscriptionRepo, invoiceRepo, planRepo, usageRepo)
	analyticsService.SetFX(fxProvider, fxFallback, reportingCurrency)
	analyticsService.SetTenantLookup(tenantRepo)

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
	qboConfigured := oauthConfigs["quickbooks"].ClientID != ""
	xeroConfigured := oauthConfigs["xero"].ClientID != ""
	switch {
	case qboConfigured && xeroConfigured:
		log.Println("Accounting sync configured for QuickBooks and Xero")
	case qboConfigured:
		log.Println("Accounting sync configured for QuickBooks (set XERO_CLIENT_ID/XERO_CLIENT_SECRET to enable Xero)")
	case xeroConfigured:
		log.Println("Accounting sync configured for Xero (set QBO_CLIENT_ID/QBO_CLIENT_SECRET to enable QuickBooks)")
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
	retryWorker.SetDunningCampaignService(dunningCampaignService)
	retryWorker.SetRecoveryRecorder(dunningRecoveryService)
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

	// Start Workers in Background
	go retryWorker.Start(context.Background())
	go webhookWorker.Start(context.Background())
	go churnWorker.Start(context.Background())
	go revrecWorker.Start(context.Background())
	go einvoiceWorker.Start(context.Background())
	go dunningCampaignWorker.Start(context.Background())
	go acctSyncWorker.Start(context.Background())

	// Distributed Locking & Redis
	var locker port.Locker
	var idempotencyStore port.IdempotencyStore
	var rdb *redis.Client

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opt, parseErr := redis.ParseURL(redisURL)
		if parseErr != nil {
			slog.Error("failed to parse REDIS_URL, falling back to in-memory", "error", parseErr)
		} else {
			rdb = redis.NewClient(opt)
			locker = redisAdapter.NewRedisLocker(rdb)
			idempotencyStore = redisAdapter.NewRedisIdempotencyStore(rdb, 24*time.Hour)
			log.Println("Using Redis for Locker and Idempotency")
		}
	}
	if locker == nil {
		locker = memory.NewNoOpLocker()
		idempotencyStore = memory.NewInMemoryIdempotencyStore(24 * time.Hour)
		log.Println("Using In-Memory Locker and Idempotency (Redis not configured)")
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

	// Ledger Reconciliation Scheduler (daily) — warns when ledger disagrees with billing records
	reconciliationScheduler := scheduler.NewReconciliationScheduler(tenantRepo, reconciliationService, locker)
	reconciliationScheduler.Start()
	defer reconciliationScheduler.Stop()

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
	shutdownSchedulers := func() {
		preChargeScheduler.Stop()
		dunningScheduler.Stop()
		cardExpiryScheduler.Stop()
		mandateDebitScheduler.Stop()
		reconciliationScheduler.Stop()
		healthAlertScheduler.Stop()
	}

	// 7. Initialize Handlers
	catalogHandler := handler.NewCatalogHandler(catalogService)
	entitlementHandler := handler.NewEntitlementHandler(entitlementService) // Entitlement Engine v1
	customerHandler := handler.NewCustomerHandler(customerService)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
	checkoutHandler := handler.NewCheckoutHandler(invoiceRepo, paymentGateway)
	usageHandler := handler.NewUsageHandler(usageRepo)
	// Phase 48: Unified Portal API Handler
	portalHandler := handler.NewPortalHandler(customerRepo, invoiceRepo, subscriptionService, invoiceService, customerService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService, genaiService)
	couponHandler := handler.NewCouponHandler(couponRepo)                                               // P7
	tenantHandler := handler.NewTenantHandler(tenantService)                                            // P8 Handler
	advancedBillingHandler := handler.NewAdvancedBillingHandler(advancedBillingService, invoiceService) // P15
	ledgerHandler := handler.NewLedgerHandler(ledgerService)                                            // P22
	reconciliationHandler := handler.NewReconciliationHandler(reconciliationService)                    // Ledger reconciliation
	creditNoteHandler := handler.NewCreditNoteHandler(creditNoteService)                                // P23
	webhookMgmtHandler := handler.NewWebhookManagementHandler(webhookService)                           // P24

	// Portal (P25)
	// PORTAL_URL is where the customer-facing portal SPA is served; magic
	// link emails point there. Defaults to the API base URL for dev.
	portalBaseURL := getEnvDefault("PORTAL_URL", baseURL)
	portalService := service.NewPortalService(customerRepo, invoiceRepo, magicLinkRepo, portalSessionRepo, giftService, emailSender, portalBaseURL)
	portalAPIHandler := handler.NewPortalAPIHandler(portalService)

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
	)
	pdfHandler := handler.NewInvoicePDFHandler(pdfService)
	gstHandler := handler.NewGSTHandler(gstConfigRepo)
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
	accountingHandler := handler.NewAccountingHandler(acctConnRepo, accountingService)
	churnHandler := handler.NewChurnHandler(churnService, database)

	// Payment Handlers
	paymentHandler := handler.NewPaymentHandler(paymentGateway, invoiceRepo)
	webhookHandler := handler.NewWebhookHandler(subscriptionService, paymentGateway, retryService, invoiceRepo, subscriptionRepo, customerRepo, notificationService, os.Getenv("STRIPE_WEBHOOK_SECRET"))
	webhookHandler.SetMandateService(mandateService)
	webhookHandler.SetOfflinePaymentService(offlinePaymentService)
	webhookHandler.SetDunningCampaignService(dunningCampaignService)
	webhookHandler.SetCreditNoteService(creditNoteService) // consume gateway refund events (refund.processed/failed, charge.refunded)

	// Revenue Recognition Handler
	revrecHandler := handler.NewRevRecHandler(revrecService)

	// Cancel Flow & Dunning Campaign Handlers
	cancelFlowHandler := handler.NewCancelFlowHandler(cancelFlowService)
	dunningCampaignHandler := handler.NewDunningCampaignHandler(dunningCampaignService)

	// 8. Setup Router
	r := gin.Default()

	// Global Middleware (Phase 47)
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecureMiddleware())
	// Rate limit (per key/IP): RATE_LIMIT_PER_MINUTE, default 500.
	rateLimit, _ := strconv.Atoi(getEnvDefault("RATE_LIMIT_PER_MINUTE", "500"))
	if rateLimit <= 0 {
		rateLimit = 500
	}
	r.Use(middleware.RateLimitMiddleware(rdb, rateLimit, time.Minute))

	// CORS Middleware - configurable origin
	allowedOrigin := os.Getenv("CORS_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173" // Vite dev server default
	}
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Idempotency-Key, accept, origin, Cache-Control, X-Requested-With")
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

		// Check Postgres
		if err := database.Ping(); err != nil {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
			components["postgres"] = gin.H{"status": "down", "error": err.Error()}
		} else {
			components["postgres"] = gin.H{"status": "up"}
		}

		// Check Redis
		if rdb != nil {
			if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
				components["redis"] = gin.H{"status": "down", "error": err.Error()}
				// Redis down is degraded, not critical
				if status == "ok" {
					status = "degraded"
				}
			} else {
				components["redis"] = gin.H{"status": "up"}
			}
		}

		// Check TigerBeetle
		if ledgerService != nil {
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

	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": version})
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
	r.GET("/portal/:customer_id", publicLimit, portalHandler.ShowDashboard)
	// Phase 48: Read-only unauthenticated portal data
	r.GET("/v1/portal/:tenant_id/:customer_id", publicLimit, portalHandler.GetPortalData)
	r.POST("/payments/order", publicLimit, paymentHandler.CreateOrder)
	r.POST("/webhooks/razorpay", webhookHandler.HandleRazorpay) // Webhooks need higher limits
	r.POST("/webhooks/stripe", webhookHandler.HandleStripe)
	r.POST("/auth/register", publicLimit, tenantHandler.Register) // P8 Register Endpoint

	// Invoice PDF Public (P30 for Demo)
	r.GET("/v1/invoices/:id/pdf", publicLimit, pdfHandler.DownloadPDF)
	r.GET("/v1/invoices/:id/preview", publicLimit, pdfHandler.PreviewHTML)

	// Customer Portal Auth (P25)
	r.POST("/portal/auth/request", publicLimit, portalAPIHandler.RequestMagicLink)
	r.GET("/portal/auth/verify", publicLimit, portalAPIHandler.VerifyMagicLink)

	// Protected Customer Portal Routes
	portal := r.Group("/portal/api")
	portal.Use(middleware.PortalAuthMiddleware(portalService))
	{
		portal.GET("/profile", portalAPIHandler.GetProfile)
		portal.GET("/invoices", portalAPIHandler.GetInvoices)
		portal.POST("/redeem", portalAPIHandler.RedeemGift)
		portal.POST("/logout", portalAPIHandler.Logout)
	}

	// Protected Routes (API Key)
	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(tenantRepo))
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
		v1.GET("/subscriptions", subscriptionHandler.ListSubscriptions)
		v1.GET("/invoices", subscriptionHandler.ListInvoices)

		v1.POST("/usage/events", usageHandler.RecordEvent)

		// Analytics (Cached)
		analytics := v1.Group("/analytics")
		analytics.Use(middleware.CacheMiddleware(rdb, 5*time.Minute))
		{
			analytics.GET("/mrr", analyticsHandler.GetMRR)
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

		// Advanced Billing (P15)
		v1.POST("/subscriptions/:id/charges", advancedBillingHandler.AddUnbilledCharge)
		v1.GET("/subscriptions/:id/charges", advancedBillingHandler.ListUnbilledCharges)
		v1.POST("/subscriptions/:id/advance", advancedBillingHandler.GenerateAdvanceInvoice)

		// Ledger (P22)
		v1.GET("/ledger/accounts", ledgerHandler.ListAccounts)
		v1.GET("/ledger/entries", ledgerHandler.GetEntries)

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
		v1.GET("/events", webhookMgmtHandler.ListEvents)
		v1.GET("/events/types", webhookMgmtHandler.GetEventTypes)
		v1.GET("/events/:id/deliveries", webhookMgmtHandler.ListEventDeliveries)
		v1.POST("/events/:id/redeliver", webhookMgmtHandler.RedeliverEvent)

		// GST Settings (P30)
		v1.GET("/settings/gst", gstHandler.GetConfig)
		v1.PUT("/settings/gst", gstHandler.UpdateConfig)
		v1.POST("/settings/gst/validate", gstHandler.ValidateGSTIN)

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

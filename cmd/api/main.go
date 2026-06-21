package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/recur-so/recurso/internal/adapter/accounting"
	"github.com/recur-so/recurso/internal/adapter/ai"
	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/adapter/email"
	"github.com/recur-so/recurso/internal/adapter/gateway"
	"github.com/recur-so/recurso/internal/adapter/gsp"
	"github.com/recur-so/recurso/internal/adapter/handler"
	"github.com/recur-so/recurso/internal/adapter/memory"
	"github.com/recur-so/recurso/internal/adapter/middleware"
	"github.com/recur-so/recurso/internal/adapter/notification"
	redisAdapter "github.com/recur-so/recurso/internal/adapter/redis"
	"github.com/recur-so/recurso/internal/adapter/tigerbeetle"
	"github.com/recur-so/recurso/internal/adapter/worker"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/scheduler"
	"github.com/recur-so/recurso/internal/service"
	"github.com/redis/go-redis/v9"
)

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
	defer database.Close()

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

	emailSender := email.NewConsoleSender() // Use SMTPSender in production
	notificationService := service.NewNotificationService(emailSender, "http://localhost:8080")
	// notificationService is wired to subscriptionService, webhookHandler, schedulers, and cancellationHandler below

	// Ledger (P5)
	// We wrap in try-catch in case TB is not running (since docker failed)
	var ledgerService *service.LedgerService
	ledgerClient, err := tigerbeetle.NewLedgerClient(0, []string{"127.0.0.1:3001"})
	if err == nil {
		ledgerService = service.NewLedgerService(ledgerClient)
		defer ledgerClient.Close()
	} else {
		slog.Warn("TigerBeetle not connected — ledger disabled", "error", err)
		ledgerService = service.NewLedgerService(nil)
	}

	// 5. Initialize Gateways
	// Using MockGateway as "Razorpay" for dev environment validation without keys
	razorpayGateway := gateway.NewMockGateway()

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

	// 4. Initialize Core Services (Invoice)
	invoiceService := service.NewInvoiceService(invoiceRepo, planRepo, customerRepo, unbilledChargeRepo, subscriptionRepo, gspAdapter) // P15, P25

	catalogService := service.NewCatalogService(planRepo)
	customerService := service.NewCustomerService(customerRepo)
	tenantService := service.NewTenantService(tenantRepo)                           // P8 Service
	creditNoteService := service.NewCreditNoteService(creditNoteRepo, customerRepo) // P23
	txManager := db.NewTxManager(database)

	// Revenue Recognition (P5)
	revrecRepo := db.NewRevRecRepository(database)
	revrecService := service.NewRevRecService(revrecRepo, ledgerService)

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
	)

	// P25: E-Invoice Service
	einvoiceService := service.NewEInvoiceService(gspAdapter, invoiceRepo, customerRepo, irpConfigRepo, gstConfigRepo)
	invoiceService.EInvoiceService = einvoiceService
	subscriptionService.SetEInvoiceService(einvoiceService)
	subscriptionService.SetNotificationService(notificationService)

	// AI Service (P45)
	dunningRepo := db.NewDunningRepository(database)
	retryService := service.NewSmartRetryService(dunningRepo)
	churnService := service.NewChurnService(customerRepo, invoiceRepo)

	// Analytics
	analyticsService := service.NewAnalyticsService(subscriptionRepo, invoiceRepo, planRepo, usageRepo)

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

	// Accounting (P41)
	accountingGateway := accounting.NewMockAccountingAdapter()
	accountingService := service.NewAccountingService(accountingGateway, customerRepo, invoiceRepo, planRepo)
	_ = accountingService // Ready to be used by handlers or workers

	// Referral (P42)
	referralRepo := db.NewReferralRepository(dbx)
	referralService := service.NewReferralService(referralRepo, customerRepo)
	referralHandler := handler.NewReferralHandler(referralService)

	// Gift (P43)
	giftRepo := db.NewGiftRepository(dbx)
	giftService := service.NewGiftService(giftRepo, subscriptionRepo, invoiceService, planRepo)
	giftHandler := handler.NewGiftHandler(giftService)

	// 6. Initialize Workers
	retryWorker := worker.NewRetryWorker(invoiceRepo, retryService, paymentGateway, notifier)
	webhookWorker := worker.NewWebhookWorker(eventDeliveryRepo, webhookEndpointRepo, eventRepo)
	churnWorker := worker.NewChurnWorker(churnService, customerRepo, tenantRepo, 24*time.Hour)
	revrecWorker := worker.NewRevRecWorker(revrecService, 24*time.Hour)

	// P25: E-Invoice Retry Worker
	einvoiceWorker := worker.NewEInvoiceRetryWorker(invoiceRepo, einvoiceService)

	// Start Workers in Background
	go retryWorker.Start(context.Background())
	go webhookWorker.Start(context.Background())
	go churnWorker.Start(context.Background())
	go revrecWorker.Start(context.Background())
	go einvoiceWorker.Start(context.Background())

	// Distributed Locking & Redis
	var locker port.Locker
	var idempotencyStore port.IdempotencyStore
	var rdb *redis.Client

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opt, _ := redis.ParseURL(redisURL)
		rdb = redis.NewClient(opt)
		locker = redisAdapter.NewRedisLocker(rdb)
		idempotencyStore = redisAdapter.NewRedisIdempotencyStore(rdb, 24*time.Hour)
		log.Println("Using Redis for Locker and Idempotency")
	} else {
		locker = memory.NewNoOpLocker()
		idempotencyStore = memory.NewInMemoryIdempotencyStore(24 * time.Hour)
		log.Println("Using In-Memory Locker and Idempotency (Redis not configured)")
	}

	// Pre-charge Scheduler (P30 - RBI compliance: 24hr notifications)
	preChargeScheduler := scheduler.NewPreChargeScheduler(
		subscriptionRepo.(*db.SubscriptionRepository),
		notificationService,
		locker,
		"http://localhost:8080",
	)
	preChargeScheduler.Start()
	defer preChargeScheduler.Stop()

	// Dunning Scheduler (P30 - payment retry and escalation)
	dunningScheduler := scheduler.NewDunningScheduler(
		invoiceRepo.(*db.InvoiceRepository),
		notificationService,
		locker,
		scheduler.DefaultDunningConfig(),
		"http://localhost:8080",
	)
	dunningScheduler.Start()
	defer dunningScheduler.Stop()

	// Graceful shutdown handler
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down gracefully...")
		preChargeScheduler.Stop()
		dunningScheduler.Stop()
		os.Exit(0)
	}()

	// 7. Initialize Handlers
	catalogHandler := handler.NewCatalogHandler(catalogService)
	customerHandler := handler.NewCustomerHandler(customerService)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
	checkoutHandler := handler.NewCheckoutHandler(invoiceRepo)
	usageHandler := handler.NewUsageHandler(usageRepo)
	// Phase 48: Unified Portal API Handler
	portalHandler := handler.NewPortalHandler(customerRepo, invoiceRepo, subscriptionService, invoiceService, customerService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService, genaiService)
	couponHandler := handler.NewCouponHandler(couponRepo)                                               // P7
	tenantHandler := handler.NewTenantHandler(tenantService)                                            // P8 Handler
	advancedBillingHandler := handler.NewAdvancedBillingHandler(advancedBillingService, invoiceService) // P15
	ledgerHandler := handler.NewLedgerHandler(ledgerService)                                            // P22
	creditNoteHandler := handler.NewCreditNoteHandler(creditNoteService)                                // P23
	webhookMgmtHandler := handler.NewWebhookManagementHandler(webhookService)                           // P24

	// Portal (P25)
	portalService := service.NewPortalService(customerRepo, invoiceRepo, magicLinkRepo, portalSessionRepo, giftService)
	portalAPIHandler := handler.NewPortalAPIHandler(portalService)

	// Quotes (P27)
	quoteService := service.NewQuoteService(quoteRepo, invoiceRepo)
	quoteHandler := handler.NewQuoteHandler(quoteService)

	// GST & PDF (P30)
	pdfService := service.NewInvoicePDFService(
		"Your Company Name",
		"123 Business Street, City, State - 000000",
		"", // GSTIN - Configure in settings
		"", // PAN
		"", // State
		"Bank: HDFC Bank\nAccount: 00000000000000\nIFSC: HDFC0000000",
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
	dunningHandler := handler.NewDunningHandler(dunningAnalyticsSvc)

	// Payment Handlers
	paymentHandler := handler.NewPaymentHandler(paymentGateway, invoiceRepo)
	webhookHandler := handler.NewWebhookHandler(subscriptionService, paymentGateway, retryService, invoiceRepo, subscriptionRepo, customerRepo, notificationService, os.Getenv("STRIPE_WEBHOOK_SECRET"))

	// 8. Setup Router
	r := gin.Default()

	// Global Middleware (Phase 47)
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecureMiddleware())
	// Rate Limit: 100 requests per minute
	r.Use(middleware.RateLimitMiddleware(rdb, 100, time.Minute))

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
			"version":    "1.0.0",
			"components": components,
		})
	})

	// Public Routes — stricter rate limit (20 req/min per IP)
	publicLimit := middleware.RateLimitMiddleware(rdb, 20, time.Minute)

	r.GET("/checkout/:id", publicLimit, checkoutHandler.ShowCheckout)
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

		v1.POST("/customers", customerHandler.CreateCustomer)
		v1.GET("/customers", customerHandler.ListCustomers)

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

		// Credit Notes (P23)
		v1.POST("/credit-notes", creditNoteHandler.CreateCreditNote)
		v1.GET("/credit-notes", creditNoteHandler.ListCreditNotes)

		// Webhooks & Events (P24)
		v1.POST("/webhooks", webhookMgmtHandler.CreateEndpoint)
		v1.GET("/webhooks", webhookMgmtHandler.ListEndpoints)
		v1.DELETE("/webhooks/:id", webhookMgmtHandler.DeleteEndpoint)

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
		v1.GET("/gifts", giftHandler.ListGifts)

		// Gift API (P43)
		v1.POST("/gifts/purchase", giftHandler.PurchaseGift)
		v1.POST("/gifts/redeem", giftHandler.RedeemGift)
	}

	// 9. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Starting Recurso API on %s", serverAddr)
	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

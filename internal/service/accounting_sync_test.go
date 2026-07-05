package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/accounting"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// --- Mock AccountingConnectionRepository ---

type acctSyncConnRepo struct {
	conns    []*domain.AccountingConnection
	updates  []domain.AccountingConnection // snapshots at Update time
	syncLogs []*domain.AccountingSyncLog
}

func (m *acctSyncConnRepo) Create(ctx context.Context, conn *domain.AccountingConnection) error {
	return nil
}

func (m *acctSyncConnRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AccountingConnection, error) {
	return nil, errors.New("not found")
}

func (m *acctSyncConnRepo) GetByTenantAndProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.AccountingConnection, error) {
	return nil, errors.New("not found")
}

func (m *acctSyncConnRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.AccountingConnection, error) {
	return m.conns, nil
}

func (m *acctSyncConnRepo) Update(ctx context.Context, conn *domain.AccountingConnection) error {
	m.updates = append(m.updates, *conn)
	return nil
}

func (m *acctSyncConnRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func (m *acctSyncConnRepo) GetActiveConnections(ctx context.Context) ([]*domain.AccountingConnection, error) {
	return m.conns, nil
}

func (m *acctSyncConnRepo) CreateSyncLog(ctx context.Context, log *domain.AccountingSyncLog) error {
	m.syncLogs = append(m.syncLogs, log)
	return nil
}

func (m *acctSyncConnRepo) ListSyncLogs(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.AccountingSyncLog, error) {
	return m.syncLogs, nil
}

// --- Mock CustomerRepository ---

type acctSyncCustomerRepo struct {
	customer *domain.Customer
}

func (m *acctSyncCustomerRepo) Create(ctx context.Context, c *domain.Customer) error { return nil }

func (m *acctSyncCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	if m.customer == nil {
		return nil, errors.New("customer not found")
	}
	return m.customer, nil
}

func (m *acctSyncCustomerRepo) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return m.GetByID(ctx, id)
}

func (m *acctSyncCustomerRepo) GetByReferralCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Customer, error) {
	return nil, errors.New("not found")
}

func (m *acctSyncCustomerRepo) List(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error) {
	if m.customer == nil || filter.Offset > 0 {
		return nil, nil
	}
	return []*domain.Customer{m.customer}, nil
}

func (m *acctSyncCustomerRepo) FindByEmailAcrossTenants(ctx context.Context, email string) ([]*domain.Customer, error) {
	return nil, nil
}

func (m *acctSyncCustomerRepo) Update(ctx context.Context, c *domain.Customer) error { return nil }

func (m *acctSyncCustomerRepo) UpdateRisk(ctx context.Context, customerID uuid.UUID, score int, factors map[string]interface{}) error {
	return nil
}

func (m *acctSyncCustomerRepo) UpdatePaymentMethod(ctx context.Context, customerID uuid.UUID, brand, last4 string, expMonth, expYear int) error {
	return nil
}

// --- Mock InvoiceRepository ---

type acctSyncInvoiceRepo struct {
	invoice *domain.Invoice
}

func (m *acctSyncInvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error { return nil }

func (m *acctSyncInvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if m.invoice == nil {
		return nil, errors.New("invoice not found")
	}
	return m.invoice, nil
}

func (m *acctSyncInvoiceRepo) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return m.GetByID(ctx, id)
}

func (m *acctSyncInvoiceRepo) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *acctSyncInvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error) {
	if m.invoice == nil {
		return nil, nil
	}
	return []*domain.Invoice{m.invoice}, nil
}

func (m *acctSyncInvoiceRepo) Update(ctx context.Context, inv *domain.Invoice) error { return nil }

func (m *acctSyncInvoiceRepo) GetDueForRetry(ctx context.Context) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *acctSyncInvoiceRepo) UpdateRetryInfo(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int) error {
	return nil
}

func (m *acctSyncInvoiceRepo) UpdateRetryInfoWithDunning(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int, managedBy string) error {
	return nil
}

func (m *acctSyncInvoiceRepo) MarkAsUncollectible(ctx context.Context, invoiceID uuid.UUID) error {
	return nil
}

func (m *acctSyncInvoiceRepo) GetOverdueInvoices(ctx context.Context) ([]domain.OverdueInvoice, error) {
	return nil, nil
}

func (m *acctSyncInvoiceRepo) GetFailedEInvoices(ctx context.Context) ([]*domain.Invoice, error) {
	return nil, nil
}

func (m *acctSyncInvoiceRepo) UpdateEInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status, irn, ackNo, signedQR, ackDate, errorMsg string) error {
	return nil
}

// --- Mock PlanRepository ---

type acctSyncPlanRepo struct {
	plan *domain.Plan
}

func (m *acctSyncPlanRepo) Create(ctx context.Context, plan *domain.Plan) error { return nil }

func (m *acctSyncPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	if m.plan == nil {
		return nil, errors.New("plan not found")
	}
	return m.plan, nil
}

func (m *acctSyncPlanRepo) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Plan, error) {
	return nil, errors.New("not found")
}

func (m *acctSyncPlanRepo) List(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error) {
	return nil, nil
}

// --- Mock AccountingMappingRepository ---

type acctSyncMappingRepo struct {
	mappings map[string]*domain.AccountingEntityMapping
	upserts  []*domain.AccountingEntityMapping
}

func newAcctSyncMappingRepo() *acctSyncMappingRepo {
	return &acctSyncMappingRepo{mappings: map[string]*domain.AccountingEntityMapping{}}
}

func acctMappingKey(connectionID uuid.UUID, entityType string, entityID uuid.UUID) string {
	return connectionID.String() + "|" + entityType + "|" + entityID.String()
}

func (m *acctSyncMappingRepo) Upsert(ctx context.Context, mapping *domain.AccountingEntityMapping) error {
	m.upserts = append(m.upserts, mapping)
	m.mappings[acctMappingKey(mapping.ConnectionID, mapping.EntityType, mapping.EntityID)] = mapping
	return nil
}

func (m *acctSyncMappingRepo) Get(ctx context.Context, connectionID uuid.UUID, entityType string, entityID uuid.UUID) (*domain.AccountingEntityMapping, error) {
	return m.mappings[acctMappingKey(connectionID, entityType, entityID)], nil
}

func (m *acctSyncMappingRepo) seed(conn *domain.AccountingConnection, entityType string, entityID uuid.UUID, externalID string) {
	m.mappings[acctMappingKey(conn.ID, entityType, entityID)] = &domain.AccountingEntityMapping{
		ID:           uuid.New(),
		TenantID:     conn.TenantID,
		ConnectionID: conn.ID,
		EntityType:   entityType,
		EntityID:     entityID,
		ExternalID:   externalID,
	}
}

// --- Recording gateway used via the adapter factory ---

type acctSyncRecordingGateway struct {
	err                error
	customers          []*domain.Customer
	invoices           []*domain.Invoice
	invoiceCustomerIDs []string // customerExternalID passed to each SyncInvoice
	plans              []*domain.Plan
}

func acctExtCustomerID(c *domain.Customer) string { return "ext-cust-" + c.ID.String() }
func acctExtInvoiceID(inv *domain.Invoice) string { return "ext-inv-" + inv.ID.String() }
func acctExtProductID(p *domain.Plan) string      { return "ext-plan-" + p.ID.String() }

func (g *acctSyncRecordingGateway) SyncCustomer(ctx context.Context, c *domain.Customer) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	g.customers = append(g.customers, c)
	return acctExtCustomerID(c), nil
}

func (g *acctSyncRecordingGateway) SyncInvoice(ctx context.Context, inv *domain.Invoice, customerExternalID string) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	g.invoices = append(g.invoices, inv)
	g.invoiceCustomerIDs = append(g.invoiceCustomerIDs, customerExternalID)
	return acctExtInvoiceID(inv), nil
}

func (g *acctSyncRecordingGateway) SyncProduct(ctx context.Context, plan *domain.Plan) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	g.plans = append(g.plans, plan)
	return acctExtProductID(plan), nil
}

// --- Helpers ---

func newAcctSyncService(connRepo *acctSyncConnRepo, custRepo *acctSyncCustomerRepo, invRepo *acctSyncInvoiceRepo, planRepo *acctSyncPlanRepo) *AccountingService {
	svc := NewAccountingService(accounting.NewMockAccountingAdapter(), custRepo, invRepo, planRepo)
	if connRepo != nil {
		svc.SetConnectionRepo(connRepo)
	}
	return svc
}

func acctSyncConn(tenantID uuid.UUID, provider string, expiresIn time.Duration, active bool) *domain.AccountingConnection {
	expiresAt := time.Now().Add(expiresIn)
	return &domain.AccountingConnection{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Provider:       provider,
		AccessToken:    "old-access",
		RefreshToken:   "old-refresh",
		TokenExpiresAt: &expiresAt,
		RealmID:        "realm-1",
		SyncStatus:     "idle",
		IsActive:       active,
	}
}

// --- Tests: refresh decision logic ---

func TestAccountingSyncAllRefreshesExpiringTokenBeforeSync(t *testing.T) {
	tokenCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		if got := r.PostFormValue("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", got)
		}
		if got := r.PostFormValue("refresh_token"); got != "old-refresh" {
			t.Errorf("refresh_token = %q, want old-refresh", got)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "test-client" || pass != "test-secret" {
			t.Errorf("basic auth = %q/%q, want test-client/test-secret", user, pass)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`)
	}))
	defer server.Close()

	tenantID := uuid.New()
	conn := acctSyncConn(tenantID, "quickbooks", 2*time.Minute, true) // inside 5-minute window
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})
	svc.SetOAuthConfigs(map[string]*accounting.OAuthConfig{
		"quickbooks": {ClientID: "test-client", ClientSecret: "test-secret", TokenURL: server.URL},
	})

	var tokenAtAdapterBuild string
	var updatesBeforeAdapterBuild int
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway {
		tokenAtAdapterBuild = c.AccessToken
		updatesBeforeAdapterBuild = len(connRepo.updates)
		return &acctSyncRecordingGateway{}
	}

	if err := svc.SyncAllForTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("SyncAllForTenant returned error: %v", err)
	}

	if tokenCalls != 1 {
		t.Fatalf("token endpoint called %d times, want 1", tokenCalls)
	}
	if tokenAtAdapterBuild != "new-access" {
		t.Errorf("adapter built with token %q, want new-access", tokenAtAdapterBuild)
	}
	if updatesBeforeAdapterBuild < 1 {
		t.Error("rotated tokens were not persisted before syncing")
	}
	if len(connRepo.updates) == 0 {
		t.Fatal("connection was never persisted")
	}
	first := connRepo.updates[0]
	if first.AccessToken != "new-access" || first.RefreshToken != "new-refresh" {
		t.Errorf("persisted tokens = %q/%q, want new-access/new-refresh", first.AccessToken, first.RefreshToken)
	}
	if conn.TokenExpiresAt == nil || time.Until(*conn.TokenExpiresAt) < 50*time.Minute {
		t.Error("token expiry was not extended by refresh")
	}
}

func TestAccountingSyncAllSkipsRefreshWhenTokenFresh(t *testing.T) {
	tokenCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tenantID := uuid.New()
	conn := acctSyncConn(tenantID, "quickbooks", time.Hour, true) // well outside window
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})
	svc.SetOAuthConfigs(map[string]*accounting.OAuthConfig{
		"quickbooks": {ClientID: "test-client", ClientSecret: "test-secret", TokenURL: server.URL},
	})

	var tokenAtAdapterBuild string
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway {
		tokenAtAdapterBuild = c.AccessToken
		return &acctSyncRecordingGateway{}
	}

	if err := svc.SyncAllForTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("SyncAllForTenant returned error: %v", err)
	}

	if tokenCalls != 0 {
		t.Errorf("token endpoint called %d times, want 0", tokenCalls)
	}
	if tokenAtAdapterBuild != "old-access" {
		t.Errorf("adapter built with token %q, want old-access", tokenAtAdapterBuild)
	}
}

func TestAccountingSyncAllInvalidGrantDeactivatesConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":"invalid_grant","error_description":"Token invalid"}`)
	}))
	defer server.Close()

	tenantID := uuid.New()
	conn := acctSyncConn(tenantID, "xero", -time.Minute, true) // already expired
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})
	svc.SetOAuthConfigs(map[string]*accounting.OAuthConfig{
		"xero": {ClientID: "test-client", ClientSecret: "test-secret", TokenURL: server.URL},
	})

	adapterBuilt := false
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway {
		adapterBuilt = true
		return &acctSyncRecordingGateway{}
	}

	if err := svc.SyncAllForTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("SyncAllForTenant returned error: %v", err)
	}

	if adapterBuilt {
		t.Error("adapter was built despite refresh failure — would have synced with a dead token")
	}
	if conn.IsActive {
		t.Error("connection still active after invalid_grant")
	}
	if conn.SyncStatus != "error" {
		t.Errorf("sync status = %q, want error", conn.SyncStatus)
	}
	if !strings.Contains(conn.LastError, "invalid_grant") {
		t.Errorf("last error %q does not mention invalid_grant", conn.LastError)
	}
	if len(connRepo.updates) == 0 {
		t.Fatal("error state was not persisted")
	}
	last := connRepo.updates[len(connRepo.updates)-1]
	if last.IsActive {
		t.Error("persisted connection still active after invalid_grant")
	}
}

func TestAccountingSyncAllTransientRefreshFailureKeepsConnectionActive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tenantID := uuid.New()
	conn := acctSyncConn(tenantID, "quickbooks", time.Minute, true)
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})
	svc.SetOAuthConfigs(map[string]*accounting.OAuthConfig{
		"quickbooks": {ClientID: "test-client", ClientSecret: "test-secret", TokenURL: server.URL},
	})

	adapterBuilt := false
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway {
		adapterBuilt = true
		return &acctSyncRecordingGateway{}
	}

	if err := svc.SyncAllForTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("SyncAllForTenant returned error: %v", err)
	}

	if adapterBuilt {
		t.Error("adapter was built despite refresh failure")
	}
	if !conn.IsActive {
		t.Error("connection deactivated on a transient refresh failure; should stay active for retry")
	}
	if conn.SyncStatus != "error" {
		t.Errorf("sync status = %q, want error", conn.SyncStatus)
	}
}

func TestAccountingSyncAllExpiredTokenWithoutRefreshTokenIsSkipped(t *testing.T) {
	tenantID := uuid.New()
	conn := acctSyncConn(tenantID, "quickbooks", -time.Hour, true)
	conn.RefreshToken = ""
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})

	adapterBuilt := false
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway {
		adapterBuilt = true
		return &acctSyncRecordingGateway{}
	}

	if err := svc.SyncAllForTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("SyncAllForTenant returned error: %v", err)
	}

	if adapterBuilt {
		t.Error("adapter was built for a connection with an expired token and no refresh token")
	}
	if conn.SyncStatus != "error" || conn.LastError == "" {
		t.Errorf("connection not marked errored: status=%q lastError=%q", conn.SyncStatus, conn.LastError)
	}
}

// --- Tests: single-entity sync routing ---

func TestSyncCustomerRoutesThroughActiveConnections(t *testing.T) {
	tenantID := uuid.New()
	name := "Acme"
	customer := &domain.Customer{ID: uuid.New(), TenantID: tenantID, Email: "acme@example.com", Name: &name}

	active := acctSyncConn(tenantID, "xero", time.Hour, true)
	inactive := acctSyncConn(tenantID, "quickbooks", time.Hour, false)
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{active, inactive}}

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{customer: customer}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})

	gw := &acctSyncRecordingGateway{}
	var factoryProviders []string
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway {
		factoryProviders = append(factoryProviders, c.Provider)
		return gw
	}

	if err := svc.SyncCustomer(context.Background(), customer.ID); err != nil {
		t.Fatalf("SyncCustomer returned error: %v", err)
	}

	if len(gw.customers) != 1 || gw.customers[0].ID != customer.ID {
		t.Fatalf("gateway received %d customers, want the 1 requested", len(gw.customers))
	}
	if len(factoryProviders) != 1 || factoryProviders[0] != "xero" {
		t.Errorf("adapters built for %v, want only the active xero connection", factoryProviders)
	}
	if len(connRepo.syncLogs) != 1 {
		t.Fatalf("got %d sync logs, want 1", len(connRepo.syncLogs))
	}
	logEntry := connRepo.syncLogs[0]
	if logEntry.EntityType != "customer" || logEntry.EntityID != customer.ID || logEntry.Status != "success" {
		t.Errorf("unexpected sync log: %+v", logEntry)
	}
	if logEntry.ExternalID != acctExtCustomerID(customer) {
		t.Errorf("sync log external id = %q, want %q", logEntry.ExternalID, acctExtCustomerID(customer))
	}
}

func TestSyncInvoiceRecordsErrorAndReturnsIt(t *testing.T) {
	tenantID := uuid.New()
	name := "Acme"
	customer := &domain.Customer{ID: uuid.New(), TenantID: tenantID, Email: "acme@example.com", Name: &name}
	invoice := &domain.Invoice{ID: uuid.New(), TenantID: tenantID, CustomerID: customer.ID}

	conn := acctSyncConn(tenantID, "quickbooks", time.Hour, true)
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{customer: customer}, &acctSyncInvoiceRepo{invoice: invoice}, &acctSyncPlanRepo{})

	gw := &acctSyncRecordingGateway{err: errors.New("QuickBooks API error: status 401")}
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway { return gw }

	err := svc.SyncInvoice(context.Background(), invoice.ID)
	if err == nil {
		t.Fatal("SyncInvoice returned nil, want gateway error")
	}
	// The customer sync (attempted first) fails, then the invoice sync is
	// recorded as failed too.
	if len(connRepo.syncLogs) != 2 {
		t.Fatalf("got %d sync logs, want 2 (customer error + invoice error)", len(connRepo.syncLogs))
	}
	custLog := connRepo.syncLogs[0]
	if custLog.EntityType != "customer" || custLog.Status != "error" || custLog.ErrorMessage == "" {
		t.Errorf("unexpected customer sync log: %+v", custLog)
	}
	invLog := connRepo.syncLogs[1]
	if invLog.EntityType != "invoice" || invLog.Status != "error" || invLog.ErrorMessage == "" {
		t.Errorf("unexpected invoice sync log: %+v", invLog)
	}
}

func TestSyncProductRoutesThroughConnections(t *testing.T) {
	tenantID := uuid.New()
	plan := &domain.Plan{ID: uuid.New(), TenantID: tenantID, Name: "Pro Monthly"}

	conn := acctSyncConn(tenantID, "quickbooks", time.Hour, true)
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{plan: plan})

	gw := &acctSyncRecordingGateway{}
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway { return gw }

	if err := svc.SyncProduct(context.Background(), plan.ID.String()); err != nil {
		t.Fatalf("SyncProduct returned error: %v", err)
	}
	if len(gw.plans) != 1 || gw.plans[0].ID != plan.ID {
		t.Fatalf("gateway received %d plans, want the 1 requested", len(gw.plans))
	}
	if len(connRepo.syncLogs) != 1 || connRepo.syncLogs[0].EntityType != "product" {
		t.Errorf("expected a product sync log, got %+v", connRepo.syncLogs)
	}
}

func TestSyncCustomerWithoutConnRepoErrors(t *testing.T) {
	tenantID := uuid.New()
	customer := &domain.Customer{ID: uuid.New(), TenantID: tenantID, Email: "acme@example.com"}

	svc := newAcctSyncService(nil, &acctSyncCustomerRepo{customer: customer}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})

	if err := svc.SyncCustomer(context.Background(), customer.ID); err == nil {
		t.Fatal("SyncCustomer returned nil without a connection repository, want error")
	}
}

// --- Tests: external-ID mapping ---

// acctSyncInvoiceFixture wires a service with one active connection, a
// mapping repo, and a recording gateway for invoice sync tests.
func acctSyncInvoiceFixture(t *testing.T) (*AccountingService, *acctSyncConnRepo, *acctSyncMappingRepo, *acctSyncRecordingGateway, *domain.AccountingConnection, *domain.Customer, *domain.Invoice) {
	t.Helper()
	tenantID := uuid.New()
	name := "Acme"
	customer := &domain.Customer{ID: uuid.New(), TenantID: tenantID, Email: "acme@example.com", Name: &name}
	invoice := &domain.Invoice{ID: uuid.New(), TenantID: tenantID, CustomerID: customer.ID, InvoiceNumber: "INV-001"}

	conn := acctSyncConn(tenantID, "quickbooks", time.Hour, true)
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}
	mappingRepo := newAcctSyncMappingRepo()

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{customer: customer}, &acctSyncInvoiceRepo{invoice: invoice}, &acctSyncPlanRepo{})
	svc.SetMappingRepo(mappingRepo)

	gw := &acctSyncRecordingGateway{}
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway { return gw }
	return svc, connRepo, mappingRepo, gw, conn, customer, invoice
}

func TestSyncInvoiceSyncsCustomerFirstAndPassesExternalID(t *testing.T) {
	svc, connRepo, mappingRepo, gw, conn, customer, invoice := acctSyncInvoiceFixture(t)

	if err := svc.SyncInvoice(context.Background(), invoice.ID); err != nil {
		t.Fatalf("SyncInvoice returned error: %v", err)
	}

	if len(gw.customers) != 1 || gw.customers[0].ID != customer.ID {
		t.Fatalf("customer was not synced before the invoice (got %d customer syncs)", len(gw.customers))
	}
	if len(gw.invoices) != 1 {
		t.Fatalf("got %d invoice syncs, want 1", len(gw.invoices))
	}
	if gw.invoiceCustomerIDs[0] != acctExtCustomerID(customer) {
		t.Errorf("invoice sync received customer external id %q, want %q", gw.invoiceCustomerIDs[0], acctExtCustomerID(customer))
	}

	// Both mappings persisted from the gateway-returned IDs.
	custMapping, _ := mappingRepo.Get(context.Background(), conn.ID, "customer", customer.ID)
	if custMapping == nil || custMapping.ExternalID != acctExtCustomerID(customer) {
		t.Errorf("customer mapping = %+v, want external id %q", custMapping, acctExtCustomerID(customer))
	}
	invMapping, _ := mappingRepo.Get(context.Background(), conn.ID, "invoice", invoice.ID)
	if invMapping == nil || invMapping.ExternalID != acctExtInvoiceID(invoice) {
		t.Errorf("invoice mapping = %+v, want external id %q", invMapping, acctExtInvoiceID(invoice))
	}
	if custMapping != nil && custMapping.TenantID != conn.TenantID {
		t.Errorf("mapping tenant = %s, want %s", custMapping.TenantID, conn.TenantID)
	}

	// Sync logs carry the external IDs.
	if len(connRepo.syncLogs) != 2 {
		t.Fatalf("got %d sync logs, want 2 (customer + invoice)", len(connRepo.syncLogs))
	}
	custLog, invLog := connRepo.syncLogs[0], connRepo.syncLogs[1]
	if custLog.EntityType != "customer" || custLog.Status != "success" || custLog.ExternalID != acctExtCustomerID(customer) {
		t.Errorf("unexpected customer sync log: %+v", custLog)
	}
	if invLog.EntityType != "invoice" || invLog.Status != "success" || invLog.ExternalID != acctExtInvoiceID(invoice) {
		t.Errorf("unexpected invoice sync log: %+v", invLog)
	}
}

func TestSyncInvoiceUsesExistingCustomerMapping(t *testing.T) {
	svc, connRepo, mappingRepo, gw, conn, customer, invoice := acctSyncInvoiceFixture(t)
	mappingRepo.seed(conn, "customer", customer.ID, "qb-cust-42")

	if err := svc.SyncInvoice(context.Background(), invoice.ID); err != nil {
		t.Fatalf("SyncInvoice returned error: %v", err)
	}

	if len(gw.customers) != 0 {
		t.Errorf("customer was re-synced despite an existing mapping (%d syncs)", len(gw.customers))
	}
	if len(gw.invoiceCustomerIDs) != 1 || gw.invoiceCustomerIDs[0] != "qb-cust-42" {
		t.Errorf("invoice sync received customer external id %v, want [qb-cust-42]", gw.invoiceCustomerIDs)
	}
	// Only the invoice should be logged; the implicit customer lookup is not.
	if len(connRepo.syncLogs) != 1 || connRepo.syncLogs[0].EntityType != "invoice" {
		t.Errorf("unexpected sync logs: %+v", connRepo.syncLogs)
	}
}

func TestSyncInvoiceExistingMappingSkipsRecreate(t *testing.T) {
	svc, connRepo, mappingRepo, gw, conn, _, invoice := acctSyncInvoiceFixture(t)
	mappingRepo.seed(conn, "invoice", invoice.ID, "qb-inv-7")

	if err := svc.SyncInvoice(context.Background(), invoice.ID); err != nil {
		t.Fatalf("SyncInvoice returned error: %v", err)
	}

	if len(gw.invoices) != 0 || len(gw.customers) != 0 {
		t.Errorf("gateway was called despite existing invoice mapping (invoices=%d customers=%d)", len(gw.invoices), len(gw.customers))
	}
	if len(mappingRepo.upserts) != 0 {
		t.Errorf("mapping was re-upserted on skip: %+v", mappingRepo.upserts)
	}
	if len(connRepo.syncLogs) != 1 {
		t.Fatalf("got %d sync logs, want 1", len(connRepo.syncLogs))
	}
	logEntry := connRepo.syncLogs[0]
	if logEntry.Action != "skip" || logEntry.Status != "exists" || logEntry.ExternalID != "qb-inv-7" {
		t.Errorf("unexpected skip log: %+v", logEntry)
	}
}

func TestSyncCustomerUpsertsMappingAndSkipsSecondSync(t *testing.T) {
	tenantID := uuid.New()
	name := "Acme"
	customer := &domain.Customer{ID: uuid.New(), TenantID: tenantID, Email: "acme@example.com", Name: &name}

	conn := acctSyncConn(tenantID, "xero", time.Hour, true)
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}
	mappingRepo := newAcctSyncMappingRepo()

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{customer: customer}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})
	svc.SetMappingRepo(mappingRepo)

	gw := &acctSyncRecordingGateway{}
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway { return gw }

	if err := svc.SyncCustomer(context.Background(), customer.ID); err != nil {
		t.Fatalf("first SyncCustomer returned error: %v", err)
	}
	if len(mappingRepo.upserts) != 1 || mappingRepo.upserts[0].ExternalID != acctExtCustomerID(customer) {
		t.Fatalf("mapping not upserted from gateway id: %+v", mappingRepo.upserts)
	}

	// Second sync must not re-create on the provider.
	if err := svc.SyncCustomer(context.Background(), customer.ID); err != nil {
		t.Fatalf("second SyncCustomer returned error: %v", err)
	}
	if len(gw.customers) != 1 {
		t.Errorf("gateway called %d times, want 1 (create-once)", len(gw.customers))
	}
	if len(connRepo.syncLogs) != 2 {
		t.Fatalf("got %d sync logs, want 2", len(connRepo.syncLogs))
	}
	second := connRepo.syncLogs[1]
	if second.Action != "skip" || second.Status != "exists" || second.ExternalID != acctExtCustomerID(customer) {
		t.Errorf("unexpected second sync log: %+v", second)
	}
}

func TestSyncProductUpsertsMappingAndSkipsSecondSync(t *testing.T) {
	tenantID := uuid.New()
	plan := &domain.Plan{ID: uuid.New(), TenantID: tenantID, Name: "Pro Monthly"}

	conn := acctSyncConn(tenantID, "quickbooks", time.Hour, true)
	connRepo := &acctSyncConnRepo{conns: []*domain.AccountingConnection{conn}}
	mappingRepo := newAcctSyncMappingRepo()

	svc := newAcctSyncService(connRepo, &acctSyncCustomerRepo{}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{plan: plan})
	svc.SetMappingRepo(mappingRepo)

	gw := &acctSyncRecordingGateway{}
	svc.adapterFactory = func(c *domain.AccountingConnection) port.AccountingGateway { return gw }

	if err := svc.SyncProduct(context.Background(), plan.ID.String()); err != nil {
		t.Fatalf("first SyncProduct returned error: %v", err)
	}
	if err := svc.SyncProduct(context.Background(), plan.ID.String()); err != nil {
		t.Fatalf("second SyncProduct returned error: %v", err)
	}

	if len(gw.plans) != 1 {
		t.Errorf("gateway called %d times, want 1 (create-once)", len(gw.plans))
	}
	m, _ := mappingRepo.Get(context.Background(), conn.ID, "product", plan.ID)
	if m == nil || m.ExternalID != acctExtProductID(plan) {
		t.Errorf("product mapping = %+v, want external id %q", m, acctExtProductID(plan))
	}
	if len(connRepo.syncLogs) != 2 || connRepo.syncLogs[1].Status != "exists" {
		t.Errorf("unexpected sync logs: %+v", connRepo.syncLogs)
	}
}

func TestSyncInvoiceFailedCustomerSyncDoesNotUpsertMapping(t *testing.T) {
	svc, _, mappingRepo, gw, _, _, invoice := acctSyncInvoiceFixture(t)
	gw.err = errors.New("QuickBooks API error: status 401")

	if err := svc.SyncInvoice(context.Background(), invoice.ID); err == nil {
		t.Fatal("SyncInvoice returned nil, want gateway error")
	}
	if len(mappingRepo.upserts) != 0 {
		t.Errorf("mappings upserted despite gateway failure: %+v", mappingRepo.upserts)
	}
}

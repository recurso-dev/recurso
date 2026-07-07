package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/gsp"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// --- In-memory add-on repository (tenant-scoped) ---

type memAddonRepo struct {
	addons  []*domain.SubscriptionAddon
	listErr error
}

func (m *memAddonRepo) Create(ctx context.Context, a *domain.SubscriptionAddon) error {
	m.addons = append(m.addons, a)
	return nil
}

func (m *memAddonRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionAddon, error) {
	for _, a := range m.addons {
		if a.ID == id && a.TenantID == tenantID {
			return a, nil
		}
	}
	return nil, nil
}

func (m *memAddonRepo) ListBySubscriptionID(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.SubscriptionAddon, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var out []*domain.SubscriptionAddon
	for _, a := range m.addons {
		if a.TenantID == tenantID && a.SubscriptionID == subscriptionID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *memAddonRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	kept := m.addons[:0]
	for _, a := range m.addons {
		if a.ID == id && a.TenantID == tenantID {
			continue
		}
		kept = append(kept, a)
	}
	m.addons = kept
	return nil
}

// --- Service: Add / List / Remove add-ons ---

func addonTestService(t *testing.T, sub *domain.Subscription, plans map[uuid.UUID]*domain.Plan, addonRepo *memAddonRepo) *SubscriptionService {
	t.Helper()
	svc := newTestSubscriptionService(
		&subMockSubRepo{sub: sub},
		&subMockInvoiceRepo{},
		&multiPlanRepo{plans: plans},
		&subMockCustomerRepo{customer: &domain.Customer{ID: sub.CustomerID}},
		&subMockCouponRepo{},
		&subMockGateway{},
	)
	svc.SetAddonRepository(addonRepo)
	return svc
}

func TestAddAddon_Success(t *testing.T) {
	tenantID := uuid.New()
	basePlanID, addonPlanID := uuid.New(), uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(), PlanID: basePlanID}
	plans := map[uuid.UUID]*domain.Plan{
		basePlanID:  {ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}},
		addonPlanID: {ID: addonPlanID, Prices: []domain.Price{{Amount: 50000, Currency: "INR"}}},
	}
	repo := &memAddonRepo{}
	svc := addonTestService(t, sub, plans, repo)

	addon, err := svc.AddAddon(context.Background(), tenantID, sub.ID, addonPlanID, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addon.PlanID != addonPlanID || addon.Quantity != 2 || addon.SubscriptionID != sub.ID || addon.TenantID != tenantID {
		t.Errorf("addon fields wrong: %+v", addon)
	}
	if len(repo.addons) != 1 {
		t.Fatalf("expected 1 persisted add-on, got %d", len(repo.addons))
	}
}

func TestAddAddon_CurrencyMismatch(t *testing.T) {
	tenantID := uuid.New()
	basePlanID, addonPlanID := uuid.New(), uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(), PlanID: basePlanID}
	plans := map[uuid.UUID]*domain.Plan{
		basePlanID:  {ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}},
		addonPlanID: {ID: addonPlanID, Prices: []domain.Price{{Amount: 9900, Currency: "USD"}}},
	}
	svc := addonTestService(t, sub, plans, &memAddonRepo{})

	_, err := svc.AddAddon(context.Background(), tenantID, sub.ID, addonPlanID, 1)
	if !errors.Is(err, ErrAddonCurrencyMismatch) {
		t.Fatalf("expected ErrAddonCurrencyMismatch, got %v", err)
	}
}

func TestAddAddon_PlanNotFound(t *testing.T) {
	tenantID := uuid.New()
	basePlanID := uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(), PlanID: basePlanID}
	plans := map[uuid.UUID]*domain.Plan{
		basePlanID: {ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}},
	}
	svc := addonTestService(t, sub, plans, &memAddonRepo{})

	_, err := svc.AddAddon(context.Background(), tenantID, sub.ID, uuid.New(), 1)
	if !errors.Is(err, ErrPlanNotFound) {
		t.Fatalf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestAddAddon_InvalidQuantity(t *testing.T) {
	tenantID := uuid.New()
	basePlanID, addonPlanID := uuid.New(), uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(), PlanID: basePlanID}
	plans := map[uuid.UUID]*domain.Plan{
		basePlanID:  {ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}},
		addonPlanID: {ID: addonPlanID, Prices: []domain.Price{{Amount: 50000, Currency: "INR"}}},
	}
	svc := addonTestService(t, sub, plans, &memAddonRepo{})

	if _, err := svc.AddAddon(context.Background(), tenantID, sub.ID, addonPlanID, 0); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("expected ErrInvalidQuantity, got %v", err)
	}
}

func TestAddAddon_TenantIsolation(t *testing.T) {
	ownerTenant, otherTenant := uuid.New(), uuid.New()
	basePlanID, addonPlanID := uuid.New(), uuid.New()
	// Subscription belongs to ownerTenant.
	sub := &domain.Subscription{ID: uuid.New(), TenantID: ownerTenant, CustomerID: uuid.New(), PlanID: basePlanID}
	plans := map[uuid.UUID]*domain.Plan{
		basePlanID:  {ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}},
		addonPlanID: {ID: addonPlanID, Prices: []domain.Price{{Amount: 50000, Currency: "INR"}}},
	}
	svc := addonTestService(t, sub, plans, &memAddonRepo{})

	// A different tenant must not be able to attach an add-on to the sub.
	_, err := svc.AddAddon(context.Background(), otherTenant, sub.ID, addonPlanID, 1)
	if !errors.Is(err, ErrSubscriptionNotFound) {
		t.Fatalf("expected ErrSubscriptionNotFound for cross-tenant, got %v", err)
	}
}

func TestListAddons_OwnershipAndResults(t *testing.T) {
	tenantID := uuid.New()
	basePlanID, addonPlanID := uuid.New(), uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(), PlanID: basePlanID}
	plans := map[uuid.UUID]*domain.Plan{
		basePlanID:  {ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}},
		addonPlanID: {ID: addonPlanID, Prices: []domain.Price{{Amount: 50000, Currency: "INR"}}},
	}
	repo := &memAddonRepo{addons: []*domain.SubscriptionAddon{
		{ID: uuid.New(), TenantID: tenantID, SubscriptionID: sub.ID, PlanID: addonPlanID, Quantity: 3},
	}}
	svc := addonTestService(t, sub, plans, repo)

	addons, err := svc.ListAddons(context.Background(), tenantID, sub.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addons) != 1 || addons[0].Quantity != 3 {
		t.Fatalf("unexpected add-ons: %+v", addons)
	}

	// Cross-tenant list is denied at the ownership guard.
	if _, err := svc.ListAddons(context.Background(), uuid.New(), sub.ID); !errors.Is(err, ErrSubscriptionNotFound) {
		t.Fatalf("expected ErrSubscriptionNotFound cross-tenant, got %v", err)
	}
}

func TestRemoveAddon_Success(t *testing.T) {
	tenantID := uuid.New()
	basePlanID := uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(), PlanID: basePlanID}
	addonID := uuid.New()
	repo := &memAddonRepo{addons: []*domain.SubscriptionAddon{
		{ID: addonID, TenantID: tenantID, SubscriptionID: sub.ID, PlanID: uuid.New(), Quantity: 1},
	}}
	plans := map[uuid.UUID]*domain.Plan{basePlanID: {ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}}
	svc := addonTestService(t, sub, plans, repo)

	if err := svc.RemoveAddon(context.Background(), tenantID, sub.ID, addonID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.addons) != 0 {
		t.Fatalf("expected add-on removed, %d remain", len(repo.addons))
	}
}

func TestRemoveAddon_WrongSubscription(t *testing.T) {
	tenantID := uuid.New()
	basePlanID := uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(), PlanID: basePlanID}
	// Add-on belongs to a DIFFERENT subscription (same tenant).
	addonID := uuid.New()
	repo := &memAddonRepo{addons: []*domain.SubscriptionAddon{
		{ID: addonID, TenantID: tenantID, SubscriptionID: uuid.New(), PlanID: uuid.New(), Quantity: 1},
	}}
	plans := map[uuid.UUID]*domain.Plan{basePlanID: {ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}}
	svc := addonTestService(t, sub, plans, repo)

	if err := svc.RemoveAddon(context.Background(), tenantID, sub.ID, addonID); !errors.Is(err, ErrAddonNotFound) {
		t.Fatalf("expected ErrAddonNotFound, got %v", err)
	}
	if len(repo.addons) != 1 {
		t.Fatalf("add-on must not be removed, %d remain", len(repo.addons))
	}
}

// --- Invoice generation with add-ons ---

// invoiceServiceWithAddons builds an InvoiceService whose plan repo resolves
// both the base plan and every add-on plan by ID, with the given add-ons
// attached. taxResolver is nil -> env-default IN/TN (matches other invoice
// arithmetic tests).
func invoiceServiceWithAddons(base *domain.Plan, addonPlans []*domain.Plan, addons []*domain.SubscriptionAddon) (*InvoiceService, *mockInvoiceRepoForInvAmt) {
	plans := map[uuid.UUID]*domain.Plan{base.ID: base}
	for _, p := range addonPlans {
		plans[p.ID] = p
	}
	invRepo := &mockInvoiceRepoForInvAmt{}
	svc := NewInvoiceService(
		invRepo,
		&multiPlanRepo{plans: plans},
		&mockCustomerRepoForInvAmt{},
		&mockUCRepoForInvAmt{},
		&mockSubRepoForInvAmt{},
		gsp.NewMockGSPAdapter(),
		nil,
	)
	svc.AddonRepo = &memAddonRepo{addons: addons}
	return svc, invRepo
}

func TestGenerateInvoice_OneAddon_SummedPerLineTax(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()
	basePlanID, addonPlanID := uuid.New(), uuid.New()

	base := &domain.Plan{ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}
	addonPlan := &domain.Plan{ID: addonPlanID, Prices: []domain.Price{{Amount: 50000, Currency: "INR"}}}
	addons := []*domain.SubscriptionAddon{
		{ID: uuid.New(), TenantID: tenantID, SubscriptionID: subID, PlanID: addonPlanID, Quantity: 2},
	}

	svc, _ := invoiceServiceWithAddons(base, []*domain.Plan{addonPlan}, addons)
	svc.CustomerRepo = &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID: customerID, PlaceOfSupply: domain.StringPtr("KA"), // inter-state vs org TN -> IGST
	}}

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: subID, TenantID: tenantID, CustomerID: customerID, PlanID: basePlanID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Base 100000 + add-on line 50000*2 = 200000 subtotal.
	if inv.Subtotal != 200000 {
		t.Errorf("Subtotal = %d, want 200000", inv.Subtotal)
	}
	// 18% IGST on base (18000) + on add-on line 100000 (18000) = 36000.
	if inv.TaxAmount != 36000 || inv.IGSTAmount != 36000 {
		t.Errorf("TaxAmount/IGST = %d/%d, want 36000/36000", inv.TaxAmount, inv.IGSTAmount)
	}
	if inv.CGSTAmount != 0 || inv.SGSTAmount != 0 {
		t.Errorf("CGST/SGST = %d/%d, want 0/0", inv.CGSTAmount, inv.SGSTAmount)
	}
	if inv.Total != 236000 || inv.Total != inv.Subtotal+inv.TaxAmount {
		t.Errorf("Total = %d, want 236000 (= subtotal+tax)", inv.Total)
	}
}

func TestGenerateInvoice_MultipleAddons_ComponentSummation(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()
	basePlanID, addon1ID, addon2ID := uuid.New(), uuid.New(), uuid.New()

	base := &domain.Plan{ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}
	addon1 := &domain.Plan{ID: addon1ID, Prices: []domain.Price{{Amount: 50000, Currency: "INR"}}}
	addon2 := &domain.Plan{ID: addon2ID, Prices: []domain.Price{{Amount: 25000, Currency: "INR"}}}
	addons := []*domain.SubscriptionAddon{
		{ID: uuid.New(), TenantID: tenantID, SubscriptionID: subID, PlanID: addon1ID, Quantity: 1},
		{ID: uuid.New(), TenantID: tenantID, SubscriptionID: subID, PlanID: addon2ID, Quantity: 2},
	}

	svc, _ := invoiceServiceWithAddons(base, []*domain.Plan{addon1, addon2}, addons)
	svc.CustomerRepo = &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID: customerID, PlaceOfSupply: domain.StringPtr("TN"), // intra-state -> CGST+SGST
	}}

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: subID, TenantID: tenantID, CustomerID: customerID, PlanID: basePlanID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Subtotal = 100000 + 50000 + 25000*2 = 200000.
	if inv.Subtotal != 200000 {
		t.Errorf("Subtotal = %d, want 200000", inv.Subtotal)
	}
	// Intra-state 18% split evenly: 36000 total = 18000 CGST + 18000 SGST.
	if inv.TaxAmount != 36000 {
		t.Errorf("TaxAmount = %d, want 36000", inv.TaxAmount)
	}
	if inv.CGSTAmount != 18000 || inv.SGSTAmount != 18000 {
		t.Errorf("CGST/SGST = %d/%d, want 18000/18000", inv.CGSTAmount, inv.SGSTAmount)
	}
	if inv.IGSTAmount != 0 {
		t.Errorf("IGST = %d, want 0 for intra-state", inv.IGSTAmount)
	}
	if inv.Total != 236000 {
		t.Errorf("Total = %d, want 236000", inv.Total)
	}
}

func TestGenerateInvoice_MismatchedCurrencyAddonSkipped(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()
	basePlanID, usdAddonID := uuid.New(), uuid.New()

	base := &domain.Plan{ID: basePlanID, Prices: []domain.Price{{Amount: 100000, Currency: "INR"}}}
	usdAddon := &domain.Plan{ID: usdAddonID, Prices: []domain.Price{{Amount: 9900, Currency: "USD"}}}
	addons := []*domain.SubscriptionAddon{
		{ID: uuid.New(), TenantID: tenantID, SubscriptionID: subID, PlanID: usdAddonID, Quantity: 1},
	}

	svc, _ := invoiceServiceWithAddons(base, []*domain.Plan{usdAddon}, addons)
	svc.CustomerRepo = &mockCustomerRepoForInvAmt{customer: &domain.Customer{
		ID: customerID, PlaceOfSupply: domain.StringPtr("KA"),
	}}

	inv, err := svc.GenerateInvoice(context.Background(), &domain.Subscription{
		ID: subID, TenantID: tenantID, CustomerID: customerID, PlanID: basePlanID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The USD add-on must not be summed into the INR invoice.
	if inv.Subtotal != 100000 {
		t.Errorf("Subtotal = %d, want 100000 (mismatched-currency add-on skipped)", inv.Subtotal)
	}
	if inv.TaxAmount != 18000 {
		t.Errorf("TaxAmount = %d, want 18000 (base only)", inv.TaxAmount)
	}
}

// TestGenerateInvoice_ZeroAddons_ByteIdentical is the money-path regression
// guard: an invoice generated with the add-on repo wired but no add-ons
// attached must be numerically identical to one generated with add-ons
// disabled entirely (nil repo).
func TestGenerateInvoice_ZeroAddons_ByteIdentical(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	subID := uuid.New()
	basePlanID := uuid.New()

	newSvc := func(withRepo bool) *InvoiceService {
		invRepo := &mockInvoiceRepoForInvAmt{}
		planRepo := &multiPlanRepo{plans: map[uuid.UUID]*domain.Plan{
			basePlanID: {ID: basePlanID, Prices: []domain.Price{{Amount: 123450, Currency: "INR"}}},
		}}
		svc := NewInvoiceService(invRepo, planRepo,
			&mockCustomerRepoForInvAmt{customer: &domain.Customer{ID: customerID, PlaceOfSupply: domain.StringPtr("KA")}},
			&mockUCRepoForInvAmt{}, &mockSubRepoForInvAmt{}, gsp.NewMockGSPAdapter(), nil)
		if withRepo {
			svc.AddonRepo = &memAddonRepo{} // wired but empty
		}
		return svc
	}

	sub := &domain.Subscription{ID: subID, TenantID: tenantID, CustomerID: customerID, PlanID: basePlanID}

	invNoRepo, err := newSvc(false).GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error (no repo): %v", err)
	}
	invEmptyRepo, err := newSvc(true).GenerateInvoice(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error (empty repo): %v", err)
	}

	if invNoRepo.Subtotal != invEmptyRepo.Subtotal ||
		invNoRepo.TaxAmount != invEmptyRepo.TaxAmount ||
		invNoRepo.Total != invEmptyRepo.Total ||
		invNoRepo.IGSTAmount != invEmptyRepo.IGSTAmount ||
		invNoRepo.CGSTAmount != invEmptyRepo.CGSTAmount ||
		invNoRepo.SGSTAmount != invEmptyRepo.SGSTAmount ||
		invNoRepo.Currency != invEmptyRepo.Currency {
		t.Errorf("zero-add-on invoice diverged:\n no-repo   = %+v\n empty-repo= %+v",
			invNoRepo, invEmptyRepo)
	}
}

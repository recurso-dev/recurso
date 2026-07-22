package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// CancelFlowService handles cancel flow business logic
type CancelFlowService struct {
	flowRepo            port.CancelFlowRepository
	subscriptionService *SubscriptionService
	notificationService *NotificationService
	logger              *slog.Logger
}

func NewCancelFlowService(
	flowRepo port.CancelFlowRepository,
	subscriptionService *SubscriptionService,
	notificationService *NotificationService,
) *CancelFlowService {
	return &CancelFlowService{
		flowRepo:            flowRepo,
		subscriptionService: subscriptionService,
		notificationService: notificationService,
		logger:              slog.Default().With("service", "cancel_flow"),
	}
}

// StartSessionInput contains the input for starting a cancel session
type StartSessionInput struct {
	TenantID       uuid.UUID
	CustomerID     uuid.UUID
	SubscriptionID uuid.UUID
}

// StartSessionResult contains the result of starting a cancel session
type StartSessionResult struct {
	SessionID uuid.UUID               `json:"session_id"`
	FlowID    uuid.UUID               `json:"flow_id"`
	Steps     []domain.CancelFlowStep `json:"steps"`
	FirstStep *domain.CancelFlowStep  `json:"first_step"`
}

// StartSession creates a new cancel flow session and returns the flow steps
func (s *CancelFlowService) StartSession(ctx context.Context, input StartSessionInput) (*StartSessionResult, error) {
	// Get the default flow for the tenant
	flow, err := s.flowRepo.GetDefaultFlowForTenant(ctx, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default flow: %w", err)
	}
	if flow == nil {
		// Auto-create default flow
		flow, err = s.EnsureDefaultFlow(ctx, input.TenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to create default flow: %w", err)
		}
	}

	// Check cooldown — reject if customer accepted an offer within CooldownDays
	recentSession, err := s.flowRepo.GetRecentSessionByCustomer(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check cooldown: %w", err)
	}
	if recentSession != nil && domain.CooldownActive(recentSession.CompletedAt, flow.CooldownDays) {
		return nil, fmt.Errorf("cooldown active: customer accepted an offer recently, please try again later")
	}

	// Create session
	session := &domain.CancelFlowSession{
		ID:               uuid.New(),
		TenantID:         input.TenantID,
		CustomerID:       input.CustomerID,
		SubscriptionID:   input.SubscriptionID,
		FlowID:           flow.ID,
		Status:           domain.SessionStatusInProgress,
		CurrentStepIndex: 0,
		StartedAt:        time.Now().UTC(),
	}

	if err := s.flowRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	result := &StartSessionResult{
		SessionID: session.ID,
		FlowID:    flow.ID,
		Steps:     flow.Steps,
	}
	if len(flow.Steps) > 0 {
		result.FirstStep = &flow.Steps[0]
	}

	s.logger.Info("cancel flow session started",
		"session_id", session.ID,
		"customer_id", input.CustomerID,
		"subscription_id", input.SubscriptionID,
		"flow_id", flow.ID,
	)

	return result, nil
}

// SubmitStepInput contains the input for submitting a step response
type SubmitStepInput struct {
	TenantID  uuid.UUID
	SessionID uuid.UUID
	StepIndex int
	Response  json.RawMessage
}

// SubmitStepResult contains the result of submitting a step
type SubmitStepResult struct {
	SessionID    uuid.UUID                      `json:"session_id"`
	Status       domain.CancelFlowSessionStatus `json:"status"`
	NextStep     *domain.CancelFlowStep         `json:"next_step,omitempty"`
	SavedByOffer bool                           `json:"saved_by_offer"`
	Completed    bool                           `json:"completed"`
}

// SubmitStep processes a step response in the cancel flow
func (s *CancelFlowService) SubmitStep(ctx context.Context, input SubmitStepInput) (*SubmitStepResult, error) {
	session, err := s.flowRepo.GetSessionByID(ctx, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil || session.TenantID != input.TenantID {
		return nil, fmt.Errorf("session not found")
	}
	if session.Status != domain.SessionStatusInProgress {
		return nil, fmt.Errorf("session is not in progress")
	}

	// Get flow with steps
	flow, err := s.flowRepo.GetFlowByID(ctx, session.FlowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}
	if flow == nil {
		return nil, fmt.Errorf("flow not found")
	}

	if input.StepIndex >= len(flow.Steps) {
		return nil, fmt.Errorf("invalid step index")
	}

	// Enforce sequential step progression
	if input.StepIndex != session.CurrentStepIndex {
		return nil, fmt.Errorf("expected step index %d, got %d", session.CurrentStepIndex, input.StepIndex)
	}

	// Atomically claim this step BEFORE running any side effects. Two concurrent
	// SubmitStep calls both pass the in-memory checks above (they read the same
	// pre-update row), so without this a retention offer (trial extension / pause /
	// plan switch) would be applied twice. ClaimStep advances the step in a single
	// conditional UPDATE; only the winner proceeds (PHASE2 #2).
	claimed, err := s.flowRepo.ClaimStep(ctx, session.ID, input.TenantID, input.StepIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to claim cancel-flow step: %w", err)
	}
	if !claimed {
		return nil, fmt.Errorf("step %d was already processed", input.StepIndex)
	}
	// Reflect the claim in the in-memory session so the UpdateSession calls below
	// persist the advanced index consistently.
	session.CurrentStepIndex = input.StepIndex + 1

	step := flow.Steps[input.StepIndex]

	// Process based on step type
	switch step.StepType {
	case domain.StepTypeSurvey:
		if err := s.processSurveyStep(session, input.Response); err != nil {
			return nil, err
		}

	case domain.StepTypeOffer:
		accepted, err := s.processOfferStep(ctx, session, input.Response)
		if err != nil {
			return nil, err
		}
		if accepted {
			// Customer accepted the offer — session is saved
			now := time.Now().UTC()
			session.Status = domain.SessionStatusSaved
			session.SavedByOffer = true
			session.CompletedAt = &now
			if err := s.flowRepo.UpdateSession(ctx, session); err != nil {
				return nil, fmt.Errorf("failed to update session: %w", err)
			}
			return &SubmitStepResult{
				SessionID:    session.ID,
				Status:       session.Status,
				SavedByOffer: true,
				Completed:    true,
			}, nil
		}

	case domain.StepTypeConfirmation:
		// Finalize cancellation
		if err := s.processConfirmationStep(ctx, session); err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		session.Status = domain.SessionStatusCompleted
		session.CompletedAt = &now
		if err := s.flowRepo.UpdateSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}
		return &SubmitStepResult{
			SessionID: session.ID,
			Status:    session.Status,
			Completed: true,
		}, nil
	}

	// Advance to next step
	session.CurrentStepIndex = input.StepIndex + 1
	if err := s.flowRepo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	result := &SubmitStepResult{
		SessionID: session.ID,
		Status:    session.Status,
	}

	// Return next step if available
	if session.CurrentStepIndex < len(flow.Steps) {
		nextStep := flow.Steps[session.CurrentStepIndex]
		result.NextStep = &nextStep
	}

	return result, nil
}

func (s *CancelFlowService) processSurveyStep(session *domain.CancelFlowSession, response json.RawMessage) error {
	var surveyResponse struct {
		Reason   string `json:"reason"`
		Feedback string `json:"feedback"`
	}
	if err := json.Unmarshal(response, &surveyResponse); err != nil {
		return fmt.Errorf("invalid survey response: %w", err)
	}

	session.CancellationReason = surveyResponse.Reason
	session.Feedback = surveyResponse.Feedback
	return nil
}

func (s *CancelFlowService) processOfferStep(ctx context.Context, session *domain.CancelFlowSession, response json.RawMessage) (bool, error) {
	var offerResponse struct {
		Accepted bool                  `json:"accepted"`
		Offer    domain.RetentionOffer `json:"offer"`
	}
	if err := json.Unmarshal(response, &offerResponse); err != nil {
		return false, fmt.Errorf("invalid offer response: %w", err)
	}

	// Store the offer that was presented
	offerJSON, _ := json.Marshal(offerResponse.Offer)
	session.OfferPresented = offerJSON
	session.OfferAccepted = offerResponse.Accepted

	if !offerResponse.Accepted {
		return false, nil
	}

	// Apply the offer
	if err := s.applyOffer(ctx, session, offerResponse.Offer); err != nil {
		s.logger.Error("failed to apply retention offer", "error", err, "session_id", session.ID)
		return false, fmt.Errorf("failed to apply offer: %w", err)
	}

	return true, nil
}

func (s *CancelFlowService) applyOffer(ctx context.Context, session *domain.CancelFlowSession, offer domain.RetentionOffer) error {
	switch offer.Type {
	case domain.OfferTypePause:
		// Schedule an automatic resume PauseMonths from now (issue #111). A
		// non-positive PauseMonths means an open-ended pause (resume manually).
		var resumeAt *time.Time
		if offer.PauseMonths > 0 {
			t := time.Now().UTC().AddDate(0, offer.PauseMonths, 0)
			resumeAt = &t
		}
		_, err := s.subscriptionService.PauseSubscription(ctx, session.TenantID, session.SubscriptionID, resumeAt)
		if err != nil {
			return fmt.Errorf("failed to pause subscription: %w", err)
		}
		s.logger.Info("subscription paused via retention offer",
			"subscription_id", session.SubscriptionID,
			"pause_months", offer.PauseMonths,
			"resume_at", resumeAt,
		)

	case domain.OfferTypePlanSwitch:
		if offer.SwitchToPlanID == nil {
			return fmt.Errorf("switch_to_plan_id required for plan_switch offer")
		}
		_, err := s.subscriptionService.UpdateSubscription(ctx, session.TenantID, session.SubscriptionID, *offer.SwitchToPlanID)
		if err != nil {
			return fmt.Errorf("failed to switch plan: %w", err)
		}
		s.logger.Info("subscription plan switched via retention offer",
			"subscription_id", session.SubscriptionID,
			"new_plan_id", offer.SwitchToPlanID,
		)

	case domain.OfferTypeTrialExtension:
		_, err := s.subscriptionService.ExtendCurrentPeriod(ctx, session.TenantID, session.SubscriptionID, offer.ExtensionDays)
		if err != nil {
			return fmt.Errorf("failed to extend subscription period: %w", err)
		}
		s.logger.Info("subscription trial extended via retention offer",
			"subscription_id", session.SubscriptionID,
			"extension_days", offer.ExtensionDays,
		)

	case domain.OfferTypeDiscount:
		// Discount offers require coupon integration
		// For now, log the intent — can be expanded with CouponService
		s.logger.Info("discount offer accepted (coupon application pending)",
			"subscription_id", session.SubscriptionID,
			"discount_percent", offer.DiscountPercent,
			"duration_months", offer.DiscountDurationMonths,
		)

	case domain.OfferTypeCustom:
		s.logger.Info("custom offer accepted",
			"subscription_id", session.SubscriptionID,
		)
	}

	return nil
}

func (s *CancelFlowService) processConfirmationStep(ctx context.Context, session *domain.CancelFlowSession) error {
	// Finalize the cancellation by calling SubscriptionService.Cancel
	_, err := s.subscriptionService.Cancel(
		ctx,
		session.TenantID,
		session.SubscriptionID,
		false, // cancel at period end
		session.CancellationReason,
		session.Feedback,
	)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	s.logger.Info("subscription cancelled via cancel flow",
		"session_id", session.ID,
		"subscription_id", session.SubscriptionID,
		"reason", session.CancellationReason,
	)
	return nil
}

// GetFlowStats returns aggregated statistics for a cancel flow
func (s *CancelFlowService) GetFlowStats(ctx context.Context, tenantID, flowID uuid.UUID) (*domain.FlowStats, error) {
	return s.flowRepo.GetSessionStats(ctx, tenantID, flowID)
}

// EnsureDefaultFlow creates a default 3-step cancel flow if none exists for the tenant
func (s *CancelFlowService) EnsureDefaultFlow(ctx context.Context, tenantID uuid.UUID) (*domain.CancelFlow, error) {
	existing, err := s.flowRepo.GetDefaultFlowForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now().UTC()
	flow := &domain.CancelFlow{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         "Default Cancel Flow",
		IsActive:     true,
		IsDefault:    true,
		CooldownDays: 30,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.flowRepo.CreateFlow(ctx, flow); err != nil {
		return nil, fmt.Errorf("failed to create default flow: %w", err)
	}

	// Step 1: Survey
	surveyConfig, _ := json.Marshal(map[string]interface{}{
		"questions": []string{
			"Too expensive",
			"Missing features",
			"Found a better alternative",
			"No longer needed",
			"Technical issues",
			"Other",
		},
		"allow_feedback": true,
	})
	step1 := &domain.CancelFlowStep{
		ID:        uuid.New(),
		FlowID:    flow.ID,
		StepOrder: 0,
		StepType:  domain.StepTypeSurvey,
		Config:    surveyConfig,
		CreatedAt: now,
	}

	// Step 2: Offer
	offerConfig, _ := json.Marshal(map[string]interface{}{
		"offers": []map[string]interface{}{
			{"type": "discount", "discount_percent": 20, "discount_duration_months": 3},
			{"type": "pause", "pause_months": 1},
		},
		"headline": "Before you go, we'd like to offer you a deal",
	})
	step2 := &domain.CancelFlowStep{
		ID:        uuid.New(),
		FlowID:    flow.ID,
		StepOrder: 1,
		StepType:  domain.StepTypeOffer,
		Config:    offerConfig,
		CreatedAt: now,
	}

	// Step 3: Confirmation
	confirmConfig, _ := json.Marshal(map[string]interface{}{
		"message":        "Are you sure you want to cancel?",
		"confirm_button": "Yes, cancel my subscription",
		"cancel_button":  "No, keep my subscription",
	})
	step3 := &domain.CancelFlowStep{
		ID:        uuid.New(),
		FlowID:    flow.ID,
		StepOrder: 2,
		StepType:  domain.StepTypeConfirmation,
		Config:    confirmConfig,
		CreatedAt: now,
	}

	for _, step := range []*domain.CancelFlowStep{step1, step2, step3} {
		if err := s.flowRepo.CreateStep(ctx, step); err != nil {
			return nil, fmt.Errorf("failed to create default step: %w", err)
		}
	}

	flow.Steps = []domain.CancelFlowStep{*step1, *step2, *step3}

	s.logger.Info("default cancel flow created", "tenant_id", tenantID, "flow_id", flow.ID)
	return flow, nil
}

// --- Pass-through CRUD methods for handler ---

func (s *CancelFlowService) ListFlows(ctx context.Context, tenantID uuid.UUID) ([]*domain.CancelFlow, error) {
	return s.flowRepo.ListFlowsByTenant(ctx, tenantID)
}

func (s *CancelFlowService) GetFlowByID(ctx context.Context, id uuid.UUID) (*domain.CancelFlow, error) {
	return s.flowRepo.GetFlowByID(ctx, id)
}

func (s *CancelFlowService) CreateFlow(ctx context.Context, flow *domain.CancelFlow) error {
	return s.flowRepo.CreateFlow(ctx, flow)
}

func (s *CancelFlowService) UpdateFlow(ctx context.Context, flow *domain.CancelFlow) error {
	return s.flowRepo.UpdateFlow(ctx, flow)
}

func (s *CancelFlowService) CreateStep(ctx context.Context, step *domain.CancelFlowStep) error {
	return s.flowRepo.CreateStep(ctx, step)
}

func (s *CancelFlowService) UpdateStep(ctx context.Context, step *domain.CancelFlowStep, tenantID uuid.UUID) error {
	return s.flowRepo.UpdateStep(ctx, step, tenantID)
}

func (s *CancelFlowService) DeleteStep(ctx context.Context, id, tenantID uuid.UUID) error {
	return s.flowRepo.DeleteStep(ctx, id, tenantID)
}

func (s *CancelFlowService) GetSession(ctx context.Context, id uuid.UUID) (*domain.CancelFlowSession, error) {
	return s.flowRepo.GetSessionByID(ctx, id)
}

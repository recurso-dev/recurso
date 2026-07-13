package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// DunningCampaignService handles dunning campaign business logic
type DunningCampaignService struct {
	campaignRepo        port.DunningCampaignRepository
	invoiceRepo         port.InvoiceRepository
	customerRepo        port.CustomerRepository
	notificationService *NotificationService
	smsSender           port.SMSSender
	logger              *slog.Logger
}

func NewDunningCampaignService(
	campaignRepo port.DunningCampaignRepository,
	invoiceRepo port.InvoiceRepository,
	customerRepo port.CustomerRepository,
	notificationService *NotificationService,
	smsSender port.SMSSender,
) *DunningCampaignService {
	return &DunningCampaignService{
		campaignRepo:        campaignRepo,
		invoiceRepo:         invoiceRepo,
		customerRepo:        customerRepo,
		notificationService: notificationService,
		smsSender:           smsSender,
		logger:              slog.Default().With("service", "dunning_campaign"),
	}
}

// TriggerCampaign starts a dunning campaign for a failed invoice
func (s *DunningCampaignService) TriggerCampaign(ctx context.Context, invoiceID uuid.UUID, triggerEvent string) error {
	inv, err := s.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil || inv == nil {
		return fmt.Errorf("failed to get invoice %s: %w", invoiceID, err)
	}

	// Check if there's already an active execution for this invoice
	existing, err := s.campaignRepo.GetExecutionByInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to check existing execution: %w", err)
	}
	if existing != nil {
		s.logger.Info("dunning campaign already active for invoice", "invoice_id", invoiceID)
		return nil
	}

	// Find active campaign for the tenant
	campaign, err := s.campaignRepo.GetActiveCampaignForTenant(ctx, inv.TenantID, triggerEvent)
	if err != nil {
		return fmt.Errorf("failed to get active campaign: %w", err)
	}
	if campaign == nil {
		// Auto-create default campaign if none exists
		campaign, err = s.EnsureDefaultCampaign(ctx, inv.TenantID)
		if err != nil {
			return fmt.Errorf("failed to create default campaign: %w", err)
		}
	}

	if len(campaign.Steps) == 0 {
		s.logger.Warn("dunning campaign has no steps", "campaign_id", campaign.ID)
		return nil
	}

	// Calculate first step time (delay relative to campaign start)
	now := time.Now().UTC()
	firstStep := campaign.Steps[0]
	nextStepAt := now.Add(time.Duration(firstStep.DelayHours) * time.Hour)

	exec := &domain.DunningCampaignExecution{
		ID:               uuid.New(),
		TenantID:         inv.TenantID,
		InvoiceID:        invoiceID,
		CampaignID:       campaign.ID,
		CurrentStepIndex: 0,
		Status:           domain.DunningExecStatusActive,
		StartedAt:        now,
		NextStepAt:       &nextStepAt,
	}

	if err := s.campaignRepo.CreateExecution(ctx, exec); err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	// Mark invoice as managed by campaign so the legacy dunning scheduler skips it
	inv.DunningManagedBy = "campaign"
	if err := s.invoiceRepo.Update(ctx, inv); err != nil {
		s.logger.Error("failed to set dunning_managed_by on invoice", "error", err, "invoice_id", invoiceID)
	}

	s.logger.Info("dunning campaign triggered",
		"campaign_id", campaign.ID,
		"invoice_id", invoiceID,
		"first_step_at", nextStepAt,
	)

	return nil
}

// ProcessDueSteps processes all due campaign steps (called by worker)
func (s *DunningCampaignService) ProcessDueSteps(ctx context.Context) error {
	executions, err := s.campaignRepo.GetDueExecutions(ctx, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to get due executions: %w", err)
	}

	for _, exec := range executions {
		if err := s.processExecution(ctx, exec); err != nil {
			s.logger.Error("failed to process dunning execution",
				"execution_id", exec.ID,
				"invoice_id", exec.InvoiceID,
				"error", err,
			)
		}
	}

	return nil
}

func (s *DunningCampaignService) processExecution(ctx context.Context, exec *domain.DunningCampaignExecution) error {
	// Get campaign with steps
	campaign, err := s.campaignRepo.GetCampaignByID(ctx, exec.CampaignID, exec.TenantID)
	if err != nil || campaign == nil {
		return fmt.Errorf("failed to get campaign %s: %w", exec.CampaignID, err)
	}

	if exec.CurrentStepIndex >= len(campaign.Steps) {
		// No more steps
		now := time.Now().UTC()
		exec.Status = domain.DunningExecStatusExhausted
		exec.CompletedAt = &now
		exec.NextStepAt = nil
		return s.campaignRepo.UpdateExecution(ctx, exec)
	}

	step := campaign.Steps[exec.CurrentStepIndex]

	// Get invoice and customer
	inv, err := s.invoiceRepo.GetByIDPublic(ctx, exec.InvoiceID)
	if err != nil || inv == nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// The campaign worker's background context carries no tenant — inject the
	// invoice's own before the tenant-scoped customer read (tenant-context
	// bug class; without this no campaign step ever executed).
	ctx = context.WithValue(ctx, domain.TenantIDKey, inv.TenantID)

	// If invoice is already paid, mark recovered
	if inv.Status == domain.InvoiceStatusPaid {
		return s.MarkRecovered(ctx, exec.InvoiceID)
	}

	customer, err := s.customerRepo.GetByID(ctx, inv.CustomerID)
	if err != nil || customer == nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// Dispatch by channel
	switch step.Channel {
	case domain.DunningChannelEmail:
		if err := s.sendDunningCampaignEmail(ctx, step, customer, inv); err != nil {
			s.logger.Error("failed to send dunning campaign email", "error", err, "step_id", step.ID)
		}

	case domain.DunningChannelSMS:
		if s.smsSender != nil && customer.Phone != "" {
			body := step.Body
			if body == "" {
				body = fmt.Sprintf("Payment of %s for invoice %s is overdue. Please update your payment method.",
					formatAmount(inv.Total, inv.Currency), inv.InvoiceNumber)
			}
			if err := s.smsSender.Send(ctx, port.SMSMessage{
				To:   customer.Phone,
				Body: body,
			}); err != nil {
				s.logger.Error("failed to send dunning SMS", "error", err, "step_id", step.ID)
			}
		}

	case domain.DunningChannelInApp:
		if step.IsPaymentWall {
			inv.PaymentWallActive = true
			if err := s.invoiceRepo.Update(ctx, inv); err != nil {
				s.logger.Error("failed to set payment wall", "error", err, "invoice_id", inv.ID)
			}
			s.logger.Info("payment wall activated", "invoice_id", inv.ID)
		}
	}

	// Advance to next step
	exec.CurrentStepIndex++
	if exec.CurrentStepIndex < len(campaign.Steps) {
		nextStep := campaign.Steps[exec.CurrentStepIndex]
		// Delay is relative to campaign start, not current time, to avoid drift
		nextAt := exec.StartedAt.Add(time.Duration(nextStep.DelayHours) * time.Hour)
		// If calculated time is in the past (e.g. worker was delayed), schedule for now
		if nextAt.Before(time.Now().UTC()) {
			nextAt = time.Now().UTC()
		}
		exec.NextStepAt = &nextAt
	} else {
		// No more steps
		now := time.Now().UTC()
		exec.Status = domain.DunningExecStatusExhausted
		exec.CompletedAt = &now
		exec.NextStepAt = nil
	}

	return s.campaignRepo.UpdateExecution(ctx, exec)
}

func (s *DunningCampaignService) sendDunningCampaignEmail(ctx context.Context, step domain.DunningCampaignStep, customer *domain.Customer, inv *domain.Invoice) error {
	if s.notificationService == nil {
		return nil
	}

	data := DunningCampaignEmailData{
		CustomerName:  domain.PtrToString(customer.Name),
		CustomerEmail: customer.Email,
		InvoiceNumber: inv.InvoiceNumber,
		Amount:        formatAmount(inv.Total, inv.Currency),
		Subject:       step.Subject,
		Body:          step.Body,
	}

	return s.notificationService.SendDunningCampaignEmail(ctx, data)
}

// MarkRecovered marks a dunning campaign execution as recovered (payment succeeded)
func (s *DunningCampaignService) MarkRecovered(ctx context.Context, invoiceID uuid.UUID) error {
	exec, err := s.campaignRepo.GetExecutionByInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}
	if exec == nil {
		return nil // No active campaign for this invoice
	}

	now := time.Now().UTC()
	exec.Status = domain.DunningExecStatusRecovered
	exec.CompletedAt = &now
	exec.NextStepAt = nil

	if err := s.campaignRepo.UpdateExecution(ctx, exec); err != nil {
		return fmt.Errorf("failed to mark execution recovered: %w", err)
	}

	// Clear payment wall and reset dunning_managed_by
	inv, err := s.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err == nil && inv != nil {
		changed := false
		if inv.PaymentWallActive {
			inv.PaymentWallActive = false
			changed = true
		}
		if inv.DunningManagedBy == "campaign" {
			inv.DunningManagedBy = "scheduler"
			changed = true
		}
		if changed {
			if err := s.invoiceRepo.Update(ctx, inv); err != nil {
				s.logger.Error("failed to update invoice after campaign recovery", "error", err, "invoice_id", invoiceID)
			}
		}
	}

	s.logger.Info("dunning campaign marked recovered", "invoice_id", invoiceID, "execution_id", exec.ID)
	return nil
}

// GetPaymentWallStatus returns whether a payment wall is active for an invoice
// GetPaymentWallStatus is tenant-scoped via GetByID (the tenant is injected by
// the handler): a caller can only read the payment-wall flag for its own
// invoices, not probe any invoice id in the system.
func (s *DunningCampaignService) GetPaymentWallStatus(ctx context.Context, invoiceID uuid.UUID) (bool, error) {
	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return false, fmt.Errorf("failed to get invoice: %w", err)
	}
	if inv == nil {
		return false, fmt.Errorf("invoice not found")
	}
	return inv.PaymentWallActive, nil
}

// EnsureDefaultCampaign creates a default 4-step dunning campaign if none exists
func (s *DunningCampaignService) EnsureDefaultCampaign(ctx context.Context, tenantID uuid.UUID) (*domain.DunningCampaign, error) {
	existing, err := s.campaignRepo.GetActiveCampaignForTenant(ctx, tenantID, "payment_failed")
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now().UTC()
	campaign := &domain.DunningCampaign{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         "Default Payment Recovery",
		IsActive:     true,
		TriggerEvent: "payment_failed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.campaignRepo.CreateCampaign(ctx, campaign); err != nil {
		return nil, fmt.Errorf("failed to create default campaign: %w", err)
	}

	steps := []domain.DunningCampaignStep{
		{
			ID:         uuid.New(),
			CampaignID: campaign.ID,
			StepOrder:  0,
			Channel:    domain.DunningChannelEmail,
			DelayHours: 0,
			Subject:    "Payment Failed - Action Required",
			Body:       "Your recent payment has failed. Please update your payment method to continue your service.",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			CampaignID: campaign.ID,
			StepOrder:  1,
			Channel:    domain.DunningChannelEmail,
			DelayHours: 48,
			Subject:    "Reminder: Payment Still Pending",
			Body:       "We still haven't been able to process your payment. Please update your payment method to avoid service interruption.",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			CampaignID: campaign.ID,
			StepOrder:  2,
			Channel:    domain.DunningChannelSMS,
			DelayHours: 96,
			Body:       "Your payment is overdue. Please update your payment method to continue your service.",
			CreatedAt:  now,
		},
		{
			ID:            uuid.New(),
			CampaignID:    campaign.ID,
			StepOrder:     3,
			Channel:       domain.DunningChannelInApp,
			DelayHours:    168,
			IsPaymentWall: true,
			Subject:       "Service Suspended",
			Body:          "Your service has been suspended due to non-payment. Please update your payment method to restore access.",
			CreatedAt:     now,
		},
	}

	for i := range steps {
		if err := s.campaignRepo.CreateStep(ctx, &steps[i], tenantID); err != nil {
			return nil, fmt.Errorf("failed to create default campaign step: %w", err)
		}
	}

	campaign.Steps = steps
	s.logger.Info("default dunning campaign created", "tenant_id", tenantID, "campaign_id", campaign.ID)
	return campaign, nil
}

// DunningCampaignEmailData for dunning campaign emails
type DunningCampaignEmailData struct {
	CustomerName  string
	CustomerEmail string
	InvoiceNumber string
	Amount        string
	Subject       string
	Body          string
}

// SendDunningCampaignEmail sends a dunning campaign email via the notification service
func (ns *NotificationService) SendDunningCampaignEmail(ctx context.Context, data DunningCampaignEmailData) error {
	content, err := ns.renderTemplate(email.DunningCampaignEmailTemplate, data)
	if err != nil {
		return err
	}

	subject := data.Subject
	if subject == "" {
		subject = "Action Required: Payment Overdue"
	}

	html, err := ns.wrapInBaseTemplate(subject, content)
	if err != nil {
		return err
	}

	return ns.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  subject,
		HTMLBody: html,
	})
}

// --- Pass-through CRUD methods for handler ---

func (s *DunningCampaignService) ListCampaigns(ctx context.Context, tenantID uuid.UUID) ([]*domain.DunningCampaign, error) {
	return s.campaignRepo.ListCampaignsByTenant(ctx, tenantID)
}

func (s *DunningCampaignService) GetCampaignByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.DunningCampaign, error) {
	return s.campaignRepo.GetCampaignByID(ctx, id, tenantID)
}

func (s *DunningCampaignService) CreateCampaign(ctx context.Context, campaign *domain.DunningCampaign) error {
	return s.campaignRepo.CreateCampaign(ctx, campaign)
}

func (s *DunningCampaignService) UpdateCampaign(ctx context.Context, campaign *domain.DunningCampaign) error {
	return s.campaignRepo.UpdateCampaign(ctx, campaign)
}

func (s *DunningCampaignService) CreateStep(ctx context.Context, step *domain.DunningCampaignStep, tenantID uuid.UUID) error {
	return s.campaignRepo.CreateStep(ctx, step, tenantID)
}

func (s *DunningCampaignService) UpdateStep(ctx context.Context, step *domain.DunningCampaignStep, tenantID uuid.UUID) error {
	return s.campaignRepo.UpdateStep(ctx, step, tenantID)
}

func (s *DunningCampaignService) DeleteStep(ctx context.Context, id, tenantID uuid.UUID) error {
	return s.campaignRepo.DeleteStep(ctx, id, tenantID)
}

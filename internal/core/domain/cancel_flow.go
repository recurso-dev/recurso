package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OfferType represents the type of retention offer
type OfferType string

const (
	OfferTypeDiscount       OfferType = "discount"
	OfferTypePause          OfferType = "pause"
	OfferTypePlanSwitch     OfferType = "plan_switch"
	OfferTypeTrialExtension OfferType = "trial_extension"
	OfferTypeCustom         OfferType = "custom"
)

// CancelFlowStepType represents the type of a cancel flow step
type CancelFlowStepType string

const (
	StepTypeSurvey       CancelFlowStepType = "survey"
	StepTypeOffer        CancelFlowStepType = "offer"
	StepTypeConfirmation CancelFlowStepType = "confirmation"
)

// CancelFlowSessionStatus represents the status of a cancel flow session
type CancelFlowSessionStatus string

const (
	SessionStatusInProgress CancelFlowSessionStatus = "in_progress"
	SessionStatusCompleted  CancelFlowSessionStatus = "completed"
	SessionStatusCancelled  CancelFlowSessionStatus = "cancelled"
	SessionStatusSaved      CancelFlowSessionStatus = "saved"
)

// CancelFlow represents a per-tenant configurable cancellation flow
type CancelFlow struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	IsDefault    bool      `json:"is_default"`
	CooldownDays int       `json:"cooldown_days"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Steps []CancelFlowStep `json:"steps,omitempty"`
}

// CancelFlowStep represents an ordered step in a cancel flow
type CancelFlowStep struct {
	ID        uuid.UUID          `json:"id"`
	FlowID    uuid.UUID          `json:"flow_id"`
	StepOrder int                `json:"step_order"`
	StepType  CancelFlowStepType `json:"step_type"`
	Config    json.RawMessage    `json:"config"`
	CreatedAt time.Time          `json:"created_at"`
}

// CancelFlowSession tracks a customer's journey through a cancel flow
type CancelFlowSession struct {
	ID                 uuid.UUID               `json:"id"`
	TenantID           uuid.UUID               `json:"tenant_id"`
	CustomerID         uuid.UUID               `json:"customer_id"`
	SubscriptionID     uuid.UUID               `json:"subscription_id"`
	FlowID             uuid.UUID               `json:"flow_id"`
	Status             CancelFlowSessionStatus `json:"status"`
	CurrentStepIndex   int                     `json:"current_step_index"`
	CancellationReason string                  `json:"cancellation_reason,omitempty"`
	Feedback           string                  `json:"feedback,omitempty"`
	OfferPresented     json.RawMessage         `json:"offer_presented,omitempty"`
	OfferAccepted      bool                    `json:"offer_accepted"`
	SavedByOffer       bool                    `json:"saved_by_offer"`
	StartedAt          time.Time               `json:"started_at"`
	CompletedAt        *time.Time              `json:"completed_at,omitempty"`
}

// RetentionOffer contains the details of a retention offer
type RetentionOffer struct {
	Type                   OfferType  `json:"type"`
	DiscountPercent        int        `json:"discount_percent,omitempty"`
	DiscountDurationMonths int        `json:"discount_duration_months,omitempty"`
	PauseMonths            int        `json:"pause_months,omitempty"`
	SwitchToPlanID         *uuid.UUID `json:"switch_to_plan_id,omitempty"`
	ExtensionDays          int        `json:"extension_days,omitempty"`
}

// FlowStats contains aggregated statistics for a cancel flow
type FlowStats struct {
	TotalSessions   int                `json:"total_sessions"`
	CompletedCount  int                `json:"completed_count"`
	SavedCount      int                `json:"saved_count"`
	SaveRate        float64            `json:"save_rate"`
	ReasonBreakdown map[string]int     `json:"reason_breakdown"`
	OfferAcceptRate float64            `json:"offer_accept_rate"`
}

// CooldownActive checks if a customer is within the cooldown period
func CooldownActive(lastOfferAcceptedAt *time.Time, cooldownDays int) bool {
	if lastOfferAcceptedAt == nil || cooldownDays <= 0 {
		return false
	}
	cooldownEnd := lastOfferAcceptedAt.AddDate(0, 0, cooldownDays)
	return time.Now().Before(cooldownEnd)
}

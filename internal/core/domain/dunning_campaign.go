package domain

import (
	"time"

	"github.com/google/uuid"
)

// DunningChannel represents the communication channel for a dunning step
type DunningChannel string

const (
	DunningChannelEmail DunningChannel = "email"
	DunningChannelSMS   DunningChannel = "sms"
	DunningChannelInApp DunningChannel = "in_app"
)

// DunningCampaignExecutionStatus represents the status of a dunning campaign execution
type DunningCampaignExecutionStatus string

const (
	DunningExecStatusActive    DunningCampaignExecutionStatus = "active"
	DunningExecStatusCompleted DunningCampaignExecutionStatus = "completed"
	DunningExecStatusRecovered DunningCampaignExecutionStatus = "recovered"
	DunningExecStatusExhausted DunningCampaignExecutionStatus = "exhausted"
)

// DunningCampaign represents a configurable dunning campaign for a tenant
type DunningCampaign struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	TriggerEvent string    `json:"trigger_event"` // payment_failed, invoice_overdue
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Steps []DunningCampaignStep `json:"steps,omitempty"`
}

// DunningCampaignStep represents an ordered step in a dunning campaign
type DunningCampaignStep struct {
	ID            uuid.UUID      `json:"id"`
	CampaignID    uuid.UUID      `json:"campaign_id"`
	StepOrder     int            `json:"step_order"`
	Channel       DunningChannel `json:"channel"`
	DelayHours    int            `json:"delay_hours"`
	TemplateName  string         `json:"template_name,omitempty"`
	Subject       string         `json:"subject,omitempty"`
	Body          string         `json:"body,omitempty"`
	IsPaymentWall bool           `json:"is_payment_wall"`
	CreatedAt     time.Time      `json:"created_at"`
}

// DunningCampaignExecution tracks the execution of a campaign for a specific invoice
type DunningCampaignExecution struct {
	ID               uuid.UUID                      `json:"id"`
	TenantID         uuid.UUID                      `json:"tenant_id"`
	InvoiceID        uuid.UUID                      `json:"invoice_id"`
	CampaignID       uuid.UUID                      `json:"campaign_id"`
	CurrentStepIndex int                            `json:"current_step_index"`
	Status           DunningCampaignExecutionStatus `json:"status"`
	StartedAt        time.Time                      `json:"started_at"`
	NextStepAt       *time.Time                     `json:"next_step_at,omitempty"`
	CompletedAt      *time.Time                     `json:"completed_at,omitempty"`
}

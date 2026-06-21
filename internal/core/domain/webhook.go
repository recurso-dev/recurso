package domain

import (
	"time"

	"github.com/google/uuid"
)

// WebhookEndpoint represents a registered webhook URL for a tenant
type WebhookEndpoint struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	URL       string    `json:"url" db:"url"`
	Secret    string    `json:"-" db:"secret"` // Hidden in JSON responses
	Events    []string  `json:"events" db:"events"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Event represents a billing event that occurred in the system
type Event struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	TenantID   uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Type       string                 `json:"type" db:"type"`
	ObjectType string                 `json:"object_type" db:"object_type"`
	ObjectID   uuid.UUID              `json:"object_id" db:"object_id"`
	Data       map[string]interface{} `json:"data" db:"data"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// EventDelivery tracks the delivery of an event to a webhook endpoint
type EventDelivery struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	EventID           uuid.UUID  `json:"event_id" db:"event_id"`
	WebhookEndpointID uuid.UUID  `json:"webhook_endpoint_id" db:"webhook_endpoint_id"`
	StatusCode        int        `json:"status_code" db:"status_code"`
	ResponseBody      string     `json:"response_body" db:"response_body"`
	Attempt           int        `json:"attempt" db:"attempt"`
	NextRetryAt       *time.Time `json:"next_retry_at" db:"next_retry_at"`
	DeliveredAt       *time.Time `json:"delivered_at" db:"delivered_at"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// Common event types
const (
	EventTypeInvoiceCreated       = "invoice.created"
	EventTypeInvoicePaid          = "invoice.paid"
	EventTypeInvoicePaymentFailed = "invoice.payment_failed"
	EventTypeSubscriptionCreated  = "subscription.created"
	EventTypeSubscriptionCanceled = "subscription.canceled"
	EventTypeSubscriptionRenewed  = "subscription.renewed"
	EventTypeCustomerCreated      = "customer.created"
	EventTypeCustomerUpdated      = "customer.updated"
	EventTypePaymentSucceeded     = "payment.succeeded"
	EventTypePaymentFailed        = "payment.failed"
)

// AllEventTypes returns all supported event types
func AllEventTypes() []string {
	return []string{
		EventTypeInvoiceCreated,
		EventTypeInvoicePaid,
		EventTypeInvoicePaymentFailed,
		EventTypeSubscriptionCreated,
		EventTypeSubscriptionCanceled,
		EventTypeSubscriptionRenewed,
		EventTypeCustomerCreated,
		EventTypeCustomerUpdated,
		EventTypePaymentSucceeded,
		EventTypePaymentFailed,
	}
}

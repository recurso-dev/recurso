package domain

import (
	"time"

	"github.com/google/uuid"
)

// ConsentType represents different types of consent
type ConsentType string

const (
	ConsentTypeRecurringBilling ConsentType = "recurring_billing"
	ConsentTypeEmailMarketing   ConsentType = "email_marketing"
	ConsentTypeDataProcessing   ConsentType = "data_processing"
	ConsentTypeTermsOfService   ConsentType = "terms_of_service"
	ConsentTypePrivacyPolicy    ConsentType = "privacy_policy"
)

// Consent tracks user consent for legal compliance
type Consent struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	TenantID       uuid.UUID   `json:"tenant_id" db:"tenant_id"`
	CustomerID     uuid.UUID   `json:"customer_id" db:"customer_id"`
	SubscriptionID *uuid.UUID  `json:"subscription_id,omitempty" db:"subscription_id"`
	ConsentType    ConsentType `json:"consent_type" db:"consent_type"`

	// Consent details
	Granted   bool       `json:"granted" db:"granted"`
	GrantedAt time.Time  `json:"granted_at" db:"granted_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`

	// Audit trail
	IPAddress   string `json:"ip_address" db:"ip_address"`
	UserAgent   string `json:"user_agent" db:"user_agent"`
	ConsentText string `json:"consent_text" db:"consent_text"` // Exact text shown to user
	Version     string `json:"version" db:"version"`           // Version of consent text

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ConsentRecord is used when capturing consent
type ConsentRecord struct {
	CustomerID     uuid.UUID   `json:"customer_id"`
	SubscriptionID *uuid.UUID  `json:"subscription_id,omitempty"`
	ConsentType    ConsentType `json:"consent_type"`
	Granted        bool        `json:"granted"`
	IPAddress      string      `json:"ip_address"`
	UserAgent      string      `json:"user_agent"`
	ConsentText    string      `json:"consent_text"`
	Version        string      `json:"version"`
}

// RecurringBillingConsentText is the standard consent text for recurring billing
const RecurringBillingConsentText = `I authorize recurring charges to my payment method for the selected subscription plan. I understand that:
- I will be charged automatically on each billing cycle
- I will receive a reminder email 24 hours before each charge
- I can cancel my subscription at any time from my account dashboard
- Refunds are processed according to the refund policy`

// CurrentConsentVersion tracks the version of consent text
const CurrentConsentVersion = "2024.01.1"

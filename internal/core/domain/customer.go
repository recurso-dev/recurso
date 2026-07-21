package domain

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	Email           string         `json:"email"`
	Name            *string        `json:"name"`
	Phone           string         `json:"phone"`
	TaxID           *string        `json:"tax_id"`
	BillingAddress  BillingAddress `json:"billing_address"`
	LedgerAccountID uuid.UUID      `json:"ledger_account_id"` // Stored as UUID in PG, converted to u128 for TB
	GSTIN           *string        `json:"gstin"`             // P24
	TaxType         string         `json:"tax_type"`          // P24: 'business', 'consumer'
	PlaceOfSupply   *string        `json:"place_of_supply"`   // P24: State Code
	// US sales-tax exemption (Track D · D2). When TaxExempt is set, the number
	// and entity-use code are passed to the tax provider so it returns zero tax
	// and records an exempt sale, rather than the engine short-circuiting.
	TaxExempt          bool                   `json:"tax_exempt" db:"tax_exempt"`
	TaxExemptionNumber string                 `json:"tax_exemption_number" db:"tax_exemption_number"`
	TaxExemptionCode   string                 `json:"tax_exemption_code" db:"tax_exemption_code"` // provider entity-use / usage code; also the reason
	ReferralCode       *string                `json:"referral_code" db:"referral_code"`           // P42
	RiskScore          int                    `json:"risk_score" db:"risk_score"`                 // P45: 0-100
	RiskFactors        map[string]interface{} `json:"risk_factors" db:"risk_factors"`             // P45: JSON
	CardBrand          *string                `json:"card_brand,omitempty" db:"card_brand"`
	CardLast4          *string                `json:"card_last4,omitempty" db:"card_last4"`
	CardExpMonth       *int                   `json:"card_exp_month,omitempty" db:"card_exp_month"`
	CardExpYear        *int                   `json:"card_exp_year,omitempty" db:"card_exp_year"`
	CardTokenID        *string                `json:"card_token_id,omitempty" db:"card_token_id"`
	CardFingerprint    *string                `json:"card_fingerprint,omitempty" db:"card_fingerprint"`
	// Active is the soft-archive flag. Archiving is blocked while the customer
	// has active subscriptions; archived customers keep full billing history.
	Active    bool      `json:"active" db:"active"`
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is bumped by CustomerRepository.Update (the canonical edit
	// path for the fields pushed to accounting providers). The accounting
	// sync compares it against the entity's mapping to skip unchanged
	// customers; a zero value means "unknown" and always syncs.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type BillingAddress struct {
	Line1   string `json:"line1"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

type CustomerFilter struct {
	Search  string
	Email   string // For portal lookup
	Country string
	Status  string // active or inactive
	Limit   int
	Offset  int
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	Email           string                 `json:"email"`
	Name            *string                `json:"name"`
	Phone           string                 `json:"phone"`
	TaxID           *string                `json:"tax_id"`
	BillingAddress  BillingAddress         `json:"billing_address"`
	LedgerAccountID uuid.UUID              `json:"ledger_account_id"`                // Stored as UUID in PG, converted to u128 for TB
	GSTIN           *string                `json:"gstin"`                            // P24
	TaxType         string                 `json:"tax_type"`                         // P24: 'business', 'consumer'
	PlaceOfSupply   *string                `json:"place_of_supply"`                  // P24: State Code
	ReferralCode    *string                `json:"referral_code" db:"referral_code"` // P42
	RiskScore       int                    `json:"risk_score" db:"risk_score"`       // P45: 0-100
	RiskFactors     map[string]interface{} `json:"risk_factors" db:"risk_factors"`   // P45: JSON
	CardBrand       *string                `json:"card_brand,omitempty" db:"card_brand"`
	CardLast4       *string                `json:"card_last4,omitempty" db:"card_last4"`
	CardExpMonth    *int                   `json:"card_exp_month,omitempty" db:"card_exp_month"`
	CardExpYear     *int                   `json:"card_exp_year,omitempty" db:"card_exp_year"`
	CardTokenID     *string                `json:"card_token_id,omitempty" db:"card_token_id"`
	CardFingerprint *string                `json:"card_fingerprint,omitempty" db:"card_fingerprint"`
	CreatedAt       time.Time              `json:"created_at"`
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

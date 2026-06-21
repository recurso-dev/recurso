package domain

import (
	"time"

	"github.com/google/uuid"
)

// AccountType defines the nature of the account (Asset, Liability, Equity, Revenue, Expense)
type AccountType int

const (
	AccountTypeAsset     AccountType = 1
	AccountTypeLiability AccountType = 2
	AccountTypeEquity    AccountType = 3
	AccountTypeRevenue   AccountType = 4
	AccountTypeExpense   AccountType = 5
)

// LedgerAccount represents a financial account in the ledger.
type LedgerAccount struct {
	ID            uuid.UUID   `json:"id" db:"id"`
	TenantID      uuid.UUID   `json:"tenant_id" db:"tenant_id"`
	Name          string      `json:"name" db:"name"`
	Type          AccountType `json:"type" db:"type"` // asset, liability, equity, revenue, expense
	Code          int         `json:"code" db:"code"`     // TB: 1000 for Cash etc.
	LedgerID      uint32      `json:"ledger_id" db:"ledger_id"`
	UserData128   uint128     `json:"user_data_128" db:"user_data_128"`
	CreditsPosted uint64      `json:"credits_posted" db:"credits_posted"`
	DebitsPosted  uint64      `json:"debits_posted" db:"debits_posted"`
	Currency      string      `json:"currency" db:"currency"` // P26
	Balance       int64       `json:"balance" db:"balance"`   // Cached balance (snapshot)
	CreatedAt     time.Time   `json:"created_at" db:"created_at"`
}

// Transaction (Transfer) represents a movement of funds between two accounts.
type LedgerTransaction struct {
	ID              uuid.UUID `json:"id"`
	DebitAccountID  uuid.UUID `json:"debit_account_id"`
	CreditAccountID uuid.UUID `json:"credit_account_id"`
	Amount          uint64    `json:"amount"` // Atomic units
	LedgerID        uint32    `json:"ledger_id"`
	Code            uint16    `json:"code"` // Transaction Type (e.g. 1=Invoice, 2=Payment)
	Timestamp       time.Time `json:"timestamp"`
}

// Helper struct for uint128 since Go doesn't have it native, 
// strictly for domain representation before mapping to TB client.
type uint128 struct {
	High uint64
	Low  uint64
}

// UUIDToUint128 converts UUID to custom Uint128 for domain usage
func UUIDToUint128(id uuid.UUID) uint128 {
	// Not implemented perfectly here without bit shifting
	// Just a placeholder to check compilation
	return uint128{High: 0, Low: 0}
}

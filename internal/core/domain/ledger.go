package domain

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
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

var accountTypeNames = map[AccountType]string{
	AccountTypeAsset:     "asset",
	AccountTypeLiability: "liability",
	AccountTypeEquity:    "equity",
	AccountTypeRevenue:   "revenue",
	AccountTypeExpense:   "expense",
}

// String returns the human-readable account type.
func (a AccountType) String() string {
	if s, ok := accountTypeNames[a]; ok {
		return s
	}
	return strconv.Itoa(int(a))
}

// Scan reads an AccountType from the database, tolerating both the numeric
// codes ("1".."5") and the legacy human-readable words ("asset"..) that
// older code versions wrote into the varchar `type` column.
func (a *AccountType) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case nil:
		*a = 0
		return nil
	case int64:
		*a = AccountType(v)
		return nil
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("cannot scan %T into AccountType", src)
	}
	s = strings.TrimSpace(strings.ToLower(s))
	for t, name := range accountTypeNames {
		if s == name {
			*a = t
			return nil
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid AccountType %q", s)
	}
	*a = AccountType(n)
	return nil
}

// Value writes the numeric code to the database (canonical form).
func (a AccountType) Value() (driver.Value, error) {
	return int64(a), nil
}

// IsDebitNormal reports whether the account's balance normally sits on the
// debit side — assets and expenses. Liabilities, equity, and revenue are
// credit-normal. The trial balance uses this to detect an abnormal balance
// sign (e.g. a liability carrying a net debit, which signals a posting bug).
func (a AccountType) IsDebitNormal() bool {
	return a == AccountTypeAsset || a == AccountTypeExpense
}

// TrialBalanceLine is one account's posted totals plus its balance expressed on
// the account's normal side. Abnormal is true when that balance is negative.
type TrialBalanceLine struct {
	AccountID uuid.UUID   `json:"account_id"`
	Code      int         `json:"code"`
	Name      string      `json:"name"`
	Type      AccountType `json:"type"`
	Debits    int64       `json:"debits"`   // minor units posted to the debit side
	Credits   int64       `json:"credits"`  // minor units posted to the credit side
	Balance   int64       `json:"balance"`  // signed, on the account's normal side
	Abnormal  bool        `json:"abnormal"` // Balance < 0 — wrong sign for this account
}

// TrialBalance is a tenant's chart of accounts with posted totals. Balanced is
// the fundamental double-entry invariant: total debits == total credits across
// every account. It is the canonical artifact for proving the books balance.
type TrialBalance struct {
	TenantID     uuid.UUID          `json:"tenant_id"`
	Lines        []TrialBalanceLine `json:"lines"`
	TotalDebits  int64              `json:"total_debits"`
	TotalCredits int64              `json:"total_credits"`
	Balanced     bool               `json:"balanced"`
	AsOf         time.Time          `json:"as_of"`
}

// Chart of Accounts — standard account codes
const (
	AccountCodeAR                = 1100 // Accounts Receivable (Asset)
	AccountCodeCash              = 1000 // Cash (Asset)
	AccountCodeRevenue           = 4000 // Revenue (Income)
	AccountCodeDeferredRevenue   = 2100 // Deferred Revenue (Liability)
	AccountCodeRecognizedRevenue = 4100 // Recognized Revenue (Income)
	AccountCodeTaxPayable        = 2200 // Tax Payable (Liability)
	AccountCodeCustomerCredit    = 2300 // Customer Credit (Liability) — account credit owed to customers (ENG-154)
	AccountCodeRefunds           = 5000 // Refunds (Expense)
	AccountCodeCreditsIssued     = 5100 // Credits & Adjustments (Expense) — cost of manually-issued account credit (ENG-154)
)

// Ledger transaction codes (LedgerTransaction.Code): 1 = invoice, 2 = revenue
// recognition, 3 = payment. LedgerCodeOutputTax reclassifies collected GST out
// of revenue into Tax Payable (ENG-159); it's a distinct code so it doesn't
// collide with the invoice's Code-1 row under uq_ledger_tx_reference_code.
const LedgerCodeOutputTax uint16 = 6

// StandardChartOfAccounts returns the default accounts for a tenant
func TenantChartOfAccounts(tenantID uuid.UUID) []*LedgerAccount {
	return []*LedgerAccount{
		{ID: uuid.New(), TenantID: tenantID, Name: "Cash", Type: AccountTypeAsset, Code: AccountCodeCash, LedgerID: 1},
		{ID: uuid.New(), TenantID: tenantID, Name: "Accounts Receivable", Type: AccountTypeAsset, Code: AccountCodeAR, LedgerID: 1},
		{ID: uuid.New(), TenantID: tenantID, Name: "Deferred Revenue", Type: AccountTypeLiability, Code: AccountCodeDeferredRevenue, LedgerID: 1},
		{ID: uuid.New(), TenantID: tenantID, Name: "Tax Payable", Type: AccountTypeLiability, Code: AccountCodeTaxPayable, LedgerID: 1},
		{ID: uuid.New(), TenantID: tenantID, Name: "Customer Credit", Type: AccountTypeLiability, Code: AccountCodeCustomerCredit, LedgerID: 1},
		{ID: uuid.New(), TenantID: tenantID, Name: "Revenue", Type: AccountTypeRevenue, Code: AccountCodeRevenue, LedgerID: 1},
		{ID: uuid.New(), TenantID: tenantID, Name: "Recognized Revenue", Type: AccountTypeRevenue, Code: AccountCodeRecognizedRevenue, LedgerID: 1},
		{ID: uuid.New(), TenantID: tenantID, Name: "Refunds", Type: AccountTypeExpense, Code: AccountCodeRefunds, LedgerID: 1},
		{ID: uuid.New(), TenantID: tenantID, Name: "Credits & Adjustments", Type: AccountTypeExpense, Code: AccountCodeCreditsIssued, LedgerID: 1},
	}
}

// DeferredRollforward is the movement of the Deferred Revenue account over a
// period, sourced straight from the ledger: the opening balance, new deferrals
// booked (credits), amounts recognized or reversed out (debits), and the
// closing balance. Opening + Added - Released == Closing. The canonical
// deferred-revenue rollforward an auditor ties to the trial balance.
type DeferredRollforward struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Opening     int64     `json:"opening"`  // deferred balance at period start, minor units
	Added       int64     `json:"added"`    // new deferrals booked in period (credits)
	Released    int64     `json:"released"` // recognized/reversed in period (debits)
	Closing     int64     `json:"closing"`  // Opening + Added - Released
}

// GeneralLedgerRow is one posted transaction flattened for export: the two
// accounts value moved between (by code and name), the amount, and its
// provenance (code, reference, description). Used for the read-only GL export.
type GeneralLedgerRow struct {
	TransactionID     uuid.UUID `json:"transaction_id"`
	Timestamp         time.Time `json:"timestamp"`
	Code              uint16    `json:"code"`
	DebitAccountCode  int       `json:"debit_account_code"`
	DebitAccountName  string    `json:"debit_account_name"`
	CreditAccountCode int       `json:"credit_account_code"`
	CreditAccountName string    `json:"credit_account_name"`
	Amount            int64     `json:"amount"` // minor units
	ReferenceID       uuid.UUID `json:"reference_id"`
	Description       string    `json:"description"`
}

// LedgerAccount represents a financial account in the ledger.
type LedgerAccount struct {
	ID            uuid.UUID   `json:"id" db:"id"`
	TenantID      uuid.UUID   `json:"tenant_id" db:"tenant_id"`
	Name          string      `json:"name" db:"name"`
	Type          AccountType `json:"type" db:"type"` // asset, liability, equity, revenue, expense
	Code          int         `json:"code" db:"code"` // TB: 1000 for Cash etc.
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
	ID              uuid.UUID `json:"id" db:"id"`
	DebitAccountID  uuid.UUID `json:"debit_account_id" db:"debit_account_id"`
	CreditAccountID uuid.UUID `json:"credit_account_id" db:"credit_account_id"`
	Amount          uint64    `json:"amount" db:"amount"`
	LedgerID        uint32    `json:"ledger_id" db:"ledger_id"`
	Code            uint16    `json:"code" db:"code"`
	ReferenceID     uuid.UUID `json:"reference_id" db:"reference_id"` // Invoice/Payment ID
	Description     string    `json:"description" db:"description"`
	Timestamp       time.Time `json:"timestamp" db:"created_at"`
}

// Helper struct for uint128 since Go doesn't have it native,
// strictly for domain representation before mapping to TB client.
type uint128 struct {
	High uint64
	Low  uint64
}

// UUIDToUint128 converts UUID to custom Uint128 for domain usage via bit-shifting
func UUIDToUint128(id uuid.UUID) uint128 {
	b := id[:]
	var high, low uint64
	for i := 0; i < 8; i++ {
		high = (high << 8) | uint64(b[i])
	}
	for i := 8; i < 16; i++ {
		low = (low << 8) | uint64(b[i])
	}
	return uint128{High: high, Low: low}
}

// Uint128ToUUID converts a uint128 back to a UUID
func Uint128ToUUID(v uint128) uuid.UUID {
	var b [16]byte
	for i := 7; i >= 0; i-- {
		b[i] = byte(v.High & 0xFF)
		v.High >>= 8
	}
	for i := 15; i >= 8; i-- {
		b[i] = byte(v.Low & 0xFF)
		v.Low >>= 8
	}
	return uuid.UUID(b)
}

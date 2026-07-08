package recurso

import (
	"context"
	"net/http"
	"time"
)

// LedgerAccount is a double-entry ledger account.
type LedgerAccount struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Code          int       `json:"code"`
	LedgerID      int       `json:"ledger_id"`
	CreditsPosted int64     `json:"credits_posted"`
	DebitsPosted  int64     `json:"debits_posted"`
	Currency      string    `json:"currency"`
	Balance       int64     `json:"balance"`
	CreatedAt     time.Time `json:"created_at"`
}

// LedgerTransaction is a ledger transfer touching an account.
type LedgerTransaction struct {
	ID              string    `json:"id"`
	DebitAccountID  string    `json:"debit_account_id"`
	CreditAccountID string    `json:"credit_account_id"`
	Amount          int64     `json:"amount"`
	LedgerID        int       `json:"ledger_id"`
	Code            int       `json:"code"`
	ReferenceID     string    `json:"reference_id"`
	Description     string    `json:"description"`
	Timestamp       time.Time `json:"timestamp"`
}

// LedgerEntriesParams selects the account whose entries to list.
type LedgerEntriesParams struct {
	AccountID string
}

// LedgerService groups the finance ledger endpoints.
type LedgerService struct{ client *Client }

// Accounts lists the tenant's ledger accounts.
func (s *LedgerService) Accounts(ctx context.Context) ([]LedgerAccount, error) {
	return getData[[]LedgerAccount](ctx, s.client, http.MethodGet, "/ledger/accounts", nil)
}

// Entries lists ledger transfers touching the given account.
func (s *LedgerService) Entries(ctx context.Context, params *LedgerEntriesParams) ([]LedgerTransaction, error) {
	path := "/ledger/entries"
	if params != nil {
		path = newQuery().str("account_id", params.AccountID).apply(path)
	}
	return getData[[]LedgerTransaction](ctx, s.client, http.MethodGet, path, nil)
}

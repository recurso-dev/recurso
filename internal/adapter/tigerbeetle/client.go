package tigerbeetle

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

type LedgerClient struct {
	client tb.Client
}

func NewLedgerClient(clusterID uint32, addresses []string) (*LedgerClient, error) {
	// 3rd arg is max_concurrency, 0 means default
	client, err := tb.NewClient(tbtypes.ToUint128(uint64(clusterID)), addresses, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create tb client: %w", err)
	}
	return &LedgerClient{client: client}, nil
}

func (c *LedgerClient) Close() {
	c.client.Close()
}

// CreateAccounts creates new accounts in the ledger.
// Idempotent: If account exists, it returns an error result for that account, but we might treat it leniently.
func (c *LedgerClient) CreateAccounts(ctx context.Context, accounts []*domain.LedgerAccount) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("ledger client not connected")
	}

	tbAccounts := make([]tbtypes.Account, len(accounts))

	for i, acc := range accounts {
		// Convert UUID to Uint128
		id, err := tbtypes.HexStringToUint128(acc.ID.String())
		if err != nil {
			return fmt.Errorf("invalid uuid %s: %w", acc.ID, err)
		}

		tbAccounts[i] = tbtypes.Account{
			ID:             id,
			DebitsPosted:   tbtypes.ToUint128(acc.DebitsPosted),
			CreditsPosted:  tbtypes.ToUint128(acc.CreditsPosted),
			UserData128:    tbtypes.ToUint128(acc.UserData128.Low), // P4: Simplification if older lib doesn't support High/Low in ToUint128?
			UserData64:     0,
			UserData32:     0,
			Reserved:       0,
			Ledger:         acc.LedgerID,
			Code:           uint16(acc.Code),
			Flags:          0, // No specific flags for now (e.g. History)
			Timestamp:      0, // TB sets this
		}
	}

	results, err := c.client.CreateAccounts(tbAccounts)
	if err != nil {
		return fmt.Errorf("tb driver error: %w", err)
	}

	for _, res := range results {
		// Ignore "AccountExists" error for idempotency?
		// For now, let's log everything that isn't success.
		if res.Result != tbtypes.AccountExists {
			log.Printf("Error creating account index %d: %s", res.Index, res.Result)
			return fmt.Errorf("failed to create account %d: %s", res.Index, res.Result)
		}
	}

	return nil
}

// CreateTransfers executes double-entry transfers.
func (c *LedgerClient) CreateTransfers(ctx context.Context, transfers []*domain.LedgerTransaction) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("ledger client not connected")
	}

	tbTransfers := make([]tbtypes.Transfer, len(transfers))

	for i, tx := range transfers {
		id, _ := tbtypes.HexStringToUint128(tx.ID.String())
		dr, _ := tbtypes.HexStringToUint128(tx.DebitAccountID.String())
		cr, _ := tbtypes.HexStringToUint128(tx.CreditAccountID.String())

		// Store reference ID (invoice/payment) in UserData128 for traceability
		refU128 := tbtypes.Uint128{}
		if tx.ReferenceID != (uuid.UUID{}) {
			ref, refErr := tbtypes.HexStringToUint128(tx.ReferenceID.String())
			if refErr == nil {
				refU128 = ref
			}
		}

		tbTransfers[i] = tbtypes.Transfer{
			ID:              id,
			DebitAccountID:  dr,
			CreditAccountID: cr,
			Amount:          tbtypes.ToUint128(tx.Amount),
			PendingID:       tbtypes.Uint128{}, // Not pending
			UserData128:     refU128,
			UserData64:      0,
			UserData32:      0,
			Timeout:         0,
			Ledger:          tx.LedgerID,
			Code:            tx.Code,
			Flags:           0,
			Timestamp:       0,
		}
	}

	results, err := c.client.CreateTransfers(tbTransfers)
	if err != nil {
		return fmt.Errorf("tb driver error: %w", err)
	}

	for _, res := range results {
		log.Printf("Error creating transfer index %d: %s", res.Index, res.Result)
		return fmt.Errorf("failed to create transfer: %s", res.Result)
	}

	return nil
}
// GetAccountTransfers fetches transfers for a given account.
func (c *LedgerClient) GetAccountTransfers(ctx context.Context, accountID uuid.UUID) ([]tbtypes.Transfer, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("ledger client not connected")
	}

	id, err := tbtypes.HexStringToUint128(accountID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid uuid %s: %w", accountID, err)
	}

	filter := tbtypes.AccountFilter{
		AccountID:    id,
		TimestampMin: 0, // From beginning
		TimestampMax: 0, // To end
		Limit:        50, // Default limit
	}

	transfers, err := c.client.GetAccountTransfers(filter)
	if err != nil {
		return nil, fmt.Errorf("tb driver error: %w", err)
	}

	return transfers, nil
}

package tigerbeetle

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// tbMaxPageLimit is the page size used when enumerating transfers. A single
// TigerBeetle reply is bounded by the message size (~8190 transfers); the
// server clamps larger limits, so enumeration never assumes a full page
// means "requested limit was honored" — it only stops on an empty page.
const tbMaxPageLimit = 8190

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

// Connected reports whether the client holds a live TigerBeetle connection.
func (c *LedgerClient) Connected() bool {
	return c != nil && c.client != nil
}

// uuidToUint128 maps a UUID onto TigerBeetle's little-endian Uint128 by
// treating the UUID's canonical big-endian bytes as one 128-bit integer.
// uint128ToUUID is its exact inverse, so the same ID round-trips between
// Postgres (uuid) and TigerBeetle (u128) losslessly.
//
// Note: tbtypes.HexStringToUint128(id.String()) does NOT work here —
// uuid.String() is 36 chars with dashes, which the hex parser rejects.
func uuidToUint128(id uuid.UUID) tbtypes.Uint128 {
	var b [16]byte
	for i := range b {
		b[i] = id[15-i]
	}
	return tbtypes.BytesToUint128(b)
}

// uint128ToUUID converts a TigerBeetle Uint128 back to the UUID whose
// big-endian bytes encode the same 128-bit integer. Inverse of uuidToUint128.
func uint128ToUUID(v tbtypes.Uint128) uuid.UUID {
	b := v.Bytes() // little-endian
	var id uuid.UUID
	for i := range id {
		id[i] = b[15-i]
	}
	return id
}

// uint128Low64 returns the low 64 bits of a TigerBeetle amount. Every amount
// this codebase writes fits in uint64 (see CreateTransfers); if the high bits
// are somehow set, saturate to MaxUint64 so the truncated value can never
// silently equal a legitimate amount.
func uint128Low64(v tbtypes.Uint128) uint64 {
	b := v.Bytes() // little-endian
	for _, hi := range b[8:] {
		if hi != 0 {
			return math.MaxUint64
		}
	}
	return binary.LittleEndian.Uint64(b[:8])
}

// TransferRecord is a TigerBeetle transfer flattened into the UUID/uint64
// vocabulary the rest of the codebase speaks.
type TransferRecord struct {
	ID              uuid.UUID
	DebitAccountID  uuid.UUID
	CreditAccountID uuid.UUID
	Amount          uint64
	Ledger          uint32
	Code            uint16
	ReferenceID     uuid.UUID // from UserData128; uuid.Nil when unset
	Timestamp       uint64    // TigerBeetle-assigned, unique per transfer
}

func transferToRecord(t tbtypes.Transfer) TransferRecord {
	return TransferRecord{
		ID:              uint128ToUUID(t.ID),
		DebitAccountID:  uint128ToUUID(t.DebitAccountID),
		CreditAccountID: uint128ToUUID(t.CreditAccountID),
		Amount:          uint128Low64(t.Amount),
		Ledger:          t.Ledger,
		Code:            t.Code,
		ReferenceID:     uint128ToUUID(t.UserData128),
		Timestamp:       t.Timestamp,
	}
}

// CreateAccounts creates new accounts in the ledger.
// Idempotent: If account exists, it returns an error result for that account, but we might treat it leniently.
func (c *LedgerClient) CreateAccounts(ctx context.Context, accounts []*domain.LedgerAccount) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("ledger client not connected")
	}

	tbAccounts := make([]tbtypes.Account, len(accounts))

	for i, acc := range accounts {
		tbAccounts[i] = tbtypes.Account{
			ID:            uuidToUint128(acc.ID),
			DebitsPosted:  tbtypes.ToUint128(acc.DebitsPosted),
			CreditsPosted: tbtypes.ToUint128(acc.CreditsPosted),
			UserData128:   tbtypes.ToUint128(acc.UserData128.Low), // P4: Simplification if older lib doesn't support High/Low in ToUint128?
			UserData64:    0,
			UserData32:    0,
			Reserved:      0,
			Ledger:        acc.LedgerID,
			Code:          uint16(acc.Code),
			Flags:         0, // No specific flags for now (e.g. History)
			Timestamp:     0, // TB sets this
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
		// Store reference ID (invoice/payment) in UserData128 for traceability
		refU128 := tbtypes.Uint128{}
		if tx.ReferenceID != (uuid.UUID{}) {
			refU128 = uuidToUint128(tx.ReferenceID)
		}

		tbTransfers[i] = tbtypes.Transfer{
			ID:              uuidToUint128(tx.ID),
			DebitAccountID:  uuidToUint128(tx.DebitAccountID),
			CreditAccountID: uuidToUint128(tx.CreditAccountID),
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
	return c.GetAccountTransfersPaged(ctx, accountID, 0, 50)
}

// GetAccountTransfersPaged fetches one chronological page of transfers for an
// account, starting at timestampMin (inclusive). Both debit and credit sides
// are included. The server may clamp limit to its batch maximum.
func (c *LedgerClient) GetAccountTransfersPaged(ctx context.Context, accountID uuid.UUID, timestampMin uint64, limit uint32) ([]tbtypes.Transfer, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("ledger client not connected")
	}

	filter := tbtypes.AccountFilter{
		AccountID:    uuidToUint128(accountID),
		TimestampMin: timestampMin,
		TimestampMax: 0, // To end
		Limit:        limit,
		Flags: tbtypes.AccountFilterFlags{
			Debits:  true,
			Credits: true,
			// Reversed=false: chronological order, required for paging by
			// advancing TimestampMin.
		}.ToUint32(),
	}

	transfers, err := c.client.GetAccountTransfers(filter)
	if err != nil {
		return nil, fmt.Errorf("tb driver error: %w", err)
	}

	return transfers, nil
}

// EnumerateAccountTransfers walks every transfer touching an account, in
// chronological order, by fetching pages and advancing TimestampMin past the
// last-seen transfer timestamp (TigerBeetle timestamps are unique per
// transfer). tigerbeetle-go v0.15.x has no QueryTransfers/QueryFilter API, so
// per-account AccountFilter paging is the only exhaustive enumeration path.
//
// maxTransfers bounds memory: if the account holds more than maxTransfers
// transfers, an error is returned rather than a truncated (and therefore
// misleading) result.
func (c *LedgerClient) EnumerateAccountTransfers(ctx context.Context, accountID uuid.UUID, maxTransfers int) ([]TransferRecord, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("ledger client not connected")
	}

	var records []TransferRecord
	var timestampMin uint64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		page, err := c.GetAccountTransfersPaged(ctx, accountID, timestampMin, tbMaxPageLimit)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return records, nil
		}
		if len(records)+len(page) > maxTransfers {
			return nil, fmt.Errorf("account %s has more than %d transfers; enumeration aborted", accountID, maxTransfers)
		}

		for _, t := range page {
			records = append(records, transferToRecord(t))
		}

		last := page[len(page)-1].Timestamp
		if last == math.MaxUint64 || last < timestampMin {
			// Cannot advance further (or server returned out-of-range data);
			// stop rather than loop forever.
			return records, nil
		}
		timestampMin = last + 1
	}
}

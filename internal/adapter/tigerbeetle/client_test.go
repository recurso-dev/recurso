package tigerbeetle

import (
	"math"
	"strings"
	"testing"

	"github.com/google/uuid"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func TestUUIDUint128RoundTrip(t *testing.T) {
	cases := []uuid.UUID{
		uuid.Nil,
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		uuid.MustParse("00ffee00-0000-4000-8000-0000000000ff"), // leading zero byte
		uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		uuid.New(),
		uuid.New(),
	}
	for _, id := range cases {
		got := uint128ToUUID(uuidToUint128(id))
		if got != id {
			t.Errorf("round trip changed %s into %s", id, got)
		}
	}
}

// The conversion must agree with the library's own big-endian hex parsing
// (which is what a dash-less uuid hex string encodes), so IDs written before
// and after this helper existed mean the same 128-bit value.
func TestUUIDToUint128MatchesHexParsing(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	hexStr := strings.ReplaceAll(id.String(), "-", "")
	want, err := tbtypes.HexStringToUint128(hexStr)
	if err != nil {
		t.Fatalf("HexStringToUint128(%q): %v", hexStr, err)
	}
	if got := uuidToUint128(id); got != want {
		t.Errorf("uuidToUint128(%s) = %v, want %v", id, got, want)
	}
}

func TestUint128Low64(t *testing.T) {
	if got := uint128Low64(tbtypes.ToUint128(0)); got != 0 {
		t.Errorf("uint128Low64(0) = %d, want 0", got)
	}
	if got := uint128Low64(tbtypes.ToUint128(123456789)); got != 123456789 {
		t.Errorf("uint128Low64(123456789) = %d, want 123456789", got)
	}
	if got := uint128Low64(tbtypes.ToUint128(math.MaxUint64)); got != math.MaxUint64 {
		t.Errorf("uint128Low64(MaxUint64) = %d, want MaxUint64", got)
	}

	// High bits set: must saturate, never silently truncate to a
	// plausible-looking amount.
	var high [16]byte
	high[8] = 1 // 2^64
	if got := uint128Low64(tbtypes.BytesToUint128(high)); got != math.MaxUint64 {
		t.Errorf("uint128Low64(2^64) = %d, want MaxUint64 saturation", got)
	}
}

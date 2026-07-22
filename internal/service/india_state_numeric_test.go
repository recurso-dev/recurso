package service

import "testing"

// normalizeINStateNumeric must yield the two-digit numeric GST state code the
// NIC e-invoice schema requires, from any of the forms PlaceOfSupply is stored
// in (abbreviation, full name, already-numeric).
func TestNormalizeINStateNumeric(t *testing.T) {
	cases := map[string]string{
		"TN":         "33", // abbreviation
		"tn":         "33", // case-insensitive
		"Tamil Nadu": "33", // full name
		"33":         "33", // already numeric
		"9":          "09", // single-digit padded (UP)
		"KA":         "29",
		"MH":         "27",
		"AP":         "37", // pinned to the post-reorganization code, not legacy 28
		"":           "",   // empty passes through
		"ZZ":         "ZZ", // unknown passes through (surfaces as an IRP error)
	}
	for in, want := range cases {
		if got := normalizeINStateNumeric(in); got != want {
			t.Errorf("normalizeINStateNumeric(%q) = %q, want %q", in, got, want)
		}
	}
}

// The reverse map must round-trip against the forward alpha map for every code
// (AP excepted, which is intentionally pinned).
func TestGSTStateCodeMapsRoundTrip(t *testing.T) {
	for num, alpha := range gstNumericStateToAlpha {
		if alpha == "AP" {
			continue // 28 and 37 both map to AP; reverse is pinned to 37
		}
		if back := gstAlphaToNumericState[alpha]; back != num {
			t.Errorf("reverse[%q] = %q, want %q", alpha, back, num)
		}
	}
	if gstAlphaToNumericState["AP"] != "37" {
		t.Errorf("AP must pin to 37, got %q", gstAlphaToNumericState["AP"])
	}
}

package mcp

import "testing"

func TestDefaultTiers_MoneyPathOffByDefault(t *testing.T) {
	allow := defaultTiers()
	if !Tier1.enabled(allow) {
		t.Error("Tier1 (reads) should be enabled by default")
	}
	if !Tier2.enabled(allow) {
		t.Error("Tier2 (idempotent writes) should be enabled by default")
	}
	if Tier3.enabled(allow) {
		t.Error("Tier3 (money-path/destructive) must be OFF by default — opt-in only")
	}
}

func TestTierAnnotations_ReflectRisk(t *testing.T) {
	if a := Tier1.annotations("x"); !a.ReadOnlyHint {
		t.Error("Tier1 must carry ReadOnlyHint")
	}
	a2 := Tier2.annotations("x")
	if !a2.IdempotentHint || a2.DestructiveHint == nil || *a2.DestructiveHint {
		t.Error("Tier2 must be idempotent and non-destructive")
	}
	a3 := Tier3.annotations("x")
	if a3.DestructiveHint == nil || !*a3.DestructiveHint {
		t.Error("Tier3 must carry a destructive hint")
	}
}

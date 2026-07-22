// Report-scope helpers for finance reports across legal entities (Multi-Entity
// Books). Kept separate from the component so the component file only exports a
// component (react-refresh).

// Scope sentinels — kept distinct from a legal-entity id.
export const SCOPE_ALL = "all";
export const SCOPE_CONSOLIDATED = "consolidated";

// scopeToParams turns the selected scope into the trial-balance query params:
// "all" → the per-entity breakdown (no param), "consolidated" → a by-code
// rollup, otherwise a single entity's ledger.
export function scopeToParams(scope) {
  if (scope === SCOPE_CONSOLIDATED) return { consolidated: true };
  if (scope && scope !== SCOPE_ALL) return { entity_id: scope };
  return {};
}

// scopeEntityId returns the selected entity id, or "" for the whole-tenant
// scopes ("all" / "consolidated") — used to scope the GL export.
export function scopeEntityId(scope) {
  return scope === SCOPE_ALL || scope === SCOPE_CONSOLIDATED ? "" : scope || "";
}

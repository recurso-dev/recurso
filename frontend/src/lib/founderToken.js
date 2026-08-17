// In-memory holder for the founder (platform) token — the cross-tenant operator
// credential that gates /platform/metrics.
//
// Like the tenant API key (see authToken.js), it is deliberately NOT persisted
// to localStorage: a token in storage is readable by any injected script (XSS)
// and survives across sessions. Holding it in module memory scopes the exposure
// to the tab's lifetime — it is gone on refresh (the founder re-pastes it) and
// cannot be lifted out of storage after the fact.
let founderToken = ''

// Purge any token a prior build may have persisted. One-time cleanup on load.
try {
  window.localStorage.removeItem('recurso_founder_token')
} catch {
  // localStorage unavailable — nothing to clean up.
}

export const getFounderToken = () => founderToken
export const setFounderToken = (t) => {
  founderToken = t || ''
}
export const clearFounderToken = () => {
  founderToken = ''
}

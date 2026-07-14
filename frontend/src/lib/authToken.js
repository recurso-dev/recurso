// In-memory holder for the legacy tenant API key (the paste-a-key auth mode).
//
// This is deliberately NOT persisted to localStorage. An auth token in
// localStorage is readable by any injected script (XSS) and survives across
// sessions, so a single XSS payload can exfiltrate it and impersonate the
// tenant indefinitely. Holding it in module memory scopes the exposure to the
// current tab's lifetime: it is gone on refresh (the user re-pastes the key)
// and cannot be lifted out of storage after the fact. The primary auth path is
// the httpOnly session cookie, which JavaScript cannot read at all.
let apiKey = ''

// Purge any key persisted by an older build so the vulnerable artifact does not
// linger in storage. One-time cleanup on module load.
try {
  window.localStorage.removeItem('recurso_api_key')
} catch {
  // localStorage unavailable (SSR / privacy mode) — nothing to clean up.
}

export const getApiKey = () => apiKey
export const setApiKey = (key) => {
  apiKey = key || ''
}
export const clearApiKey = () => {
  apiKey = ''
}

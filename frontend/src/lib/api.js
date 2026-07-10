import axios from 'axios';

// Single source of truth for API locations. In dev both resolve to relative
// paths served by the Vite proxy; in prod set VITE_API_BASE_URL (e.g.
// "https://api.recurso.dev/v1").
export const API_BASE = import.meta.env.VITE_API_BASE_URL || '/v1';
// Server root for non-/v1 routes (/auth, /portal, /checkout).
export const API_ROOT = API_BASE.replace(/\/v1\/?$/, '');

// Send the httpOnly session cookie on every request (same-origin behind the
// nginx proxy). Applies to the `api` instance and direct axios calls (/auth).
axios.defaults.withCredentials = true;

const api = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('recurso_api_key');
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`; // Backend expects "Bearer <api_key>" or just "<api_key>"? Middleware check needed.
      // Checking middleware: "strings.TrimPrefix(authHeader, "Bearer ")" is standard, let's assume standard.
      // Wait, let's check middleware/auth.go to be sure.
    }
    return config;
  },
  (error) => Promise.reject(error)
);

export const endpoints = {
  // --- Auth (session, cookie-based) ---
  authRegister: (data) => axios.post(`${API_ROOT}/auth/register`, data),
  authLogin: (email, password) =>
    axios.post(`${API_ROOT}/auth/login`, { email, password }),
  // Second step of two-step login: exchange the short-lived mfa_token + a TOTP
  // (or backup) code for a session cookie.
  loginMfa: (mfa_token, code) =>
    axios.post(`${API_ROOT}/auth/login/mfa`, { mfa_token, code }),
  authLogout: () => axios.post(`${API_ROOT}/auth/logout`),
  authMe: () => axios.get(`${API_ROOT}/auth/me`),
  // Password reset (public, cookie-less).
  forgotPassword: (email) =>
    axios.post(`${API_ROOT}/auth/forgot-password`, { email }),
  resetPassword: (token, password) =>
    axios.post(`${API_ROOT}/auth/reset-password`, { token, password }),
  // --- MFA management (authed, session-scoped) ---
  mfaSetup: () => api.post('/auth/mfa/setup'),
  mfaVerify: (code) => api.post('/auth/mfa/verify', { code }),
  mfaDisable: (code) => api.post('/auth/mfa/disable', { code }),
  // --- Active sessions (authed) ---
  getSessions: () => api.get('/auth/sessions'),
  revokeSession: (id) => api.delete(`/auth/sessions/${id}`),
  revokeOtherSessions: () => api.delete('/auth/sessions'),
  // --- OAuth social login (public) ---
  // Which providers are configured on this server. Buttons link (full-page
  // redirect, not axios) to `${API_ROOT}/auth/oauth/{name}/start`.
  getOAuthProviders: () => axios.get(`${API_ROOT}/auth/oauth/providers`),
  // --- SAML SSO connection (authed, owner/admin) ---
  getSSOConnection: () => api.get('/sso/connection'),
  updateSSOConnection: (data) => api.put('/sso/connection', data),
  deleteSSOConnection: () => api.delete('/sso/connection'),
  // --- Team members (tenant-scoped) ---
  getUsers: () => api.get('/users'),
  createUser: (data) => api.post('/users', data),
  updateUserRole: (id, role) => api.patch(`/users/${id}`, { role }),
  deleteUser: (id) => api.delete(`/users/${id}`),
  getPlans: (params) => api.get('/plans', { params }),
  getAccount: () => api.get('/account'),
  updateAccount: (data) => api.put('/account', data),
  getCustomers: (params) => api.get('/customers', { params }),
  getSubscriptions: (params) => api.get('/subscriptions', { params }),
  getInvoices: (params) => api.get('/invoices', { params }),
  // Tenant-scoped (session or API key); fetched as a blob so the auth header
  // is sent — a plain <a href> would only work for cookie sessions.
  getInvoicePdf: (id) => api.get(`/invoices/${id}/pdf`, { responseType: 'blob' }),
  getMRR: () => api.get('/analytics/mrr'),
  getUsageStats: () => api.get('/analytics/usage'),
  getLedgerEntries: (params) => api.get('/ledger/entries', { params }),
  getLedgerAccounts: () => api.get('/ledger/accounts'),
  // On-demand ledger reconciliation (computed per request, never persisted).
  runReconciliation: () => api.get('/finance/reconciliation'),
  
  // Developer
  getAPIKeys: () => api.get('/developer/keys'),
  createKey: (data) => api.post('/developer/keys', data),
  register: (data) => axios.post('/auth/register', data),
  
  createCustomer: (data) => api.post('/customers', data),
  createPlan: (data) => api.post('/plans', data),
  getPlanEntitlements: (id) => api.get(`/plans/${id}/entitlements`),
  // PUT semantics: the body is the plan's full desired entitlement set;
  // entries absent from the array are removed server-side.
  setPlanEntitlements: (id, entitlements) => api.put(`/plans/${id}/entitlements`, entitlements),
  createSubscription: (data) => api.post('/subscriptions', data),
  updateSubscription: (id, data) => api.put(`/subscriptions/${id}`, data),
  previewPlanChange: (id, planId) =>
    api.get(`/subscriptions/${id}/preview-change`, { params: { plan_id: planId } }),
  getSubscriptionAddons: (id) => api.get(`/subscriptions/${id}/addons`),
  addSubscriptionAddon: (id, data) => api.post(`/subscriptions/${id}/addons`, data),
  removeSubscriptionAddon: (id, addonId) =>
    api.delete(`/subscriptions/${id}/addons/${addonId}`),
  cancelSubscription: (id) => api.post(`/subscriptions/${id}/cancel`),
  pauseSubscription: (id) => api.post(`/subscriptions/${id}/pause`),
  resumeSubscription: (id) => api.post(`/subscriptions/${id}/resume`),

  // Credit Notes
  getCreditNotes: (params) => api.get('/credit-notes', { params }),
  createCreditNote: (data) => api.post('/credit-notes', data),

  // Coupons
  getCoupons: () => api.get('/coupons'),
  createCoupon: (data) => api.post('/coupons', data),

  // Webhooks & Events (P24)
  getWebhooks: () => api.get('/webhooks'),
  createWebhook: (data) => api.post('/webhooks', data),
  deleteWebhook: (id) => api.delete(`/webhooks/${id}`),
  getEvents: (params) => api.get('/events', { params }),
  getEventTypes: () => api.get('/events/types'),
  // Per-endpoint delivery rows for a single event (derived status, attempts, retry).
  getEventDeliveries: (eventId) => api.get(`/events/${eventId}/deliveries`),
  // Recent deliveries for one webhook endpoint (supports limit/offset/status).
  getWebhookDeliveries: (id, params) => api.get(`/webhooks/${id}/deliveries`, { params }),
  // Queue a re-delivery of an event to its subscribed endpoints; returns 202.
  redeliverEvent: (eventId) => api.post(`/events/${eventId}/redeliver`),

  // Quotes (P27)
  getQuotes: (params) => api.get('/quotes', { params }),
  getQuote: (id) => api.get(`/quotes/${id}`),
  createQuote: (data) => api.post('/quotes', data),
  updateQuote: (id, data) => api.put(`/quotes/${id}`, data),
  deleteQuote: (id) => api.delete(`/quotes/${id}`),
  sendQuote: (id) => api.post(`/quotes/${id}/send`),
  acceptQuote: (id) => api.post(`/quotes/${id}/accept`),
  declineQuote: (id) => api.post(`/quotes/${id}/decline`),
  convertQuoteToInvoice: (id) => api.post(`/quotes/${id}/convert`),

  // Gifts (P25)
  getGifts: () => api.get('/gifts'),
  purchaseGift: (data) => api.post('/gifts/purchase', data),
  redeemGift: (data) => api.post('/gifts/redeem', data),

  // Referrals (P25)
  getReferrals: () => api.get('/referrals'),
  createReferral: (data) => api.post('/referrals', data),
  generateReferralCode: (data) => api.post('/referrals/generate-code', data),

  // Checkout (public, uses base URL without /v1)
  getCheckoutInvoice: (id) => axios.get(`${API_ROOT}/checkout/${id}`),
  initiateCheckoutPayment: (id) => axios.post(`${API_ROOT}/checkout/${id}/pay`),

  // Smart Dunning Analytics
  getDunningOverview: () => api.get('/analytics/dunning/overview'),
  getDunningWeights: () => api.get('/analytics/dunning/weights'),
  getDunningHistory: (params) => api.get('/analytics/dunning/history', { params }),
  getDunningRecovered: () => api.get('/analytics/dunning/recovered'),

  // E-Invoice (P25)
  getEInvoiceStatus: (invoiceId) => api.get(`/invoices/${invoiceId}/einvoice`),
  retryEInvoice: (invoiceId) => api.post(`/invoices/${invoiceId}/einvoice/retry`),
  cancelEInvoice: (invoiceId, data) => api.post(`/invoices/${invoiceId}/einvoice/cancel`, data),
  getIRPConfig: () => api.get('/settings/irp'),
  updateIRPConfig: (data) => api.put('/settings/irp', data),
  testIRPConfig: () => api.post('/settings/irp/test'),
};

export default api;

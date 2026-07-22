import axios from 'axios';
import { getApiKey } from './authToken';

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
    // Legacy API-key mode: the key lives in memory only (see lib/authToken.js),
    // never in localStorage. The backend accepts "Bearer <api_key>".
    const token = getApiKey();
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`;
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
  // Public-sandbox entry: only exists when the server runs DEMO_MODE.
  authDemo: () => axios.post(`${API_ROOT}/auth/demo`),
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
  // Invite a teammate: no password — they get an email to set their own.
  inviteUser: (data) => api.post('/users/invite', data),
  updateUserRole: (id, role) => api.patch(`/users/${id}`, { role }),
  deleteUser: (id) => api.delete(`/users/${id}`),
  getPlans: (params) => api.get('/plans', { params }),
  getAccount: () => api.get('/account'),
  updateAccount: (data) => api.put('/account', data),
  getCustomers: (params) => api.get('/customers', { params }),
  getCustomer: (id) => api.get(`/customers/${id}`),
  // Partial update; { active: false } archives (blocked while subscriptions
  // are active), { active: true } restores.
  updateCustomer: (id, data) => api.put(`/customers/${id}`, data),
  // Consent audit trail (GDPR): list a customer's recorded consents and revoke one.
  getCustomerConsents: (id) => api.get(`/customers/${id}/consents`),
  revokeConsent: (consentId) => api.post('/consents/revoke', { consent_id: consentId }),
  getSubscriptions: (params) => api.get('/subscriptions', { params }),
  getInvoices: (params) => api.get('/invoices', { params }),
  // Tenant-scoped (session or API key); fetched as a blob so the auth header
  // is sent — a plain <a href> would only work for cookie sessions.
  getInvoicePdf: (id) => api.get(`/invoices/${id}/pdf`, { responseType: 'blob' }),
  getMRR: () => api.get('/analytics/mrr'),
  // MRR movement between two dates (new/expansion/contraction/churned/reactivation).
  getMRRWaterfall: (start, end) =>
    api.get('/analytics/mrr/waterfall', { params: { start, end } }),
  // Outstanding receivables bucketed by days past due.
  getInvoiceAging: () => api.get('/analytics/invoice-aging'),
  // ARPA / ARPU / LTV.
  getUnitEconomics: () => api.get('/analytics/unit-economics'),
  // MRR split across plans.
  getRevenueByPlan: () => api.get('/analytics/revenue-by-plan'),
  // MRR split across customer countries.
  getRevenueByGeography: () => api.get('/analytics/revenue-by-geography'),
  getUsageStats: () => api.get('/analytics/usage'),
  getLedgerEntries: (params) => api.get('/ledger/entries', { params }),
  getLedgerAccounts: () => api.get('/ledger/accounts'),
  // On-demand ledger reconciliation (computed per request, never persisted).
  runReconciliation: () => api.get('/finance/reconciliation'),
  // Deferred-revenue rollforward: recognized in the period, deferred balance,
  // the month-by-month release schedule, and the per-currency split.
  getRevenueRecognition: (month, year) =>
    api.get('/finance/revrec/report', { params: { month, year } }),
  // Provable-ledger auditor reports (ENG-192): trial balance, GL CSV export,
  // the recognition waterfall, and the deferred-revenue rollforward.
  getTrialBalance: () => api.get('/ledger/trial-balance'),
  exportGeneralLedger: () => api.get('/ledger/export', { responseType: 'blob' }),
  getRevenueWaterfall: () => api.get('/finance/revrec/waterfall'),
  getDeferredRollforward: (month, year) =>
    api.get('/ledger/deferred-rollforward', { params: { month, year } }),
  // Month-end close pack (B2): trial balance + reconciliation + deferred
  // rollforward + GL export pointer + a ready-to-close verdict, in one call.
  getClosePack: (month, year) =>
    api.get('/finance/close-pack', { params: { month, year } }),

  // Developer
  getAPIKeys: () => api.get('/developer/keys'),
  createKey: (data) => api.post('/developer/keys', data),
  // Soft-deactivates the key; it stops authenticating immediately.
  revokeKey: (id) => api.delete(`/developer/keys/${id}`),
  register: (data) => axios.post('/auth/register', data),
  
  createCustomer: (data) => api.post('/customers', data),
  createPlan: (data) => api.post('/plans', data),
  getPlan: (id) => api.get(`/plans/${id}`),
  // Partial update; set { active: false } to archive, { active: true } to restore.
  updatePlan: (id, data) => api.put(`/plans/${id}`, data),
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
  reactivateSubscription: (id) => api.post(`/subscriptions/${id}/reactivate`),
  // Generate an advance invoice covering the next N periods (1-60).
  advanceSubscription: (id, periods) => api.post(`/subscriptions/${id}/advance`, { periods }),
  // Minimum commitment per period, minor units; 0 clears it.
  setSubscriptionCommitment: (id, amount) => api.put(`/subscriptions/${id}/commitment`, { amount }),
  // Ad-hoc one-off charge folded onto the subscription's next invoice.
  addSubscriptionCharge: (id, body) => api.post(`/subscriptions/${id}/charges`, body),
  getSubscriptionCharges: (id) => api.get(`/subscriptions/${id}/charges`),

  // Credit Notes
  getCreditNotes: (params) => api.get('/credit-notes', { params }),
  createCreditNote: (data) => api.post('/credit-notes', data),
  approveCreditNote: (id) => api.post(`/credit-notes/${id}/approve`),
  rejectCreditNote: (id) => api.post(`/credit-notes/${id}/reject`),

  // Coupons
  getCoupons: () => api.get('/coupons'),
  createCoupon: (data) => api.post('/coupons', data),
  // active:false blocks new redemptions; existing subscriptions keep the discount.
  setCouponActive: (id, active) => api.put(`/coupons/${id}`, { active }),

  // Webhooks & Events (P24)
  getWebhooks: () => api.get('/webhooks'),
  createWebhook: (data) => api.post('/webhooks', data),
  deleteWebhook: (id) => api.delete(`/webhooks/${id}`),
  // Pause ("inactive") / resume ("active") deliveries for one endpoint.
  setWebhookStatus: (id, status) => api.put(`/webhooks/${id}/status`, { status }),
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
  // Marks the referral as qualified (reward becomes claimable).
  qualifyReferral: (id) => api.post(`/referrals/${id}/qualify`),

  // Checkout (public, uses base URL without /v1)
  getCheckoutInvoice: (id) => axios.get(`${API_ROOT}/checkout/${id}`),
  initiateCheckoutPayment: (id) => axios.post(`${API_ROOT}/checkout/${id}/pay`),

  // Smart Dunning Analytics
  getDunningOverview: () => api.get('/analytics/dunning/overview'),
  getDunningWeights: () => api.get('/analytics/dunning/weights'),
  getDunningHistory: (params) => api.get('/analytics/dunning/history', { params }),
  getDunningRecovered: () => api.get('/analytics/dunning/recovered'),

  // Payment mandates (UPI Autopay)
  getMandates: () => api.get('/mandates'),
  createMandate: (data) => api.post('/mandates', data),
  revokeMandate: (id) => api.post(`/mandates/${id}/revoke`),

  // Invoice disputes (admin)
  getDisputes: (status) => api.get('/disputes', { params: status ? { status } : {} }),
  resolveDispute: (id, note) => api.post(`/disputes/${id}/resolve`, { note }),

  // Offline payments + virtual accounts
  getOfflinePayments: () => api.get('/payments/offline'),
  recordOfflinePayment: (data) => api.post('/payments/offline', data),
  getVirtualAccounts: () => api.get('/virtual-accounts'),
  createVirtualAccount: (data) => api.post('/virtual-accounts', data),

  // Churn risk
  getChurnAlerts: () => api.get('/churn/alerts'),
  acknowledgeChurnAlert: (id) => api.post(`/churn/alerts/${id}/ack`),
  getHighRiskCustomers: (threshold) =>
    api.get('/churn/high-risk', { params: threshold ? { threshold } : {} }),

  // Cancellation / retention flows (list/get/stats return the payload directly)
  getCancelFlows: () => api.get('/cancel-flows'),
  getCancelFlow: (id) => api.get(`/cancel-flows/${id}`),
  createCancelFlow: (data) => api.post('/cancel-flows', data),
  updateCancelFlow: (id, data) => api.put(`/cancel-flows/${id}`, data),
  createCancelFlowStep: (flowId, data) => api.post(`/cancel-flows/${flowId}/steps`, data),
  updateCancelFlowStep: (stepId, data) => api.put(`/cancel-flows/steps/${stepId}`, data),
  deleteCancelFlowStep: (stepId) => api.delete(`/cancel-flows/steps/${stepId}`),
  getCancelFlowStats: (flowId) => api.get('/cancel-flows/stats', { params: { flow_id: flowId } }),

  // Dunning campaign config (list/get return the payload directly, not { data })
  getDunningCampaigns: () => api.get('/dunning-campaigns'),
  getDunningCampaign: (id) => api.get(`/dunning-campaigns/${id}`),
  createDunningCampaign: (data) => api.post('/dunning-campaigns', data),
  updateDunningCampaign: (id, data) => api.put(`/dunning-campaigns/${id}`, data),
  createDunningStep: (campaignId, data) => api.post(`/dunning-campaigns/${campaignId}/steps`, data),
  updateDunningStep: (stepId, data) => api.put(`/dunning-campaigns/steps/${stepId}`, data),
  deleteDunningStep: (stepId) => api.delete(`/dunning-campaigns/steps/${stepId}`),

  // E-Invoice (P25)
  getEInvoiceStatus: (invoiceId) => api.get(`/invoices/${invoiceId}/einvoice`),
  retryEInvoice: (invoiceId) => api.post(`/invoices/${invoiceId}/einvoice/retry`),
  cancelEInvoice: (invoiceId, data) => api.post(`/invoices/${invoiceId}/einvoice/cancel`, data),
  getIRPConfig: () => api.get('/settings/irp'),
  updateIRPConfig: (data) => api.put('/settings/irp', data),
  testIRPConfig: () => api.post('/settings/irp/test'),
  // EU e-invoicing config (Track C): opt-in + EN 16931 seller identity.
  getEUEInvoiceConfig: () => api.get('/settings/eu-einvoice'),
  updateEUEInvoiceConfig: (data) => api.put('/settings/eu-einvoice', data),

  getMCPSettings: () => api.get('/settings/mcp'),
  updateMCPSettings: (data) => api.put('/settings/mcp', data),

  // Usage-based billing (metering)
  getBillableMetrics: () => api.get('/billable-metrics'),
  createBillableMetric: (data) => api.post('/billable-metrics', data),
  // Same input shape as create.
  updateBillableMetric: (id, data) => api.put(`/billable-metrics/${id}`, data),
  deleteBillableMetric: (id) => api.delete(`/billable-metrics/${id}`),
  getPlanCharges: (planId) => api.get(`/plans/${planId}/charges`),
  setPlanCharges: (planId, charges) => api.put(`/plans/${planId}/charges`, charges),
  // Read-only pricing simulator: rate a proposed charge set against sample
  // usage, no persistence. Body: { currency?, subscription_id?, charges[], usage[] }.
  simulateCharges: (planId, body) => api.post(`/plans/${planId}/simulate-charges`, body),
  getUsageAmount: (subId) => api.get(`/subscriptions/${subId}/usage-amount`),
  // Progressive billing: generate an interim invoice now for accrued usage past
  // the threshold (A5). No-op (204) when nothing is due.
  billUsageNow: (subId) => api.post(`/subscriptions/${subId}/bill-usage`),

  // Prepaid wallets
  getWallets: (params) => api.get('/wallets', { params }),
  createWallet: (data) => api.post('/wallets', data),
  getWallet: (id) => api.get(`/wallets/${id}`),
  topUpWallet: (id, data) => api.post(`/wallets/${id}/top-up`, data),
  getWalletTransactions: (id, params) => api.get(`/wallets/${id}/transactions`, { params }),
  setWalletAutoRecharge: (id, data) => api.put(`/wallets/${id}/auto-recharge`, data),

  // Usage alerts
  // Raw ingestion stream, newest first (debugging: did my events land?).
  getUsageEvents: (params) => api.get('/usage/events', { params }),
  getUsageAlerts: (params) => api.get('/usage-alerts', { params }),
  createUsageAlert: (data) => api.post('/usage-alerts', data),
  deleteUsageAlert: (id) => api.delete(`/usage-alerts/${id}`),

  // Audit trail
  getAuditLogs: (params) => api.get('/audit-logs', { params }),

  // Organizations (multi-tenant admin)
  getOrganizations: () => api.get('/organizations'),
  getOrganization: (id) => api.get(`/organizations/${id}`),
  createOrganization: (data) => api.post('/organizations', data),
  updateOrganization: (id, data) => api.put(`/organizations/${id}`, data),
  deleteOrganization: (id) => api.delete(`/organizations/${id}`),
  getOrgTenants: (id) => api.get(`/organizations/${id}/tenants`),
  addOrgTenant: (id, tenantId) => api.post(`/organizations/${id}/tenants`, { tenant_id: tenantId }),
  removeOrgTenant: (id, tenantId) => api.delete(`/organizations/${id}/tenants/${tenantId}`),
  getOrgMRR: (id) => api.get(`/organizations/${id}/analytics/mrr`),

  // US sales-tax nexus
  getTaxNexus: () => api.get('/settings/tax/nexus'),
  // Full replacement of the declared-state set.
  setTaxNexus: (states) => api.put('/settings/tax/nexus', { states }),
  getTaxNexusStatus: (year) =>
    api.get('/settings/tax/nexus/status', { params: year ? { year } : {} }),
  // Per-state US sales-tax liability for a filing period (D3).
  getTaxLiability: (params) =>
    api.get('/settings/tax/liability', { params: params || {} }),
  // US sales-tax registrations (D4).
  getTaxRegistrations: () => api.get('/settings/tax/registrations'),
  setTaxRegistrations: (registrations) =>
    api.put('/settings/tax/registrations', { registrations }),

  // India GST config
  getGSTConfig: () => api.get('/settings/gst'),
  validateGSTIN: (gstin) => api.post('/settings/gst/validate', { gstin }),
  updateGSTConfig: (config) => api.put('/settings/gst', config),
  // India GST returns (readable sections + GSTN upload JSON)
  getGSTR1: (month, year) => api.get('/india/gstr1', { params: { month, year } }),
  getGSTR3B: (month, year) => api.get('/india/gstr3b', { params: { month, year } }),

  // GenAI analytics: natural-language question -> { data, query }
  askAnalytics: (question) => api.post('/analytics/ask', { question }),

  // Accounting integrations (QuickBooks / Xero)
  getAccountingConnections: () => api.get('/accounting/connections'),
  // Returns { auth_url } — redirect the browser there to start OAuth.
  connectAccounting: (provider) => api.post(`/accounting/connect/${provider}`),
  // NetSuite (account_id + access_token) and Tally (empty body) — no OAuth redirect.
  connectAccountingToken: (provider, body) => api.post(`/accounting/connect-token/${provider}`, body),
  disconnectAccounting: (id) => api.delete(`/accounting/connections/${id}`),
  triggerAccountingSync: () => api.post('/accounting/sync'),
  getAccountingSyncStatus: () => api.get('/accounting/sync/status'),

  // BYO payment gateways (Stripe / Razorpay) — per-tenant credentials.
  getGatewayConnections: () => api.get('/gateway-connections'),
  connectGateway: (body) => api.post('/gateway-connections', body),
  setGatewayWebhookSecret: (provider, webhook_secret) =>
    api.put(`/gateway-connections/${provider}/webhook-secret`, { webhook_secret }),
  disconnectGateway: (provider) => api.delete(`/gateway-connections/${provider}`),

  // BYO tax/CRM/storage integrations — per-tenant credentials.
  getIntegrationConnections: () => api.get('/integration-connections'),
  connectIntegration: (body) => api.post('/integration-connections', body),
  disconnectIntegration: (category, provider) =>
    api.delete(`/integration-connections/${category}/${provider}`),
};


export default api;

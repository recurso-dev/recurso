import axios from 'axios';

// Single source of truth for API locations. In dev both resolve to relative
// paths served by the Vite proxy; in prod set VITE_API_BASE_URL (e.g.
// "https://api.recurso.dev/v1").
export const API_BASE = import.meta.env.VITE_API_BASE_URL || '/v1';
// Server root for non-/v1 routes (/auth, /portal, /checkout).
export const API_ROOT = API_BASE.replace(/\/v1\/?$/, '');

const api = axios.create({
  baseURL: API_BASE,
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
  getPlans: (params) => api.get('/plans', { params }),
  getAccount: () => api.get('/account'),
  updateAccount: (data) => api.put('/account', data),
  getCustomers: (params) => api.get('/customers', { params }),
  getSubscriptions: (params) => api.get('/subscriptions', { params }),
  getInvoices: (params) => api.get('/invoices', { params }),
  getMRR: () => api.get('/analytics/mrr'),
  getUsageStats: () => api.get('/analytics/usage'),
  getLedgerEntries: (params) => api.get('/ledger/entries', { params }),
  getLedgerAccounts: () => api.get('/ledger/accounts'),
  
  // Developer
  getAPIKeys: () => api.get('/developer/keys'),
  createKey: (data) => api.post('/developer/keys', data),
  register: (data) => axios.post('/auth/register', data),
  
  createCustomer: (data) => api.post('/customers', data),
  createPlan: (data) => api.post('/plans', data),
  createSubscription: (data) => api.post('/subscriptions', data),
  updateSubscription: (id, data) => api.put(`/subscriptions/${id}`, data),
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

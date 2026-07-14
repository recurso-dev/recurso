"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Recurso = exports.DEFAULT_BASE_URL = void 0;
const axios_1 = __importDefault(require("axios"));
/** Default API base URL, used when none is supplied at construction. */
exports.DEFAULT_BASE_URL = 'http://localhost:8080';
/**
 * Official Node.js SDK for the Recurso billing API.
 *
 * Method coverage mirrors the REST surface: list endpoints accept filter
 * params, mutations are grouped per resource, and lifecycle actions
 * (cancel, pause, resume, ...) live on their resource.
 */
class Recurso {
    /**
     * @param apiKey  API key used for Bearer authentication.
     * @param options Client options. Pass `{ baseUrl }` to target a specific
     *                environment; a bare string is also accepted for backward
     *                compatibility. Defaults to {@link DEFAULT_BASE_URL}.
     */
    constructor(apiKey, options = {}) {
        this.get = async (path, params) => (await this.client.get(path, { params })).data;
        this.post = async (path, data) => (await this.client.post(path, data)).data;
        this.put = async (path, data) => (await this.client.put(path, data)).data;
        this.del = async (path) => (await this.client.delete(path)).data;
        this.account = {
            get: () => this.get('/v1/account'),
            update: (data) => this.put('/v1/account', data),
        };
        this.customers = {
            create: (data) => this.post('/v1/customers', { country: 'US', ...data }),
            list: (params) => this.get('/v1/customers', params),
            updatePaymentMethod: (id, data) => this.put(`/v1/customers/${id}/payment-method`, data),
            churn: (id) => this.get(`/v1/customers/${id}/churn`),
            consents: (id) => this.get(`/v1/customers/${id}/consents`),
        };
        this.plans = {
            create: (data) => this.post('/v1/plans', { interval_count: 1, ...data }),
            list: (params) => this.get('/v1/plans', params),
        };
        this.subscriptions = {
            create: (data) => this.post('/v1/subscriptions', data),
            list: (params) => this.get('/v1/subscriptions', params),
            update: (id, data) => this.put(`/v1/subscriptions/${id}`, data),
            cancel: (id, data) => this.post(`/v1/subscriptions/${id}/cancel`, data),
            pause: (id, data) => this.post(`/v1/subscriptions/${id}/pause`, data),
            resume: (id) => this.post(`/v1/subscriptions/${id}/resume`),
            reactivate: (id) => this.post(`/v1/subscriptions/${id}/reactivate`),
            /** Bill N future periods immediately (advance invoicing). */
            advance: (id, data) => this.post(`/v1/subscriptions/${id}/advance`, data),
            charges: (id) => this.get(`/v1/subscriptions/${id}/charges`),
            addCharge: (id, data) => this.post(`/v1/subscriptions/${id}/charges`, data),
            /**
             * Current billing period's usage per dimension plus lifetime
             * totals, with the customer's entitlement limit/remaining joined
             * in where a feature_key matches the dimension name.
             */
            usage: (id) => this.get(`/v1/subscriptions/${id}/usage`),
        };
        this.invoices = {
            list: (params) => this.get('/v1/invoices', params),
            /** Public PDF download URL for an invoice. */
            pdfUrl: (id) => `${this.client.defaults.baseURL}/v1/invoices/${id}/pdf`,
            eInvoiceStatus: (id) => this.get(`/v1/invoices/${id}/einvoice`),
            retryEInvoice: (id) => this.post(`/v1/invoices/${id}/einvoice/retry`),
            cancelEInvoice: (id, data) => this.post(`/v1/invoices/${id}/einvoice/cancel`, data),
        };
        this.coupons = {
            create: (data) => this.post('/v1/coupons', data),
            list: (params) => this.get('/v1/coupons', params),
        };
        this.usage = {
            /** Record a metered usage event against a subscription. */
            record: (data) => this.post('/v1/usage/events', data),
            /**
             * Time-windowed usage buckets: {data: [{period, dimension,
             * quantity}], from, to, granularity}. At least one of
             * subscription_id or customer_id is required; the window defaults
             * to the last 30 days at day granularity.
             */
            query: (params) => this.get('/v1/usage', params),
            /** The tenant's dimension catalog with first/last seen and event counts. */
            dimensions: () => this.get('/v1/usage/dimensions'),
        };
        this.creditNotes = {
            create: (data) => this.post('/v1/credit-notes', data),
            list: (params) => this.get('/v1/credit-notes', params),
        };
        this.quotes = {
            create: (data) => this.post('/v1/quotes', data),
            list: (params) => this.get('/v1/quotes', params),
            get: (id) => this.get(`/v1/quotes/${id}`),
            update: (id, data) => this.put(`/v1/quotes/${id}`, data),
            send: (id) => this.post(`/v1/quotes/${id}/send`),
            accept: (id) => this.post(`/v1/quotes/${id}/accept`),
            decline: (id) => this.post(`/v1/quotes/${id}/decline`),
            /** Convert an accepted quote into a subscription. */
            convert: (id) => this.post(`/v1/quotes/${id}/convert`),
            delete: (id) => this.del(`/v1/quotes/${id}`),
        };
        this.webhooks = {
            /** Register an endpoint to receive event deliveries. */
            create: (data) => this.post('/v1/webhooks', data),
            list: () => this.get('/v1/webhooks'),
            delete: (id) => this.del(`/v1/webhooks/${id}`),
            /**
             * Recent delivery attempts to an endpoint, newest first. Filter by
             * derived status (pending | succeeded | failed) and paginate with
             * limit/offset.
             */
            deliveries: (id, params) => this.get(`/v1/webhooks/${id}/deliveries`, params),
        };
        this.events = {
            list: (params) => this.get('/v1/events', params),
            types: () => this.get('/v1/events/types'),
            /** Delivery attempts of an event across all webhook endpoints. */
            deliveries: (id) => this.get(`/v1/events/${id}/deliveries`),
            /**
             * Re-enqueue delivery of an event to every active subscribed
             * endpoint (202: {event_id, deliveries_queued}). Idempotent.
             */
            redeliver: (id) => this.post(`/v1/events/${id}/redeliver`),
        };
        this.mandates = {
            create: (data) => this.post('/v1/mandates', data),
            list: (params) => this.get('/v1/mandates', params),
            get: (id) => this.get(`/v1/mandates/${id}`),
            revoke: (id) => this.post(`/v1/mandates/${id}/revoke`),
        };
        this.gifts = {
            purchase: (data) => this.post('/v1/gifts/purchase', data),
            redeem: (data) => this.post('/v1/gifts/redeem', data),
            list: (params) => this.get('/v1/gifts', params),
        };
        this.referrals = {
            create: (data) => this.post('/v1/referrals', data),
            list: (params) => this.get('/v1/referrals', params),
            generateCode: (data) => this.post('/v1/referrals/generate-code', data),
            qualify: (id) => this.post(`/v1/referrals/${id}/qualify`),
        };
        this.entitlements = {
            /**
             * Replace a plan's full entitlement set (PUT semantics: feature
             * keys absent from the list are removed).
             */
            setForPlan: (planId, list) => this.put(`/v1/plans/${planId}/entitlements`, list),
            getForPlan: (planId) => this.get(`/v1/plans/${planId}/entitlements`),
            /**
             * Effective entitlements for a customer: the union over the plans
             * of their active/trialing subscriptions (boolean: any-true wins;
             * limit: max across plans).
             */
            forCustomer: (customerId) => this.get(`/v1/customers/${customerId}/entitlements`),
            /** Fast single-feature check: {feature_key, granted, limit_value}. */
            check: (customerId, feature) => this.get('/v1/entitlements/check', { customer_id: customerId, feature }),
        };
        this.analytics = {
            /**
             * Monthly recurring revenue, FX-normalized to the tenant's reporting
             * currency: {mrr, normalized_mrr, reporting_currency, breakdown[],
             * fx: {rates, source, as_of}}.
             */
            mrr: () => this.get('/v1/analytics/mrr'),
        };
        this.ledger = {
            accounts: () => this.get('/v1/ledger/accounts'),
            entries: (params) => this.get('/v1/ledger/entries', params),
        };
        const baseURL = (typeof options === 'string' ? options : options.baseUrl) ?? exports.DEFAULT_BASE_URL;
        this.client = axios_1.default.create({
            baseURL,
            headers: {
                Authorization: `Bearer ${apiKey}`,
                'Content-Type': 'application/json',
            },
        });
    }
}
exports.Recurso = Recurso;

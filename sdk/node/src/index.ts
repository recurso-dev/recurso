import axios, { AxiosInstance } from 'axios';
import type { components, operations } from './schema';

/** Default API base URL, used when none is supplied at construction. */
export const DEFAULT_BASE_URL = 'http://localhost:8080';

/** Options accepted by the {@link Recurso} constructor. */
export interface RecursoOptions {
    /** API base URL. Defaults to {@link DEFAULT_BASE_URL} when omitted. */
    baseUrl?: string;
}

/** Common list-endpoint query parameters (all optional, server-side). */
export interface ListParams {
    page?: number;
    limit?: number;
    q?: string;
    status?: string;
    [key: string]: unknown;
}

/**
 * A JSON value returned by (or sent to) the API. The API speaks JSON, so any
 * payload or response is one of these shapes.
 */
export type JsonValue =
    | string
    | number
    | boolean
    | null
    | JsonValue[]
    | { [key: string]: JsonValue };

/** A JSON object body — the shape of every request payload and object response. */
export type JsonObject = { [key: string]: JsonValue };

/**
 * Generic API response. Used only as the fallback for endpoints the spec does
 * not model; typed endpoints resolve to their concrete resource shape instead.
 */
export type ApiResponse = JsonObject;

// --- Spec-derived types (generated from cmd/api/openapi.yaml into ./schema) ---
//
// Response and request shapes below are derived from the OpenAPI spec, the same
// source of truth the Python SDK is generated from. Regenerate `schema.d.ts`
// with `npm run generate` whenever the API changes so these can never drift.

/** Every resource/request model defined by the API, keyed by schema name. */
export type Schemas = components['schemas'];

// Ergonomic aliases for the resources these methods return, so callers can
// name them directly (e.g. `const s: Subscription = await ...`).
export type Customer = Schemas['Customer'];
export type Plan = Schemas['Plan'];
export type Price = Schemas['Price'];
export type Subscription = Schemas['Subscription'];
export type UnbilledCharge = Schemas['UnbilledCharge'];
export type SubscriptionUsage = Schemas['SubscriptionUsage'];
export type Invoice = Schemas['Invoice'];
export type Coupon = Schemas['Coupon'];
export type CreditNote = Schemas['CreditNote'];
export type Quote = Schemas['Quote'];
export type QuoteActionResponse = Schemas['QuoteActionResponse'];
export type WebhookEndpoint = Schemas['WebhookEndpoint'];
export type Event = Schemas['Event'];
export type EventDelivery = Schemas['EventDelivery'];
export type Mandate = Schemas['Mandate'];
export type Gift = Schemas['Gift'];
export type Referral = Schemas['Referral'];
export type MRRMetrics = Schemas['MRRMetrics'];
export type LedgerAccount = Schemas['LedgerAccount'];
export type LedgerTransaction = Schemas['LedgerTransaction'];
export type ChurnScoreResult = Schemas['ChurnScoreResult'];
export type Consent = Schemas['Consent'];
export type Tenant = Schemas['Tenant'];

/** The JSON body of an operation's success (2xx) response, per the spec. */
type SuccessJson<O> = O extends { responses: infer R }
    ? R extends { 200: { content: { 'application/json': infer B } } }
        ? B
        : R extends { 201: { content: { 'application/json': infer B } } }
          ? B
          : R extends { 202: { content: { 'application/json': infer B } } }
            ? B
            : ApiResponse
    : ApiResponse;

/** Response body type for a named operation. */
export type Res<K extends keyof operations> = SuccessJson<operations[K]>;

/** The JSON request body of an operation, if it defines one. */
type RequestJson<O> = O extends {
    requestBody?: { content: { 'application/json': infer B } };
}
    ? B
    : never;

/** Request body type for a named operation. */
export type Body<K extends keyof operations> = RequestJson<operations[K]>;

/** Payload for creating or updating a customer. */
export interface CustomerInput {
    email: string;
    name: string;
    line1?: string;
    city?: string;
    state?: string;
    zip?: string;
    country?: string;
    [key: string]: JsonValue | undefined;
}

/** Payload for creating a plan. */
export interface PlanInput {
    name: string;
    code: string;
    amount: number;
    currency: string;
    interval_unit: 'day' | 'week' | 'month' | 'year';
    interval_count?: number;
    [key: string]: JsonValue | undefined;
}

/** Payload for creating a subscription. */
export interface SubscriptionInput {
    customer_id: string;
    plan_id: string;
    coupon_code?: string;
    start_date?: string;
    payment_terms?: string;
    [key: string]: JsonValue | undefined;
}

/** Payload for creating a coupon. */
export interface CouponInput {
    code: string;
    discount_type: 'percent' | 'amount';
    discount_value: number;
    duration: 'forever' | 'once';
    [key: string]: JsonValue | undefined;
}

/** Payload for recording a metered usage event. */
export interface UsageEventInput {
    subscription_id: string;
    customer_id: string;
    dimension: string;
    quantity: number;
}

/** Query parameters for the time-windowed usage endpoint. */
export interface UsageQueryParams {
    subscription_id?: string;
    customer_id?: string;
    dimension?: string;
    from?: string;
    to?: string;
    granularity?: 'day' | 'month';
}

/** Payload for registering a webhook endpoint. */
export interface WebhookInput {
    url: string;
    event_types?: string[];
    [key: string]: JsonValue | undefined;
}

/** Query parameters for listing webhook deliveries. */
export interface WebhookDeliveriesParams {
    limit?: number;
    offset?: number;
    status?: 'pending' | 'succeeded' | 'failed';
}

/** Payload for redeeming a gift. */
export interface GiftRedeemInput {
    code: string;
    [key: string]: JsonValue | undefined;
}

/** A single entitlement in a plan's entitlement set. */
export interface Entitlement {
    feature_key: string;
    kind: 'boolean' | 'limit';
    bool_value?: boolean;
    limit_value?: number;
}

/** Query parameters for listing ledger entries. */
export interface LedgerEntriesParams {
    account_id?: string;
    [key: string]: JsonValue | undefined;
}

/**
 * Official Node.js SDK for the Recurso billing API.
 *
 * Method coverage mirrors the REST surface: list endpoints accept filter
 * params, mutations are grouped per resource, and lifecycle actions
 * (cancel, pause, resume, ...) live on their resource. Return types are
 * derived from the OpenAPI spec, so responses carry full field-level types.
 */
export class Recurso {
    private client: AxiosInstance;

    /**
     * @param apiKey  API key used for Bearer authentication.
     * @param options Client options. Pass `{ baseUrl }` to target a specific
     *                environment; a bare string is also accepted for backward
     *                compatibility. Defaults to {@link DEFAULT_BASE_URL}.
     */
    constructor(apiKey: string, options: string | RecursoOptions = {}) {
        const baseURL =
            (typeof options === 'string' ? options : options.baseUrl) ?? DEFAULT_BASE_URL;
        this.client = axios.create({
            baseURL,
            headers: {
                Authorization: `Bearer ${apiKey}`,
                'Content-Type': 'application/json',
            },
        });
    }

    private get = async <T = ApiResponse>(path: string, params?: object): Promise<T> =>
        (await this.client.get<T>(path, { params })).data;
    private post = async <T = ApiResponse>(path: string, data?: unknown): Promise<T> =>
        (await this.client.post<T>(path, data)).data;
    private put = async <T = ApiResponse>(path: string, data?: unknown): Promise<T> =>
        (await this.client.put<T>(path, data)).data;
    private del = async <T = ApiResponse>(path: string): Promise<T> =>
        (await this.client.delete<T>(path)).data;

    public account = {
        get: () => this.get<Res<'getAccount'>>('/v1/account'),
        update: (data: Body<'updateAccount'>) =>
            this.put<Res<'updateAccount'>>('/v1/account', data),
    };

    public customers = {
        create: (data: CustomerInput) =>
            this.post<Res<'createCustomer'>>('/v1/customers', { country: 'US', ...data }),
        list: (params?: ListParams) => this.get<Res<'listCustomers'>>('/v1/customers', params),
        updatePaymentMethod: (id: string, data: Body<'updateCustomerPaymentMethod'>) =>
            this.put<Res<'updateCustomerPaymentMethod'>>(
                `/v1/customers/${id}/payment-method`,
                data,
            ),
        churn: (id: string) => this.get<Res<'getCustomerChurn'>>(`/v1/customers/${id}/churn`),
        consents: (id: string) =>
            this.get<Res<'listCustomerConsents'>>(`/v1/customers/${id}/consents`),
    };

    public plans = {
        create: (data: PlanInput) =>
            this.post<Res<'createPlan'>>('/v1/plans', { interval_count: 1, ...data }),
        list: (params?: ListParams) => this.get<Res<'listPlans'>>('/v1/plans', params),
    };

    public subscriptions = {
        create: (data: SubscriptionInput) =>
            this.post<Res<'createSubscription'>>('/v1/subscriptions', data),
        list: (params?: ListParams) =>
            this.get<Res<'listSubscriptions'>>('/v1/subscriptions', params),
        update: (id: string, data: Body<'updateSubscription'>) =>
            this.put<Res<'updateSubscription'>>(`/v1/subscriptions/${id}`, data),
        cancel: (id: string, data?: Body<'cancelSubscription'>) =>
            this.post<Res<'cancelSubscription'>>(`/v1/subscriptions/${id}/cancel`, data),
        pause: (id: string, data?: Body<'pauseSubscription'>) =>
            this.post<Res<'pauseSubscription'>>(`/v1/subscriptions/${id}/pause`, data),
        resume: (id: string) =>
            this.post<Res<'resumeSubscription'>>(`/v1/subscriptions/${id}/resume`),
        reactivate: (id: string) =>
            this.post<Res<'reactivateSubscription'>>(`/v1/subscriptions/${id}/reactivate`),
        /** Bill N future periods immediately (advance invoicing). */
        advance: (id: string, data: Body<'generateAdvanceInvoice'>) =>
            this.post<Res<'generateAdvanceInvoice'>>(`/v1/subscriptions/${id}/advance`, data),
        charges: (id: string) =>
            this.get<Res<'listUnbilledCharges'>>(`/v1/subscriptions/${id}/charges`),
        addCharge: (id: string, data: Body<'addUnbilledCharge'>) =>
            this.post<Res<'addUnbilledCharge'>>(`/v1/subscriptions/${id}/charges`, data),
        /**
         * Current billing period's usage per dimension plus lifetime
         * totals, with the customer's entitlement limit/remaining joined
         * in where a feature_key matches the dimension name.
         */
        usage: (id: string) =>
            this.get<Res<'getSubscriptionUsage'>>(`/v1/subscriptions/${id}/usage`),
    };

    public invoices = {
        list: (params?: ListParams) => this.get<Res<'listInvoices'>>('/v1/invoices', params),
        /** Public PDF download URL for an invoice. */
        pdfUrl: (id: string) => `${this.client.defaults.baseURL}/v1/invoices/${id}/pdf`,
        eInvoiceStatus: (id: string) =>
            this.get<Res<'getEInvoiceStatus'>>(`/v1/invoices/${id}/einvoice`),
        retryEInvoice: (id: string) =>
            this.post<Res<'retryEInvoice'>>(`/v1/invoices/${id}/einvoice/retry`),
        cancelEInvoice: (id: string, data?: Body<'cancelEInvoice'>) =>
            this.post<Res<'cancelEInvoice'>>(`/v1/invoices/${id}/einvoice/cancel`, data),
    };

    public coupons = {
        create: (data: CouponInput) => this.post<Res<'createCoupon'>>('/v1/coupons', data),
        list: (params?: ListParams) => this.get<Res<'listCoupons'>>('/v1/coupons', params),
    };

    public usage = {
        /** Record a metered usage event against a subscription. */
        record: (data: UsageEventInput) =>
            this.post<Res<'recordUsageEvent'>>('/v1/usage/events', data),
        /**
         * Time-windowed usage buckets: {data: [{period, dimension,
         * quantity}], from, to, granularity}. At least one of
         * subscription_id or customer_id is required; the window defaults
         * to the last 30 days at day granularity.
         */
        query: (params: UsageQueryParams) => this.get<Res<'queryUsage'>>('/v1/usage', params),
        /** The tenant's dimension catalog with first/last seen and event counts. */
        dimensions: () => this.get<Res<'listUsageDimensions'>>('/v1/usage/dimensions'),
    };

    public creditNotes = {
        create: (data: Body<'createCreditNote'>) =>
            this.post<Res<'createCreditNote'>>('/v1/credit-notes', data),
        list: (params?: ListParams) =>
            this.get<Res<'listCreditNotes'>>('/v1/credit-notes', params),
    };

    public quotes = {
        create: (data: Body<'createQuote'>) => this.post<Res<'createQuote'>>('/v1/quotes', data),
        list: (params?: ListParams) => this.get<Res<'listQuotes'>>('/v1/quotes', params),
        get: (id: string) => this.get<Res<'getQuote'>>(`/v1/quotes/${id}`),
        update: (id: string, data: Body<'updateQuote'>) =>
            this.put<Res<'updateQuote'>>(`/v1/quotes/${id}`, data),
        send: (id: string) => this.post<Res<'sendQuote'>>(`/v1/quotes/${id}/send`),
        accept: (id: string) => this.post<Res<'acceptQuote'>>(`/v1/quotes/${id}/accept`),
        decline: (id: string) => this.post<Res<'declineQuote'>>(`/v1/quotes/${id}/decline`),
        /** Convert an accepted quote into a subscription. */
        convert: (id: string) =>
            this.post<Res<'convertQuoteToInvoice'>>(`/v1/quotes/${id}/convert`),
        delete: (id: string) => this.del<Res<'deleteQuote'>>(`/v1/quotes/${id}`),
    };

    public webhooks = {
        /** Register an endpoint to receive event deliveries. */
        create: (data: WebhookInput) =>
            this.post<Res<'createWebhookEndpoint'>>('/v1/webhooks', data),
        list: () => this.get<Res<'listWebhookEndpoints'>>('/v1/webhooks'),
        delete: (id: string) => this.del<Res<'deleteWebhookEndpoint'>>(`/v1/webhooks/${id}`),
        /**
         * Recent delivery attempts to an endpoint, newest first. Filter by
         * derived status (pending | succeeded | failed) and paginate with
         * limit/offset.
         */
        deliveries: (id: string, params?: WebhookDeliveriesParams) =>
            this.get<Res<'listWebhookEndpointDeliveries'>>(
                `/v1/webhooks/${id}/deliveries`,
                params,
            ),
    };

    public events = {
        list: (params?: ListParams) => this.get<Res<'listEvents'>>('/v1/events', params),
        types: () => this.get<Res<'listEventTypes'>>('/v1/events/types'),
        /** Delivery attempts of an event across all webhook endpoints. */
        deliveries: (id: string) =>
            this.get<Res<'listEventDeliveries'>>(`/v1/events/${id}/deliveries`),
        /**
         * Re-enqueue delivery of an event to every active subscribed
         * endpoint (202: {event_id, deliveries_queued}). Idempotent.
         */
        redeliver: (id: string) => this.post<Res<'redeliverEvent'>>(`/v1/events/${id}/redeliver`),
    };

    public mandates = {
        create: (data: Body<'createMandate'>) =>
            this.post<Res<'createMandate'>>('/v1/mandates', data),
        list: (params?: ListParams) => this.get<Res<'listMandates'>>('/v1/mandates', params),
        get: (id: string) => this.get<Res<'getMandate'>>(`/v1/mandates/${id}`),
        revoke: (id: string) => this.post<Res<'revokeMandate'>>(`/v1/mandates/${id}/revoke`),
    };

    public gifts = {
        purchase: (data: Body<'purchaseGift'>) =>
            this.post<Res<'purchaseGift'>>('/v1/gifts/purchase', data),
        redeem: (data: GiftRedeemInput) => this.post<Res<'redeemGift'>>('/v1/gifts/redeem', data),
        list: (params?: ListParams) => this.get<Res<'listGifts'>>('/v1/gifts', params),
    };

    public referrals = {
        create: (data: Body<'createReferral'>) =>
            this.post<Res<'createReferral'>>('/v1/referrals', data),
        list: (params?: ListParams) => this.get<Res<'listReferrals'>>('/v1/referrals', params),
        generateCode: (data: Body<'generateReferralCode'>) =>
            this.post<Res<'generateReferralCode'>>('/v1/referrals/generate-code', data),
        qualify: (id: string) =>
            this.post<Res<'qualifyReferral'>>(`/v1/referrals/${id}/qualify`),
    };

    public entitlements = {
        /**
         * Replace a plan's full entitlement set (PUT semantics: feature
         * keys absent from the list are removed).
         */
        setForPlan: (planId: string, list: Entitlement[]) =>
            this.put<Res<'setPlanEntitlements'>>(`/v1/plans/${planId}/entitlements`, list),
        getForPlan: (planId: string) =>
            this.get<Res<'getPlanEntitlements'>>(`/v1/plans/${planId}/entitlements`),
        /**
         * Effective entitlements for a customer: the union over the plans
         * of their active/trialing subscriptions (boolean: any-true wins;
         * limit: max across plans).
         */
        forCustomer: (customerId: string) =>
            this.get<Res<'getCustomerEntitlements'>>(`/v1/customers/${customerId}/entitlements`),
        /** Fast single-feature check: {feature_key, granted, limit_value}. */
        check: (customerId: string, feature: string) =>
            this.get<Res<'checkEntitlement'>>('/v1/entitlements/check', {
                customer_id: customerId,
                feature,
            }),
    };

    public analytics = {
        /**
         * Monthly recurring revenue, FX-normalized to the tenant's reporting
         * currency: {mrr, normalized_mrr, reporting_currency, breakdown[],
         * fx: {rates, source, as_of}}.
         */
        mrr: () => this.get<Res<'getMRR'>>('/v1/analytics/mrr'),
    };

    public ledger = {
        accounts: () => this.get<Res<'listLedgerAccounts'>>('/v1/ledger/accounts'),
        entries: (params?: LedgerEntriesParams) =>
            this.get<Res<'listLedgerEntries'>>('/v1/ledger/entries', params),
    };
}

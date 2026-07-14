import axios, { AxiosInstance } from 'axios';

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
 * Generic API response. Endpoints return either a single resource object or a
 * paginated envelope; both are JSON objects, so callers can narrow as needed.
 */
export type ApiResponse = JsonObject;

/** Paginated list envelope returned by `list` endpoints. */
export interface ListResponse<T = JsonObject> {
    data: T[];
    page?: number;
    limit?: number;
    total?: number;
    [key: string]: JsonValue | T[] | undefined;
}

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
 * (cancel, pause, resume, ...) live on their resource.
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

    private get = async (path: string, params?: object): Promise<ApiResponse> =>
        (await this.client.get<ApiResponse>(path, { params })).data;
    private post = async (path: string, data?: object): Promise<ApiResponse> =>
        (await this.client.post<ApiResponse>(path, data)).data;
    private put = async (path: string, data?: object): Promise<ApiResponse> =>
        (await this.client.put<ApiResponse>(path, data)).data;
    private del = async (path: string): Promise<ApiResponse> =>
        (await this.client.delete<ApiResponse>(path)).data;

    public account = {
        get: () => this.get('/v1/account'),
        update: (data: JsonObject) => this.put('/v1/account', data),
    };

    public customers = {
        create: (data: CustomerInput) =>
            this.post('/v1/customers', { country: 'US', ...data }),
        list: (params?: ListParams) => this.get('/v1/customers', params),
        updatePaymentMethod: (id: string, data: JsonObject) =>
            this.put(`/v1/customers/${id}/payment-method`, data),
        churn: (id: string) => this.get(`/v1/customers/${id}/churn`),
        consents: (id: string) => this.get(`/v1/customers/${id}/consents`),
    };

    public plans = {
        create: (data: PlanInput) =>
            this.post('/v1/plans', { interval_count: 1, ...data }),
        list: (params?: ListParams) => this.get('/v1/plans', params),
    };

    public subscriptions = {
        create: (data: SubscriptionInput) => this.post('/v1/subscriptions', data),
        list: (params?: ListParams) => this.get('/v1/subscriptions', params),
        update: (id: string, data: JsonObject) =>
            this.put(`/v1/subscriptions/${id}`, data),
        cancel: (id: string, data?: JsonObject) =>
            this.post(`/v1/subscriptions/${id}/cancel`, data),
        pause: (id: string, data?: JsonObject) =>
            this.post(`/v1/subscriptions/${id}/pause`, data),
        resume: (id: string) => this.post(`/v1/subscriptions/${id}/resume`),
        reactivate: (id: string) => this.post(`/v1/subscriptions/${id}/reactivate`),
        /** Bill N future periods immediately (advance invoicing). */
        advance: (id: string, data: { periods: number }) =>
            this.post(`/v1/subscriptions/${id}/advance`, data),
        charges: (id: string) => this.get(`/v1/subscriptions/${id}/charges`),
        addCharge: (id: string, data: JsonObject) =>
            this.post(`/v1/subscriptions/${id}/charges`, data),
        /**
         * Current billing period's usage per dimension plus lifetime
         * totals, with the customer's entitlement limit/remaining joined
         * in where a feature_key matches the dimension name.
         */
        usage: (id: string) => this.get(`/v1/subscriptions/${id}/usage`),
    };

    public invoices = {
        list: (params?: ListParams) => this.get('/v1/invoices', params),
        /** Public PDF download URL for an invoice. */
        pdfUrl: (id: string) =>
            `${this.client.defaults.baseURL}/v1/invoices/${id}/pdf`,
        eInvoiceStatus: (id: string) => this.get(`/v1/invoices/${id}/einvoice`),
        retryEInvoice: (id: string) => this.post(`/v1/invoices/${id}/einvoice/retry`),
        cancelEInvoice: (id: string, data?: JsonObject) =>
            this.post(`/v1/invoices/${id}/einvoice/cancel`, data),
    };

    public coupons = {
        create: (data: CouponInput) => this.post('/v1/coupons', data),
        list: (params?: ListParams) => this.get('/v1/coupons', params),
    };

    public usage = {
        /** Record a metered usage event against a subscription. */
        record: (data: UsageEventInput) => this.post('/v1/usage/events', data),
        /**
         * Time-windowed usage buckets: {data: [{period, dimension,
         * quantity}], from, to, granularity}. At least one of
         * subscription_id or customer_id is required; the window defaults
         * to the last 30 days at day granularity.
         */
        query: (params: UsageQueryParams) => this.get('/v1/usage', params),
        /** The tenant's dimension catalog with first/last seen and event counts. */
        dimensions: () => this.get('/v1/usage/dimensions'),
    };

    public creditNotes = {
        create: (data: JsonObject) => this.post('/v1/credit-notes', data),
        list: (params?: ListParams) => this.get('/v1/credit-notes', params),
    };

    public quotes = {
        create: (data: JsonObject) => this.post('/v1/quotes', data),
        list: (params?: ListParams) => this.get('/v1/quotes', params),
        get: (id: string) => this.get(`/v1/quotes/${id}`),
        update: (id: string, data: JsonObject) =>
            this.put(`/v1/quotes/${id}`, data),
        send: (id: string) => this.post(`/v1/quotes/${id}/send`),
        accept: (id: string) => this.post(`/v1/quotes/${id}/accept`),
        decline: (id: string) => this.post(`/v1/quotes/${id}/decline`),
        /** Convert an accepted quote into a subscription. */
        convert: (id: string) => this.post(`/v1/quotes/${id}/convert`),
        delete: (id: string) => this.del(`/v1/quotes/${id}`),
    };

    public webhooks = {
        /** Register an endpoint to receive event deliveries. */
        create: (data: WebhookInput) => this.post('/v1/webhooks', data),
        list: () => this.get('/v1/webhooks'),
        delete: (id: string) => this.del(`/v1/webhooks/${id}`),
        /**
         * Recent delivery attempts to an endpoint, newest first. Filter by
         * derived status (pending | succeeded | failed) and paginate with
         * limit/offset.
         */
        deliveries: (id: string, params?: WebhookDeliveriesParams) =>
            this.get(`/v1/webhooks/${id}/deliveries`, params),
    };

    public events = {
        list: (params?: ListParams) => this.get('/v1/events', params),
        types: () => this.get('/v1/events/types'),
        /** Delivery attempts of an event across all webhook endpoints. */
        deliveries: (id: string) => this.get(`/v1/events/${id}/deliveries`),
        /**
         * Re-enqueue delivery of an event to every active subscribed
         * endpoint (202: {event_id, deliveries_queued}). Idempotent.
         */
        redeliver: (id: string) => this.post(`/v1/events/${id}/redeliver`),
    };

    public mandates = {
        create: (data: JsonObject) => this.post('/v1/mandates', data),
        list: (params?: ListParams) => this.get('/v1/mandates', params),
        get: (id: string) => this.get(`/v1/mandates/${id}`),
        revoke: (id: string) => this.post(`/v1/mandates/${id}/revoke`),
    };

    public gifts = {
        purchase: (data: JsonObject) => this.post('/v1/gifts/purchase', data),
        redeem: (data: GiftRedeemInput) => this.post('/v1/gifts/redeem', data),
        list: (params?: ListParams) => this.get('/v1/gifts', params),
    };

    public referrals = {
        create: (data: JsonObject) => this.post('/v1/referrals', data),
        list: (params?: ListParams) => this.get('/v1/referrals', params),
        generateCode: (data: JsonObject) =>
            this.post('/v1/referrals/generate-code', data),
        qualify: (id: string) => this.post(`/v1/referrals/${id}/qualify`),
    };

    public entitlements = {
        /**
         * Replace a plan's full entitlement set (PUT semantics: feature
         * keys absent from the list are removed).
         */
        setForPlan: (planId: string, list: Entitlement[]) =>
            this.put(`/v1/plans/${planId}/entitlements`, list),
        getForPlan: (planId: string) => this.get(`/v1/plans/${planId}/entitlements`),
        /**
         * Effective entitlements for a customer: the union over the plans
         * of their active/trialing subscriptions (boolean: any-true wins;
         * limit: max across plans).
         */
        forCustomer: (customerId: string) =>
            this.get(`/v1/customers/${customerId}/entitlements`),
        /** Fast single-feature check: {feature_key, granted, limit_value}. */
        check: (customerId: string, feature: string) =>
            this.get('/v1/entitlements/check', { customer_id: customerId, feature }),
    };

    public analytics = {
        /**
         * Monthly recurring revenue, FX-normalized to the tenant's reporting
         * currency: {mrr, normalized_mrr, reporting_currency, breakdown[],
         * fx: {rates, source, as_of}}.
         */
        mrr: () => this.get('/v1/analytics/mrr'),
    };

    public ledger = {
        accounts: () => this.get('/v1/ledger/accounts'),
        entries: (params?: LedgerEntriesParams) =>
            this.get('/v1/ledger/entries', params),
    };
}
